package d2util

import "testing"

func TestTurboContinue(t *testing.T) {
	cases := []struct {
		name          string
		ticks         int
		spent, budget float64
		force, want   bool
	}{
		{"first tick always runs", 0, 1, 0.012, false, true},
		{"within budget", 5, 0.005, 0.012, false, true},
		{"budget spent", 5, 0.012, 0.012, false, false},
		{"forced draw ends the frame", 5, 0, 0.012, true, false},
		{"tick cap", TurboMaxTicksPerFrame, 0, 0.012, false, false},
	}
	for _, c := range cases {
		if got := TurboContinue(c.ticks, c.spent, c.budget, c.force); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestTurboShouldDraw(t *testing.T) {
	cases := []struct {
		frame, every int
		forced, want bool
	}{
		{0, 10, false, true},
		{1, 10, false, false},
		{10, 10, false, true},
		{7, 10, true, true},
		{7, 1, false, true},
	}
	for _, c := range cases {
		if got := TurboShouldDraw(c.frame, c.every, c.forced); got != c.want {
			t.Errorf("%+v: got %v", c, got)
		}
	}
}

func TestTurboClockDeterministic(t *testing.T) {
	t.Setenv("OD2_TURBO", "1")
	turboOn = true // env is read once per process; force for the test
	defer func() { turboOn = false }()

	a := Now()
	TurboAdvance()
	TurboAdvance()

	if d := Now() - a; d < 2*TurboTickSeconds-1e-6 || d > 2*TurboTickSeconds+1e-6 {
		t.Fatalf("two ticks moved the clock by %v", d)
	}
}
