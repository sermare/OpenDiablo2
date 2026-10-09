package d2path

import "testing"

func TestPlaceClusterSpacingAndWalls(t *testing.T) {
	g := NewCellGrid(0, 0, 40, 40)
	for x := 0; x < 40; x++ {
		g.Set(x, 20, FlagWalk) // a wall through the centre row
	}

	pts := PlaceCluster(g, MaskMonster, Point{20, 20}, 8, 2, 12)
	if len(pts) != 8 {
		t.Fatalf("placed %d of 8", len(pts))
	}

	for i, a := range pts {
		if Blocked(g, a.X, a.Y, MaskMonster) {
			t.Errorf("%v is blocked", a)
		}

		for _, b := range pts[i+1:] {
			if abs(a.X-b.X) < 2 && abs(a.Y-b.Y) < 2 {
				t.Errorf("%v and %v too close", a, b)
			}
		}
	}
}

func TestPlaceClusterCramped(t *testing.T) {
	g := NewCellGrid(0, 0, 3, 3)
	if got := PlaceCluster(g, MaskMonster, Point{1, 1}, 5, 2, 5); len(got) >= 5 {
		t.Errorf("a 3x3 room cannot hold 5 spaced units, got %d", len(got))
	}
}

func TestLayeredGridUnits(t *testing.T) {
	base := NewCellGrid(0, 0, 10, 10)
	l := NewLayeredGrid(base)
	l.Set(3, 3, FlagMonster)
	l.Set(4, 3, FlagPlayer)
	l.Set(4, 3, FlagCorpse)

	if Blocked(l, 3, 3, MaskMonster) {
		t.Error("footprint bits are not in the monster mask")
	}

	if !Blocked(l, 3, 3, MaskMonster|MaskUnits) || !Blocked(l, 4, 3, MaskUnits) {
		t.Error("units must block with MaskUnits")
	}

	l.Clear(4, 3, FlagPlayer)

	if Blocked(l, 4, 3, MaskUnits) || l.Overlay(4, 3) != FlagCorpse {
		t.Errorf("corpse bit must remain and not block: %x", l.Overlay(4, 3))
	}

	l.Clear(4, 3, FlagCorpse)

	if l.Overlay(4, 3) != 0 {
		t.Error("overlay should be empty")
	}

	if l.Flags(-1, -1) != OutOfGrid {
		t.Error("out of grid passes through")
	}
}
