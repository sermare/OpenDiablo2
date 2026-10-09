package d2gs

import (
	"bytes"
	"errors"
	"math/rand"
	"reflect"
	"testing"
)

func TestExpectedSizeServer(t *testing.T) {
	// Values marked (G) were read directly from the dword table at 0x72e958 in Ghidra.
	tests := []struct {
		id   byte
		n    int
		kind SizeKind
	}{
		{0x00, 1, SizeFixed},    // (G)
		{0x01, 8, SizeFixed},    // (G)
		{0x03, 12, SizeFixed},   // (G)
		{0x13, 14, SizeFixed},   // (G)
		{0x16, 0, SizeVariable}, // (G) -1
		{0x17, 0, SizeInvalid},  // (G) 0
		{0x18, 15, SizeFixed},   // (G)
		{0x24, 90, SizeFixed},   // (G)
		{0x28, 103, SizeFixed},  // (G)
		{0x29, 97, SizeFixed},   // (G)
		{0x2b, 0, SizeInvalid},  // (G) 0
		{0x3e, 0, SizeVariable}, // (G)
		{0xae, 0, SizeVariable}, // (G)
		{0xb2, 53, SizeFixed},   // (G)
		{0xb4, 5, SizeFixed},    // (G)
		{0xb5, 0, SizeInvalid},  // > 0xb4
		{0xff, 0, SizeInvalid},
	}
	for _, tc := range tests {
		n, k := ExpectedSize(tc.id, ServerToClient)
		if n != tc.n || k != tc.kind {
			t.Errorf("id %#x: got (%d,%v) want (%d,%v)", tc.id, n, k, tc.n, tc.kind)
		}
	}
}

func TestExpectedSizeClient(t *testing.T) {
	tests := []struct {
		id   byte
		n    int
		kind SizeKind
	}{
		{0x00, 0, SizeInvalid},
		{0x01, 0, SizeUnknown},
		{0x13, 9, SizeFixed}, // (G) handler 0x548990 rejects len != 9
		{0x16, 13, SizeFixed},
		{0x1c, 3, SizeFixed},
		{0x33, 17, SizeFixed}, // (G) handler 0x549960 rejects len != 0x11
		{0x35, 17, SizeFixed}, // (G) 0x5499a0 compares with 0x11
		{0x37, 5, SizeFixed},
		{0x2b, 0, SizeInvalid}, // NULL handler
		{0x4a, 0, SizeInvalid},
		{0x66, 0, SizeUnknown},
		{0x67, 0, SizeInvalid}, // control lane, not an in-game id
		{0xfa, 0, SizeInvalid},
	}
	for _, tc := range tests {
		n, k := ExpectedSize(tc.id, ClientToServer)
		if n != tc.n || k != tc.kind {
			t.Errorf("id %#x: got (%d,%v) want (%d,%v)", tc.id, n, k, tc.n, tc.kind)
		}
	}
}

// sample builds a valid server packet for id.
func sampleServer(id byte, rng *rand.Rand) []byte {
	n, k := ExpectedSize(id, ServerToClient)
	if k == SizeFixed {
		b := make([]byte, n)
		rng.Read(b)
		b[0] = id
		return b
	}
	var b []byte
	switch id {
	case 0x16:
		b = make([]byte, 20)
		b[1], b[2] = 20, 0
	case 0x26:
		b = make([]byte, 10)
		b = append(b, []byte("bob\x00hello\x00")...)
	case 0x3e:
		b = make([]byte, 6)
		b[1] = 6
	case 0x5b:
		b = make([]byte, 40)
		b[1], b[2] = 40, 0
	case 0x94:
		b = make([]byte, 12) // (2+2)*3
		b[1] = 2
	case 0x9c, 0x9d:
		b = make([]byte, 14)
		b[2] = 14
	case 0xa6:
		b = make([]byte, 9)
		b[2], b[3] = 9, 0
	case 0xa8, 0xaa:
		b = make([]byte, 15)
		b[6] = 15
	case 0xac:
		b = make([]byte, 20)
		b[0xc] = 20
	case 0xae:
		b = make([]byte, 3+5)
		b[1], b[2] = 5, 0
	case 0xaf:
		b = []byte{0, 0}
	case 0xb3:
		b = make([]byte, 7+4)
		b[1] = 4
	default:
		panic("no sample")
	}
	b[0] = id
	return b
}

func allServerIDs() []byte {
	var ids []byte
	for id := 0; id <= MaxServerID; id++ {
		if _, k := ExpectedSize(byte(id), ServerToClient); k != SizeInvalid {
			ids = append(ids, byte(id))
		}
	}
	return ids
}

func TestServerVariableLengths(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, id := range allServerIDs() {
		b := sampleServer(id, rng)
		if err := Validate(ServerToClient, b); err != nil {
			t.Errorf("id %#x sample invalid: %v", id, err)
		}
	}
	// 0xaf: nonzero byte@1 -> byte@1+1.
	if n, _, err := serverLength([]byte{0xaf, 4, 0, 0, 0}); err != nil || n != 5 {
		t.Errorf("0xaf: n=%d err=%v", n, err)
	}
	// 0xae clamps >0x1fd payload to 0 -> length 3.
	if n, _, err := serverLength([]byte{0xae, 0xff, 0xff}); err != nil || n != 3 {
		t.Errorf("0xae clamp: n=%d err=%v", n, err)
	}
	// 0x26 = len1 + len2 + 12.
	b := sampleServer(0x26, rand.New(rand.NewSource(1)))
	if n, _, _ := serverLength(b); n != 3+5+12 {
		t.Errorf("0x26 len %d", n)
	}
	// Bad: length below header, above max.
	for _, bad := range [][]byte{
		{0x3e, 1}, {0x9c, 0, 0xff, 0}, {0xa8, 0, 0, 0, 0, 0, 3}, {0x16, 0, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	} {
		if _, _, err := serverLength(bad); err == nil {
			// 0x9c with 0xff is legal (255); skip that one.
			if bad[0] == 0x9c {
				continue
			}
			t.Errorf("expected error for % x", bad)
		}
	}
}

func TestDecoderStreamAnySplit(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	ids := allServerIDs()
	var want [][]byte
	var stream []byte
	for i := 0; i < 400; i++ {
		p := sampleServer(ids[rng.Intn(len(ids))], rng)
		want = append(want, p)
		stream = append(stream, p...)
	}
	for _, chunk := range []int{1, 2, 3, 7, 64, 513, len(stream)} {
		d := NewDecoder(ServerToClient)
		var got [][]byte
		for off := 0; off < len(stream); off += chunk {
			end := off + chunk
			if end > len(stream) {
				end = len(stream)
			}
			d.Write(stream[off:end])
			for {
				p, ok, err := d.Next()
				if err != nil {
					t.Fatalf("chunk %d: %v", chunk, err)
				}
				if !ok {
					break
				}
				got = append(got, p.Data)
			}
		}
		if !reflect.DeepEqual(got, want) || d.Buffered() != 0 {
			t.Fatalf("chunk %d: mismatch (%d vs %d packets, %d left)", chunk, len(got), len(want), d.Buffered())
		}
	}
}

// Every truncation point of a valid stream must yield a prefix of the packets,
// never an error or panic, and the remainder must be the partial tail.
func TestDecoderTruncation(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	ids := allServerIDs()
	var lens []int
	var stream []byte
	for i := 0; i < 60; i++ {
		p := sampleServer(ids[rng.Intn(len(ids))], rng)
		stream = append(stream, p...)
		lens = append(lens, len(p))
	}
	for cut := 0; cut <= len(stream); cut++ {
		pkts, rest, err := SplitAll(ServerToClient, stream[:cut])
		if err != nil {
			t.Fatalf("cut %d: %v", cut, err)
		}
		total := 0
		for _, p := range pkts {
			total += len(p.Data)
		}
		if total+len(rest) != cut {
			t.Fatalf("cut %d: %d + %d", cut, total, len(rest))
		}
		// total must be the largest packet boundary <= cut.
		bound, acc := 0, 0
		for _, l := range lens {
			if acc+l > cut {
				break
			}
			acc += l
			bound = acc
		}
		if total != bound {
			t.Fatalf("cut %d: consumed %d want %d", cut, total, bound)
		}
	}
}

func TestDecoderErrors(t *testing.T) {
	tests := []struct {
		name string
		dir  Direction
		in   []byte
		want error
	}{
		{"s2c invalid id", ServerToClient, []byte{0x17, 0, 0}, ErrInvalidID},
		{"s2c id too high", ServerToClient, []byte{0xb5}, ErrInvalidID},
		{"s2c bad variable", ServerToClient, []byte{0x3e, 1}, ErrBadLength},
		{"c2s unknown size", ClientToServer, []byte{0x01, 0, 0, 0, 0}, ErrUnknownSize},
		{"c2s control id", ClientToServer, []byte{0x67}, ErrInvalidID},
		{"c2s zero id", ClientToServer, []byte{0x00}, ErrInvalidID},
	}
	for _, tc := range tests {
		d := NewDecoder(tc.dir)
		d.Write(tc.in)
		_, ok, err := d.Next()
		if ok || !errors.Is(err, tc.want) {
			t.Errorf("%s: ok=%v err=%v want %v", tc.name, ok, err, tc.want)
		}
		// Sticky until Reset.
		if _, _, err2 := d.Next(); err2 == nil {
			t.Errorf("%s: error not sticky", tc.name)
		}
		d.Reset()
		if _, _, err2 := d.Next(); err2 != nil {
			t.Errorf("%s: error after Reset: %v", tc.name, err2)
		}
	}
}

func TestDecoderRandomBytesNeverPanics(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for i := 0; i < 20000; i++ {
		b := make([]byte, rng.Intn(80))
		rng.Read(b)
		for _, dir := range []Direction{ServerToClient, ClientToServer} {
			pkts, rest, _ := SplitAll(dir, b)
			total := len(rest)
			for _, p := range pkts {
				total += len(p.Data)
				if len(p.Data) > MaxPacketSize || len(p.Data) == 0 {
					t.Fatalf("bad packet len %d", len(p.Data))
				}
			}
			if total != len(b) {
				t.Fatalf("lost bytes: %d vs %d", total, len(b))
			}
		}
	}
}

func FuzzDecoder(f *testing.F) {
	f.Add([]byte{0x01, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{0x9c, 0, 5, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) {
		for _, dir := range []Direction{ServerToClient, ClientToServer} {
			SplitAll(dir, b)
		}
	})
}

func TestClientMessagesRoundTrip(t *testing.T) {
	msgs := []Message{
		&InteractUnit{1, 0xdeadbeef},
		&PickUpUnit{4, 77, 1},
		&DropCursorItem{9},
		&ItemToContainer{1, 2, 3, 4},
		&PickFromContainer{5},
		&ItemToBody{6, 7},
		&SwapTwoHandedBody{6, 8},
		&PickFromBody{3},
		&SwapBodyItem{6, 9},
		&SwapWeaponHands{6, 4},
		&SwapContainerItem{1, 2, 3, 4},
		&UseItem{1, 2, 3},
		&StackItems{1, 2},
		&UnstackItem{1},
		&ItemToBelt{1, 2},
		&PickFromBelt{1},
		&SwapBeltItem{1, 2},
		&UseBeltItem{1, 2, 3},
		&IdentifyItem{1, 2},
		&InsertSocketItem{1, 2},
		&ScrollToTome{1, 2},
		&ItemToCube{1, 2},
		&NpcInit{1, 2},
		&NpcCancel{1, 2},
		&QuestMessage{1, 2},
		&BuyItem{1, 2, 0x80000000, 1500},
		&SellItem{1, 2, 3, 1500},
		&IdentifyItems{1},
		&RepairItem{1, 2, 3, 99},
		&HireMercenary{1, 2},
		&GambleItem{1},
	}
	seen := map[byte]bool{}
	for _, m := range msgs {
		b := m.MarshalPacket()
		if b[0] != m.PacketID() {
			t.Errorf("%T id mismatch", m)
		}
		if n, k := ExpectedSize(b[0], ClientToServer); k != SizeFixed || n != len(b) {
			t.Errorf("%T len %d vs table (%d,%v)", m, len(b), n, k)
		}
		if err := Validate(ClientToServer, b); err != nil {
			t.Errorf("%T: %v", m, err)
		}
		got, err := DecodeClient(b)
		if err != nil {
			t.Errorf("%T: %v", m, err)
			continue
		}
		if !reflect.DeepEqual(got, m) {
			t.Errorf("%T round trip: got %+v want %+v", m, got, m)
		}
		seen[b[0]] = true
	}
	// Every verified fixed client id has a typed message.
	for id := range clientSizes {
		if !seen[id] {
			t.Errorf("client id %#x has no typed message", id)
		}
	}
}

func TestClientMessageLayout(t *testing.T) {
	got := (&ItemToContainer{ItemID: 0x01020304, X: 5, Y: 6, Page: 4}).MarshalPacket()
	want := []byte{0x18, 4, 3, 2, 1, 5, 0, 0, 0, 6, 0, 0, 0, 4, 0, 0, 0}
	if !bytes.Equal(got, want) {
		t.Errorf("% x vs % x", got, want)
	}
	got = (&SellItem{NpcID: 1, ItemID: 2, Tab: 0x0304, Cost: 9}).MarshalPacket()
	want = []byte{0x33, 1, 0, 0, 0, 2, 0, 0, 0, 4, 3, 0, 0, 9, 0, 0, 0}
	if !bytes.Equal(got, want) {
		t.Errorf("% x vs % x", got, want)
	}
}

func TestClientDecodeTruncation(t *testing.T) {
	for id := range clientSizes {
		n := clientSizes[id]
		full := make([]byte, n)
		full[0] = id
		if _, err := DecodeClient(full); err != nil {
			t.Errorf("id %#x full: %v", id, err)
		}
		for cut := 0; cut < n; cut++ {
			if _, err := DecodeClient(full[:cut]); err == nil {
				t.Errorf("id %#x cut %d: expected error", id, cut)
			}
		}
		if _, err := DecodeClient(append(full, 0)); err == nil {
			t.Errorf("id %#x oversize: expected error", id)
		}
	}
	if _, err := DecodeClient([]byte{0x01, 0, 0, 0, 0}); !errors.Is(err, ErrUnknownSize) {
		t.Errorf("untyped id: %v", err)
	}
}

func TestEncoderChunks(t *testing.T) {
	e := NewEncoder(ServerToClient)
	p := make([]byte, 90)
	p[0] = 0x24
	for i := 0; i < 10; i++ { // 900 bytes: 5 per chunk (450), then 5
		if err := e.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	cs := e.Chunks()
	if len(cs) != 2 || len(cs[0].Data) != 450 || len(cs[1].Data) != 450 {
		t.Fatalf("chunks: %d", len(cs))
	}
	out := e.Flush()
	if len(out) != 900 || len(e.Chunks()) != 0 {
		t.Fatal("flush")
	}
	pkts, rest, err := SplitAll(ServerToClient, out)
	if err != nil || len(pkts) != 10 || len(rest) != 0 {
		t.Fatalf("decode: %d %d %v", len(pkts), len(rest), err)
	}
	// A packet that no longer fits starts a new chunk.
	e = NewEncoder(ServerToClient)
	big := make([]byte, 0x1f0)
	big[0], big[1], big[2] = 0x16, 0xf0, 0x01 // u16 length 0x1f0
	_ = e.Add(big)
	_ = e.Add(p)
	if len(e.Chunks()) != 2 {
		t.Errorf("want 2 chunks, got %d", len(e.Chunks()))
	}
	// Invalid packets are rejected.
	if err := e.Add([]byte{0x01, 0}); !errors.Is(err, ErrWrongSize) {
		t.Errorf("wrong size: %v", err)
	}
	if err := e.Add([]byte{0x17}); !errors.Is(err, ErrInvalidID) {
		t.Errorf("bad id: %v", err)
	}
	if err := e.Add(make([]byte, MaxPacketSize+1)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("too large: %v", err)
	}
}

func TestFrameLength(t *testing.T) {
	for _, n := range []int{0, 1, 0xef, 0xf0, 0x100, 0x204, 0xfff} {
		b, err := AppendFrameLength(nil, n)
		if err != nil {
			t.Fatal(err)
		}
		wantPrefix := 1
		if n >= 0xf0 {
			wantPrefix = 2
		}
		got, prefix := ReadFrameLength(b)
		if got != n || prefix != wantPrefix {
			t.Errorf("n=%d: got (%d,%d) bytes % x", n, got, prefix, b)
		}
		if prefix == 2 {
			if _, p := ReadFrameLength(b[:1]); p != 0 {
				t.Errorf("n=%d: partial prefix accepted", n)
			}
		}
	}
	if _, err := AppendFrameLength(nil, 0x1000); err == nil {
		t.Error("expected error")
	}
}

func TestPacketName(t *testing.T) {
	if n, v := PacketName(0x32, ClientToServer); n != "BuyItem" || !v {
		t.Errorf("%s %v", n, v)
	}
	if n, v := PacketName(0x0f, ServerToClient); n != "PlayerMove" || v {
		t.Errorf("%s %v", n, v)
	}
	if n, _ := PacketName(0xee, ServerToClient); n != "UnknownEE" {
		t.Error(n)
	}
}
