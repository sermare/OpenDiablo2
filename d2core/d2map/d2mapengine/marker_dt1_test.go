package d2mapengine

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
)

func TestIsMarkerDT1(t *testing.T) {
	for file, want := range map[string]bool{
		"act1/barracks/warp.dt1":       true,
		"act1/barracks/inviswal.dt1":   true,
		"act4/mesa/inv_wall.dt1":       true,
		"expansion/town/collision.dt1": true,
		"act1\\Barracks\\Warp.dt1":     true,
		"act1/town/floor.dt1":          false,
		"act2/town/walls.dt1":          false,
		"act1/outdoors/blank.dt1":      false,
		"expansion/town/townwest.dt1x": false,
	} {
		if got := isMarkerDT1(file); got != want {
			t.Errorf("%s: %v want %v", file, got, want)
		}
	}
}

func TestStripMarkerGraphics(t *testing.T) {
	tiles := []d2dt1.Tile{
		{Type: 10, Width: 160, Height: -128, Blocks: make([]d2dt1.Block, 3)},
		{Type: 0, Width: 160, Height: -128, Blocks: make([]d2dt1.Block, 3)},
	}

	stripMarkerGraphics(tiles)

	if tiles[0].Blocks != nil || tiles[0].Height != 0 || tiles[0].Width != 0 {
		t.Errorf("wall marker kept graphics: %+v", tiles[0])
	}

	if tiles[1].Blocks != nil || tiles[1].Width != 160 {
		t.Errorf("floor marker must keep its size but lose blocks: %+v", tiles[1])
	}
}
