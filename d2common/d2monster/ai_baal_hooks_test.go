package d2monster

import "testing"

func TestBaalTeleportDest(t *testing.T) {
	tests := []struct {
		name         string
		self, target Point
		dist         int
		want         Point
	}{
		{"away east", Point{100, 100}, Point{90, 100}, 10, Point{125, 100}},
		{"away diagonal", Point{100, 100}, Point{90, 90}, 15, Point{116, 116}},
		{"zero distance reads as one", Point{100, 100}, Point{100, 100}, 0, Point{100, 100}},
		{"west", Point{100, 100}, Point{110, 100}, 10, Point{75, 100}},
	}

	for _, tt := range tests {
		if got := BaalTeleportDest(tt.self, tt.target, tt.dist); got != tt.want {
			t.Errorf("%s: %v want %v", tt.name, got, tt.want)
		}
	}
}

type fakePlacer struct {
	*fakeBaal
	ok   bool
	cast []Point
}

func (f *fakePlacer) PlaceNear(_ *Brain, p Point) (Point, bool) { return p, f.ok }
func (f *fakePlacer) CastSkillAt(_ *Brain, skill int, _ Mode, p Point) bool {
	f.cast = append(f.cast, p)

	return skill == SkillBaalTeleport
}

func TestBaalTeleportAction(t *testing.T) {
	p := skills(profile("BaalCrab"), "Baal Nova", "Baal Inferno", "Baal Tentacle", "Baal Cold Missiles", "Baal Teleport")
	f := newFake(10, false)

	for _, ok := range []bool{true, false} {
		w := &fakePlacer{fakeBaal: &fakeBaal{fakeWorld: f}, ok: ok}
		b := brainAt(p)
		tg := f.target
		c := &Ctx{B: b, W: w, Params: Params{Target: &tg, Dist: 10}}
		baalTeleportAway(c)

		switch {
		case ok && (len(w.cast) != 1 || w.cast[0].X != 75):
			t.Errorf("hop not cast away from target: %v", w.cast)
		case !ok && len(w.cast) != 0:
			t.Errorf("hop cast without a free spot: %v", w.cast)
		}
	}

	// no teleport skill: a plain wait
	w := &fakePlacer{fakeBaal: &fakeBaal{fakeWorld: f}, ok: true}
	c := &Ctx{B: brainAt(skills(profile("BaalCrab"), "Baal Nova")), W: w, Params: Params{Target: &f.target, Dist: 10}}
	baalTeleportAway(c)

	if len(w.cast) != 0 {
		t.Error("cast without the slot")
	}
}

func TestBaalDangerousSkill(t *testing.T) {
	tests := []struct {
		id, lvl int
		want    bool
	}{
		{59, 1, true}, {56, 1, true}, {51, 3, false}, {51, 4, true}, {27, 7, false}, {27, 8, true}, {36, 20, false},
	}
	for _, tt := range tests {
		if got := BaalDangerousSkill(tt.id, tt.lvl); got != tt.want {
			t.Errorf("skill %d lvl %d: %v", tt.id, tt.lvl, got)
		}
	}
}

func TestBaalHomeCloneHelpers(t *testing.T) {
	if far, vf := BaalHomeFlags(75); far || vf {
		t.Error("75 is not far")
	}

	if far, vf := BaalHomeFlags(76); !far || vf {
		t.Error("76 is far")
	}

	if far, vf := BaalHomeFlags(101); !far || !vf {
		t.Error("101 is very far")
	}

	if hp, max := BaalCloneStats(3000, 9001); hp != 1000 || max != 3000 {
		t.Errorf("clone life %d/%d", hp, max)
	}

	if dx, dy := BaalCloneOffset(0, 23); dx != -12 || dy != 11 {
		t.Errorf("offset %d,%d", dx, dy)
	}
}

func TestMonsterMoveSkills(t *testing.T) {
	if MulDiv(20, 100, 100) != 20 || MulDiv(20, 150, 100) != 30 || MulDiv(3, 50, 100) != 2 || MulDiv(1, 1, 0) != 0 {
		t.Error("MulDiv")
	}

	if DiabRunSpeed(20, 100) != 20 {
		t.Error("run speed")
	}

	// skills.txt DiabRun Param1..Param6 = 8, 14, 5, 13, 16, 6
	r := DiabRunParamsOf([7]int{0, 8, 14, 5, 13, 16, 6})
	if r.StartFrame != 5 || r.LoopAt != 13 || r.LoopTo != 6 || r.Total != 16 {
		t.Fatalf("%+v", r)
	}

	if s := r.Step(4, false, false, false); s.Start {
		t.Error("started early")
	}

	if s := r.Step(5, false, false, false); !s.Start {
		t.Error("no start at frame 5")
	}

	if s := r.Step(13, true, true, true); !s.Rewind || !s.Strike {
		t.Errorf("loop frame: %+v", s)
	}

	if s := r.Step(8, true, true, false); s.Strike {
		t.Error("struck out of melee range")
	}

	if RunAdvance(30, 7, 8) != 8 || RunAdvance(10, 7, 8) != 3 || RunAdvance(5, 7, 8) != 0 {
		t.Error("RunAdvance")
	}

	if got := JumpReflect(Point{10, 10}, Point{8, 9}); got != (Point{14, 12}) {
		t.Errorf("reflect %v", got)
	}
}
