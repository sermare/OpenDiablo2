package d2state

import "testing"

// Tests pinning rules read from Game.exe (see the "Verified" list in doc.go).

func TestPoisonBurnReplaceRules(t *testing.T) {
	s := New()
	s.AddStream(0, "poison", 512, 100, "a", 1)
	s.AddStream(10, "poison", 256, 500, "b", 2) // weaker: ignored entirely

	if st := s.Streams(11); len(st) != 1 || st[0].PerFrame != 512 || st[0].Until != 100 {
		t.Errorf("weaker poison must be ignored: %+v", st)
	}

	s.AddStream(20, "poison", 512, 30, "c", 3) // equal: replaces, even if shorter

	if st := s.Streams(21); len(st) != 1 || st[0].Until != 50 || st[0].Source != "c" {
		t.Errorf("equal poison must replace: %+v", st)
	}

	s.AddStream(25, "poison", 1024, 10, "d", 4) // stronger replaces

	if st := s.Streams(26); len(st) != 1 || st[0].PerFrame != 1024 || st[0].Until != 35 {
		t.Errorf("stronger poison must replace: %+v", st)
	}

	s.AddStream(40, "poison", 100, 10, "e", 5) // the old one ended: a weak one starts

	if st := s.Streams(41); len(st) != 1 || st[0].PerFrame != 100 {
		t.Errorf("expired stream must not block: %+v", st)
	}

	s.AddStream(41, "burn", 50, 10, "f", 6)

	if len(s.Streams(42)) != 2 {
		t.Error("burn is independent of poison")
	}
}

func TestChillFreezeOnlyExtend(t *testing.T) {
	s := New()
	s.ApplyHit(0, Hit{ColdLen: 100, HasColdEffect: true, ColdEffect: -50})
	s.ApplyHit(10, Hit{ColdLen: 20, HasColdEffect: true, ColdEffect: -30})

	if !s.Active(99, Chill) || s.SpeedPct(50) != -50 {
		t.Error("a shorter chill keeps the old end and the first slow")
	}

	s.ApplyHit(10, Hit{ColdLen: 200, HasColdEffect: true, ColdEffect: -30})

	if !s.Active(209, Chill) || s.SpeedPct(50) != -50 {
		t.Error("a longer chill extends the end but keeps the slow")
	}

	f := New()
	f.ApplyHit(0, Hit{FreezeLen: 100})
	f.ApplyHit(10, Hit{FreezeLen: 20})

	if !f.Active(99, Freeze) {
		t.Error("a shorter freeze must not shorten")
	}
}

func TestColdEffectGates(t *testing.T) {
	cases := []struct {
		name                string
		eff                 int
		chill, freeze, stun bool
	}{
		{"zero: no chill, freeze; stun still works", 0, false, false, true},
		{"negative: all", -50, true, true, true},
		{"positive: chilled but not frozen", 20, true, false, true},
	}

	for _, c := range cases {
		s := New()
		s.ApplyHit(0, Hit{ColdLen: 50, FreezeLen: 50, StunLen: 50, HasColdEffect: true, ColdEffect: c.eff})

		if s.Active(1, Chill) != c.chill || s.Active(1, Freeze) != c.freeze || s.Active(1, Stun) != c.stun {
			t.Errorf("%s: chill/freeze/stun = %v/%v/%v", c.name, s.Active(1, Chill), s.Active(1, Freeze), s.Active(1, Stun))
		}
	}
}

func TestDifficultyLengthDivisors(t *testing.T) {
	s := New()
	s.ApplyHit(0, Hit{ColdLen: 7, FreezeLen: 50, HasColdEffect: true, ColdEffect: -50, ChillDiv: 2, FreezeDiv: 5})

	if !s.Active(2, Chill) || s.Active(3, Chill) { // 7/2 = 3 frames
		t.Error("chill length is integer-divided")
	}

	if !s.Active(9, Freeze) || s.Active(10, Freeze) { // 50/5 = 10 frames
		t.Error("freeze length is divided")
	}

	t2 := New()
	t2.ApplyHit(0, Hit{ColdLen: 1, HasColdEffect: true, ColdEffect: -50, ChillDiv: 4})

	if !t2.Active(0, Chill) || t2.Active(1, Chill) {
		t.Error("chill lasts at least 1 frame")
	}
}

func TestCurseRoundingAndSkillLevelRules(t *testing.T) {
	if got := CurseLength(5, 50); got != 3 { // 5 - trunc(2.5)
		t.Errorf("curse length rounds up: %d", got)
	}

	s := New()
	s.defs = Defs{"weaken": {ID: 1, Name: "weaken", Curse: true}, "lifetap": {ID: 2, Name: "lifetap", Curse: true}}
	s.Apply(0, Instance{Name: "weaken", Until: 100, SkillID: 66, Level: 5, Mods: []StatMod{{"x", 1}}})

	// same skill, same level: refresh the end only
	s.Apply(10, Instance{Name: "weaken", Until: 50, SkillID: 66, Level: 5, Mods: []StatMod{{"x", 9}}})

	if in := s.Get(11, "weaken"); in == nil || in.Until != 50 || in.Mods[0].Value != 1 {
		t.Errorf("refresh only: %+v", in)
	}

	// same skill, lower level: rejected
	if ok, _ := s.ApplyTimed(12, Instance{Name: "weaken", Until: 500, SkillID: 66, Level: 4}); ok || s.Get(13, "weaken").Until != 50 {
		t.Error("lower level of the same skill must be rejected")
	}

	// higher level: rebuilt
	s.Apply(14, Instance{Name: "weaken", Until: 200, SkillID: 66, Level: 6, Mods: []StatMod{{"x", 2}}})

	if in := s.Get(15, "weaken"); in.Until != 200 || in.Mods[0].Value != 2 {
		t.Errorf("higher level rebuilds: %+v", in)
	}

	// another curse removes this one, whatever its level
	s.Apply(16, Instance{Name: "lifetap", Until: 90, SkillID: 72, Level: 1})

	if s.Active(17, "weaken") || !s.Active(17, "lifetap") {
		t.Error("one curse at a time")
	}
}

func TestColorShiftIgnoresBlue(t *testing.T) {
	s := New()
	s.defs = Defs{
		"a": {ID: 3, Name: "a", ColorPri: 50, ColorShift: 10, Blue: true},
		"b": {ID: 5, Name: "b", ColorPri: 60, ColorShift: 20},
		"c": {ID: 4, Name: "c", ColorPri: 60, ColorShift: 30},
		"z": {ID: 0, Name: "z", ColorPri: 99, ColorShift: 40},
	}

	for n := range s.defs {
		s.Apply(0, Instance{Name: n})
	}

	if got, _ := s.ColorShift(1); got != 30 { // pri 60 beats blue 50; tie goes to id 4; id 0 never colours
		t.Errorf("shift = %d, want 30", got)
	}

	s.defs["p"] = Def{ID: 6, Name: "p", ColorPri: 70, ColorShift: 104}
	s.Apply(0, Instance{Name: "p"})

	if got, _ := s.ColorShiftLocal(1, true); got != 0 {
		t.Errorf("local player 3D must not get shift 104, got %d", got)
	}

	if got, _ := s.ColorShiftLocal(1, false); got != 104 {
		t.Errorf("software mode keeps 104, got %d", got)
	}
}

func TestRealBitStaysDeath(t *testing.T) {
	d := realDefs(t)

	if !d.BitStays("freeze", "monster") || d.BitStays("freeze", "boss") || d.stays("freeze", "boss") != d.stays("freeze", "monster") {
		t.Error("freeze: monster bit stays, boss bit goes, boss statlist follows monster")
	}
}
