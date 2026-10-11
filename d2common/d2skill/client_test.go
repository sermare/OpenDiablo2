package d2skill

import "testing"

func TestMultiShotStep(t *testing.T) {
	cases := []struct {
		name         string
		caster, tgt  SubPoint
		wantX, wantY int
	}{
		{"east", SubPoint{0, 0}, SubPoint{10, 0}, 0, -1},
		{"south", SubPoint{0, 0}, SubPoint{0, 10}, 1, 0},
		{"west", SubPoint{0, 0}, SubPoint{-8, 0}, 0, 1},
		{"short east", SubPoint{5, 5}, SubPoint{6, 5}, 0, -1},
		{"diagonal", SubPoint{0, 0}, SubPoint{7, 7}, 1, -1},
	}
	for _, c := range cases {
		x, y := MultiShotStep(c.caster, c.tgt)
		if x != c.wantX || y != c.wantY {
			t.Errorf("%s: step (%d,%d), want (%d,%d)", c.name, x, y, c.wantX, c.wantY)
		}
	}
}

func TestMultiShotPlan(t *testing.T) {
	// firing east at (10,0): step (0,-1); 5 arrows start at y = 0 - (-1*5/2) = 2
	got := MultiShotPlan(SubPoint{0, 0}, SubPoint{10, 0}, 5, 1)
	if len(got) != 5 {
		t.Fatalf("%d shots", len(got))
	}

	for i, s := range got {
		want := SubPoint{10, 2 - i}
		if s.To != want || s.From != (SubPoint{}) || s.Kind != KindMultiShot || s.Loop != i {
			t.Errorf("arrow %d = %+v, want aim %v", i, s, want)
		}
	}

	if MultiShotPlan(SubPoint{}, SubPoint{1, 1}, 0, 1) != nil {
		t.Error("zero count must give nothing")
	}

	if MultiShotSlot(1, true) != 1 || MultiShotSlot(3, true) != 2 || MultiShotSlot(3, false) != 1 {
		t.Error("slot choice")
	}
}

func TestChargedStrikePlan(t *testing.T) {
	got := ChargedStrikePlan(SubPoint{2, 3}, SubPoint{6, 5}, 3, 1)
	if len(got) != 3 {
		t.Fatal(len(got))
	}

	for i, s := range got {
		if s.From != (SubPoint{6, 5}) || s.To != (SubPoint{10, 7}) || s.Loop != i || s.Kind != KindAimed {
			t.Errorf("bolt %d = %+v", i, s)
		}
	}
}

func TestGuidedArrowPlan(t *testing.T) {
	u := GuidedArrowPlan(SubPoint{10, 10}, SubPoint{14, 7}, true, 1)
	if u.Kind != KindHomingUnit || u.Data28 != 1 || u.Data2C != (-3)*0x10000+4 {
		t.Errorf("unit: %+v", u)
	}

	g := GuidedArrowPlan(SubPoint{10, 10}, SubPoint{14, 7}, false, 1)
	if g.Kind != KindHomingGround || g.Data28 != 2 {
		t.Errorf("ground: %+v", g)
	}
}

func TestLightningStrikePlan(t *testing.T) {
	main := ChainTarget{1, SubPoint{10, 10}}
	others := []ChainTarget{main, {2, SubPoint{30, 10}}, {3, SubPoint{14, 10}}, {4, SubPoint{12, 12}}}

	s, ok := LightningStrikePlan(main, others, 5, 4, 1)
	if !ok || s.To != (SubPoint{12, 12}) || s.Data28 != 4 || s.Parent != 1 || s.Kind != 0x20 {
		t.Errorf("%v %+v", ok, s)
	}

	if _, ok := LightningStrikePlan(main, others[:2], 5, 4, 1); ok {
		t.Error("nothing within range")
	}
}

func TestStrafeClientPlan(t *testing.T) {
	cur := ChainTarget{1, SubPoint{5, 0}}
	others := []ChainTarget{cur, {2, SubPoint{9, 0}}, {3, SubPoint{3, 0}}, {4, SubPoint{40, 0}}}

	s, left, next, ok := StrafeClientPlan(SubPoint{0, 0}, cur, 3, others, 10, 1)
	if s.To != cur.Pos || left != 2 || !ok || next.ID != 3 {
		t.Errorf("%+v %d %+v %v", s, left, next, ok)
	}

	if _, left, _, ok = StrafeClientPlan(SubPoint{}, cur, 1, others, 10, 1); left != 0 || ok {
		t.Error("last arrow ends the burst")
	}

	if _, _, _, ok = StrafeClientPlan(SubPoint{}, cur, 3, []ChainTarget{cur}, 10, 1); ok {
		t.Error("a lone enemy has no next target")
	}
}

func TestSynergyMissileSlot(t *testing.T) {
	cases := []struct {
		bits4, stat, want int
		has               bool
	}{
		{0, 5, 1, true}, {4, 5, 1, false}, {4, 1, 1, true}, {4, 2, 2, true}, {4, 3, 3, true}, {4, 9, 3, true},
	}
	for _, c := range cases {
		if got := SynergyMissileSlot(c.bits4, c.has, c.stat); got != c.want {
			t.Errorf("%+v got %d", c, got)
		}
	}
}
