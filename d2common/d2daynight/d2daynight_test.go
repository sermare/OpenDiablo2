package d2daynight

import "testing"

func TestClassify(t *testing.T) {
	want := map[int]TimeOfDay{0: Evening, 1: Morning, 2: Day, 3: Day, 4: Evening, 5: Evening}
	for phase, w := range want {
		if got := Classify(phase); got != w {
			t.Errorf("Classify(%d)=%d want %d", phase, got, w)
		}
	}
}

func TestAdvancePhases(t *testing.T) {
	const tpd = TicksPerDegreeReal

	tests := []struct {
		name      string
		ticks     int // from the start of phase 0 (degree 320)
		wantPhase int
	}{
		{"start", 0, 0},
		{"before dawn", 19*tpd - 1, 0},
		{"dawn", 20 * tpd, 1},
		{"day", 40 * tpd, 2},
		{"midday", 140 * tpd, 2},
		{"phase3", 200 * tpd, 3},
		{"dusk", 220 * tpd, 4},
		{"night", 240 * tpd, 5},
		{"back to phase 0", 360 * tpd, 0},
	}

	for _, tc := range tests {
		e := NewCycling()
		e.Advance(tc.ticks)

		if e.Phase() != tc.wantPhase {
			t.Errorf("%s: after %d ticks phase %d, want %d (tick %d)", tc.name, tc.ticks, e.Phase(), tc.wantPhase, e.Tick())
		}
	}
}

func TestFullDayWraps(t *testing.T) {
	e := NewCycling()
	start := e.Tick()
	seen := map[int]bool{}

	day := TicksPerDegreeReal * DegreesPerDay
	for i := 0; i < day; i += 64 {
		e.Advance(64)
		seen[e.Phase()] = true
	}

	if e.Tick() != start || len(seen) != PhaseCount {
		t.Errorf("tick %d want %d; phases seen %v", e.Tick(), start, seen)
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
		{2, -1, false, 500}, {4, 99999999, true, 0},
	} {
		if ok := e.SetPhaseAndTick(tc.phase, tc.tick); ok != tc.ok || (ok && e.Tick() != tc.wantTick) {
			t.Errorf("Set(%d,%d)=%v tick %d", tc.phase, tc.tick, ok, e.Tick())
		}
	}
}

func TestAmbientColours(t *testing.T) {
	for phase, want := range map[int]RGB{
		PhaseNight0: {0x7d, 0x90, 0xf3}, PhaseDawn: {0xd0, 0xb8, 0x83}, PhaseDay: {255, 255, 255},
		PhaseDusk: {255, 255, 255}, PhaseNight4: {0xc2, 0x98, 0xc1}, PhaseNight5: {0x7d, 0x90, 0xf3},
	} {
		if got := NewFixed(phase).Ambient(); got != want {
			t.Errorf("fixed phase %d ambient = %v want %v", phase, got, want)
		}

		e := NewCycling()
		e.SetPhase(phase)

		if got := e.Ambient(); got != want {
			t.Errorf("cycling phase %d start ambient = %v want %v", phase, got, want)
		}
	}

	// halfway from dawn orange (340) to day white (360 == 0)
	e := NewCycling()
	e.SetPhase(PhaseDawn)
	e.Advance(10 * TicksPerDegreeReal)

	if got := e.Ambient(); got.R <= 0xd0 || got.B <= 0x83 {
		t.Errorf("dawn midpoint ambient = %v", got)
	}
}

func TestIntensity(t *testing.T) {
	at := func(deg int) int {
		e := NewCycling()
		e.SetPhaseAndTick(PhaseDay, deg*TicksPerDegreeReal)

		return e.Intensity()
	}

	// first half: 128+128*cos; second half: 128+64*cos
	for deg, want := range map[int]int{0: 255, 90: 128, 180: 64, 270: 128, 200: 68, 340: 188} {
		if got := at(deg); got != want {
			t.Errorf("deg %d intensity %d want %d", deg, got, want)
		}
	}

	if NewSpecial().Intensity() != 32 {
		t.Error("special intensity must be 32")
	}
}
