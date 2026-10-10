package d2realm

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ProtocolVersion is the version of the realm extension messages (OUR
// extension, not part of any original protocol).
const ProtocolVersion = 1

// Client -> server tunnel message types (carried in d2gs 0x6c).
const (
	MsgHello       byte = 0x80
	MsgListGames   byte = 0x81
	MsgCreateGame  byte = 0x82
	MsgJoinGame    byte = 0x83
	MsgUploadChar  byte = 0x84
	MsgListChars   byte = 0x85
	MsgSelectChar  byte = 0x86
	MsgLevelChange byte = 0x87 // level change inside a game
)

// Server -> client tunnel message types (carried in d2gs 0xAE).
const (
	MsgHelloAck    byte = 0x90
	MsgResult      byte = 0x91
	MsgGameList    byte = 0x92
	MsgCharList    byte = 0x93
	MsgCharData    byte = 0x94
	MsgPresence    byte = 0x95
	MsgPlayerLevel byte = 0x96
	MsgGameJoined  byte = 0x97
)

// Code is the outcome of a request.
type Code byte

// Result codes.
const (
	CodeOK Code = iota
	CodeBadRequest
	CodeNameTaken
	CodeNoHello
	CodeNoCharacter
	CodeGameNotFound
	CodeGameExists
	CodeGameFull
	CodeBadPassword
	CodeDifficultyLocked
	CodeLevelTooLow
	CodeLevelTooHigh
	CodeCharInvalid
	CodeAlreadyInGame
	CodeNotInGame
	CodeServerFull
	CodeModeMismatch // hardcore / expansion differ from the game
	CodeInternal
)

var codeNames = [...]string{"ok", "bad request", "name taken", "say hello first", "no character selected",
	"game not found", "game exists", "game full", "bad password", "difficulty locked",
	"level too low", "level too high", "character rejected", "already in a game", "not in a game",
	"server full", "hardcore/expansion mismatch", "internal error"}

func (c Code) String() string {
	if int(c) < len(codeNames) {
		return codeNames[c]
	}

	return fmt.Sprintf("code %d", byte(c))
}

// Limits of the game screens (unverified for the original; they are the
// limits this realm enforces).
const (
	MaxGameName    = 15
	MaxPassword    = 15
	MaxDescription = 31
	MinPlayers     = 1
	MaxPlayers     = 8
	MaxAccountName = 15
	maxUpload      = 1 << 16
)

// Difficulties.
const (
	Normal byte = iota
	Nightmare
	Hell
)

// GameInfo is one row of the game list.
type GameInfo struct {
	Name        string
	Description string
	Creator     string
	Difficulty  byte
	Players     byte
	MaxPlayers  byte
	MinLevel    byte // 0 = no limit
	MaxLevel    byte // 0 = no limit
	Hardcore    bool
	Expansion   bool
	HasPassword bool
}

// CharInfo is one row of the stored character list.
type CharInfo struct {
	Name  string
	Class byte
	Level byte
}

// ErrMalformed is returned for truncated or oversized messages.
var ErrMalformed = errors.New("d2realm: malformed message")

type wr struct{ b []byte }

func (w *wr) u8(v byte) { w.b = append(w.b, v) }
func (w *wr) u16(v uint16) {
	var t [2]byte
	binary.LittleEndian.PutUint16(t[:], v)
	w.b = append(w.b, t[:]...)
}

func (w *wr) u32(v uint32) {
	var t [4]byte
	binary.LittleEndian.PutUint32(t[:], v)
	w.b = append(w.b, t[:]...)
}

func (w *wr) bool(v bool) {
	if v {
		w.u8(1)
	} else {
		w.u8(0)
	}
}

func (w *wr) str(s string) {
	if len(s) > 255 {
		s = s[:255]
	}

	w.u8(byte(len(s)))
	w.b = append(w.b, s...)
}

func (w *wr) bytes(p []byte) {
	w.u32(uint32(len(p)))
	w.b = append(w.b, p...)
}

type rd struct {
	b   []byte
	err error
}

func (r *rd) take(n int) []byte {
	if r.err != nil || n < 0 || n > len(r.b) {
		r.err = ErrMalformed

		return make([]byte, 4) // zeros keep callers safe; err is sticky
	}

	p := r.b[:n]
	r.b = r.b[n:]

	return p
}

func (r *rd) u8() byte    { return r.take(1)[0] }
func (r *rd) u16() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }
func (r *rd) u32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }
func (r *rd) bool() bool  { return r.u8() != 0 }
func (r *rd) str() string { return string(r.take(int(r.u8()))) }

func (r *rd) bytes() []byte {
	n := binary.LittleEndian.Uint32(r.take(4))
	if n > maxUpload {
		r.err = ErrMalformed

		return nil
	}

	return append([]byte(nil), r.take(int(n))...)
}

func (r *rd) done() error {
	if r.err == nil && len(r.b) != 0 {
		r.err = ErrMalformed
	}

	return r.err
}

// ---- client -> server ----

// Hello opens a session. Account is the lobby name and owns the saves.
type Hello struct {
	Version byte
	Account string
}

// CreateGame carries the fields of the original TCP/IP "create game" screen
// plus level restrictions.
type CreateGame struct {
	Name, Password, Description string
	Difficulty                  byte
	MaxPlayers                  byte
	MinLevel, MaxLevel          byte
}

// JoinGame names a game and gives its password.
type JoinGame struct{ Name, Password string }

// LevelChange tells the server the player entered a level.
type LevelChange struct {
	Act   byte
	Level uint16
}

// ListGamesReq asks for the game list.
type ListGamesReq struct{}

// ListCharsReq asks for the account's stored characters.
type ListCharsReq struct{}

// UploadChar carries a .d2s for validation and storage.
type UploadChar struct{ Data []byte }

// SelectChar loads a stored character as the active one.
type SelectChar struct{ Name string }

// ---- server -> client ----

// HelloAck answers Hello. Roster lists the players in the lobby.
type HelloAck struct {
	Code    Code
	Message string
	Roster  []string
}

// Result reports the outcome of a request (Op is the request type).
type Result struct {
	Op      byte
	Code    Code
	Message string
}

// GameJoined confirms a create/join: the player's unit id and the game seed.
type GameJoined struct {
	Game   GameInfo
	UnitID uint32
	Seed   uint32
}

// Presence announces a lobby arrival or departure.
type Presence struct {
	Joined bool
	Name   string
}

// PlayerLevel tells a game's members where a player is.
type PlayerLevel struct {
	UnitID uint32
	Act    byte
	Level  uint16
}

// GameList is the game list.
type GameList struct{ Games []GameInfo }

// CharList is the stored character list.
type CharList struct{ Chars []CharInfo }

// CharData is a stored character's file.
type CharData struct {
	Name string
	Data []byte
}

func putGame(w *wr, g GameInfo) {
	w.str(g.Name)
	w.str(g.Description)
	w.str(g.Creator)
	w.u8(g.Difficulty)
	w.u8(g.Players)
	w.u8(g.MaxPlayers)
	w.u8(g.MinLevel)
	w.u8(g.MaxLevel)
	w.bool(g.Hardcore)
	w.bool(g.Expansion)
	w.bool(g.HasPassword)
}

func getGame(r *rd) GameInfo {
	var g GameInfo

	g.Name, g.Description, g.Creator = r.str(), r.str(), r.str()
	g.Difficulty, g.Players, g.MaxPlayers, g.MinLevel, g.MaxLevel = r.u8(), r.u8(), r.u8(), r.u8(), r.u8()
	g.Hardcore, g.Expansion, g.HasPassword = r.bool(), r.bool(), r.bool()

	return g
}

// Encode serialises a message and returns its tunnel type and body.
func Encode(msg interface{}) (typ byte, body []byte) {
	w := &wr{}

	switch m := msg.(type) {
	case Hello:
		typ = MsgHello
		w.u8(m.Version)
		w.str(m.Account)
	case ListGamesReq:
		typ = MsgListGames
	case CreateGame:
		typ = MsgCreateGame
		w.str(m.Name)
		w.str(m.Password)
		w.str(m.Description)
		w.u8(m.Difficulty)
		w.u8(m.MaxPlayers)
		w.u8(m.MinLevel)
		w.u8(m.MaxLevel)
	case JoinGame:
		typ = MsgJoinGame
		w.str(m.Name)
		w.str(m.Password)
	case UploadChar:
		typ = MsgUploadChar
		w.bytes(m.Data)
	case ListCharsReq:
		typ = MsgListChars
	case SelectChar:
		typ = MsgSelectChar
		w.str(m.Name)
	case LevelChange:
		typ = MsgLevelChange
		w.u8(m.Act)
		w.u16(m.Level)
	case HelloAck:
		typ = MsgHelloAck
		w.u8(byte(m.Code))
		w.str(m.Message)
		w.u16(uint16(len(m.Roster)))

		for _, n := range m.Roster {
			w.str(n)
		}
	case Result:
		typ = MsgResult
		w.u8(m.Op)
		w.u8(byte(m.Code))
		w.str(m.Message)
	case GameList:
		typ = MsgGameList
		w.u16(uint16(len(m.Games)))

		for _, g := range m.Games {
			putGame(w, g)
		}
	case CharList:
		typ = MsgCharList
		w.u16(uint16(len(m.Chars)))

		for _, c := range m.Chars {
			w.str(c.Name)
			w.u8(c.Class)
			w.u8(c.Level)
		}
	case CharData:
		typ = MsgCharData
		w.str(m.Name)
		w.bytes(m.Data)
	case Presence:
		typ = MsgPresence
		w.bool(m.Joined)
		w.str(m.Name)
	case PlayerLevel:
		typ = MsgPlayerLevel
		w.u32(m.UnitID)
		w.u8(m.Act)
		w.u16(m.Level)
	case GameJoined:
		typ = MsgGameJoined
		putGame(w, m.Game)
		w.u32(m.UnitID)
		w.u32(m.Seed)
	default:
		panic(fmt.Sprintf("d2realm: cannot encode %T", msg))
	}

	return typ, w.b
}

// Decode parses a message of the given tunnel type.
func Decode(typ byte, body []byte) (interface{}, error) {
	r := &rd{b: body}

	var msg interface{}

	switch typ {
	case MsgHello:
		msg = Hello{Version: r.u8(), Account: r.str()}
	case MsgListGames:
		msg = ListGamesReq{}
	case MsgCreateGame:
		msg = CreateGame{Name: r.str(), Password: r.str(), Description: r.str(),
			Difficulty: r.u8(), MaxPlayers: r.u8(), MinLevel: r.u8(), MaxLevel: r.u8()}
	case MsgJoinGame:
		msg = JoinGame{Name: r.str(), Password: r.str()}
	case MsgUploadChar:
		msg = UploadChar{Data: r.bytes()}
	case MsgListChars:
		msg = ListCharsReq{}
	case MsgSelectChar:
		msg = SelectChar{Name: r.str()}
	case MsgLevelChange:
		msg = LevelChange{Act: r.u8(), Level: r.u16()}
	case MsgHelloAck:
		m := HelloAck{Code: Code(r.u8()), Message: r.str()}

		for n := int(r.u16()); n > 0 && r.err == nil; n-- {
			m.Roster = append(m.Roster, r.str())
		}

		msg = m
	case MsgResult:
		msg = Result{Op: r.u8(), Code: Code(r.u8()), Message: r.str()}
	case MsgGameList:
		var m GameList

		for n := int(r.u16()); n > 0 && r.err == nil; n-- {
			m.Games = append(m.Games, getGame(r))
		}

		msg = m
	case MsgCharList:
		var m CharList

		for n := int(r.u16()); n > 0 && r.err == nil; n-- {
			m.Chars = append(m.Chars, CharInfo{Name: r.str(), Class: r.u8(), Level: r.u8()})
		}

		msg = m
	case MsgCharData:
		msg = CharData{Name: r.str(), Data: r.bytes()}
	case MsgPresence:
		msg = Presence{Joined: r.bool(), Name: r.str()}
	case MsgPlayerLevel:
		msg = PlayerLevel{UnitID: r.u32(), Act: r.u8(), Level: r.u16()}
	case MsgGameJoined:
		msg = GameJoined{Game: getGame(r), UnitID: r.u32(), Seed: r.u32()}
	default:
		return nil, fmt.Errorf("%w: unknown type %#x", ErrMalformed, typ)
	}

	if err := r.done(); err != nil {
		return nil, err
	}

	return msg, nil
}

// LevelSeed derives the seed of one level from the game seed, so every
// client of a game generates the same level (OUR extension).
func LevelSeed(gameSeed uint32, level uint16) uint32 {
	h := gameSeed ^ (uint32(level)+1)*0x9E3779B1
	h ^= h >> 16
	h *= 0x85EBCA6B
	h ^= h >> 13
	h *= 0xC2B2AE35
	h ^= h >> 16

	return h
}
