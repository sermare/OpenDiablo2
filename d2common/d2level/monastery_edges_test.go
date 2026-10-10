package d2level

import "testing"

// The Warp -1 Vis links of the Monastery are walked across the border of the neighbouring level rectangles
// (Outer Cloister is north of the Gate, the Cathedral north of Inner Cloister, the Barracks west of Courtyard 1
// for exit side 0), with the rectangles of seed 1 from drlgoutdoor.MonasteryRects.
func TestMonasteryEdgeExit(t *testing.T) {
	rects := map[int]Rect{26: {3000, 1000, 64, 18}, 27: {3000, 960, 56, 40}, 28: {2940, 966, 60, 56},
		32: {4000, 1000, 18, 20}, 33: {3996, 966, 28, 34}}

	tests := []struct {
		name     string
		from     int
		wx, wy   float64
		wantTo   int
		wantEdge bool
	}{
		{"gate north border", 26, 3020, 1000.5, 27, true},
		{"gate far from border", 26, 3020, 1010, 0, false},
		{"cloister south border", 27, 3020, 999.5, 26, true},
		{"cloister west border to barracks", 27, 3000.5, 980, 28, true},
		{"barracks east border", 28, 2999.5, 980, 27, true},
		{"inner cloister north border", 32, 4010, 1000.5, 33, true},
		{"cathedral south border", 33, 4010, 999.5, 32, true},
	}

	for _, tc := range tests {
		to, ok := EdgeExit(rects, tc.from, tc.wx, tc.wy, 1.5)
		if ok != tc.wantEdge || (ok && to != tc.wantTo) {
			t.Errorf("%s: EdgeExit = %d %v, want %d %v", tc.name, to, ok, tc.wantTo, tc.wantEdge)
		}
	}

	found := func(a, b int) bool {
		for _, n := range EdgeNeighbors(a) {
			if n == b {
				return true
			}
		}

		return false
	}

	for _, p := range [][2]int{{26, 27}, {27, 28}, {32, 33}} {
		if !found(p[0], p[1]) || !found(p[1], p[0]) {
			t.Errorf("no edge between %d and %d", p[0], p[1])
		}
	}

	// the new borders must not turn the dungeon levels into "outdoor" levels for the tile rule
	for _, tc := range [][3]int{{28, 1, 29}, {33, 1, 34}} {
		if to, ok := TileDestination(tc[0], tc[1]); !ok || to != tc[2] {
			t.Errorf("TileDestination(%d, %d) = %d %v, want %d", tc[0], tc[1], to, ok, tc[2])
		}
	}

	if isOutdoor(27) || isOutdoor(28) || isOutdoor(32) || isOutdoor(33) || !isOutdoor(26) {
		t.Error("only the Monastery Gate keeps its outdoor border rule")
	}
}
