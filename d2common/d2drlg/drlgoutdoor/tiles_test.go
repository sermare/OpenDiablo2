package drlgoutdoor

import (
	"encoding/binary"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// synthDT1 builds a version 7 DT1 with the given tile headers (orient, style, seq, rarity).
func synthDT1(t *testing.T, name string, tiles [][4]int32) *DT1 {
	t.Helper()

	b := make([]byte, 0x114+0x60*len(tiles))
	binary.LittleEndian.PutUint32(b[0:], 7)
	binary.LittleEndian.PutUint32(b[4:], 6)
	binary.LittleEndian.PutUint32(b[0x10c:], uint32(len(tiles)))
	binary.LittleEndian.PutUint32(b[0x110:], 0x114)

	for i, tl := range tiles {
		h := b[0x114+0x60*i:]
		binary.LittleEndian.PutUint32(h[0x14:], uint32(tl[0]))
		binary.LittleEndian.PutUint32(h[0x18:], uint32(tl[1]))
		binary.LittleEndian.PutUint32(h[0x1c:], uint32(tl[2]))
		binary.LittleEndian.PutUint32(h[0x20:], uint32(tl[3]))
	}

	d, err := ParseDT1(name, b)
	if err != nil {
		t.Fatal(err)
	}

	return d
}

func TestDT1QueryOrder(t *testing.T) {
	// three tiles with one key: a query lists the most recently loaded first
	a := synthDT1(t, "a.dt1", [][4]int32{{0, 5, 1, 1}, {0, 5, 1, 2}, {0, 6, 1, 7}})
	b := synthDT1(t, "b.dt1", [][4]int32{{0, 5, 1, 4}})
	lb := &Library{slots: []*DT1{a, b}}

	got := lb.query(0, 5, 1, 40)
	want := []struct {
		f string
		i int
	}{{"a.dt1", 1}, {"a.dt1", 0}, {"b.dt1", 0}}

	if len(got) != len(want) {
		t.Fatalf("got %d tiles, want %d", len(got), len(want))
	}

	for i, w := range want {
		if got[i].File() != w.f || got[i].Idx != w.i {
			t.Errorf("result %d: %s/%d, want %s/%d", i, got[i].File(), got[i].Idx, w.f, w.i)
		}
	}

	if n := len(lb.query(0, 5, 1, 2)); n != 2 {
		t.Errorf("limit 2 returned %d tiles", n)
	}

	if n := len(lb.query(3, 5, 1, 40)); n != 0 {
		t.Errorf("unknown orientation returned %d tiles", n)
	}
}

func TestPickWeights(t *testing.T) {
	// rarities 1, 3 in query order (sum 4, a power of two: the roll is a mask)
	d := synthDT1(t, "w.dt1", [][4]int32{{0, 1, 0, 3}, {0, 1, 0, 1}})
	lb := &Library{slots: []*DT1{d}}
	// query order is tile 1 (rarity 1), tile 0 (rarity 3)

	counts := map[int]int{}

	for s := uint32(1); s <= 4000; s++ {
		var seed d2rand.Seed
		seed.Init(s)

		tb := &tileBuilder{rt: &RoomTiles{Lib: lb}, rs: &seed}
		ref := tb.pick(0, 1<<20|0) // style 1, sequence 0
		counts[ref.Idx]++
	}

	if counts[1] < 800 || counts[1] > 1200 || counts[0] < 2800 {
		t.Errorf("weights 1:3 not respected: %v", counts)
	}
}

func TestPickFallbackAndSingle(t *testing.T) {
	d := synthDT1(t, "f.dt1", [][4]int32{{10, 0, 0, 1}, {0, 2, 0, 5}})
	lb := &Library{slots: []*DT1{d}}

	var seed d2rand.Seed
	seed.Init(7)

	before := seed
	tb := &tileBuilder{rt: &RoomTiles{Lib: lb}, rs: &seed}

	// no tile for (orientation 0, style 9): falls back to orientation 10, no roll
	if ref := tb.pick(0, 9<<20); ref.Idx != 0 {
		t.Errorf("fallback tile %d, want 0", ref.Idx)
	}

	if seed != before {
		t.Error("fallback consumed a room-seed step")
	}

	// a single match still rolls once (sum 5)
	if ref := tb.pick(0, 2<<20); ref.Idx != 1 {
		t.Errorf("single tile %d, want 1", ref.Idx)
	}

	if seed == before {
		t.Error("a pick with matches must consume a room-seed step")
	}
}

func TestRecordFlags(t *testing.T) {
	// a floor cell 0x112008c2: bit 7 set (flag 1), bit 28 (0x102), (v>>18)&3 = 0 -> 0x4000
	if got := flagsCommon(((0x112008c2>>0x12)&3)*0x4000+0x4000, 0x112008c2, 0); got != 0x4103 {
		t.Errorf("flags %#x, want 0x4103", got)
	}
	// ring floor 0x40006: bit 2 -> 0x2000, (v>>18)&3 = 1 -> 0x8000
	if got := flagsCommon(((0x40006>>0x12)&3)*0x4000+0x4000, 0x40006, 0); got != 0xa000 {
		t.Errorf("flags %#x, want 0xa000", got)
	}
}
