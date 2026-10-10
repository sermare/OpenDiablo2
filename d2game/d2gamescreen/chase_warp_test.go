package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
)

// Act 2 playthrough (batch 1 verify): in the Halls of the Dead the hero arrives on the warp tile that leads back
// to Dry Hills. A monster that came to the stairs and was chased there counted as a click on the tile
// (OnPlayerMove -> targetWarpAt) and the fight ended in the wrong level.
func TestChaseDoesNotTargetAWarpTile(t *testing.T) {
	v := &Game{}
	v.levels.warps = []d2mapengine.WarpTile{{TileX: 43, TileY: 31}}

	var moved [][2]float64

	record := func(x, y float64) { moved = append(moved, [2]float64{x, y}) }

	// a stale target from an earlier order is dropped, and the monster's position selects nothing
	v.levels.warpTarget = &v.levels.warps[0]
	v.chaseWith(43.5, 31.5, record)

	if v.levels.warpTarget != nil {
		t.Errorf("a chase order on the stairs targeted the warp tile %+v", *v.levels.warpTarget)
	}

	if len(moved) != 1 || moved[0] != [2]float64{43.5, 31.5} {
		t.Errorf("the chase order was not passed on: %v", moved)
	}

	// the click path still selects the tile, so the test above is about the chase and not about the data
	v.targetWarpAt(43.5, 31.5)

	if v.levels.warpTarget == nil {
		t.Fatal("a click on the stairs no longer targets the warp tile")
	}
}
