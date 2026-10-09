package d2level

import "testing"

// The Act 2 desert is a chain of seamless levels: Lut Gholein - Rocky Waste - Dry Hills - Far Oasis -
// Lost City - Valley of Snakes. The Canyon of the Magi stands alone (playtest bug: walkto:exit=41 from
// Lut Gholein found "no exit", the Act 2 borders were not in the link table).
func TestAct2DesertEdges(t *testing.T) {
	chain := []int{40, 41, 42, 43, 44, 45}

	for i := 0; i+1 < len(chain); i++ {
		a, b := chain[i], chain[i+1]

		var fwd, back bool

		for _, n := range EdgeNeighbors(a) {
			fwd = fwd || n == b
		}

		for _, n := range EdgeNeighbors(b) {
			back = back || n == a
		}

		if !fwd || !back {
			t.Errorf("levels %d and %d are not seamless neighbours (%v %v)", a, b, fwd, back)
		}
	}

	if got := EdgeNeighbors(46); len(got) != 0 {
		t.Errorf("the Canyon of the Magi has no seamless neighbour, got %v", got)
	}

	// Lut Gholein (56x56 at 1000,1000) with Rocky Waste (80x80) to its west
	rects := map[int]Rect{40: {1000, 1000, 56, 56}, 41: {920, 1000, 80, 80}}

	if to, ok := EdgeExit(rects, 40, 1001, 1020, 1.5); !ok || to != 41 {
		t.Errorf("walking out of the west gate: got %d %v", to, ok)
	}

	if _, ok := EdgeExit(rects, 40, 1030, 1020, 1.5); ok {
		t.Error("the middle of the town is no exit")
	}
}

// Each desert level has one tomb, lair or temple behind it, and the dungeon levels of Act 2 resolve their
// stairs like the Act 1 ones (up first).
func TestAct2SingleTileDestination(t *testing.T) {
	for level, want := range map[int]int{41: 55, 42: 56, 43: 62, 44: 65, 45: 58} {
		if got, ok := SingleTileDestination(level); !ok || got != want {
			t.Errorf("level %d leads to %d %v, want %d", level, got, ok, want)
		}
	}

	// the town (sewers and palace) and the canyon (seven tombs) have several destinations
	for _, level := range []int{40, 46} {
		if _, ok := SingleTileDestination(level); ok {
			t.Errorf("level %d has several destinations", level)
		}
	}

	// Halls of the Dead 1 (56): stairs up lead back to Dry Hills, the stairs down to level 57
	if to, ok := TileDestination(56, 0); !ok || to != 42 {
		t.Errorf("Halls of the Dead 1 up stairs: %d %v", to, ok)
	}

	if to, ok := TileDestination(56, 4); !ok || to != 57 {
		t.Errorf("Halls of the Dead 1 down stairs: %d %v", to, ok)
	}
}
