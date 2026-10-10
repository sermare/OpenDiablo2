package d2gamescreen

import (
	"math"
	"testing"
)

func TestAutoSubsteps(t *testing.T) {
	cases := []struct {
		total float64
		want  int
	}{{0, 1}, {0.016, 1}, {0.04, 1}, {0.041, 2}, {0.064, 2}, {0.5, 13}, {100, maxAutoSubsteps}}
	for _, c := range cases {
		if got := autoSubsteps(c.total); got != c.want {
			t.Fatalf("total %v: %d steps, want %d", c.total, got, c.want)
		}
	}

	// no simulated time is lost: the steps add up to the total, each at most one tick (until the bound)
	for _, total := range []float64{0.016, 0.064, 0.25, 0.5, 1.0} {
		n := autoSubsteps(total)
		per := total / float64(n)

		if per > maxAutoSimStep+1e-9 || math.Abs(per*float64(n)-total) > 1e-9 {
			t.Fatalf("total %v: %d steps of %v", total, n, per)
		}
	}
}
