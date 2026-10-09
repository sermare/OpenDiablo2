package d2level

import "testing"

// the Act 1 world of game seed 0 (drlgworld golden): the town sits on top of
// Blood Moor, Cold Plains to the west of Blood Moor.
var seed0 = map[int]Rect{
	1: {1136, 864, 56, 40},
	2: {1096, 904, 96, 56},
	3: {1016, 920, 80, 80},
	4: {1000, 1000, 80, 80},
}

func TestSharedBorder(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want Border
		ok   bool
	}{
		{"town to blood moor", 1, 2, Border{South, 1136, 1192, 904}, true},
		{"blood moor to town", 2, 1, Border{North, 1136, 1192, 904}, true},
		{"blood moor to cold plains", 2, 3, Border{West, 920, 960, 1096}, true},
		{"cold plains to blood moor", 3, 2, Border{East, 920, 960, 1096}, true},
		{"town to cold plains do not touch", 1, 3, Border{}, false},
	}

	for _, tc := range tests {
		got, ok := SharedBorder(seed0[tc.a], seed0[tc.b])
		if ok != tc.ok || got != tc.want {
			t.Errorf("%s: got %+v %v, want %+v %v", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}

func TestEdgeExit(t *testing.T) {
	tests := []struct {
		name      string
		from      int
		wx, wy, m float64
		want      int
		exists    bool
	}{
		{"blood moor top, inside the shared range", 2, 1160, 904.5, 1.5, 1, true},
		{"blood moor top, left of the town", 2, 1110, 904.5, 1.5, 0, false},
		{"blood moor top, too far from the border", 2, 1160, 908, 1.5, 0, false},
		{"blood moor west edge, in the shared range", 2, 1097, 940, 1.5, 3, true},
		{"blood moor west edge, below the range", 2, 1097, 961, 1.5, 0, false},
		{"town bottom", 1, 1150, 902.5, 1.5, 2, true},
		{"town middle", 1, 1150, 880, 1.5, 0, false},
		{"unknown level", 99, 0, 0, 1.5, 0, false},
	}

	for _, tc := range tests {
		got, ok := EdgeExit(seed0, tc.from, tc.wx, tc.wy, tc.m)
		if ok != tc.exists || (ok && got != tc.want) {
			t.Errorf("%s: got %d %v, want %d %v", tc.name, got, ok, tc.want, tc.exists)
		}
	}
}

func TestEdgeArrivalIsInsideAndOutsideTheMargin(t *testing.T) {
	// leaving the town through its bottom edge lands inset tiles below the border,
	// on the same world column
	x, y, ok := EdgeArrival(seed0, 1, 2, 1150, 902.8, 3)
	if !ok || x != 1150 || y != 907 {
		t.Errorf("town to blood moor: got (%v,%v) %v", x, y, ok)
	}

	// the arrival must not be inside the exit margin of the way back
	if _, back := EdgeExit(seed0, 2, x, y, 1.5); back {
		t.Error("arrival is inside the margin of the way back")
	}

	// a position beside the shared range is clamped into it
	x, _, _ = EdgeArrival(seed0, 2, 1, 1120, 904.2, 3)
	if x != 1136.5 {
		t.Errorf("clamped column = %v, want 1136.5", x)
	}

	if _, _, ok := EdgeArrival(seed0, 1, 3, 0, 0, 3); ok {
		t.Error("levels that do not touch have no arrival")
	}
}

func TestEdgeTarget(t *testing.T) {
	x, y, ok := EdgeTarget(seed0, 2, 1, 2)
	if !ok || x != 1164 || y != 906 {
		t.Errorf("blood moor to town target = (%v,%v) %v", x, y, ok)
	}
}

func TestEdgeNeighborsAreTheDRLGLinks(t *testing.T) {
	got := EdgeNeighbors(3)
	want := map[int]bool{2: true, 4: true, 17: true}

	if len(got) != len(want) {
		t.Fatalf("cold plains neighbours = %v", got)
	}

	for _, n := range got {
		if !want[n] {
			t.Errorf("unexpected neighbour %d", n)
		}
	}
}
