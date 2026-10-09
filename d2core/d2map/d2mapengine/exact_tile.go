package d2mapengine

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

// ExactTile is a tile a level generator chose itself: tile Index of the DT1
// file File (as the generator's room tile library holds it), instead of a
// (style, sequence, orientation) key the engine would resolve with a random
// variant.
type ExactTile struct {
	File  string
	Index int
}

// exactComponent resolves an ExactTile in the engine's DT1 set. The DT1 file is
// loaded when the level type does not list it (the generator's library always
// holds Blank.dt1, InvisWal.dt1 and Warp.dt1).
func (m *MapEngine) exactComponent(t ExactTile) (c d2ds1.Tile, dt *d2dt1.Tile, ok bool) {
	file := strings.ToLower(strings.ReplaceAll(t.File, "\\", "/"))

	slot := m.dt1Slot(file)
	if slot < 0 {
		m.addDT1(file)
		slot = m.dt1Slot(file)
	}

	if slot < 0 {
		return c, nil, false
	}

	end := len(m.dt1TileData)
	if slot+1 < len(m.dt1Starts) {
		end = m.dt1Starts[slot+1]
	}

	g := m.dt1Starts[slot] + t.Index
	if t.Index < 0 || g >= end {
		return c, nil, false
	}

	dt = &m.dt1TileData[g]

	// the renderer and PrepareTile look a tile up as GetTiles(style, sequence,
	// type)[RandomIndex]: this tile's position in that list
	m.extendRank()

	rnd := m.dt1Rank[g]

	if rnd > 255 {
		return c, nil, false
	}

	c.Style, c.Sequence, c.RandomIndex = byte(dt.Style), byte(dt.Sequence), byte(rnd)
	c.Prop1 = 1
	c.Type = d2enum.TileType(dt.Type)

	return c, dt, true
}

func (m *MapEngine) dt1Slot(file string) int {
	for i := range m.dt1Files {
		if m.dt1Files[i] == file {
			return i
		}
	}

	return -1
}

// SetExactTiles puts the tiles a generator picked from its own DT1 library on
// one map cell: floors, walls and shadows, each already resolved to a DT1 tile.
// The sub-tile flags are those of the chosen tiles. Tiles outside the map are
// ignored; records whose DT1 tile cannot be found are skipped.
//
// keepMarkers keeps the logical marker walls (special type 10/11 walls that are
// hidden or have a style of 8 and above: cave entrance cells, start markers) a
// DS1 stamp put on the cell, because the game keeps those as warp objects, not
// as tile records.
func (m *MapEngine) SetExactTiles(x, y int, region d2enum.RegionIdType, keepMarkers bool, floors, walls, shadows []ExactTile) {
	if x < 0 || y < 0 || x >= m.size.Width || y >= m.size.Height {
		return
	}

	var comp d2mapstamp.Tile

	t := &m.tiles[m.tileCoordinateToIndex(x, y)]

	var markers []d2ds1.Tile

	if keepMarkers {
		for _, w := range t.Components.Walls {
			if w.Type.Special() && (w.Hidden() || w.Style >= 8) {
				markers = append(markers, w)
			}
		}
	}

	*t = MapTile{RegionType: region}

	add := func(list []ExactTile, dst *[]d2ds1.Tile) {
		for _, e := range list {
			c, dt, ok := m.exactComponent(e)
			if !ok {
				continue
			}

			*dst = append(*dst, c)

			for i := range t.SubTiles {
				t.SubTiles[i].Combine(dt.SubTileFlags[i])
			}
		}
	}

	add(floors, &comp.Floors)
	add(walls, &comp.Walls)
	add(shadows, &comp.Shadows)

	comp.Walls = append(comp.Walls, markers...)
	t.Components = comp
}

// extendRank fills dt1Rank for the DT1 tiles loaded since the last call.
func (m *MapEngine) extendRank() {
	if m.dt1KeyCount == nil {
		m.dt1KeyCount = map[[3]int32]int{}
	}

	for i := len(m.dt1Rank); i < len(m.dt1TileData); i++ {
		t := &m.dt1TileData[i]
		k := [3]int32{t.Style, t.Sequence, t.Type}
		m.dt1Rank = append(m.dt1Rank, m.dt1KeyCount[k])
		m.dt1KeyCount[k]++
	}
}
