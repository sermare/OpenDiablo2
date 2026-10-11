package d2monsters

import "testing"

func TestMosquitoBites(t *testing.T) {
	cases := []struct{ c1, c2, roll, want int }{
		{4, 7, 0, 4}, {4, 7, 2, 6}, {4, 7, 5, 6}, {4, 4, 9, 4}, {0, 0, 3, 1}, {-2, 1, 1, 1},
	}
	for _, c := range cases {
		if got := mosquitoBites(c.c1, c.c2, c.roll); got != c.want {
			t.Errorf("%+v got %d", c, got)
		}
	}
}
