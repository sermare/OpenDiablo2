package d2replay

import (
	"bytes"
	"sort"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type toyUnit struct{ x, y, hp int }

// toySim is a miniature of the monster director: units in a map keyed by id,
// one d2rand stream, moves from inputs. ordered selects the deterministic
// (sorted) iteration the real code uses; unordered iterates the map directly,
// which is the bug class the harness must catch.
type toySim struct {
	rng     *d2rand.Seed
	units   map[uint32]*toyUnit
	ordered bool
}

func newToy(ordered bool) Factory {
	return func(seed uint32) Sim {
		s := &toySim{rng: d2rand.New(seed), units: map[uint32]*toyUnit{}, ordered: ordered}
		for id := uint32(1); id <= 24; id++ {
			s.units[id] = &toyUnit{x: int(id), y: int(id * 3 % 7), hp: 100}
		}

		return s
	}
}

func (s *toySim) ids() []uint32 {
	ids := make([]uint32, 0, len(s.units))
	for id := range s.units {
		ids = append(ids, id)
	}

	if s.ordered {
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}

	return ids
}

func (s *toySim) Step(frame int, inputs []Input) {
	for _, in := range inputs {
		if u := s.units[in.Actor]; u != nil && in.Kind == "move" {
			u.x, u.y = in.X, in.Y
		}
	}

	for _, id := range s.ids() { // each unit rolls once per frame
		u := s.units[id]
		u.x += int(s.rng.Roll(3)) - 1
		u.hp -= int(s.rng.Roll(4))
	}
}

func (s *toySim) Hash(h *Hasher) {
	ids := make([]uint32, 0, len(s.units))
	for id := range s.units {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		u := s.units[id]
		h.U32(id)
		h.I(u.x)
		h.I(u.y)
		h.I(u.hp)
	}
}

func sampleLog(seed uint32) Log {
	rec := NewRecorder(seed)
	rec.Record(Input{Frame: 3, Kind: "move", Actor: 2, X: 10, Y: 11})
	rec.Record(Input{Frame: 20, Kind: "move", Actor: 5, X: -4, Y: 9})

	return rec.Finish(60)
}

func TestSameSeedSameInputsSameHashes(t *testing.T) {
	for _, seed := range []uint32{0, 1, 0xdeadbeef} {
		if err := Verify(newToy(true), sampleLog(seed)); err != nil {
			t.Fatalf("seed %#x: %v", seed, err)
		}
	}
}

func TestDifferentSeedOrInputsDiffer(t *testing.T) {
	base := Run(newToy(true), sampleLog(1))

	if Compare(base, Run(newToy(true), sampleLog(2))) < 0 {
		t.Error("different seeds produced the same trace")
	}

	l := sampleLog(1)
	l.Inputs[1].X++

	if f := Compare(base, Run(newToy(true), l)); f != 20 {
		t.Errorf("changed input at frame 20 first diverges at %d, want 20", f)
	}
}

// The harness must notice map-order dependence (many units, many frames: a
// chance of two runs agreeing by luck is negligible).
func TestHarnessCatchesMapIterationOrder(t *testing.T) {
	if err := Verify(newToy(false), sampleLog(7)); err == nil {
		t.Error("map-iteration-order dependence went undetected")
	}
}

type clockSim struct{ n int64 }

func (c *clockSim) Step(int, []Input) { c.n += time.Now().UnixNano() }
func (c *clockSim) Hash(h *Hasher)    { h.U64(uint64(c.n)) }

func TestHarnessCatchesTimeDependence(t *testing.T) {
	if err := Verify(func(uint32) Sim { return &clockSim{} }, Log{Frames: 3}); err == nil {
		t.Error("time dependence went undetected")
	}
}

func TestLogRoundTrip(t *testing.T) {
	l := sampleLog(9)

	var buf bytes.Buffer
	if err := l.Write(&buf); err != nil {
		t.Fatal(err)
	}

	got, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if Compare(Run(newToy(true), l), Run(newToy(true), got)) >= 0 {
		t.Error("replaying a decoded log differs from the original")
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b []uint64
		want int
	}{
		{nil, nil, -1},
		{[]uint64{1, 2}, []uint64{1, 2}, -1},
		{[]uint64{1, 2}, []uint64{1, 3}, 1},
		{[]uint64{1}, []uint64{1, 2}, 1},
	}

	for _, tt := range tests {
		if got := Compare(tt.a, tt.b); got != tt.want {
			t.Errorf("Compare(%v,%v)=%d want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
