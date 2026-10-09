package d2gs

import (
	"bufio"
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

func TestHuffmanTable(t *testing.T) {
	// Kraft sum: a complete prefix code sums to <= 1
	var sum float64
	for _, l := range huffLengths {
		sum += 1 / float64(uint32(1)<<l)
	}

	if sum > 1.0000001 {
		t.Fatalf("Kraft sum %v > 1: not a prefix code", sum)
	}

	h := NewHuffman()
	if len(h.decode) != 256 {
		t.Fatalf("%d distinct codes, want 256 (collision)", len(h.decode))
	}

	for sym, l := range huffLengths {
		// the original keeps each code in one byte: no code may need more
		if h.code[sym] > 0xff {
			t.Errorf("symbol %#x: code %#x does not fit the original's byte table", sym, h.code[sym])
		}

		if uint32(h.code[sym]) >= 1<<l {
			t.Errorf("symbol %#x: code %#x longer than %d bits", sym, h.code[sym], l)
		}
	}
	// the most frequent byte (0x00, length 1) has the one-bit code 1
	if h.code[0] != 1 {
		t.Errorf("code of 0x00 = %#x", h.code[0])
	}
}

func TestHuffmanRoundTrip(t *testing.T) {
	h := NewHuffman()
	rng := rand.New(rand.NewSource(1))

	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}

	cases := [][]byte{{}, {0}, {0xff}, all, bytes.Repeat([]byte{0}, 100), {0x15, 0x01, 0x00, 'h', 'i', 0, 0}}

	for i := 0; i < 200; i++ {
		b := make([]byte, rng.Intn(600))
		rng.Read(b)
		cases = append(cases, b)
	}

	for i, in := range cases {
		comp := h.Compress(nil, in)

		out, err := h.Decompress(nil, comp)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}

		if !bytes.Equal(in, out) {
			t.Fatalf("case %d: round trip differs (%d in, %d out)", i, len(in), len(out))
		}
	}
}

func TestHuffmanPacking(t *testing.T) {
	h := NewHuffman()
	// 0x00 is the 1-bit code "1": eight of them make 0xff, MSB first
	if got := h.Compress(nil, bytes.Repeat([]byte{0}, 8)); !bytes.Equal(got, []byte{0xff}) {
		t.Errorf("got %x", got)
	}
	// a partial last byte is zero padded: three zero bytes = 111 00000
	if got := h.Compress(nil, []byte{0, 0, 0}); !bytes.Equal(got, []byte{0xe0}) {
		t.Errorf("got %x", got)
	}
}

func TestBlobFraming(t *testing.T) {
	big := bytes.Repeat([]byte{0x7b}, 600) // 0x7b = a long code: compresses past 0xF0 bytes
	p1 := (MoveToLocation{X: 10, Y: 20}).MarshalPacket()

	blob, err := EncodeBlob(p1, ClientChat("hello"), big[:5])
	if err != nil {
		t.Fatal(err)
	}

	two, err := EncodeBlob(bytes.Repeat([]byte{0xA0}, 300)) // random-ish bytes: > 0xF0 compressed
	if err != nil {
		t.Fatal(err)
	}

	if two[0]&0xF0 != 0xF0 {
		t.Errorf("long blob should use the 2-byte prefix, first byte %#x", two[0])
	}

	r := bufio.NewReader(bytes.NewReader(append(blob, two...)))

	plain, err := ReadBlob(r)
	if err != nil {
		t.Fatal(err)
	}

	pkts, ignored := SplitClient(plain)
	if len(pkts) != 2 || !bytes.Equal(pkts[0], p1) {
		t.Fatalf("packets %v ignored %d", pkts, ignored)
	}

	if txt, _ := ParseClientChat(pkts[1]); txt != "hello" {
		t.Errorf("chat %q", txt)
	}

	if ignored != 5 { // the 5 trailing 0x7b bytes are an unknown client id
		t.Errorf("ignored %d", ignored)
	}

	plain, err = ReadBlob(r)
	if err != nil || len(plain) != 300 {
		t.Fatalf("second blob: %v len %d", err, len(plain))
	}
}

func TestServerMessagesRoundTrip(t *testing.T) {
	flags := GameFlags{Difficulty: 2, Hardcore: true, Expansion: true}
	load := LoadAct{Act: 0, Seed: 0xdeadbeef, StartLevel: 1, Aux: 7}
	in := AssignPlayer{UnitID: 9, Class: 1, Name: "Maricon", X: 5100, Y: 5200}
	mv := PlayerMove{UnitID: 9, Run: true, TargetX: 100, TargetY: 200, CurX: 90, CurY: 80}
	sk := UnitSkillOnLocation{UnitID: 9, Skill: 36, Level: 1, X: 5, Y: 6}
	chat := ChatMessage{Type: 1, UnitID: 9, Name: "Maricon", Text: "hi there"}

	for _, b := range [][]byte{flags.Marshal(), load.Marshal(), in.Marshal(), mv.Marshal(), sk.Marshal(),
		(PlayerLeave{UnitID: 9}).Marshal(), chat.Marshal(), GameExit()} {
		if err := Validate(ServerToClient, b); err != nil {
			t.Errorf("id %#x fails the size table: %v", b[0], err)
		}
	}

	if g, err := ParseGameFlags(flags.Marshal()); err != nil || g != flags {
		t.Errorf("flags %+v %v", g, err)
	}

	if g, err := ParseLoadAct(load.Marshal()); err != nil || g != load {
		t.Errorf("loadact %+v %v", g, err)
	}

	if g, err := ParseAssignPlayer(in.Marshal()); err != nil || g != in {
		t.Errorf("ingame %+v %v", g, err)
	}

	if g, err := ParsePlayerMove(mv.Marshal()); err != nil || g != mv {
		t.Errorf("move %+v %v", g, err)
	}

	if g, err := ParseUnitSkillOnLocation(sk.Marshal()); err != nil || g != sk {
		t.Errorf("skill %+v %v", g, err)
	}

	if g, err := ParseChatMessage(chat.Marshal()); err != nil || g != chat {
		t.Errorf("chat %+v %v", g, err)
	}

	if g, err := ParsePlayerLeave((PlayerLeave{UnitID: 77}).Marshal()); err != nil || g.UnitID != 77 {
		t.Errorf("leave %+v %v", g, err)
	}

	j := JoinGame{Name: "A very long hero name", Class: 3, Level: 80, Difficulty: 2}
	if g, err := ParseJoinGame(j.Marshal()); err != nil || g.Name != "A very long her" || g.Class != 3 {
		t.Errorf("join %+v %v", g, err)
	}
}

func TestTunnel(t *testing.T) {
	data := make([]byte, 5000)
	rand.New(rand.NewSource(3)).Read(data)

	for _, id := range []byte{S2CMetaAE, CtlTunnel} {
		pkts := Tunnel(id, 4, data)
		if len(pkts) < 10 {
			t.Fatalf("expected chunking, got %d packets", len(pkts))
		}

		var a TunnelAssembler

		var got []byte

		for i, p := range pkts {
			if id == S2CMetaAE {
				if err := Validate(ServerToClient, p); err != nil {
					t.Fatalf("chunk %d: %v", i, err)
				}
			}

			typ, d, done, err := a.Add(p)
			if err != nil {
				t.Fatal(err)
			}

			if done != (i == len(pkts)-1) {
				t.Fatalf("chunk %d done=%v", i, done)
			}

			if done {
				got = d
				if typ != 4 {
					t.Errorf("type %d", typ)
				}
			}
		}

		if !reflect.DeepEqual(got, data) {
			t.Error("reassembled data differs")
		}
	}

	if p := Tunnel(S2CMetaAE, 9, nil); len(p) != 1 || len(p[0]) != 5 {
		t.Errorf("empty message: %v", p)
	}
}
