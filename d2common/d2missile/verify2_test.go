package d2missile

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// UNIT_GetDistanceToUnit (0x642b10): subtile distance max + min/2, unit sizes
// subtracted (half each), used by the Guided Arrow 4..24 window and by the
// retarget 'closer than 25' test.
func TestDistanceIsMaxPlusHalfMin(t *testing.T) {
	for _, tc := range []struct {
		x2, y2 float64
		s1, s2 int
		want   int
	}{
		{3, 4, 0, 0, 5},       // (3 + 2*4) / 2
		{10, 0, 0, 0, 10},     // straight
		{20, 10, 0, 0, 25},    // euclid would be 22
		{-6, -8, 0, 0, 11},    // symmetric in sign: (6 + 16) / 2
		{10, 0, 0, 4, 8},      // target size 4 -> 2 off each axis, floored at 0
		{10, 10, 2, 2, 12},    // both sizes: 1 off each, then 2 off: (8 + 16) / 2
		{0.9, 0.9, 0, 0, 0},   // same subtile
		{1.1, 0.9, 0, 0, 1},   // dx 1, dy 0: (0 + 2) / 2
		{24.5, 3.5, 0, 0, 25}, // just outside the Guided Arrow window
		{23.5, 3.5, 0, 0, 24}, // just inside
	} {
		got := Distance(0, 0, tc.s1, tc.x2, tc.y2, tc.s2)
		if got != tc.want {
			t.Errorf("Distance(0,0 -> %v,%v sizes %d,%d) = %d, want %d", tc.x2, tc.y2, tc.s1, tc.s2, got, tc.want)
		}
	}
}

// The homing window is max+min/2, not euclidean: a target 24.2 away in a
// straight line by euclid but 25 by the exe's metric is not steered at.
func TestGuidedArrowWindowUsesTheExeMetric(t *testing.T) {
	for _, tc := range []struct {
		x, y  float64
		turns bool
	}{
		{23.5, 3.5, true},  // octile 24
		{24.5, 3.5, false}, // octile 25 although euclid is 24.7
	} {
		w := newWorld()
		goal := &posTarget{fakeTarget: newTarget("g", int(tc.x), int(tc.y)), px: tc.x, py: tc.y}
		w.targets = []*fakeTarget{goal.fakeTarget}
		s := NewSim(w, nil)
		m, _ := s.Create(CreateParams{Spec: guidedSpec(), Level: 1, X: 0.5, Y: 0.5, DestX: 40, DestY: 0.5, Home: goal})
		dy := m.DY
		m.Life = 100 // a multiple of Param1 = 5: the re-aim frame

		if !s.guidedTurn(m) {
			t.Fatal("vanished")
		}

		turned := m.DY != dy
		if turned != tc.turns {
			t.Errorf("target (%v,%v): turned=%v want %v", tc.x, tc.y, turned, tc.turns)
		}
	}
}

type serialTarget struct {
	*posTarget
	serial int
}

func (s serialTarget) Serial() int { return s.serial }

// The re-target callback 0x569a40 keeps the candidate with the lowest unit id,
// not the nearest one.
func TestGuidedArrowRetargetPicksLowestUnitID(t *testing.T) {
	base := newWorld()
	near := serialTarget{&posTarget{fakeTarget: newTarget("near", 12, 1), px: 12.5, py: 1.5}, 900}
	old := serialTarget{&posTarget{fakeTarget: newTarget("old", 14, 8), px: 14.5, py: 8.5}, 7}
	fw := &finderWorld{fakeWorld: base, found: near, more: []Target{old}}
	base.targets = []*fakeTarget{near.fakeTarget, old.fakeTarget}
	s := NewSim(fw, nil)
	sp := guidedSpec()
	sp.Range = 40

	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 10, DestY: 0, ClampToDest: true})

	for i := 0; i < 50 && m.HomeMode != 5; i++ {
		base.frame++
		s.Step()
	}

	if m.HomeMode != 5 || m.Home == nil || m.Home.ID() != "old" {
		t.Fatalf("mode %d home %v, want the lowest id (old)", m.HomeMode, m.Home)
	}

	if fw.gotR != 15 {
		t.Errorf("search radius %d, want Param2 = 15", fw.gotR)
	}
}

// 0x6513e0 reads the cell through the missile's block mask: bits outside the
// type's mask are invisible, so a moving missile flies over them.
func TestMovingMissileIgnoresCellBitsOutsideItsMask(t *testing.T) {
	for _, tc := range []struct {
		ct   int
		bit  uint16
		blk  bool
		name string
	}{
		{3, d2path.FlagWalk, false, "walk bit, type 3 (0x184)"},
		{1, d2path.FlagWalk, false, "walk bit, type 1 (0x84)"},
		{6, d2path.FlagWalk, false, "walk bit, type 6 (0x4)"},
		{8, d2path.FlagWalk, true, "walk bit, type 8 (0x185)"},
		{7, d2path.FlagWall, false, "wall bit, type 7 (0x40)"},
		{3, d2path.FlagWall, true, "wall bit, type 3"},
	} {
		w := newWorld()
		w.grid.Set(5, 0, tc.bit)
		sp := fireBolt()
		sp.CollideType = tc.ct
		s := NewSim(w, nil)
		m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})
		k := kinds(run(s, w, 12))

		if blocked := k[EventWall] == 1; blocked != tc.blk || k[EventVanish] != 0 {
			t.Errorf("%s: events %v blocked=%v want %v", tc.name, k, blocked, tc.blk)
		}

		if !tc.blk && m.X < 6 {
			t.Errorf("%s: stopped at %v", tc.name, m.X)
		}
	}
}

type alignedTarget struct {
	*fakeTarget
	aligned bool
}

func (a alignedTarget) PlayerAligned() bool { return a.aligned }

type alignedWorld struct {
	*fakeWorld
	at alignedTarget
}

func (w *alignedWorld) IsEnemy(Owner, Target) bool { return true }

func (w *alignedWorld) Targets(x, y int) []Target {
	if x == w.at.x && y == w.at.y {
		return []Target{w.at}
	}

	return nil
}

// CollideType 1 (predicate 0x5a6210): players, and monsters that carry state
// 105 with stat 172 == 2; other monsters are ignored. Type 3 hits every unit.
func TestCollideTypeOneTakesAlignedMonsters(t *testing.T) {
	for _, tc := range []struct {
		ct      int
		aligned bool
		hit     bool
	}{{1, false, false}, {1, true, true}, {3, false, true}} {
		base := newWorld()
		mon := alignedTarget{newTarget("m", 8, 0), tc.aligned}
		w := &alignedWorld{fakeWorld: base, at: mon}
		sp := fireBolt()
		sp.CollideType = tc.ct
		s := NewSim(w, nil)
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

		k := kinds(run(s, base, 15))
		if (k[EventHit] == 1) != tc.hit {
			t.Errorf("type %d aligned=%v: events %v", tc.ct, tc.aligned, k)
		}
	}
}

// SrvDoFunc 2 / 6 (0x5abf30 / 0x5ac1b0): SubMissile1 is created at the
// missile's subtile once per subtile entered, one frame late (path flag 8 is
// from the previous step); 6 also destroys a missile without SubMissile1 or
// owner at once, with no hit function.
func TestSrvDoFuncTwoAndSixSpawnTheTrail(t *testing.T) {
	for _, fn := range []int{2, 6} {
		w := newWorld()
		trail := &Spec{ID: 2, Name: "trail", Range: 30, CollideType: 3}
		tbl := specs{"trail": trail}
		jav := &Spec{ID: 1, Name: "jav", SrvDoFunc: fn, Vel: 16, MaxVel: 16, Range: 20, CollideType: 3,
			SubMissile: [3]string{"trail"}}
		s := NewSim(w, tbl)
		evs := collect(s)

		m, _ := s.Create(CreateParams{Spec: jav, Level: 1, SkillID: 7, X: 0.5, Y: 0.5, DestX: 60, DestY: 0.5})

		for i := 0; i < 10; i++ {
			w.frame++
			s.Step()
		}

		created := 0

		for _, e := range *evs {
			if e.Kind == EventCreate && e.Missile.Spec.Name == "trail" {
				created++

				if e.Missile.SkillID != 7 || e.Missile.Level != 1 {
					t.Errorf("func %d: trail skill/level %d/%d", fn, e.Missile.SkillID, e.Missile.Level)
				}

				if x := e.Missile.X; x != float64(int(x))+0.5 {
					t.Errorf("func %d: trail not at a subtile centre: %v", fn, x)
				}
			}
		}

		// 16*0.75/16 = 0.75 subtile a frame: 10 frames cross 7 subtile borders, the
		// last one is seen a frame later
		if want := int(m.X-0.5) - 0; created < want-1 || created > want+1 || created < 5 {
			t.Errorf("func %d: %d trail missiles for a missile at x=%v", fn, created, m.X)
		}
	}

	// func 6 without SubMissile1: destroyed at once, no hit function
	w := newWorld()
	bad := &Spec{ID: 3, Name: "maker", SrvDoFunc: 6, Vel: 16, Range: 20, CollideType: 3, SrvHitFunc: 1}
	s := NewSim(w, nil)
	m, _ := s.Create(CreateParams{Spec: bad, Level: 1, DestX: 60})
	k := kinds(run(s, w, 1))

	if !m.Dead() || k[EventVanish] != 1 || k[EventArea] != 0 {
		t.Errorf("func 6 without sub missile: dead=%v events %v", m.Dead(), k)
	}
}
