package d2level

import "testing"

// Lut Gholein's two sewer entrances (the manhole, style 2, and the dock stairs, style 3) lead to Sewers Level 1.
func TestLutGholeinSewerTiles(t *testing.T) {
	for _, style := range []int{2, 3} {
		if to, ok := TileDestination(40, style); !ok || to != 47 {
			t.Errorf("TileDestination(40, %d) = %d, %v; want 47", style, to, ok)
		}
	}

	// the other special tiles of the town (palace doors, ...) are not sewer entrances
	if to, ok := TileDestination(40, 13); ok && to == 47 {
		t.Errorf("style 13 must not lead to the sewers")
	}
}
