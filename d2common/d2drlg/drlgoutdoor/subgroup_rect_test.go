package drlgoutdoor

import "testing"

// DRLG_AddSubThemeGroupObjects (0x6726b0) compares sub-tile coordinates with strict inequalities on both ends.
func TestInSubGroupRect(t *testing.T) {
	// group at tile (2,3), 4x2 tiles: sub-tiles x 10..30 and y 15..25, exclusive
	tests := []struct {
		name   string
		ox, oy int
		want   bool
	}{
		{"inside", 12, 17, true},
		{"first column is out", 10, 17, false},
		{"first row is out", 12, 15, false},
		{"one past the first column", 11, 16, true},
		{"last column inside", 29, 24, true},
		{"right edge out", 30, 20, false},
		{"bottom edge out", 20, 25, false},
		{"far outside", 100, 100, false},
	}

	for _, tc := range tests {
		if got := InSubGroupRect(tc.ox, tc.oy, 2, 3, 4, 2); got != tc.want {
			t.Errorf("%s: (%d,%d) = %v, want %v", tc.name, tc.ox, tc.oy, got, tc.want)
		}
	}
}
