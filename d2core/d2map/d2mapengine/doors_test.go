package d2mapengine

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2geom"
)

// newBareEngine builds a map engine without assets: 4x4 tiles of open floor,
// with one wall tile whose sub-tiles all block.
func newBareEngine() *MapEngine {
	m := &MapEngine{size: d2geom.Size{Width: 4, Height: 4}, objBlock: NewObjectCollision()}
	m.tiles = make([]MapTile, 16)

	for i := range m.tiles[1+1*4].SubTiles {
		m.tiles[1+1*4].SubTiles[i].BlockWalk = true // tile (1,1)
	}

	return m
}

func TestWalkBlocked(t *testing.T) {
	m := newBareEngine()

	cases := []struct {
		name string
		x, y int
		want bool
	}{
		{"open floor", 2, 2, false},
		{"wall tile", 5, 5, true},
		{"wall tile edge", 9, 9, true},
		{"next to wall", 10, 5, false},
		{"negative", -1, 3, true},
		{"past the right edge", 20, 3, true},
		{"past the bottom", 3, 20, true},
	}
	for _, tc := range cases {
		if got := m.WalkBlocked(tc.x, tc.y); got != tc.want {
			t.Errorf("%s: WalkBlocked(%d,%d) = %v, want %v", tc.name, tc.x, tc.y, got, tc.want)
		}
	}

	// a closed door adds blocked cells; clearing it frees them
	m.objBlock.Set("door", 12, 12, 1, 3)

	if !m.WalkBlocked(12, 13) || !m.DoorBlocks(12, 14) || m.WalkBlocked(13, 13) {
		t.Error("door footprint not applied")
	}

	m.objBlock.Clear("door")

	if m.WalkBlocked(12, 13) {
		t.Error("door cells must be free after clearing")
	}
}

func TestNearestWalkable(t *testing.T) {
	m := newBareEngine()

	if x, y, ok := m.NearestWalkable(2, 2, 3); !ok || x != 2 || y != 2 {
		t.Errorf("free start should be returned as is, got %d,%d,%v", x, y, ok)
	}

	// inside the 5x5 wall tile (subtiles 5..9): the nearest free cell is 3 away at most
	x, y, ok := m.NearestWalkable(7, 7, 6)
	if !ok || m.WalkBlocked(x, y) {
		t.Fatalf("expected a walkable cell, got %d,%d,%v", x, y, ok)
	}

	if d := maxInt(abs(x-7), abs(y-7)); d != 3 {
		t.Errorf("nearest free cell is %d away, want 3", d)
	}

	if _, _, ok := m.NearestWalkable(7, 7, 1); ok {
		t.Error("radius 1 around the middle of a wall must find nothing")
	}
}
