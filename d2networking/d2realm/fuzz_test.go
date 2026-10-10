package d2realm

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2gs"
)

// All seeds are generated here (synthetic messages and a synthetic new-character header).

func sampleMessages() []interface{} {
	hero, _ := d2s.NewCharacter("Hero", d2s.Amazon, d2s.NewCharacterFlags{Expansion: true, Created: time.Unix(1700000000, 0)},
		d2s.DefaultAppearance(d2s.Amazon))

	return []interface{}{
		Hello{Version: 1, Account: "alice"},
		UploadChar{Data: hero},
		SelectChar{Name: "Hero"},
		CreateGame{Name: "g0", Difficulty: 0, MaxPlayers: 8, MinLevel: 1, MaxLevel: 99},
		JoinGame{Name: "g0"},
		Hello{Version: 1, Account: "alice"},
		ListGamesReq{},
		CreateGame{Name: "g1", Password: "pw", Description: "d", Difficulty: 1, MaxPlayers: 4, MinLevel: 1, MaxLevel: 99},
		JoinGame{Name: "g1", Password: "pw"},
		UploadChar{Data: []byte("not a save")},
		ListCharsReq{},
		SelectChar{Name: "Hero"},
		LevelChange{Act: 1, Level: 3},
		HelloAck{Code: CodeOK, Message: "hi", Roster: []string{"a", "b"}},
		Result{Op: 1, Code: CodeBadRequest, Message: "no"},
		GameList{Games: []GameInfo{{Name: "g1", Players: 1, MaxPlayers: 8}}},
		CharList{Chars: []CharInfo{{Name: "Hero", Class: 1, Level: 5}}},
		CharData{Name: "Hero", Data: []byte{1, 2, 3}},
		Presence{Joined: true, Name: "alice"},
		PlayerLevel{UnitID: 9, Act: 1, Level: 2},
		GameJoined{Game: GameInfo{Name: "g1"}, UnitID: 3, Seed: 77},
	}
}

func FuzzDecode(f *testing.F) {
	for _, m := range sampleMessages() {
		typ, body := Encode(m)
		f.Add(typ, body)
	}

	f.Add(byte(0), []byte{})
	f.Add(byte(0xff), []byte{0xff, 0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, typ byte, body []byte) {
		msg, err := Decode(typ, body)
		if err != nil {
			return
		}

		// what decodes must re-encode and decode to the same message
		typ2, body2 := Encode(msg)

		msg2, err := Decode(typ2, body2)
		if err != nil {
			t.Fatalf("re-decode of %T failed: %v", msg, err)
		}

		if !reflect.DeepEqual(msg, msg2) {
			t.Fatalf("round trip changed %#v into %#v", msg, msg2)
		}
	})
}

func FuzzValidate(f *testing.F) {
	for _, class := range []d2s.Class{d2s.Amazon, d2s.Necromancer} {
		if b, err := d2s.NewCharacter("Tester", class, d2s.NewCharacterFlags{Expansion: true, Created: time.Unix(1700000000, 0)},
			d2s.DefaultAppearance(class)); err == nil {
			f.Add(b)
		}
	}

	f.Add([]byte{})

	v := &Validator{}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = v.Validate(data)
	})
}

func FuzzChatAndNames(f *testing.F) {
	f.Add(ClientChat("hello", ""), "alice", "g1")
	f.Add(ClientChat("psst", "bob"), "", "")
	f.Add([]byte{d2gs.C2SChat}, "a\x00b", "\xff")

	f.Fuzz(func(t *testing.T, p []byte, account, game string) {
		_, _, _ = ParseChat(p)
		_ = ValidAccount(account)
		_ = ValidGameName(game)
		_ = cleanText(account)
	})
}

// FuzzServerPackets feeds arbitrary client byte streams to the lobby dispatcher of two sessions of
// one server, with no sockets: it must not panic or deadlock, whatever the order of messages.
func FuzzServerPackets(f *testing.F) {
	var stream []byte

	for _, m := range sampleMessages() {
		typ, body := Encode(m)
		for _, p := range tunnelFor(typ, body) {
			stream = append(stream, p...)
		}
	}

	f.Add(stream)
	f.Add(append([]byte(nil), stream[:len(stream)/2]...))
	f.Add(ClientChat("hello", ""))

	f.Fuzz(func(t *testing.T, data []byte) {
		srv := New(Config{Rand: func() uint32 { return 1 }})
		defer srv.Close()

		var sess [2]*session

		for i := range sess {
			a, b := net.Pipe()
			defer a.Close()
			defer b.Close()

			sess[i] = &session{srv: srv, conn: a, out: make(chan []byte, 4096)}
			srv.sessions[sess[i]] = struct{}{}
		}

		pkts, _ := d2gs.SplitClient(data)
		for i, p := range pkts {
			if len(p) == 0 {
				continue
			}

			func() {
				srv.mu.Lock()
				defer srv.mu.Unlock() // also when the dispatcher panics, so the failure is reported

				srv.packet(sess[i%2], p)
			}()
		}

		for _, s := range sess {
			srv.drop(s)
		}
	})
}
