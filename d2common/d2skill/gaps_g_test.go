package d2skill

import "testing"

func TestFendZealCounts(t *testing.T) {
	cases := []struct{ calc1, found, want int }{{5, 3, 3}, {2, 9, 2}, {4, 0, 0}, {0, 3, 0}}
	for _, c := range cases {
		if got := StrikeCount(c.calc1, c.found); got != c.want {
			t.Errorf("StrikeCount(%d,%d)=%d want %d", c.calc1, c.found, got, c.want)
		}
	}

	if FendReach(2) != 6 || FendEnemyMask != 0x20003 {
		t.Error("reach or mask")
	}
}

func TestBoneWallAllowed(t *testing.T) {
	cases := []struct{ room, town, bit, want bool }{
		{true, false, false, true}, {true, true, false, false}, {true, true, true, true}, {false, false, true, false},
	}
	for _, c := range cases {
		if got := BoneWallAllowed(c.room, c.town, c.bit); got != c.want {
			t.Errorf("%+v got %v", c, got)
		}
	}
}

func TestClientGate(t *testing.T) {
	bw := &Skill{SrvDoFunc: doBoneWall}
	if _, ok := clientGate(bw, Target{TownRoom: true}); ok {
		t.Error("town room should be refused")
	}

	bw.AllowTownRoom = true
	if _, ok := clientGate(bw, Target{TownRoom: true}); !ok {
		t.Error("bit 0x100 allows the town room")
	}

	golem := &Skill{SrvDoFunc: doGolemIron}
	owned := &ItemTarget{OnGround: true, GolemItem: true, FlagOK: true, InRoom: true, Owned: true}

	if r, ok := clientGate(golem, Target{Item: owned}); ok || r != ReasonTarget {
		t.Error("owned item refused")
	}

	free := &ItemTarget{OnGround: true, GolemItem: true, FlagOK: true, InRoom: true}
	if _, ok := clientGate(golem, Target{Item: free}); !ok {
		t.Error("ground item accepted")
	}

	if _, ok := clientGate(golem, Target{}); !ok {
		t.Error("no data refuses nothing")
	}

	rev := &Skill{SrvDoFunc: doRevive}
	bad := &ReviveTarget{IsMonster: true, Flag8: true, UsableCorpse: true}

	if r, ok := clientGate(rev, Target{Revive: bad}); ok || r != ReasonNoCorpse {
		t.Error("unrevivable refused")
	}

	if !ReviveTargetOK(ReviveTarget{true, true, true, true}) {
		t.Error("revive ok")
	}
}

func TestCanInterrupt(t *testing.T) {
	base := InterruptInput{CurrentMode: 7, NewMode: 8, Flag: true, HasUsedSkill: true, ActionDue: true}
	mod := func(f func(*InterruptInput)) InterruptInput {
		in := base
		f(&in)

		return in
	}
	hold := func(f func(*InterruptInput)) InterruptInput {
		return mod(func(i *InterruptInput) { i.SkillBits4 = Bits4Hold; f(i) })
	}

	cases := []struct {
		name    string
		in      InterruptInput
		ok, rst bool
	}{
		{"state 0x36", mod(func(i *InterruptInput) { i.StateBlocked = true }), false, false},
		{"dead", mod(func(i *InterruptInput) { i.CurrentMode = 0x11 }), false, false},
		{"death", mod(func(i *InterruptInput) { i.CurrentMode = 0 }), false, false},
		{"no new mode", mod(func(i *InterruptInput) { i.NewMode = 0; i.ActionDue = false }), true, false},
		{"no skill in use", mod(func(i *InterruptInput) { i.HasUsedSkill = false; i.ActionDue = false }), true, false},
		{"repeat 0x43", mod(func(i *InterruptInput) { i.SameSkill = true; i.SrvDoFunc = 0x43; i.ActionDue = false }), true, false},
		{"allowed mode near frame", base, true, false},
		{"allowed mode far", mod(func(i *InterruptInput) { i.ActionDue = false }), false, false},
		{"mode 9 refused", mod(func(i *InterruptInput) { i.NewMode = 9 }), false, false},
		{"mode 0x12", mod(func(i *InterruptInput) { i.NewMode = 0x12 }), true, false},
		{"neutral restarts", mod(func(i *InterruptInput) { i.CurrentMode = 1; i.NewMode = 2 }), true, true},
		{"hold bit, no states", hold(func(i *InterruptInput) {}), true, false},
		{"hold bit, state f", hold(func(i *InterruptInput) { i.State0f = true }), false, false},
		{"hold bit, state f neutral", hold(func(i *InterruptInput) { i.State0f = true; i.CurrentMode = 1 }), true, true},
		{"hold bit, 2a roll under", hold(func(i *InterruptInput) { i.State2a, i.State2aChance, i.Roll = true, 50, 10 }), false, false},
		{"hold bit, 2a roll over", hold(func(i *InterruptInput) { i.State2a, i.State2aChance, i.Roll = true, 50, 60 }), true, false},
	}

	for _, c := range cases {
		ok, rs := CanInterrupt(c.in)
		if ok != c.ok || rs != c.rst {
			t.Errorf("%s: got %v,%v want %v,%v", c.name, ok, rs, c.ok, c.rst)
		}
	}

	if !ActionDue(10, 5) || ActionDue(11, 5) {
		t.Error("ActionDue slack")
	}
}
