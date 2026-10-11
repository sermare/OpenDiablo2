package d2monsters

import "testing"

func TestRestoredLife(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		hp, max, elapsed, want int
	}{
		{"just left", 40, 100, 0, 40},
		{"one frame short", 40, 100, InactiveLifeFrames - 1, 40},
		{"exactly the limit", 40, 100, InactiveLifeFrames, 100},
		{"long away", 1, 100, 50000, 100},
		{"full stays", 100, 100, 10, 100},
		{"inconsistent hp falls back to max", 150, 100, 10, 100},
	} {
		if got := RestoredLife(tc.hp, tc.max, tc.elapsed); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}
