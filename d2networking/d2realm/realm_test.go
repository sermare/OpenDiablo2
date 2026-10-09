package d2realm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

const wait = 3 * time.Second

func newServer(t *testing.T, cfg Config) (*Server, string) {
	t.Helper()

	if cfg.Store == nil {
		cfg.Store = NewMemStore()
	}

	if cfg.Rand == nil {
		n := uint32(0x1000)
		cfg.Rand = func() uint32 { n += 0x111; return n }
	}

	s := New(cfg)

	addr, err := s.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(s.Close)

	return s, addr.String()
}

func dial(t *testing.T, addr, account string) *Client {
	t.Helper()

	c, err := Dial(addr)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = c.Close() })

	if _, err := c.Hello(account); err != nil {
		t.Fatalf("hello %s: %v", account, err)
	}

	return c
}

func newChar(t *testing.T, name string, class d2s.Class, flags d2s.NewCharacterFlags) []byte {
	t.Helper()

	b, err := d2s.NewCharacter(name, class, flags, d2s.DefaultAppearance(class))
	if err != nil {
		t.Fatal(err)
	}

	return b
}

// player logs in and uploads a fresh expansion softcore character.
func player(t *testing.T, addr, account, char string) *Client {
	t.Helper()

	c := dial(t, addr, account)
	if err := c.Upload(newChar(t, char, d2s.Sorceress, d2s.NewCharacterFlags{Expansion: true})); err != nil {
		t.Fatalf("upload %s: %v", char, err)
	}

	return c
}

func waitFor(t *testing.T, c *Client, what string, match func(interface{}) bool) interface{} {
	t.Helper()

	ev, err := c.Wait(wait, match)
	if err != nil {
		t.Fatalf("waiting for %s: %v (queued: %#v)", what, err, c.Pending())
	}

	return ev
}

func isChat(text string) func(interface{}) bool {
	return func(e interface{}) bool { m, ok := e.(Chat); return ok && m.Text == text }
}

func noEvent(t *testing.T, c *Client, what string, match func(interface{}) bool) {
	t.Helper()

	// a round trip to the server orders everything sent before it
	if _, err := c.Games(); err != nil {
		t.Fatal(err)
	}

	for _, e := range c.Pending() {
		if match(e) {
			t.Fatalf("unexpected %s: %#v", what, e)
		}
	}
}

func wantCode(t *testing.T, err error, want Code) {
	t.Helper()

	var re *RequestError
	if !errors.As(err, &re) || re.Code != want {
		t.Fatalf("got error %v, want code %s", err, want)
	}
}

func TestProtocolRoundTrip(t *testing.T) {
	game := GameInfo{Name: "g", Description: "d", Creator: "c", Difficulty: 2, Players: 3, MaxPlayers: 8,
		MinLevel: 5, MaxLevel: 90, Hardcore: true, Expansion: true, HasPassword: true}
	msgs := []interface{}{
		Hello{Version: 1, Account: "bob"}, ListGamesReq{}, ListCharsReq{},
		CreateGame{Name: "n", Password: "p", Description: "d", Difficulty: 1, MaxPlayers: 4, MinLevel: 2, MaxLevel: 9},
		JoinGame{Name: "n", Password: "p"}, UploadChar{Data: []byte{1, 2, 3}}, SelectChar{Name: "x"},
		LevelChange{Act: 2, Level: 0x1234},
		HelloAck{Code: CodeOK, Message: "hi", Roster: []string{"a", "b"}},
		Result{Op: MsgJoinGame, Code: CodeGameFull, Message: "full"},
		GameList{Games: []GameInfo{game, game}}, CharList{Chars: []CharInfo{{"a", 1, 2}}},
		CharData{Name: "a", Data: []byte{9, 8}}, Presence{Joined: true, Name: "z"},
		PlayerLevel{UnitID: 7, Act: 3, Level: 99}, GameJoined{Game: game, UnitID: 5, Seed: 0xdeadbeef},
	}

	for _, m := range msgs {
		typ, body := Encode(m)

		got, err := Decode(typ, body)
		if err != nil {
			t.Fatalf("%T: %v", m, err)
		}

		if !reflect.DeepEqual(got, m) {
			t.Errorf("%T: got %#v want %#v", m, got, m)
		}

		if len(body) > 0 { // truncation and trailing bytes are errors
			if _, err := Decode(typ, body[:len(body)-1]); err == nil {
				t.Errorf("%T: truncated body accepted", m)
			}
		}

		if _, err := Decode(typ, append(append([]byte(nil), body...), 0)); err == nil {
			t.Errorf("%T: trailing byte accepted", m)
		}
	}

	if _, err := Decode(0x01, nil); err == nil {
		t.Error("unknown type accepted")
	}

	// an upload that claims more than the limit
	huge := binary.LittleEndian.AppendUint32(nil, maxUpload+1)
	if _, err := Decode(MsgUploadChar, huge); err == nil {
		t.Error("oversized upload accepted")
	}
}

func TestLevelSeed(t *testing.T) {
	seen := map[uint32]bool{}

	for lvl := uint16(0); lvl < 200; lvl++ {
		s := LevelSeed(0xabcdef, lvl)
		if s != LevelSeed(0xabcdef, lvl) {
			t.Fatal("not deterministic")
		}

		if seen[s] {
			t.Fatalf("level %d collides", lvl)
		}

		seen[s] = true
	}

	if LevelSeed(1, 5) == LevelSeed(2, 5) {
		t.Error("game seed ignored")
	}
}

func TestValidateHeaderRules(t *testing.T) {
	good := newChar(t, "Validhero", d2s.Paladin, d2s.NewCharacterFlags{Expansion: true})
	fix := func(b []byte) []byte {
		binary.LittleEndian.PutUint32(b[0x0C:], 0)
		binary.LittleEndian.PutUint32(b[0x0C:], d2s.Checksum(b))

		return b
	}
	edit := func(f func(b []byte)) []byte {
		b := append([]byte(nil), good...)
		f(b)

		return fix(b)
	}

	tests := []struct {
		name string
		data []byte
		ok   bool
	}{
		{"valid", good, true},
		{"empty", nil, false},
		{"truncated", good[:100], false},
		{"renamed", edit(func(b []byte) { b[0x14] ^= 1 }), true},
		{"bad checksum raw", func() []byte { b := append([]byte(nil), good...); b[0x2B] = 9; return b }(), false},
		{"level 100", edit(func(b []byte) { b[0x2B] = 100 }), false},
		{"level 0", edit(func(b []byte) { b[0x2B] = 0 }), false},
		{"new char level 5", edit(func(b []byte) { b[0x2B] = 5 }), false},
		{"bad class", edit(func(b []byte) { b[0x28] = 9 }), false},
		{"classic druid", edit(func(b []byte) { b[0x28] = byte(d2s.Druid); b[0x24] &^= byte(d2s.StatusExpansion) }), false},
		{"name with digit", edit(func(b []byte) { b[0x14+2] = '9' }), false},
		{"oversized", append(append([]byte(nil), good...), make([]byte, maxUpload)...), false},
	}

	v := Validator{}

	for _, tc := range tests {
		_, err := v.Validate(tc.data)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err=%v, want ok=%v", tc.name, err, tc.ok)
		}

		if err != nil && !errors.Is(err, ErrInvalidCharacter) {
			t.Errorf("%s: error %v does not wrap ErrInvalidCharacter", tc.name, err)
		}
	}
}

func TestDifficultyAllowed(t *testing.T) {
	h := &d2s.Header{}
	if !DifficultyAllowed(h, Normal) || DifficultyAllowed(h, Nightmare) {
		t.Error("never-played character: only Normal")
	}

	h.Difficulty[1] = 0x80 | 2 // nightmare, act III
	for d, want := range []bool{true, true, false} {
		if got := DifficultyAllowed(h, byte(d)); got != want {
			t.Errorf("difficulty %d: %v, want %v", d, got, want)
		}
	}
}

func TestCanPlay(t *testing.T) {
	mk := func(level uint8, status uint32, diff [3]byte) *d2s.Character {
		return &d2s.Character{Header: &d2s.Header{Level: level, Status: status, Difficulty: diff}}
	}
	exp := d2s.StatusExpansion
	nm := [3]byte{0, 0x80, 0}

	tests := []struct {
		name string
		c    *d2s.Character
		g    GameInfo
		want Code
	}{
		{"ok", mk(30, exp, nm), GameInfo{Expansion: true, Difficulty: Nightmare, MinLevel: 20, MaxLevel: 40}, CodeOK},
		{"too low", mk(10, exp, nm), GameInfo{Expansion: true, MinLevel: 20}, CodeLevelTooLow},
		{"too high", mk(50, exp, nm), GameInfo{Expansion: true, MaxLevel: 40}, CodeLevelTooHigh},
		{"hell locked", mk(50, exp, nm), GameInfo{Expansion: true, Difficulty: Hell}, CodeDifficultyLocked},
		{"hc vs sc", mk(50, exp|d2s.StatusHardcore, nm), GameInfo{Expansion: true}, CodeModeMismatch},
		{"classic vs exp", mk(50, 0, nm), GameInfo{Expansion: true}, CodeModeMismatch},
		{"dead hc", mk(50, exp|d2s.StatusHardcore|d2s.StatusDied, nm), GameInfo{Expansion: true, Hardcore: true}, CodeCharInvalid},
	}

	for _, tc := range tests {
		if code, _ := canPlay(tc.c, tc.g); code != tc.want {
			t.Errorf("%s: %s, want %s", tc.name, code, tc.want)
		}
	}
}

func testStore(t *testing.T, st Store) {
	t.Helper()

	if err := st.Save("alice", "Hero", []byte("one")); err != nil {
		t.Fatal(err)
	}

	if err := st.Save("Alice", "hero", []byte("two")); err != nil { // case-insensitive overwrite
		t.Fatal(err)
	}

	if b, err := st.Load("ALICE", "HERO"); err != nil || string(b) != "two" {
		t.Fatalf("load: %q %v", b, err)
	}

	if err := st.Save("bob", "Hero", []byte("x")); !errors.Is(err, ErrOwnedElsewhere) {
		t.Fatalf("stealing a name: %v", err)
	}

	if _, err := st.Load("alice", "nobody"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}

	for _, bad := range []string{"../evil", "a/b", "", "x", "a b", strings.Repeat("a", 16)} {
		if err := st.Save(bad, "ok", nil); err == nil {
			t.Errorf("account %q accepted", bad)
		}

		if err := st.Save("alice", bad, nil); err == nil {
			t.Errorf("character %q accepted", bad)
		}
	}

	_ = st.Save("alice", "Second", []byte("s"))

	names, err := st.List("alice")
	if err != nil || !reflect.DeepEqual(names, []string{"hero", "second"}) {
		t.Fatalf("list: %v %v", names, err)
	}

	if names, _ := st.List("nobody"); len(names) != 0 {
		t.Fatalf("list of empty account: %v", names)
	}
}

func TestMemStore(t *testing.T) { testStore(t, NewMemStore()) }

func TestDirStore(t *testing.T) {
	dir := t.TempDir()

	st, err := NewDirStore(filepath.Join(dir, "saves"))
	if err != nil {
		t.Fatal(err)
	}

	testStore(t, st)

	// survives a new store on the same folder
	st2, _ := NewDirStore(filepath.Join(dir, "saves"))
	if b, err := st2.Load("alice", "hero"); err != nil || string(b) != "two" {
		t.Fatalf("reopen: %q %v", b, err)
	}

	if m, _ := filepath.Glob(filepath.Join(dir, "saves", "*", "*.tmp")); len(m) != 0 {
		t.Fatalf("temp files left: %v", m)
	}
}

func TestLobbyPresenceAndChat(t *testing.T) {
	_, addr := newServer(t, Config{ServerName: "test realm"})

	a := dial(t, addr, "Alice")
	b, err := Dial(addr)
	if err != nil {
		t.Fatal(err)
	}

	defer b.Close()

	// chat and requests before Hello are refused
	_ = b.Send(ListGamesReq{})
	waitFor(t, b, "no-hello result", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeNoHello })

	ack, err := b.Hello("Bob")
	if err != nil || !reflect.DeepEqual(ack.Roster, []string{"Alice"}) || ack.Message != "test realm" {
		t.Fatalf("hello: %+v %v", ack, err)
	}

	waitFor(t, a, "Bob arrives", func(e interface{}) bool { p, ok := e.(Presence); return ok && p.Joined && p.Name == "Bob" })

	// duplicate account (case-insensitive) and invalid names
	dup, _ := Dial(addr)
	defer dup.Close()

	if _, err := dup.Hello("alice"); err == nil {
		t.Error("duplicate account accepted")
	}

	for _, bad := range []string{"x", "a b", "../x", strings.Repeat("n", 16)} {
		if _, err := dup.Hello(bad); err == nil {
			t.Errorf("account %q accepted", bad)
		}
	}

	// lobby chat reaches everybody, including the sender (authoritative echo)
	if err := a.Say("hello lobby", ""); err != nil {
		t.Fatal(err)
	}

	for _, c := range []*Client{a, b} {
		m := waitFor(t, c, "lobby chat", isChat("hello lobby")).(Chat)
		if m.Name != "Alice" || m.Type != ChatNormal {
			t.Errorf("chat: %+v", m)
		}
	}

	// whisper reaches only the recipient; unknown recipient gets a system line
	c := dial(t, addr, "Carol")
	_ = a.Say("psst", "bob")
	m := waitFor(t, b, "whisper", isChat("psst")).(Chat)

	if m.Type != ChatWhisper || m.Name != "Alice" {
		t.Errorf("whisper: %+v", m)
	}

	_ = a.Say("anyone", "Nobody")
	waitFor(t, a, "system reply", func(e interface{}) bool {
		m, ok := e.(Chat)
		return ok && m.Type == ChatSystem && strings.Contains(m.Text, "Nobody")
	})
	noEvent(t, c, "whisper leak", isChat("psst"))

	// control characters and empty text are dropped
	_ = a.Say("a\x01b", "")
	waitFor(t, b, "sanitised chat", isChat("ab"))

	_ = a.Say("   ", "")
	noEvent(t, b, "blank chat", func(e interface{}) bool { m, ok := e.(Chat); return ok && strings.TrimSpace(m.Text) == "" })

	// departure is announced
	_ = b.Close()
	waitFor(t, a, "Bob leaves", func(e interface{}) bool { p, ok := e.(Presence); return ok && !p.Joined && p.Name == "Bob" })
}

func TestGameLifecycle(t *testing.T) {
	srv, addr := newServer(t, Config{})

	a := player(t, addr, "alice", "Amy")
	b := player(t, addr, "bob", "Bill")
	c := dial(t, addr, "carol") // stays in the lobby

	if g, _ := a.Games(); len(g) != 0 {
		t.Fatalf("games before create: %v", g)
	}

	joined, err := a.Create(CreateGame{Name: "Baal Run", Password: "pw", Description: "fast", MaxPlayers: 3, MinLevel: 0})
	if err != nil {
		t.Fatal(err)
	}

	want := GameInfo{Name: "Baal Run", Description: "fast", Creator: "Amy", Players: 1, MaxPlayers: 3,
		Expansion: true, HasPassword: true}
	if joined.Game != want {
		t.Fatalf("created game %+v, want %+v", joined.Game, want)
	}

	// the creator also got the real packets: flags and act with the seed
	flags := waitFor(t, a, "game flags", func(e interface{}) bool { _, ok := e.(GameFlags); return ok }).(GameFlags)
	act := waitFor(t, a, "load act", func(e interface{}) bool { _, ok := e.(LoadAct); return ok }).(LoadAct)

	if !flags.Expansion || flags.Hardcore || act.Seed != joined.Seed || joined.Seed == 0 {
		t.Fatalf("flags %+v act %+v seed %#x", flags, act, joined.Seed)
	}

	// lobby members see the game and the creator leaving the lobby
	waitFor(t, c, "Alice leaves lobby", func(e interface{}) bool { p, ok := e.(Presence); return ok && !p.Joined && p.Name == "alice" })

	games, err := c.Games()
	if err != nil || len(games) != 1 || games[0].Name != "Baal Run" || games[0].Players != 1 {
		t.Fatalf("game list %+v %v", games, err)
	}

	// join errors
	_, err = b.Join("baal run", "nope")
	wantCode(t, err, CodeBadPassword)
	_, err = b.Join("Missing", "")
	wantCode(t, err, CodeGameNotFound)

	// drop-in: same seed, both sides learn about each other
	bj, err := b.Join("BAAL RUN", "pw")
	if err != nil {
		t.Fatal(err)
	}

	if bj.Seed != joined.Seed || bj.Game.Players != 2 || bj.UnitID == joined.UnitID {
		t.Fatalf("join: %+v vs %+v", bj, joined)
	}

	if bact := waitFor(t, b, "b load act", func(e interface{}) bool { _, ok := e.(LoadAct); return ok }).(LoadAct); bact.Seed != joined.Seed {
		t.Fatalf("b seed %#x", bact.Seed)
	}

	pa := waitFor(t, b, "Amy in game", func(e interface{}) bool { p, ok := e.(PlayerInGame); return ok && p.Name == "Amy" }).(PlayerInGame)
	if pa.UnitID != joined.UnitID || pa.Level != 1 || pa.Class != byte(d2s.Sorceress) {
		t.Fatalf("player info %+v", pa)
	}

	pb := waitFor(t, a, "Bill in game", func(e interface{}) bool { p, ok := e.(PlayerInGame); return ok && p.Name == "Bill" }).(PlayerInGame)
	if pb.UnitID != bj.UnitID {
		t.Fatalf("player info %+v", pb)
	}

	waitFor(t, a, "join notice", isChat("Bill joined the game."))

	// per-level sync: Bill enters level 5, Amy hears it; a later joiner is told too
	_ = b.Send(LevelChange{Act: 1, Level: 40})
	pl := waitFor(t, a, "level change", func(e interface{}) bool { _, ok := e.(PlayerLevel); return ok }).(PlayerLevel)

	if pl.UnitID != bj.UnitID || pl.Act != 1 || pl.Level != 40 {
		t.Fatalf("level change %+v", pl)
	}

	if LevelSeed(joined.Seed, pl.Level) != LevelSeed(bj.Seed, pl.Level) {
		t.Fatal("level seeds differ between clients")
	}

	// game chat stays inside the game; lobby chat stays out of it
	_ = a.Say("in-game", "")
	waitFor(t, b, "game chat", isChat("in-game"))
	waitFor(t, a, "game chat echo", isChat("in-game"))
	noEvent(t, c, "game chat in lobby", isChat("in-game"))

	_ = c.Say("lobby-only", "")
	waitFor(t, c, "lobby echo", isChat("lobby-only"))
	noEvent(t, a, "lobby chat in game", isChat("lobby-only"))

	// in-game whisper by character name
	_ = a.Say("secret", "bill")
	waitFor(t, b, "game whisper", isChat("secret"))

	// the game list shows two players
	if g, _ := c.Games(); len(g) != 1 || g[0].Players != 2 {
		t.Fatalf("game list %+v", g)
	}

	// drop-out by 0x69: Amy is told, Bill is back in the lobby and sees the roster
	if err := b.Leave(); err != nil {
		t.Fatal(err)
	}

	waitFor(t, a, "player leave", func(e interface{}) bool { p, ok := e.(PlayerLeave); return ok && p.UnitID == bj.UnitID })
	waitFor(t, a, "leave notice", isChat("Bill left the game."))
	waitFor(t, b, "leave ack", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeOK })
	waitFor(t, b, "roster has carol", func(e interface{}) bool { p, ok := e.(Presence); return ok && p.Name == "carol" })
	waitFor(t, c, "Bill back", func(e interface{}) bool { p, ok := e.(Presence); return ok && p.Joined && p.Name == "bob" })

	// leaving twice is an error, Bill can rejoin (drop-in again)
	_ = b.Leave()
	waitFor(t, b, "not in game", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeNotInGame })

	if _, err := b.Join("Baal Run", "pw"); err != nil {
		t.Fatal(err)
	}

	// drop-out by disconnect: the game survives while somebody is in it...
	_ = b.Close()
	waitFor(t, a, "disconnect leave", func(e interface{}) bool { _, ok := e.(PlayerLeave); return ok })

	if g, _ := c.Games(); len(g) != 1 || g[0].Players != 1 {
		t.Fatalf("game list %+v", g)
	}

	// ...and goes away with the last player
	_ = a.Close()

	deadline := time.Now().Add(wait)
	for {
		g, _ := c.Games()
		if len(g) == 0 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("game not removed: %+v", g)
		}

		time.Sleep(10 * time.Millisecond)
	}

	srv.mu.Lock()
	defer srv.mu.Unlock()

	if len(srv.games) != 0 || len(srv.chars) != 0 || len(srv.accounts) != 1 {
		t.Fatalf("leaked state: games=%d chars=%d accounts=%d", len(srv.games), len(srv.chars), len(srv.accounts))
	}
}

func TestGameRules(t *testing.T) {
	_, addr := newServer(t, Config{MaxGames: 2})

	exp := d2s.NewCharacterFlags{Expansion: true}
	a := dial(t, addr, "alice")

	// nothing works without a character
	_, err := a.Create(CreateGame{Name: "x"})
	wantCode(t, err, CodeNoCharacter)
	_, err = a.Join("x", "")
	wantCode(t, err, CodeNoCharacter)

	if err := a.Upload([]byte("garbage")); err == nil {
		t.Fatal("garbage accepted")
	} else {
		wantCode(t, err, CodeCharInvalid)
	}

	if err := a.Upload(newChar(t, "Amy", d2s.Sorceress, exp)); err != nil {
		t.Fatal(err)
	}

	bad := []CreateGame{
		{Name: ""}, {Name: " lead"}, {Name: strings.Repeat("n", 16)}, {Name: "ok", Password: strings.Repeat("p", 16)},
		{Name: "ok", Description: strings.Repeat("d", 32)}, {Name: "ok", Difficulty: 3}, {Name: "ok", MaxPlayers: 9},
		{Name: "ok", MinLevel: 50, MaxLevel: 40}, {Name: "ok", MaxLevel: 100}, {Name: "ok", Description: "tab\there"},
	}
	for i, g := range bad {
		if _, err := a.Create(g); err == nil {
			t.Errorf("bad create %d (%+v) accepted", i, g)
		} else {
			wantCode(t, err, CodeBadRequest)
		}
	}

	// a new character has not unlocked Nightmare
	_, err = a.Create(CreateGame{Name: "nm", Difficulty: Nightmare})
	wantCode(t, err, CodeDifficultyLocked)

	// level 1 against a level window
	_, err = a.Create(CreateGame{Name: "hi", MinLevel: 10})
	wantCode(t, err, CodeLevelTooLow)

	if _, err = a.Create(CreateGame{Name: "Two", MaxPlayers: 2}); err != nil {
		t.Fatal(err)
	}

	// already in a game
	_, err = a.Create(CreateGame{Name: "again"})
	wantCode(t, err, CodeAlreadyInGame)
	_, err = a.Join("Two", "")
	wantCode(t, err, CodeAlreadyInGame)
	wantCode(t, a.Upload(newChar(t, "Other", d2s.Sorceress, exp)), CodeAlreadyInGame)

	// re-uploading the same character inside the game is fine
	if err := a.Upload(newChar(t, "Amy", d2s.Sorceress, exp)); err != nil {
		t.Fatal(err)
	}

	b := player(t, addr, "bob", "Bill")
	if _, err := b.Create(CreateGame{Name: "TWO"}); err == nil {
		t.Fatal("duplicate game name accepted")
	} else {
		wantCode(t, err, CodeGameExists)
	}

	// hardcore / classic characters cannot join an expansion softcore game
	hc := dial(t, addr, "hank")
	if err := hc.Upload(newChar(t, "Hank", d2s.Barbarian, d2s.NewCharacterFlags{Expansion: true, Hardcore: true})); err != nil {
		t.Fatal(err)
	}

	_, err = hc.Join("Two", "")
	wantCode(t, err, CodeModeMismatch)

	cl := dial(t, addr, "clara")
	if err := cl.Upload(newChar(t, "Clara", d2s.Amazon, d2s.NewCharacterFlags{})); err != nil {
		t.Fatal(err)
	}

	_, err = cl.Join("Two", "")
	wantCode(t, err, CodeModeMismatch)

	// fill the game (2 players), the third is refused
	if _, err := b.Join("Two", ""); err != nil {
		t.Fatal(err)
	}

	d := player(t, addr, "dora", "Dora")
	_, err = d.Join("Two", "")
	wantCode(t, err, CodeGameFull)

	// room for exactly MaxGames games
	if _, err := d.Create(CreateGame{Name: "Three"}); err != nil {
		t.Fatal(err)
	}

	e := player(t, addr, "ed", "Edd")
	_, err = e.Create(CreateGame{Name: "Four"})
	wantCode(t, err, CodeServerFull)

	// a character in use cannot be used by someone else
	g := dial(t, addr, "gina")
	wantCode(t, g.Upload(newChar(t, "Amy", d2s.Sorceress, exp)), CodeNameTaken)

	// leaving frees the slot for rejoining and for a new waiting player
	_ = b.Leave()
	waitFor(t, b, "left", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeOK })

	if _, err := d.Leave2(t, "Two"); err != nil {
		t.Fatal(err)
	}
}

// Leave2 leaves the current game and joins another one.
func (c *Client) Leave2(t *testing.T, game string) (GameJoined, error) {
	t.Helper()

	_ = c.Leave()
	waitFor(t, c, "left", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeOK })

	return c.Join(game, "")
}

func TestCharacterPersistence(t *testing.T) {
	dir := t.TempDir()
	st, _ := NewDirStore(dir)
	_, addr := newServer(t, Config{Store: st})

	file := newChar(t, "Persist", d2s.Necromancer, d2s.NewCharacterFlags{Expansion: true})

	a := dial(t, addr, "alice")
	if err := a.Upload(file); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "alice", "persist.d2s")); err != nil {
		t.Fatalf("not on disk: %v", err)
	}

	_ = a.Close()

	// reconnect (after the server dropped the first session) and get it back
	var a2 *Client

	for i := 0; ; i++ {
		c, err := Dial(addr)
		if err != nil {
			t.Fatal(err)
		}

		defer c.Close()

		if _, err := c.Hello("alice"); err == nil {
			a2 = c

			break
		}

		if i > 100 {
			t.Fatal("account never freed")
		}

		time.Sleep(10 * time.Millisecond)
	}

	chars, err := a2.Chars()
	if err != nil || len(chars) != 1 || chars[0].Name != "Persist" || chars[0].Class != byte(d2s.Necromancer) || chars[0].Level != 1 {
		t.Fatalf("chars %+v %v", chars, err)
	}

	got, err := a2.Select("persist")
	if err != nil || !bytes.Equal(got, file) {
		t.Fatalf("select: %v equal=%v", err, bytes.Equal(got, file))
	}

	// the selected character can host a game without uploading again
	if _, err := a2.Create(CreateGame{Name: "stored"}); err != nil {
		t.Fatal(err)
	}

	// another account can neither take the name nor select it
	b := dial(t, addr, "bob")
	wantCode(t, b.Upload(file), CodeNameTaken)
	_, err = b.Select("Persist")
	wantCode(t, err, CodeNoCharacter)

	// class changes of a stored character are refused
	_ = a2.Leave()
	waitFor(t, a2, "left", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeOK })

	swapped := newChar(t, "Persist", d2s.Sorceress, d2s.NewCharacterFlags{Expansion: true})
	wantCode(t, a2.Upload(swapped), CodeCharInvalid)

	_, err = a2.Select("nobody")
	wantCode(t, err, CodeNoCharacter)
}

func TestManyClientsJoinAndLeave(t *testing.T) {
	srv, addr := newServer(t, Config{})

	const n = 8

	creator := player(t, addr, "host", "Host")
	if _, err := creator.Create(CreateGame{Name: "party", MaxPlayers: n}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	errs := make(chan error, n)

	for i := 1; i < n; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			c, err := Dial(addr)
			if err != nil {
				errs <- err

				return
			}

			defer c.Close()

			if _, err = c.Hello(fmt.Sprintf("acct%d", i)); err == nil {
				err = c.Upload(newChar(t, fmt.Sprintf("Char%c", 'A'+i), d2s.Amazon, d2s.NewCharacterFlags{Expansion: true}))
			}

			var j GameJoined
			if err == nil {
				j, err = c.Join("party", "")
			}

			if err == nil && (j.Seed == 0 || j.UnitID == 0) {
				err = fmt.Errorf("client %d: bad join %+v", i, j)
			}

			if err == nil {
				err = c.Say(fmt.Sprintf("hi from %d", i), "")
			}

			if err == nil {
				_, err = c.Wait(wait, isChat(fmt.Sprintf("hi from %d", i)))
			}

			if err == nil && i%2 == 0 {
				_ = c.Leave()
				_, err = c.Wait(wait, func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeOK })
			}

			errs <- err
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	// the host saw every join
	joins := 0
	for _, e := range creator.Pending() {
		if _, ok := e.(PlayerInGame); ok {
			joins++
		}
	}

	if joins != n-1 {
		t.Fatalf("host saw %d joins, want %d", joins, n-1)
	}

	// once everybody is gone the server holds no stale state
	_ = creator.Close()

	deadline := time.Now().Add(wait)
	for {
		srv.mu.Lock()
		left := len(srv.sessions) + len(srv.games) + len(srv.chars) + len(srv.accounts)
		srv.mu.Unlock()

		if left == 0 {
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("leaked state: %d entries", left)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func TestSlowAndGarbageClients(t *testing.T) {
	_, addr := newServer(t, Config{IdleTimeout: 200 * time.Millisecond})

	// an idle connection is dropped
	c, err := Dial(addr)
	if err != nil {
		t.Fatal(err)
	}

	defer c.Close()

	if _, err := c.Wait(wait, func(e interface{}) bool { _, ok := e.(Disconnected); return ok }); err != nil {
		t.Fatalf("idle client not dropped: %v", err)
	}

	// a malformed realm message is answered, not fatal
	d := dial(t, addr, "dave")
	typ, _ := Encode(JoinGame{})
	_ = d.write(tunnelFor(typ, []byte{200})...) // string longer than the body

	waitFor(t, d, "bad request", func(e interface{}) bool { r, ok := e.(Result); return ok && r.Code == CodeBadRequest })

	if _, err := d.Games(); err != nil {
		t.Fatal(err)
	}
}
