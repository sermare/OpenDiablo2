package d2player

import "testing"

func TestReplenishSeconds(t *testing.T) {
	for v, want := range map[int64]float64{30: 100.0 / 30, 100: 1, 1: 100, 0: 0, -5: 0} {
		if got := replenishSeconds(v); got != want {
			t.Errorf("replenishSeconds(%d) = %v, want %v", v, got, want)
		}
	}
}
