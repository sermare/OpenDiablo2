package d2realm

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
)

// Chat types of the 0x26 packet. Normal is the d2gs value; the others are
// OUR assignments (unverified).
const (
	ChatNormal  uint8 = 1
	ChatWhisper uint8 = 2
	ChatSystem  uint8 = 4
)

// Config configures a Server. Only Store is required.
type Config struct {
	Store  Store
	Tables *d2s.ItemTables // optional; enables item checks
	// MaxClients and MaxGames bound the server (0 = defaults 256 and 128).
	MaxClients, MaxGames int
	// IdleTimeout drops a connection that sends nothing for this long (0 = never).
	IdleTimeout time.Duration
	// Rand supplies game seeds (nil = math/rand).
	Rand func() uint32
	// Logf receives log lines (nil = silent).
	Logf func(format string, args ...interface{})
	// ServerName is shown in the Hello reply.
	ServerName string
}

type game struct {
	info     GameInfo
	password string
	seed     uint32
	members  []*session
}

type session struct {
	srv     *Server
	conn    net.Conn
	out     chan []byte
	closed  bool
	asm     d2gs.TunnelAssembler
	account string
	hello   bool

	charName string
	char     *d2s.Character
	game     *game
	unitID   uint32
	act      byte
	level    uint16
	hasLevel bool
}

// Server is the realm.
type Server struct {
	cfg Config
	val Validator

	mu       sync.Mutex
	ln       net.Listener
	sessions map[*session]struct{}
	accounts map[string]*session // lower(account) -> session
	chars    map[string]*session // lower(character) -> session
	games    map[string]*game    // lower(name) -> game
	nextUnit uint32
	shut     bool
	wg       sync.WaitGroup
}

// New creates a server.
func New(cfg Config) *Server {
	if cfg.MaxClients == 0 {
		cfg.MaxClients = 256
	}

	if cfg.MaxGames == 0 {
		cfg.MaxGames = 128
	}

	if cfg.Store == nil { // a nil Store used to panic on the first character message
		cfg.Store = NewMemStore()
	}

	if cfg.Rand == nil {
		cfg.Rand = rand.Uint32
	}

	if cfg.Logf == nil {
		cfg.Logf = func(string, ...interface{}) {}
	}

	return &Server{
		cfg: cfg, val: Validator{Tables: cfg.Tables},
		sessions: map[*session]struct{}{}, accounts: map[string]*session{},
		chars: map[string]*session{}, games: map[string]*game{}, nextUnit: 1,
	}
}

// Listen starts serving on addr ("host:port") and returns once the listener is
// open; connections are handled in the background until Close.
func (s *Server) Listen(addr string) (net.Addr, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	s.wg.Add(1)

	go s.accept(ln)

	return ln.Addr(), nil
}

// Close stops listening, disconnects everybody and waits for the handlers.
func (s *Server) Close() {
	s.mu.Lock()
	s.shut = true
	ln := s.ln

	var conns []net.Conn
	for ss := range s.sessions {
		conns = append(conns, ss.conn)
	}
	s.mu.Unlock()

	if ln != nil {
		_ = ln.Close()
	}

	for _, c := range conns {
		_ = c.Close()
	}

	s.wg.Wait()
}

func (s *Server) accept(ln net.Listener) {
	defer s.wg.Done()

	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}

		ss := &session{srv: s, conn: c, out: make(chan []byte, 256)}

		s.mu.Lock()
		if s.shut {
			s.mu.Unlock()
			_ = c.Close()

			return
		}

		s.sessions[ss] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(2)

		go ss.writeLoop()
		go ss.readLoop()
	}
}

func (ss *session) writeLoop() {
	defer ss.srv.wg.Done()

	for b := range ss.out {
		if _, err := ss.conn.Write(b); err != nil {
			_ = ss.conn.Close()

			for range ss.out { // drain until closed
			}

			return
		}
	}
}

func (ss *session) readLoop() {
	defer ss.srv.wg.Done()
	defer ss.srv.drop(ss)

	r := bufio.NewReader(ss.conn)

	for {
		if t := ss.srv.cfg.IdleTimeout; t > 0 {
			_ = ss.conn.SetReadDeadline(time.Now().Add(t))
		}

		plain, err := d2gs.ReadBlob(r)
		if err != nil {
			return
		}

		pkts, ignored := d2gs.SplitClient(plain)
		if ignored > 0 {
			ss.srv.cfg.Logf("realm: %s: dropped %d bytes of unframable data", ss.name(), ignored)
		}

		for _, p := range pkts {
			ss.srv.mu.Lock()
			ss.srv.packet(ss, p)
			closed := ss.closed
			ss.srv.mu.Unlock()

			if closed {
				return
			}
		}
	}
}

func (ss *session) name() string {
	if ss.account != "" {
		return ss.account
	}

	return ss.conn.RemoteAddr().String()
}

// send frames packets and queues them; call with s.mu held. A client that
// cannot keep up is disconnected.
func (ss *session) send(pkts ...[]byte) {
	if ss.closed || len(pkts) == 0 {
		return
	}

	b, err := d2gs.EncodeStream(pkts...)
	if err != nil {
		ss.srv.cfg.Logf("realm: encode: %v", err)

		return
	}

	select {
	case ss.out <- b:
	default:
		ss.srv.cfg.Logf("realm: %s: send queue full, dropping client", ss.name())
		_ = ss.conn.Close()
	}
}

func (ss *session) sendMsg(msg interface{}) {
	typ, body := Encode(msg)
	ss.send(d2gs.Tunnel(d2gs.S2CMetaAE, typ, body)...)
}

func (ss *session) result(op byte, code Code, format string, args ...interface{}) {
	ss.sendMsg(Result{Op: op, Code: code, Message: fmt.Sprintf(format, args...)})
}

func (ss *session) system(format string, args ...interface{}) {
	ss.send(d2gs.ChatMessage{Type: ChatSystem, Text: fmt.Sprintf(format, args...)}.Marshal())
}

// drop removes a session completely.
func (s *Server) drop(ss *session) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ss.closed {
		return
	}

	if ss.game != nil {
		s.leaveGame(ss, false)
	} else if ss.hello {
		s.lobbyBroadcast(ss, Presence{Joined: false, Name: ss.account})
	}

	if ss.hello {
		delete(s.accounts, strings.ToLower(ss.account))
	}

	if ss.charName != "" && s.chars[strings.ToLower(ss.charName)] == ss {
		delete(s.chars, strings.ToLower(ss.charName))
	}

	delete(s.sessions, ss)
	ss.closed = true
	close(ss.out)
	_ = ss.conn.Close()
}

// packet dispatches one client packet; s.mu is held.
func (s *Server) packet(ss *session, p []byte) {
	switch p[0] {
	case d2gs.CtlTunnel:
		typ, data, done, err := ss.asm.Add(p)
		if err != nil {
			ss.result(0, CodeBadRequest, "%v", err)

			return
		}

		if ss.asm.Pending() > maxUpload {
			_ = ss.conn.Close()

			return
		}

		if done {
			s.message(ss, typ, data)
		}
	case d2gs.C2SChat:
		s.chat(ss, p)
	case d2gs.CtlLeaveGame:
		if ss.game == nil {
			ss.result(d2gs.CtlLeaveGame, CodeNotInGame, "not in a game")

			return
		}

		s.leaveGame(ss, true)
	default:
		// gameplay packets are not relayed by the lobby server
	}
}

func (s *Server) message(ss *session, typ byte, body []byte) {
	msg, err := Decode(typ, body)
	if err != nil {
		ss.result(typ, CodeBadRequest, "%v", err)

		return
	}

	if _, isHello := msg.(Hello); !isHello && !ss.hello {
		ss.result(typ, CodeNoHello, "send Hello first")

		return
	}

	switch m := msg.(type) {
	case Hello:
		s.doHello(ss, m)
	case ListGamesReq:
		ss.sendMsg(GameList{Games: s.gameList()})
	case CreateGame:
		s.doCreate(ss, m)
	case JoinGame:
		s.doJoin(ss, m)
	case UploadChar:
		s.doUpload(ss, m)
	case ListCharsReq:
		s.doListChars(ss)
	case SelectChar:
		s.doSelect(ss, m)
	case LevelChange:
		s.doLevel(ss, m)
	default:
		ss.result(typ, CodeBadRequest, "not a client message")
	}
}

// ---- lobby ----

func (s *Server) lobbyMembers() []*session {
	var out []*session

	for ss := range s.sessions {
		if ss.hello && ss.game == nil {
			out = append(out, ss)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].account < out[j].account })

	return out
}

func (s *Server) lobbyBroadcast(except *session, msg interface{}) {
	for _, m := range s.lobbyMembers() {
		if m != except {
			m.sendMsg(msg)
		}
	}
}

func (s *Server) doHello(ss *session, m Hello) {
	reply := func(code Code, msg string) { ss.sendMsg(HelloAck{Code: code, Message: msg}) }

	switch {
	case ss.hello:
		reply(CodeBadRequest, "already said hello")
	case m.Version != ProtocolVersion:
		reply(CodeBadRequest, fmt.Sprintf("protocol version %d not supported (server speaks %d)", m.Version, ProtocolVersion))
	case !ValidAccount(m.Account):
		reply(CodeBadRequest, "account names are 2 to 15 letters, digits, '-' or '_'")
	case len(s.accounts) >= s.cfg.MaxClients:
		reply(CodeServerFull, "server full")
	default:
		key := strings.ToLower(m.Account)
		if _, taken := s.accounts[key]; taken {
			reply(CodeNameTaken, "that account is already online")

			return
		}

		var roster []string
		for _, p := range s.lobbyMembers() {
			roster = append(roster, p.account)
		}

		ss.account, ss.hello = m.Account, true
		s.accounts[key] = ss
		s.lobbyBroadcast(ss, Presence{Joined: true, Name: ss.account})
		ss.sendMsg(HelloAck{Code: CodeOK, Message: s.cfg.ServerName, Roster: roster})
		s.cfg.Logf("realm: %s joined the lobby", ss.account)
	}
}

// chat handles client packet 0x15: message, then recipient (empty = all).
func (s *Server) chat(ss *session, p []byte) {
	if !ss.hello {
		ss.result(d2gs.C2SChat, CodeNoHello, "send Hello first")

		return
	}

	text, to, err := ParseChat(p)
	if err != nil || strings.TrimSpace(text) == "" {
		return
	}

	scope := s.lobbyMembers()
	from, unit := ss.account, uint32(0)

	if ss.game != nil {
		scope, from, unit = ss.game.members, ss.charName, ss.unitID
	}

	name := func(m *session) string {
		if ss.game != nil {
			return m.charName
		}

		return m.account
	}

	if to == "" {
		pkt := d2gs.ChatMessage{Type: ChatNormal, UnitID: unit, Name: from, Text: text}.Marshal()
		for _, m := range scope {
			m.send(pkt)
		}

		return
	}

	for _, m := range scope {
		if strings.EqualFold(name(m), to) {
			m.send(d2gs.ChatMessage{Type: ChatWhisper, UnitID: unit, Name: from, Text: text}.Marshal())

			return
		}
	}

	ss.system("%s is not here.", to)
}

// ParseChat extracts message and recipient from a client 0x15 packet.
func ParseChat(p []byte) (text, to string, err error) {
	if len(p) < 5 || p[0] != d2gs.C2SChat {
		return "", "", d2gs.ErrShort
	}

	end := indexNUL(p, 3)
	if end < 0 {
		return "", "", d2gs.ErrBadLength
	}

	toEnd := indexNUL(p, end+1)
	if toEnd < 0 {
		return "", "", d2gs.ErrBadLength
	}

	return cleanText(string(p[3:end])), cleanText(string(p[end+1 : toEnd])), nil
}

// ClientChat builds a client chat packet with a recipient (empty = everyone).
func ClientChat(text, to string) []byte {
	if len(text) > 0xff {
		text = text[:0xff]
	}

	b := []byte{d2gs.C2SChat, ChatNormal, 0}
	b = append(b, text...)
	b = append(b, 0)
	b = append(b, to...)

	return append(b, 0)
}

func indexNUL(b []byte, from int) int {
	for i := from; i < len(b); i++ {
		if b[i] == 0 {
			return i
		}
	}

	return -1
}

func cleanText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}

		return r
	}, s)
}

// ---- characters ----

func (s *Server) doUpload(ss *session, m UploadChar) {
	const op = MsgUploadChar

	c, err := s.val.Validate(m.Data)
	if err != nil {
		ss.result(op, CodeCharInvalid, "%v", err)
		s.cfg.Logf("realm: %s: rejected upload: %v", ss.account, err)

		return
	}

	name := c.Header.Name
	if ss.game != nil && !strings.EqualFold(name, ss.charName) {
		ss.result(op, CodeAlreadyInGame, "cannot switch character inside a game")

		return
	}

	if o := s.chars[strings.ToLower(name)]; o != nil && o != ss {
		ss.result(op, CodeNameTaken, "character %s is in use", name)

		return
	}

	if old, err := s.cfg.Store.Load(ss.account, name); err == nil {
		if oc, err := s.val.Validate(old); err == nil {
			const modeBits = d2s.StatusHardcore | d2s.StatusExpansion
			if oc.Header.Class != c.Header.Class || oc.Header.Status&modeBits != c.Header.Status&modeBits {
				ss.result(op, CodeCharInvalid, "class or hardcore/expansion flags differ from the stored character")

				return
			}
		}
	}

	if err := s.cfg.Store.Save(ss.account, name, m.Data); err != nil {
		code := CodeInternal
		if errors.Is(err, ErrOwnedElsewhere) {
			code = CodeNameTaken
		}

		ss.result(op, code, "%v", err)

		return
	}

	s.setChar(ss, c)
	ss.result(op, CodeOK, "saved %s", name)
}

func (s *Server) setChar(ss *session, c *d2s.Character) {
	if ss.charName != "" && s.chars[strings.ToLower(ss.charName)] == ss {
		delete(s.chars, strings.ToLower(ss.charName))
	}

	ss.char, ss.charName = c, c.Header.Name
	s.chars[strings.ToLower(ss.charName)] = ss
}

func (s *Server) doListChars(ss *session) {
	names, err := s.cfg.Store.List(ss.account)
	if err != nil {
		ss.result(MsgListChars, CodeInternal, "%v", err)

		return
	}

	var list CharList

	for _, n := range names {
		data, err := s.cfg.Store.Load(ss.account, n)
		if err != nil {
			continue
		}

		if h, err := d2s.ParseHeader(data); err == nil {
			list.Chars = append(list.Chars, CharInfo{Name: h.Name, Class: byte(h.Class), Level: h.Level})
		}
	}

	ss.sendMsg(list)
}

func (s *Server) doSelect(ss *session, m SelectChar) {
	const op = MsgSelectChar

	if ss.game != nil {
		ss.result(op, CodeAlreadyInGame, "leave the game first")

		return
	}

	data, err := s.cfg.Store.Load(ss.account, m.Name)
	if err != nil {
		ss.result(op, CodeNoCharacter, "no stored character %q", m.Name)

		return
	}

	c, err := s.val.Validate(data)
	if err != nil {
		ss.result(op, CodeCharInvalid, "%v", err)

		return
	}

	if o := s.chars[strings.ToLower(c.Header.Name)]; o != nil && o != ss {
		ss.result(op, CodeNameTaken, "character %s is in use", c.Header.Name)

		return
	}

	s.setChar(ss, c)
	ss.sendMsg(CharData{Name: c.Header.Name, Data: data})
	ss.result(op, CodeOK, "selected %s", c.Header.Name)
}

// ---- games ----

func (s *Server) gameList() []GameInfo {
	out := make([]GameInfo, 0, len(s.games))

	for _, g := range s.games {
		gi := g.info
		gi.Players = byte(len(g.members))
		out = append(out, gi)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out
}

// ValidGameName reports whether name is acceptable: 1 to 15 printable ASCII
// characters, no leading or trailing space.
func ValidGameName(name string) bool {
	if len(name) < 1 || len(name) > MaxGameName || name != strings.TrimSpace(name) {
		return false
	}

	for i := 0; i < len(name); i++ {
		if name[i] < 0x20 || name[i] > 0x7e {
			return false
		}
	}

	return true
}

// canPlay checks the character-versus-game rules shared by create and join.
func canPlay(c *d2s.Character, gi GameInfo) (Code, string) {
	h := c.Header
	lvl := h.Level

	switch {
	case h.IsHardcore() && h.IsDead():
		return CodeCharInvalid, "dead hardcore character"
	case h.IsHardcore() != gi.Hardcore:
		return CodeModeMismatch, "hardcore and softcore characters cannot play together"
	case h.IsExpansion() != gi.Expansion:
		return CodeModeMismatch, "classic and expansion characters cannot play together"
	case !DifficultyAllowed(h, gi.Difficulty):
		return CodeDifficultyLocked, "your character has not unlocked that difficulty"
	case gi.MinLevel != 0 && lvl < gi.MinLevel:
		return CodeLevelTooLow, fmt.Sprintf("level %d or higher required", gi.MinLevel)
	case gi.MaxLevel != 0 && lvl > gi.MaxLevel:
		return CodeLevelTooHigh, fmt.Sprintf("level %d or lower required", gi.MaxLevel)
	}

	return CodeOK, ""
}

func (s *Server) doCreate(ss *session, m CreateGame) {
	const op = MsgCreateGame

	if ss.char == nil {
		ss.result(op, CodeNoCharacter, "upload or select a character first")

		return
	}

	if ss.game != nil {
		ss.result(op, CodeAlreadyInGame, "already in a game")

		return
	}

	if m.MaxPlayers == 0 {
		m.MaxPlayers = MaxPlayers
	}

	switch {
	case !ValidGameName(m.Name), len(m.Password) > MaxPassword, len(m.Description) > MaxDescription,
		m.Difficulty > Hell, m.MaxPlayers < MinPlayers, m.MaxPlayers > MaxPlayers,
		m.MaxLevel > 99, m.MaxLevel != 0 && m.MaxLevel < m.MinLevel,
		cleanText(m.Description) != m.Description, cleanText(m.Password) != m.Password:
		ss.result(op, CodeBadRequest, "invalid game fields")

		return
	}

	key := strings.ToLower(m.Name)
	if _, ok := s.games[key]; ok {
		ss.result(op, CodeGameExists, "a game with that name exists")

		return
	}

	if len(s.games) >= s.cfg.MaxGames {
		ss.result(op, CodeServerFull, "too many games")

		return
	}

	h := ss.char.Header
	g := &game{
		password: m.Password,
		seed:     s.cfg.Rand(),
		info: GameInfo{Name: m.Name, Description: m.Description, Creator: ss.charName,
			Difficulty: m.Difficulty, MaxPlayers: m.MaxPlayers, MinLevel: m.MinLevel, MaxLevel: m.MaxLevel,
			Hardcore: h.IsHardcore(), Expansion: h.IsExpansion(), HasPassword: m.Password != ""},
	}

	if code, msg := canPlay(ss.char, g.info); code != CodeOK {
		ss.result(op, code, "%s", msg)

		return
	}

	s.games[key] = g
	s.enter(ss, g)
	s.cfg.Logf("realm: %s created game %q (seed %#x)", ss.charName, g.info.Name, g.seed)
}

func (s *Server) doJoin(ss *session, m JoinGame) {
	const op = MsgJoinGame

	if ss.char == nil {
		ss.result(op, CodeNoCharacter, "upload or select a character first")

		return
	}

	if ss.game != nil {
		ss.result(op, CodeAlreadyInGame, "already in a game")

		return
	}

	g := s.games[strings.ToLower(m.Name)]

	switch {
	case g == nil:
		ss.result(op, CodeGameNotFound, "no such game")

		return
	case g.password != m.Password:
		ss.result(op, CodeBadPassword, "wrong password")

		return
	case len(g.members) >= int(g.info.MaxPlayers):
		ss.result(op, CodeGameFull, "game is full")

		return
	}

	if code, msg := canPlay(ss.char, g.info); code != CodeOK {
		ss.result(op, code, "%s", msg)

		return
	}

	s.enter(ss, g)
}

// enter puts ss into g (drop-in): the joiner learns the game and the players,
// the players learn the joiner.
func (s *Server) enter(ss *session, g *game) {
	// leave the lobby
	s.lobbyBroadcast(ss, Presence{Joined: false, Name: ss.account})

	ss.unitID, s.nextUnit = s.nextUnit, s.nextUnit+1
	ss.game = g
	ss.hasLevel = false

	_, act, _ := ss.char.Header.ActiveDifficulty()
	ss.act = byte(act)

	gi := g.info
	gi.Players = byte(len(g.members) + 1)

	ss.sendMsg(GameJoined{Game: gi, UnitID: ss.unitID, Seed: g.seed})
	ss.send(
		d2gs.GameFlags{Difficulty: g.info.Difficulty, Hardcore: g.info.Hardcore, Expansion: g.info.Expansion}.Marshal(),
		d2gs.LoadAct{Act: ss.act, Seed: g.seed}.Marshal(),
	)

	self := d2gs.AssignPlayer{UnitID: ss.unitID, Class: byte(ss.char.Header.Class),
		Name: ss.charName}.Marshal()

	for _, m := range g.members {
		ss.send(d2gs.AssignPlayer{UnitID: m.unitID, Class: byte(m.char.Header.Class),
			Name: m.charName}.Marshal())

		if m.hasLevel {
			ss.sendMsg(PlayerLevel{UnitID: m.unitID, Act: m.act, Level: m.level})
		}

		m.send(self)
		m.system("%s joined the game.", ss.charName)
	}

	g.members = append(g.members, ss)
}

// leaveGame removes ss from its game (drop-out). back reports whether the
// session stays connected and returns to the lobby.
func (s *Server) leaveGame(ss *session, back bool) {
	g := ss.game
	if g == nil {
		return
	}

	for i, m := range g.members {
		if m == ss {
			g.members = append(g.members[:i], g.members[i+1:]...)

			break
		}
	}

	ss.game = nil

	for _, m := range g.members {
		m.send(d2gs.PlayerLeave{UnitID: ss.unitID}.Marshal())
		m.system("%s left the game.", ss.charName)
	}

	if len(g.members) == 0 {
		delete(s.games, strings.ToLower(g.info.Name))
		s.cfg.Logf("realm: game %q closed", g.info.Name)
	}

	if !back {
		return
	}

	ss.result(d2gs.CtlLeaveGame, CodeOK, "left %s", g.info.Name)

	for _, m := range s.lobbyMembers() {
		ss.sendMsg(Presence{Joined: true, Name: m.account}) // roster for the returner
	}

	s.lobbyBroadcast(ss, Presence{Joined: true, Name: ss.account})
}

func (s *Server) doLevel(ss *session, m LevelChange) {
	if ss.game == nil {
		ss.result(MsgLevelChange, CodeNotInGame, "not in a game")

		return
	}

	if m.Act > 4 {
		ss.result(MsgLevelChange, CodeBadRequest, "bad act")

		return
	}

	ss.act, ss.level, ss.hasLevel = m.Act, m.Level, true

	for _, o := range ss.game.members {
		if o != ss {
			o.sendMsg(PlayerLevel{UnitID: ss.unitID, Act: m.Act, Level: m.Level})
		}
	}
}
