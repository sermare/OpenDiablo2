package d2object

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type openField struct{}

func (openField) Blocked(_, _, _, _, _ int) bool { return false }

type wallField struct{}

func (wallField) Blocked(_, _, _, _, _ int) bool { return true }

func testScatter(col Collider, defs []Def, groups map[int]Group) *Scatter {
	return &Scatter{Defs: defs, Groups: groups, Game: d2rand.New(77), Region: &Region{Total: 16}, Col: col}
}

func room(seed uint32, w, h int) *RoomCtx {
	return &RoomCtx{Rect: Rect{X: 100, Y: 200, W: w, H: h}, Seed: d2rand.New(seed)}
}

func oneGroup(id, density, prob int) map[int]Group {
	var g Group
	g.Members[0] = GroupMember{ID: id, Density: density, Prob: prob}

	return map[int]Group{1: g}
}

func TestSpawnCountFormula(t *testing.T) {
	for _, c := range []struct{ w, h, d, want int }{{64, 64, 128, 16}, {40, 40, 30, 1}, {10, 10, 128, 0}, {80, 80, 0, 0}} {
		if got := count(Rect{W: c.w, H: c.h}, c.d); got != c.want {
			t.Errorf("count(%d,%d,%d)=%d want %d", c.w, c.h, c.d, got, c.want)
		}
	}
}

func TestSlotRollSequence(t *testing.T) {
	// eight slot rolls and one member roll come off the room seed, then one step per object made
	defs := make([]Def, 10)
	defs[5] = Def{ID: 5, PopulateFn: 3, SizeX: 2, SizeY: 2}
	s := testScatter(openField{}, defs, oneGroup(5, 127, 100))
	lv := LevelRow{ID: 3, Groups: [8]int{1}, Probs: [8]int{100}}
	rc := room(9, 64, 64)
	n, err := s.Room(lv, rc)

	if err != nil || n == 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}

	ref := d2rand.New(9)
	for i := 0; i < 8; i++ {
		ref.Roll(100)
	}

	ref.Roll(100)

	for i := 0; i < n; i++ {
		ref.Step()
	}

	if rc.Seed.Lo != ref.Lo || rc.Seed.Hi != ref.Hi {
		t.Fatalf("room seed stream differs after %d objects", n)
	}
}

func TestDensityAClusterBounds(t *testing.T) {
	defs := make([]Def, 10)
	defs[5] = Def{ID: 5, PopulateFn: 3, SizeX: 1, SizeY: 1}
	s := testScatter(openField{}, defs, oneGroup(5, 127, 100))
	lv := LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}
	max := count(Rect{W: 64, H: 64}, 127)

	for seed := uint32(1); seed < 40; seed++ {
		s.Made = nil
		s.Game = d2rand.New(seed)

		if _, err := s.Room(lv, room(seed, 64, 64)); err != nil {
			t.Fatal(err)
		}

		if len(s.Made) != max {
			t.Fatalf("seed %d made %d, want %d on open ground", seed, len(s.Made), max)
		}

		for _, p := range s.Made {
			if p.X <= 100 || p.X >= 100+64-1 || p.Y <= 200 || p.Y >= 200+64-1 {
				t.Fatalf("object outside the room margin: %+v", p)
			}
		}
	}
}

func TestBlockedGroundMakesNothing(t *testing.T) {
	defs := make([]Def, 10)
	defs[5] = Def{ID: 5, PopulateFn: 3}
	s := testScatter(wallField{}, defs, oneGroup(5, 127, 100))

	if n, _ := s.Room(LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}, room(3, 64, 64)); n != 0 {
		t.Fatalf("made %d on blocked ground", n)
	}
}

func TestDensityFatal(t *testing.T) {
	defs := make([]Def, 10)
	defs[5] = Def{ID: 5, PopulateFn: 3}
	s := testScatter(openField{}, defs, oneGroup(5, 0x80, 100))

	if _, err := s.Room(LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}, room(3, 64, 64)); !errors.Is(err, ErrDensity) {
		t.Fatalf("err=%v", err)
	}
}

func TestNoObjectsRoomAndEmptyHandler(t *testing.T) {
	defs := make([]Def, 10)
	defs[5] = Def{ID: 5, PopulateFn: 0}
	s := testScatter(openField{}, defs, oneGroup(5, 10, 100))
	lv := LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}

	if n, _ := s.Room(lv, room(3, 64, 64)); n != 0 {
		t.Fatal("empty handler slot must spawn nothing")
	}

	rc := room(3, 64, 64)
	rc.NoObjects = true
	seed := *rc.Seed

	if n, _ := s.Room(lv, rc); n != 0 || *rc.Seed != seed {
		t.Fatal("a NoObjects room must not touch its seed")
	}
}

func TestSparseForcesSlotRoll(t *testing.T) {
	// a sparse level forces r = 100, so ObjPrb 99 never fires while ObjPrb 100 does
	defs := make([]Def, 10)
	defs[1] = Def{ID: 1, SubClass: 1} // the objects.txt row indexed by the GROUP id
	defs[5] = Def{ID: 5, PopulateFn: 3}
	s := testScatter(openField{}, defs, oneGroup(5, 127, 100))
	s.Region = &Region{Total: 4, Seen: 4} // (4<<7)/4 = 128 > 96, no shrine yet

	if !s.Region.Sparse() {
		t.Fatal("expected a sparse level")
	}

	if n, _ := s.Room(LevelRow{Groups: [8]int{1}, Probs: [8]int{99}}, room(5, 64, 64)); n != 0 {
		t.Fatalf("prob 99 fired on a sparse level: %d", n)
	}

	s.Region = &Region{Total: 4, Seen: 4}

	if n, _ := s.Room(LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}, room(5, 64, 64)); n == 0 {
		t.Fatal("prob 100 must still fire")
	}
}

func TestShrineQuotas(t *testing.T) {
	defs := make([]Def, 10)
	defs[5] = Def{ID: 5, PopulateFn: 2, SizeX: 1, SizeY: 1}
	s := testScatter(openField{}, defs, oneGroup(5, 5, 100))
	s.Region = &Region{Total: 24} // quota 24/8 = 3
	lv := LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}

	for i := 0; i < 60; i++ {
		s.Room(lv, room(uint32(i+1), 200, 200))
	}

	if got := len(s.Region.Large); got > 4 || got == 0 {
		t.Fatalf("large list %d, want 1..4 (Total/8 + 1)", got)
	}

	for i, a := range s.Region.Large {
		for _, b := range s.Region.Large[:i] {
			if abs(a[0]-b[0]) < 50 || abs(a[1]-b[1]) < 50 {
				t.Fatalf("shrines %v %v closer than 50", a, b)
			}
		}
	}
}

func TestGroupByClassClusters(t *testing.T) {
	defs := make([]Def, 10)
	defs[3] = Def{ID: 3, PopulateFn: 1, SizeX: 1, SizeY: 1} // barrel
	s := testScatter(openField{}, defs, oneGroup(3, 40, 100))
	lv := LevelRow{Groups: [8]int{1}, Probs: [8]int{100}}
	total := 0

	for seed := uint32(1); seed < 30; seed++ {
		s.Made = nil
		s.Room(lv, room(seed, 64, 64))
		total += len(s.Made)
	}

	if total == 0 {
		t.Fatal("no barrels placed")
	}
}

func TestThemesDrawOnly(t *testing.T) {
	s := testScatter(openField{}, make([]Def, 10), map[int]Group{})
	before := *s.Game
	s.Region = &Region{Total: 10, Seen: 9}
	s.Room(LevelRow{Themes: 60}, room(1, 40, 40))

	if *s.Game == before {
		t.Fatal("a Themes level must draw from the game seed")
	}

	if len(s.Made) != 0 {
		t.Fatal("themes never make objects")
	}
}

// TestRealTablesCounts runs every objgroup over many rooms with the real tables and checks that the totals stay
// within the density bounds. Skipped without D2_TABLES.
func TestRealTablesCounts(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	od, err1 := os.ReadFile(filepath.Join(dir, "objects.txt"))
	gd, err2 := os.ReadFile(filepath.Join(dir, "objgroup.txt"))

	if err1 != nil || err2 != nil {
		t.Skip("tables not readable")
	}

	defs, err := ParseObjects(od)
	if err != nil {
		t.Fatal(err)
	}

	gs, err := ParseGroups(gd)
	if err != nil {
		t.Fatal(err)
	}

	groups := map[int]Group{}
	for _, g := range gs {
		groups[g.Offset] = g
	}

	s := &Scatter{Defs: defs, Groups: groups, Col: openField{}}

	for gid, g := range groups {
		if gid == 0 {
			continue
		}

		bound := 1

		for _, m := range g.Members {
			if m.ID == 0 {
				continue
			}

			if m.Density >= 0x80 {
				t.Errorf("group %d member %d density %d is fatal in the exe", gid, m.ID, m.Density)
			}

			if m.ID >= len(defs) || defs[m.ID].PopulateFn >= 10 {
				t.Errorf("group %d member %d has no handler", gid, m.ID)
			}

			if b := count(Rect{W: 64, H: 64}, m.Density); b > bound {
				bound = b
			}
		}

		s.Game = d2rand.New(uint32(gid))
		s.Region = &Region{Total: 200}
		s.Made = nil
		lv := LevelRow{Groups: [8]int{gid}, Probs: [8]int{100}}

		for i := 0; i < 60; i++ {
			before := len(s.Made)

			if _, err := s.Room(lv, room(uint32(i*7919+gid), 64, 64)); err != nil {
				t.Fatal(err)
			}

			// handler 1 grows clusters (up to tries x cluster), handler 6 adds a twin: generous cap
			if made := len(s.Made) - before; made > 12*bound+4 {
				t.Fatalf("group %d room %d made %d objects, bound %d", gid, i, made, 12*bound+4)
			}
		}
	}
}
