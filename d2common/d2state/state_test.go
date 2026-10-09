package d2state

import (
	"reflect"
	"testing"
)

func TestApplyExpireAndRefresh(t *testing.T) {
	s := New()

	if s.Active(0, "x") {
		t.Fatal("empty set has an active state")
	}

	if prev := s.Apply(10, Instance{Name: "amp", Until: 20, Mods: []StatMod{{"damageresist", -100}}}); prev != nil {
		t.Fatalf("first apply returned %v", prev)
	}

	if !s.Active(19, "amp") || s.Active(20, "amp") {
		t.Error("state must be active on frames 10..19 only")
	}

	if got := s.Stat(15, "damageresist"); got != -100 {
		t.Errorf("stat = %d", got)
	}

	// re-applying refreshes and reports the previous instance
	if prev := s.Apply(15, Instance{Name: "amp", Until: 40}); prev == nil || prev.Until != 20 {
		t.Errorf("refresh returned %+v", prev)
	}

	if got := s.Stat(16, "damageresist"); got != 0 {
		t.Errorf("refreshed state without mods still gives %d", got)
	}

	if res := s.Tick(40); !reflect.DeepEqual(res.Expired, []string{"amp"}) {
		t.Errorf("expired = %v", res.Expired)
	}

	if s.Active(40, "amp") {
		t.Error("expired state still active")
	}
}

func TestIndefiniteStateAndStatSum(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "might", Mods: []StatMod{{"damagepercent", 40}}})
	s.Apply(0, Instance{Name: "conc", Mods: []StatMod{{"damagepercent", 30}, {"skill_concentration", 20}}})
	s.Apply(0, Instance{Name: "weaken", Until: 50, Mods: []StatMod{{"damagepercent", -33}}})

	if got := s.DamagePct(10); got != 37 {
		t.Errorf("damage pct = %d, want 40+30-33", got)
	}

	if got := s.DamagePct(60); got != 70 {
		t.Errorf("after weaken = %d", got)
	}

	if got := s.Names(60); !reflect.DeepEqual(got, []string{"conc", "might"}) {
		t.Errorf("names = %v", got)
	}

	s.Remove("might")

	if s.Active(60, "might") {
		t.Error("removed state is active")
	}
}

func TestPoisonTotalMatchesTooltip(t *testing.T) {
	// Poison Javelin level 1: 32 (8.8) per frame for 200 frames = 25 hp
	s := New()
	s.AddStream(0, "poison", 32, 200, "hero", 1)

	total := 0

	for f := 0; f < 260; f++ {
		total += s.Tick(f).Poison
	}

	if total != 24 && total != 25 {
		t.Errorf("total poison = %d, want 25 (24 with the last fraction dropped)", total)
	}

	if len(s.Streams(260)) != 0 {
		t.Error("finished stream is still listed")
	}
}

func TestPoisonRejectsEmptyStreams(t *testing.T) {
	s := New()
	s.AddStream(0, "poison", 0, 10, "x", 1)
	s.AddStream(0, "poison", 10, 0, "x", 1)

	if n := len(s.Streams(0)); n != 0 {
		t.Errorf("empty streams must be ignored, have %d", n)
	}
}

func TestBurnIsSeparateFromPoison(t *testing.T) {
	s := New()
	s.AddStream(0, "burn", 256, 4, "a", 1)

	res := s.Tick(1)
	if res.Burn != 1 || res.Poison != 0 {
		t.Errorf("tick = %+v", res)
	}
}

func TestApplyHit(t *testing.T) {
	s := New()
	names := s.ApplyHit(100, Hit{ColdLen: 60, FreezeLen: 30, StunLen: 10, Poison: 64, PoisonLen: 50, Burn: 100, BurnLen: 25,
		Source: "hero", SkillID: 7})

	want := []string{Stun, Freeze, Chill, "poison", "burn"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("applied %v, want %v", names, want)
	}

	if s.CanAct(105) {
		t.Error("a stunned and frozen unit can act")
	}

	if s.Active(111, Stun) {
		t.Error("stun must have ended after 10 frames")
	}

	if s.CanAct(120) {
		t.Error("the freeze (30 frames) is still running at 120")
	}

	if !s.CanAct(131) {
		t.Error("the unit must act again once stun and freeze ended")
	}

	if got := s.SpeedPct(120); got != ChillSpeedPct {
		t.Errorf("chilled speed = %d", got)
	}

	if got := s.SpeedPct(161); got != 0 {
		t.Errorf("chill must end after 60 frames, speed = %d", got)
	}

	// immunities skip the state
	s2 := New()
	if got := s2.ApplyHit(0, Hit{StunLen: 10, ColdLen: 10, FreezeLen: 10, CannotStun: true, CannotChill: true,
		CannotFreeze: true}); len(got) != 0 {
		t.Errorf("immune unit got %v", got)
	}
}

func TestDerivedQueries(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "amplifydamage", Until: 100, Mods: []StatMod{{"damageresist", -100}}})
	s.Apply(0, Instance{Name: "lowerresist", Until: 100, Mods: []StatMod{{"fireresist", -30}, {"coldresist", -30}}})
	s.Apply(0, Instance{Name: "decrepify", Until: 100, Mods: []StatMod{{"velocitypercent", -50}, {"attackrate", -50}}})
	s.Apply(0, Instance{Name: "ironmaiden", Until: 100, Mods: []StatMod{{StatIronMaiden, 75}}})
	s.Apply(0, Instance{Name: Terror, Until: 50})

	cases := []struct {
		name string
		got  int
		want int
	}{
		{"phys resist", s.ResistDelta(1, "phys"), -100},
		{"fire resist", s.ResistDelta(1, "fire"), -30},
		{"cold resist", s.ResistDelta(1, "cold"), -30},
		{"ltng resist", s.ResistDelta(1, "ltng"), 0},
		{"speed", s.SpeedPct(1), -50},
		{"attack speed", s.AttackSpeedPct(1), -50},
		{"reflect", s.ReflectPct(1), 75},
		{"unknown kind", s.ResistDelta(1, "zzz"), 0},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}

	if !s.Fleeing(10) || s.Fleeing(50) {
		t.Error("terror must last until frame 50")
	}

	if s.ReflectPct(100) != 0 {
		t.Error("expired iron maiden still reflects")
	}
}

func TestDefensePct(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "shout", Mods: []StatMod{{"skill_armor_percent", 135}}})

	if got := s.DefensePct(0); got != 135 {
		t.Errorf("defense = %d", got)
	}

	s.Apply(0, Instance{Name: "berserk", Mods: []StatMod{{"armor_override_percent", -100}}})

	if got := s.DefensePct(0); got != -100 {
		t.Errorf("berserk defense = %d, want -100 (zero defense)", got)
	}
}

func TestSpeedClamp(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "a", Mods: []StatMod{{"velocitypercent", -80}}})
	s.Apply(0, Instance{Name: "b", Mods: []StatMod{{"velocitypercent", -80}}})

	if got := s.SpeedPct(0); got != -100 {
		t.Errorf("speed = %d, want clamp at -100", got)
	}
}

func TestReset(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "a"})
	s.AddStream(0, "poison", 10, 10, "", 0)
	s.Reset()

	if len(s.Names(0)) != 0 || len(s.Streams(0)) != 0 {
		t.Error("reset must clear the set")
	}
}

func TestDrainPool(t *testing.T) {
	s := New()
	s.Apply(0, Instance{Name: "bonearmor", Mods: []StatMod{{"bonearmor", 100 * 256}, {"bonearmormax", 100 * 256}}})

	if got := s.Drain(0, "bonearmor", 30*256); got != 30*256 {
		t.Errorf("drained %d", got)
	}

	if got := s.Stat(0, "bonearmor"); got != 70*256 {
		t.Errorf("pool %d", got)
	}

	if got := s.Drain(0, "bonearmor", 500*256); got != 70*256 {
		t.Errorf("rest = %d", got)
	}

	if s.Active(0, "bonearmor") {
		t.Error("an empty pool must end the state")
	}
}
