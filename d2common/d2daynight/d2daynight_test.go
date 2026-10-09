package d2daynight

import "testing"

// dayTicks: each of the 5 changes 1->2->3->4->5->0 costs 241 steps (the tick
// snaps back by one) and 0->1 costs 1 step (next start is 0, so any tick
// exceeds it): 1206 steps, so the 1440 tick wrap never fires. This is how the
// decompiled FUN_0061bc10 reads; the quirk is probably unintended in the game.
const dayTicks = 5*241 + 1

func TestClassify(t *testing.T) {
	want := map[int]TimeOfDay{0: Evening, 1: Morning, 2: Day, 3: Day, 4: Evening, 5: Evening}
	for phase, w := range want {
		if got := Classify(phase); got != w {
			t.Errorf("Classify(%d)=%d want %d", phase, got, w)
		}
	}
}

func TestAdvancePhases(t *testing.T) {
	tests := []struct {
		ticks     int
		wantPhase int
		wantTick  int
	}{
		{0, 1, 0},
		{240, 1, 240},
		{241, 2, 240},
		{482, 3, 480},
		{723, 4, 720},
		{964, 5, 960},
		{1205, 0, 1200},
		{1206, 1, 0},
	}

	for _, tc := range tests {
		e := NewCycling()
		e.Advance(tc.ticks)

		if e.Phase() != tc.wantPhase || e.Tick() != tc.wantTick {
			t.Errorf("after %d ticks: phase %d tick %d, want %d/%d", tc.ticks, e.Phase(), e.Tick(), tc.wantPhase, tc.wantTick)
		}
	}
}

func TestFullDayWraps(t *testing.T) {
	e := NewCycling()
	e.Advance(dayTicks * 3)

	if e.Phase() != PhaseDawn || e.Tick() != 0 {
		t.Errorf("after 3 days: phase %d tick %d", e.Phase(), e.Tick())
	}

	seen := map[int]bool{}

	for i := 0; i < dayTicks; i++ {
		e.Advance(1)
		seen[e.Phase()] = true
	}

	if len(seen) != PhaseCount {
		t.Errorf("phases seen in a day: %v", seen)
	}
}

func TestFixedDoesNotAdvance(t *testing.T) {
	e := NewFixed(PhaseDay)
	e.Advance(100000)

	if e.Phase() != PhaseDay || e.Tick() != 0 || e.TimeOfDay() != Day {
		t.Errorf("fixed env moved: %d %d", e.Phase(), e.Tick())
	}

	if got := NewFixed(99).Phase(); got != PhaseDay {
		t.Errorf("invalid phase fell back to %d", got)
	}
}

func TestSetPhaseAndTick(t *testing.T) {
	e := NewCycling()

	for _, tc := range []struct {
		phase, tick int
		ok          bool
		wantTick    int
	}{
		{3, 500, true, 500}, {6, 0, false, 500}, {-1, 0, false, 500},
		{2, -1, false, 500}, {4, 99999, true, 0},
	} {
		if ok := e.SetPhaseAndTick(tc.phase, tc.tick); ok != tc.ok || (ok && e.Tick() != tc.wantTick) {
			t.Errorf("Set(%d,%d)=%v tick %d", tc.phase, tc.tick, ok, e.Tick())
		}
	}
}

func TestAmbient(t *testing.T) {
	// Fixed table, phase 2 is white and phase 3's colour is also white.
	if got := NewFixed(PhaseDay).Ambient(); got != (RGB{255, 255, 255}) {
		t.Errorf("fixed day ambient = %v", got)
	}

	// Phase 4 (c2 98 c1) at tick 0 of its span in the fixed table equals its base.
	if got := NewFixed(PhaseNight4).Ambient(); got != (RGB{0xc2, 0x98, 0xc1}) {
		t.Errorf("fixed phase4 ambient = %v", got)
	}

	// Cycling table colours are verified bytes: dawn is (0,30,244).
	if got := NewCycling().Ambient(); got != (RGB{0, 0x1e, 0xf4}) {
		t.Errorf("cycling dawn ambient = %v", got)
	}
}
