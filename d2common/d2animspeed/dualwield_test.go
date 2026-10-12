package d2animspeed

import "testing"

// TestDualWieldAdjust: the term only applies when the unit can dual wield and both hands hold weapons; the average
// truncates toward zero (0x624150).
func TestDualWieldAdjust(t *testing.T) {
	cases := []struct {
		can, both     bool
		first, second int
		want          int
	}{
		{true, true, -10, -10, 0},
		{true, true, 0, -20, -10},
		{true, true, -20, 0, 10},
		{true, true, -5, 0, 3}, // (0-5)/2 truncates to -2, minus the first (-5)
		{false, true, 0, -20, 0},
		{true, false, 0, -20, 0},
	}

	for _, c := range cases {
		if got := DualWieldAdjust(c.can, c.both, c.first, c.second); got != c.want {
			t.Errorf("%+v: %d, want %d", c, got, c.want)
		}
	}
}
