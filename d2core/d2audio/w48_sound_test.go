package d2audio

import "testing"

func TestModularAbsDistance(t *testing.T) {
	cases := []struct{ a, b, m, want int }{
		{0, 0, 4, 0}, {1, 0, 4, 1}, {3, 0, 4, 1}, {2, 0, 4, 2}, {5, 1, 4, 0}, {-1, 0, 4, 1}, {0, 0, 0, 0},
	}

	for _, c := range cases {
		if got := ModularAbsDistance(c.a, c.b, c.m); got != c.want {
			t.Errorf("ModularAbsDistance(%d,%d,%d) = %d, want %d", c.a, c.b, c.m, got, c.want)
		}
	}
}

func TestFootstepFrameDue(t *testing.T) {
	cases := []struct {
		name                 string
		count, steps, offset int
		want                 []int
	}{
		{"hero walk 8 frames, 2 steps", 8, 2, 0, []int{0, 4}},
		{"one step per cycle", 8, 1, 0, []int{0}},
		{"offset 1", 8, 2, 1, []int{1, 5}},
		{"12 frames, 2 steps", 12, 2, 0, []int{0, 6}},
		{"no steps", 8, 0, 0, nil},
	}

	for _, c := range cases {
		var got []int

		for f := 0; f < c.count; f++ {
			if FootstepFrameDue(f, c.count, c.steps, c.offset) {
				got = append(got, f)
			}
		}

		if len(got) != len(c.want) {
			t.Errorf("%s: step frames %v, want %v", c.name, got, c.want)

			continue
		}

		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: step frames %v, want %v", c.name, got, c.want)
			}
		}
	}
}

func TestFootstepTrackerTwoStepsPerCycle(t *testing.T) {
	var tr FootstepTracker

	steps := 0

	// three walk cycles of 8 frames, the game frame calling Observe twice per animation frame
	for cycle := 0; cycle < 3; cycle++ {
		for f := 0; f < 8; f++ {
			for rep := 0; rep < 2; rep++ {
				if tr.Observe(f, 8, 2, 0) {
					steps++
				}
			}
		}
	}

	if steps != 6 {
		t.Errorf("steps in 3 cycles = %d, want 6 (frames 0 and 4 of each)", steps)
	}

	tr.Reset()

	if !tr.Observe(0, 8, 2, 0) {
		t.Error("the first frame after a reset should step")
	}
}

func TestFootstepSpacing(t *testing.T) {
	cases := []struct {
		since, count, steps int
		want                bool
	}{
		{4, 8, 2, true}, {2, 8, 2, false}, {3, 8, 2, true}, {0, 8, 1, true},
	}

	for _, c := range cases {
		if got := FootstepSpacingOK(c.since, c.count, c.steps); got != c.want {
			t.Errorf("FootstepSpacingOK(%d,%d,%d) = %v, want %v", c.since, c.count, c.steps, got, c.want)
		}
	}
}

func TestBlockKickSwingRules(t *testing.T) {
	for hc, want := range map[string]string{"": BlockWeaponHandle, "hth": BlockWeaponHandle, "1hss": BlockWeaponHandle,
		"staf": BlockWeaponHandle, "bow": BlockArrowHandle, "xbow": BlockArrowHandle, "weird": ""} {
		if got := BlockHandle(hc); got != want {
			t.Errorf("BlockHandle(%q) = %q, want %q", hc, got, want)
		}
	}

	if KickDelay(6) != 4 || KickDelay(0) != 0 {
		t.Error("only the assassin delays the kick sound")
	}

	for _, c := range []struct {
		mode, class int
		want        bool
	}{{3, 4, true}, {3, 6, false}, {4, 6, true}, {4, 4, false}, {1, 4, false}} {
		if got := SwingInSkillMode(c.mode, c.class); got != c.want {
			t.Errorf("SwingInSkillMode(%d,%d) = %v", c.mode, c.class, got)
		}
	}
}
