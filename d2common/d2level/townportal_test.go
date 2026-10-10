package d2level

import "testing"

func TestTownPortalCastable(t *testing.T) {
	tests := []struct {
		level int
		want  bool
	}{
		{1, false}, {2, true}, {40, false}, {74, true}, {109, false}, {121, true},
		{133, true}, {135, true}, {136, false}, {0, false}, {137, false},
	}

	for _, tc := range tests {
		if got := TownPortalCastable(tc.level); got != tc.want {
			t.Errorf("TownPortalCastable(%d) = %v, want %v", tc.level, got, tc.want)
		}
	}
}
