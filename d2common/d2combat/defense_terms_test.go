package d2combat

import "testing"

// Pins COMBAT_GetDefense (0x6225a0): base + base*pct/100 (sign branch for a
// non-positive base) and the stat 0xb6 term on the running total.
func TestDefenseGetDefenseTerms(t *testing.T) {
	cases := []struct{ ac, dex, pct, over, want int }{
		{100, 40, 0, 0, 110},
		{100, 40, 30, 0, 143},
		{101, 7, 50, 0, 153},
		{-10, 0, 50, 0, -5}, // a bonus shrinks a negative defense toward zero
		{-9, 0, 33, 0, -7},
		{100, 0, 0, 20, 120},
		{100, 40, 30, 10, 157},
		{0, 0, 100, 50, 0},
	}

	for _, c := range cases {
		if got := DefenseOverride(c.ac, c.dex, c.pct, c.over); got != c.want {
			t.Errorf("DefenseOverride(%d,%d,%d,%d) = %d, want %d", c.ac, c.dex, c.pct, c.over, got, c.want)
		}
	}
}
