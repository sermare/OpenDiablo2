package d2state

import "testing"

func TestApplyPotionStateExtends(t *testing.T) {
	s := New()
	mods := []StatMod{{Stat: "poisonresist", Value: 50}, {Stat: "maxpoisonresist", Value: 10}}

	until, ok := s.ApplyPotionState(100, "antidote", 750, mods)
	if !ok || until != 850 {
		t.Fatalf("first drink until=%d ok=%v, want 850", until, ok)
	}

	if got := s.StatMods(200)["poisonresist"]; got != 50 {
		t.Errorf("poisonresist = %d, want 50", got)
	}

	// 250 frames later 500 are left: the second drink adds 750 to them (0x5bc990)
	if until, _ = s.ApplyPotionState(350, "antidote", 750, mods); until != 1600 {
		t.Errorf("second drink until=%d, want 1600", until)
	}

	// the stats are the potion's own value, not summed with the first drink
	if got := s.StatMods(400)["poisonresist"]; got != 50 {
		t.Errorf("poisonresist after two drinks = %d, want 50", got)
	}

	if s.Active(1600, "antidote") || !s.Active(1599, "antidote") {
		t.Error("the state ends on the computed frame")
	}

	// an expired state starts again from now
	if until, _ = s.ApplyPotionState(2000, "antidote", 750, mods); until != 2750 {
		t.Errorf("drink after expiry until=%d, want 2750", until)
	}

	if _, ok = s.ApplyPotionState(0, "antidote", 0, mods); ok {
		t.Error("length 0 does nothing")
	}
}

func TestEndStateEndsPoisonStream(t *testing.T) {
	s := New()
	s.Apply(10, Instance{Name: "poison", Until: 500})
	s.streams = append(s.streams, Stream{Kind: "poison", PerFrame: 256, Until: 500})
	s.Apply(10, Instance{Name: "freeze", Until: 300})

	if !s.EndState(20, "poison") {
		t.Fatal("poison did not end")
	}

	if s.Active(20, "poison") || len(s.streams) != 0 {
		t.Errorf("poison state or stream left: %v %v", s.Active(20, "poison"), s.streams)
	}

	if !s.Active(20, "freeze") {
		t.Error("freeze must stay")
	}

	if s.EndState(20, "cold") {
		t.Error("ending a state that is not there reports true")
	}

	if !s.EndState(20, "freeze") || s.Active(20, "freeze") {
		t.Error("freeze did not end")
	}
}
