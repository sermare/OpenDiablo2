package d2skill

import "testing"

func TestHammerConcentrationScale(t *testing.T) {
	for _, c := range []struct{ p1, dp, want int }{{4, 60, 30}, {4, 0, 0}, {4, 7, 3}, {4, -7, -3}} {
		if got := HammerConcentrationScale(c.p1, c.dp); got != c.want {
			t.Errorf("%d*%d/8 = %d, want %d", c.p1, c.dp, got, c.want)
		}
	}
}
