package d2skill

import "testing"

func TestStrafeBurst(t *testing.T) {
	// min(calc1, max(found, min(calc1, calc3)))
	for _, tc := range []struct{ c1, c3, found, want int }{
		{5, 3, 0, 3}, // nobody in range: the calc3 minimum burst
		{5, 3, 2, 3}, // fewer enemies than calc3
		{5, 3, 4, 4}, // more enemies than calc3
		{5, 3, 9, 5}, // capped by calc1
		{2, 6, 9, 2}, // calc3 above calc1
		{0, 3, 4, 0}, // nothing to fire
	} {
		if got := StrafeBurst(tc.c1, tc.c3, tc.found); got != tc.want {
			t.Errorf("StrafeBurst(%d,%d,%d) = %d, want %d", tc.c1, tc.c3, tc.found, got, tc.want)
		}
	}
}
