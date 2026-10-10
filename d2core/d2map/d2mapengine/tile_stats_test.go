package d2mapengine

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2geom"
)

// TileStats must count a cell without a floor as a void, and a tile reference
// whose (style, sequence, type) no loaded DT1 holds as a fallback.
func TestTileStats(t *testing.T) {
	tile := func(style, seq, typ int32) d2dt1.Tile {
		return d2dt1.Tile{Style: style, Sequence: seq, Type: typ}
	}

	m := &MapEngine{
		size:        d2geom.Size{Width: 3, Height: 1},
		tiles:       make([]MapTile, 3),
		dt1Files:    []string{"a.dt1"},
		dt1Starts:   []int{0},
		dt1TileData: []d2dt1.Tile{tile(1, 0, 0), tile(2, 0, 1), tile(3, 0, 13)},
	}

	// cell 0: floor 1/0 and wall 2/0 type 1 (both present); cell 1: no floor; cell 2: floor 9/9 (missing)
	m.SetExactTiles(0, 0, d2enum.RegionAct1Town, false,
		[]ExactTile{{File: "a.dt1", Index: 0}}, []ExactTile{{File: "a.dt1", Index: 1}}, []ExactTile{{File: "a.dt1", Index: 2}})
	m.tiles[2].Components.Floors = append(m.tiles[2].Components.Floors, m.tiles[0].Components.Floors[0])
	m.tiles[2].Components.Floors[0].Style, m.tiles[2].Components.Floors[0].Sequence = 9, 9

	s := m.TileStats()

	if s.Cells != 3 || s.NoFloor != 2 || s.Floors != 2 || s.Walls != 1 || s.Shadows != 1 {
		t.Errorf("counts: %s", s)
	}

	if s.FloorMissing != 1 || s.Fallbacks() != 1 || s.MissingKeys["floor:9/9"] != 1 {
		t.Errorf("fallbacks: %s", s)
	}
}
