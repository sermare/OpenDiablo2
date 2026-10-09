package d2state

import "testing"

// A single application on a fresh unit behaves as before the verified rules
// (no ColdEffect, no divisors): same states, same ends, same stream, same
// damage per frame.
func TestSingleApplicationUnchanged(t *testing.T) {
	s := New()
	got := s.ApplyHit(100, Hit{ColdLen: 60, FreezeLen: 30, StunLen: 10, Poison: 64, PoisonLen: 50, Burn: 100, BurnLen: 25})

	want := []string{Stun, Freeze, Chill, "poison", "burn"}
	if len(got) != len(want) {
		t.Fatalf("applied %v want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("applied %v want %v", got, want)
		}
	}

	for name, until := range map[string]int{Stun: 110, Freeze: 130, Chill: 160} {
		if in := s.Get(100, name); in == nil || in.Until != until {
			t.Errorf("%s: %+v want until %d", name, in, until)
		}
	}

	if v := s.SpeedPct(100); v != ChillSpeedPct {
		t.Errorf("chill slow %d want %d", v, ChillSpeedPct)
	}

	st := s.Streams(100)
	if len(st) != 2 || st[0].PerFrame != 64 || st[0].Until != 150 || st[1].PerFrame != 100 || st[1].Until != 125 {
		t.Errorf("streams %+v", st)
	}

	// whole hit points over the poison length: 64/256 per frame for 50 frames
	total := 0
	for f := 100; f < 150; f++ {
		total += s.Tick(f).Poison
	}

	if total != 12 {
		t.Errorf("poison total %d want 12", total)
	}
}

// Death by kind: players keep plrstaydeath states, monsters (bosses too)
// monstaydeath ones; the visual bit uses the boss mask for bosses.
func TestDeathKindsSynthetic(t *testing.T) {
	defs := Defs{
		"p": {ID: 1, PlrStayDeath: true},
		"m": {ID: 2, MonStayDeath: true},
		"b": {ID: 3, BossStayDeath: true},
	}

	for kind, want := range map[string][]string{"player": {"p"}, "monster": {"m"}, "boss": {"m"}} {
		s := New()
		s.SetDefs(defs)

		for n := range defs {
			s.Apply(0, Instance{Name: n})
		}

		s.Death(kind)

		got := s.Names(1)
		if len(got) != len(want) || got[0] != want[0] {
			t.Errorf("%s keeps %v want %v", kind, got, want)
		}
	}

	if !defs.BitStays("b", "boss") || defs.BitStays("b", "monster") || defs.BitStays("m", "boss") {
		t.Error("bit masks")
	}
}
