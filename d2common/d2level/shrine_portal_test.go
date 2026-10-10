package d2level

import "testing"

func TestPortalShrineDest(t *testing.T) {
	tests := []struct {
		name        string
		level, want int
	}{
		{"cold plains", 3, 1},
		{"rogue encampment itself", 1, 1},
		{"last act 1 level", 39, 1},
		{"first act 2 level", 40, 40},
		{"arcane sanctuary", 74, 40},
		{"travincal", 83, 75},
		{"river of flame", 107, 103},
		{"worldstone keep", 128, 109},
		{"pandemonium area", 134, 109},
		{"no level", 0, 0},
	}

	for _, tc := range tests {
		if got := PortalShrineDest(tc.level); got != tc.want {
			t.Errorf("%s: PortalShrineDest(%d) = %d, want %d", tc.name, tc.level, got, tc.want)
		}
	}
}
