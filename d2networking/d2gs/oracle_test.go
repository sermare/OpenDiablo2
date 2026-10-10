package d2gs

import (
	"bytes"
	"reflect"
	"testing"
)

// Byte fixtures derived from the notes (game-net.md, structs-net-skills.md) and
// from handler bodies read in Ghidra; all multi-byte values are little-endian.
var clientFixtures = []struct {
	name string
	raw  []byte
	msg  Message
}{
	{"walk to location", []byte{0x01, 0x2c, 0x01, 0xf4, 0x01}, &MoveToLocation{X: 300, Y: 500}},
	{"run to location", []byte{0x03, 0x01, 0x00, 0xff, 0xff}, &MoveToLocation{Run: true, X: 1, Y: 65535}},
	{"walk to entity", []byte{0x02, 0x01, 0, 0, 0, 0x09, 0, 0, 0}, &UnitOrder{ID: 0x02, UnitType: 1, UnitID: 9}},
	{"run to entity", []byte{0x04, 0x01, 0, 0, 0, 0x09, 0, 0, 0}, &UnitOrder{ID: 0x04, UnitType: 1, UnitID: 9}},
	{"cast left at location", []byte{0x05, 0x10, 0, 0x20, 0}, &CastOnLocation{X: 16, Y: 32}},
	{"cast right at location", []byte{0x0c, 0x10, 0, 0x20, 0}, &CastOnLocation{Right: true, X: 16, Y: 32}},
	{"cast left at entity", []byte{0x06, 0x01, 0, 0, 0, 0x78, 0x56, 0x34, 0x12}, &UnitOrder{ID: 0x06, UnitType: 1, UnitID: 0x12345678}},
	{"cast right at entity", []byte{0x0d, 0x01, 0, 0, 0, 0x78, 0x56, 0x34, 0x12}, &UnitOrder{ID: 0x0d, UnitType: 1, UnitID: 0x12345678}},
	{"interact", []byte{0x13, 0x01, 0, 0, 0, 0x07, 0, 0, 0}, &InteractUnit{UnitType: 1, UnitID: 7}},
	{"pick up", []byte{0x16, 0x04, 0, 0, 0, 0x07, 0, 0, 0, 0x01, 0, 0, 0}, &PickUpUnit{UnitType: 4, UnitID: 7, Flag: 1}},
	{"drop", []byte{0x17, 0x05, 0, 0, 0}, &DropCursorItem{ItemID: 5}},
	{"item to inventory", []byte{0x18, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0}, &ItemToContainer{ItemID: 1, X: 2, Y: 3}},
	{"use item", []byte{0x20, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}, &UseItem{ItemID: 1, X: 2, Y: 3}},
	{"npc interact", []byte{0x2f, 1, 0, 0, 0, 9, 0, 0, 0}, &NpcInit{UnitType: 1, UnitID: 9}},
	{"npc trade menu", []byte{0x38, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0}, &NpcTrade{1, 2, 3}},
	{"buy", []byte{0x32, 1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0x80, 0xdc, 5, 0, 0}, &BuyItem{1, 2, 0x80000000, 1500}},
	{"sell", []byte{0x33, 1, 0, 0, 0, 2, 0, 0, 0, 4, 3, 0, 0, 9, 0, 0, 0}, &SellItem{1, 2, 0x0304, 9}},
	{"allocate stat", []byte{0x3a, 0x02, 0x04}, &AllocateStat{Stat: 2, Extra: 4}},
	{"add skill point", []byte{0x3b, 0x24, 0x00}, &AddSkillPoint{Skill: 36}},
	{"select skill", []byte{0x3c, 0x24, 0, 0, 0x80, 0xff, 0xff, 0xff, 0xff}, &SelectSkill{Skill: 36, Right: true, ItemID: 0xffffffff}},
	{"assign hotkey", []byte{0x51, 0x24, 0x80, 0x03, 0x00, 0xff, 0xff, 0xff, 0xff}, &SetHotkey{Slot: 3, Skill: 0x24, Right: true, ItemID: 0xffffffff}},
	{"party request", []byte{0x5e, 0x05, 0x4d, 0, 0, 0}, &PartyRequest{Action: 5, PlayerID: 77}},
	{"party relation", []byte{0x5d, 0x04, 0x01, 0x4e, 0, 0, 0}, &PartyRelation{Action: 4, Flag: 1, PlayerID: 78}},
}

func TestClientFixturesRoundTrip(t *testing.T) {
	for _, tc := range clientFixtures {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.msg.MarshalPacket(); !bytes.Equal(got, tc.raw) {
				t.Fatalf("encode: % x want % x", got, tc.raw)
			}
			got, err := DecodeClient(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.msg) {
				t.Fatalf("decode: %+v want %+v", got, tc.msg)
			}
			if err := Validate(ClientToServer, tc.raw); err != nil {
				t.Fatal(err)
			}
			// the stream decoder frames the fixture and a following one identically
			pk, rest, err := SplitAll(ClientToServer, append(append([]byte{}, tc.raw...), tc.raw...))
			if err != nil || len(pk) != 2 || len(rest) != 0 {
				t.Fatalf("split: %d %d %v", len(pk), len(rest), err)
			}
		})
	}
}

func TestClientChatFixtures(t *testing.T) {
	raw := []byte{0x15, 0x01, 0x00, 'h', 'i', 0, 0}
	if got := ClientChat("hi"); !bytes.Equal(got, raw) {
		t.Fatalf("encode % x", got)
	}
	if s, err := ParseClientChat(raw); err != nil || s != "hi" {
		t.Fatalf("%q %v", s, err)
	}
	// without the recipient field the real handler (0x5484a0) drops the packet
	if _, err := ParseClientChat([]byte{0x15, 1, 0, 'h', 'i', 0}); err == nil {
		t.Fatal("packet without recipient field accepted")
	}
	// text of 256 bytes is rejected by the handler
	long := append(append([]byte{0x15, 1, 0}, bytes.Repeat([]byte{'a'}, 256)...), 0, 0)
	if _, err := ParseClientChat(long); err == nil {
		t.Fatal("256 byte text accepted")
	}
}

// serverFixtures are server packets with sizes from the table at 0x72e958.
var serverFixtures = []struct {
	name string
	raw  []byte
}{
	{"game flags", []byte{0x01, 1, 0, 0, 0, 1, 0, 0}},
	{"load act", []byte{0x03, 1, 0x78, 0x56, 0x34, 0x12, 0x01, 0, 0, 0, 0, 0}},
	{"player move", append([]byte{0x0f, 0, 9, 0, 0, 0, 1, 0x10, 0, 0x20, 0, 0}, 0x08, 0, 0x08, 0)},
	{"hp/mana 0x18", fixedPkt(0x18, 15)},
	{"life/mana 0x95", fixedPkt(0x95, 13)},
	{"walk verify 0x96", fixedPkt(0x96, 9)},
	{"attr byte 0x19", fixedPkt(0x19, 2)},
	{"attr dword 0x1f", fixedPkt(0x1f, 6)},
	{"player in game 0x59", fixedPkt(0x59, 26)},
	{"player leave 0x5c", fixedPkt(0x5c, 5)},
	{"quest info 0x28", fixedPkt(0x28, 103)},
	{"chat 0x26", append(append([]byte{0x26}, make([]byte, 9)...), []byte("bob\x00hi\x00")...)},
	{"item action 0x9c", withByte(fixedPkt(0x9c, 14), 2, 14)},
	{"add unit 0xaa", withByte(fixedPkt(0xaa, 15), 6, 15)},
	{"state 0xa8", withByte(fixedPkt(0xa8, 10), 6, 10)},
}

func fixedPkt(id byte, n int) []byte { b := make([]byte, n); b[0] = id; return b }

func withByte(b []byte, off int, v byte) []byte { b[off] = v; return b }

func TestServerFixturesFraming(t *testing.T) {
	for _, tc := range serverFixtures {
		t.Run(tc.name, func(t *testing.T) {
			b := append([]byte{}, tc.raw...)
			if err := Validate(ServerToClient, b); err != nil {
				t.Fatal(err)
			}
			pk, rest, err := SplitAll(ServerToClient, append(append([]byte{}, b...), b...))
			if err != nil || len(pk) != 2 || len(rest) != 0 {
				t.Fatalf("split %d %d %v", len(pk), len(rest), err)
			}
			// every truncation must be reported as incomplete, never framed
			for cut := 1; cut < len(b); cut++ {
				pk, _, err := SplitAll(ServerToClient, b[:cut])
				if len(pk) != 0 {
					t.Fatalf("cut %d framed a packet (err %v)", cut, err)
				}
			}
		})
	}
}

// TestUnknownIDs pins how unknown / invalid ids are treated in both directions.
func TestUnknownIDs(t *testing.T) {
	for id := 0; id < 256; id++ {
		_, ks := ExpectedSize(byte(id), ServerToClient)
		_, kc := ExpectedSize(byte(id), ClientToServer)
		if id > MaxServerID && ks != SizeInvalid {
			t.Errorf("server id %#x beyond the table is %v", id, ks)
		}
		if (id == 0 || id > MaxClientID) && kc != SizeInvalid {
			t.Errorf("client id %#x outside 1..0x66 is %v", id, kc)
		}
		if _, _, err := SplitAll(ServerToClient, []byte{byte(id)}); ks == SizeInvalid && err == nil {
			t.Errorf("server id %#x not reported invalid", id)
		}
	}
}

// TestServerHeaderBounds checks the variable-length rules at their extremes.
func TestServerHeaderBounds(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
		bad  bool
	}{
		{"0xae max payload", append([]byte{0xae, 0xfd, 0x01}, make([]byte, 0x1fd)...), false},
		{"0xae over max falls to 3", []byte{0xae, 0xfe, 0x01, 0, 0}, true},
		{"0x9c length below header", []byte{0x9c, 0, 2}, true},
		{"0x94 skill list", append([]byte{0x94, 1}, make([]byte, 7)...), false},
		{"0xb3 chunk", append([]byte{0xb3, 2}, make([]byte, 7)...), false},
		{"0x26 unterminated", append([]byte{0x26}, bytes.Repeat([]byte{'a'}, 0x300)...), true},
	}
	for _, tc := range cases {
		err := Validate(ServerToClient, tc.raw)
		if tc.bad && err == nil {
			t.Errorf("%s: expected error", tc.name)
		}
		if !tc.bad && err != nil {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}

func TestTunnelMessageIsBounded(t *testing.T) {
	var a TunnelAssembler
	chunk := Tunnel(S2CMetaAE, 7, make([]byte, maxTunnelChunk*2))[0] // not the last chunk
	chunk[4] = 0
	var err error
	for i := 0; i < maxTunnelMessage/maxTunnelChunk+10 && err == nil; i++ {
		_, _, _, err = a.Add(chunk)
	}
	if err == nil {
		t.Fatal("unbounded tunnel reassembly")
	}
}

// ---- fuzz targets: go test -fuzz=FuzzX -fuzztime=60s ----

func FuzzDecodeClient(f *testing.F) {
	for _, tc := range clientFixtures {
		f.Add(tc.raw)
	}
	f.Add(ClientChat("hello"))
	f.Fuzz(func(t *testing.T, b []byte) {
		if m, err := DecodeClient(b); err == nil {
			// a decoded message must re-encode to a packet of the table size
			out := m.MarshalPacket()
			if n, k := ExpectedSize(out[0], ClientToServer); k != SizeFixed || n != len(out) || len(out) != len(b) {
				t.Fatalf("id %#x: re-encoded %d bytes, table (%d,%v), input %d", out[0], len(out), n, k, len(b))
			}
		}
		_ = Validate(ClientToServer, b)
		_, _ = ParseClientChat(b)
		_, _ = ClientPacketLength(b)
		SplitClient(b)
	})
}

func FuzzServerParsers(f *testing.F) {
	for _, tc := range serverFixtures {
		f.Add(tc.raw)
	}
	f.Add(ChatMessage{Type: 1, UnitID: 5, Name: "a", Text: "b"}.Marshal())
	f.Add(AssignPlayer{UnitID: 1, Name: "n", X: 1, Y: 2}.Marshal())
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = ParseGameFlags(b)
		_, _ = ParseLoadAct(b)
		_, _ = ParseLifeMana(b)
		_, _ = ParsePlayerVitals(b)
		_, _ = ParseAssignPlayer(b)
		_, _ = ParsePlayerLeave(b)
		_, _ = ParsePlayerMove(b)
		_, _ = ParseUnitSkillOnLocation(b)
		_, _ = ParseJoinGame(b)
		if m, err := ParseChatMessage(b); err == nil {
			if err := Validate(ServerToClient, m.Marshal()); err != nil && len(m.Marshal()) <= MaxPacketSize {
				t.Fatalf("re-encoded chat invalid: %v", err)
			}
		}
		_ = Validate(ServerToClient, b)
		SplitServer(b)
	})
}

func FuzzDecoderChunked(f *testing.F) {
	f.Add([]byte{0x01, 0, 0, 0, 0, 0, 0, 0, 0x9c, 0, 4, 0}, byte(3))
	f.Fuzz(func(t *testing.T, b []byte, step byte) {
		for _, dir := range []Direction{ServerToClient, ClientToServer} {
			d := NewDecoder(dir)
			s := int(step%16) + 1
			total := 0
			for i := 0; i < len(b); i += s {
				e := i + s
				if e > len(b) {
					e = len(b)
				}
				d.Write(b[i:e])
				for {
					p, ok, err := d.Next()
					if err != nil || !ok {
						break
					}
					if len(p.Data) == 0 || len(p.Data) > MaxPacketSize {
						t.Fatalf("packet length %d", len(p.Data))
					}
					total += len(p.Data)
				}
				if d.Buffered() > MaxPacketSize+s {
					// a healthy decoder never holds more than one incomplete packet
					if _, _, err := d.Next(); err == nil {
						t.Fatalf("buffered %d bytes without error", d.Buffered())
					}
				}
			}
			_ = total
		}
	})
}

func FuzzTunnelAssembler(f *testing.F) {
	for _, p := range Tunnel(S2CMetaAE, 3, []byte("hello")) {
		f.Add(p)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		var a TunnelAssembler
		for i := 0; i < 3; i++ {
			_, _, _, _ = a.Add(b)
		}
	})
}

func FuzzFrameLengthAndHuffman(f *testing.F) {
	f.Add([]byte{0xf1, 0x02, 1, 2})
	h := NewHuffman()
	f.Fuzz(func(t *testing.T, b []byte) {
		n, p := ReadFrameLength(b)
		if p > len(b) || n < 0 || n > 0xfff {
			t.Fatalf("frame length %d prefix %d", n, p)
		}
		if out, err := h.Decompress(nil, b); err == nil && len(out) > 1<<20 {
			t.Fatalf("decompress expanded %d -> %d", len(b), len(out))
		}
	})
}
