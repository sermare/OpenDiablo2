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

func TestPortalShrineOpens(t *testing.T) {
	tests := []struct {
		name  string
		level int
		want  bool
	}{
		{"cold plains", 3, true},
		{"travincal", 83, true},
		{"rogue encampment: town room, the exe does nothing", 1, false},
		{"lut gholein", 40, false},
		{"no level", 0, false},
	}

	for _, tc := range tests {
		if got := PortalShrineOpens(tc.level); got != tc.want {
			t.Errorf("%s: PortalShrineOpens(%d) = %v, want %v", tc.name, tc.level, got, tc.want)
		}
	}

	if PortalShrineOffset != 5 {
		t.Errorf("PortalShrineOffset = %d, want 5", PortalShrineOffset)
	}
}
