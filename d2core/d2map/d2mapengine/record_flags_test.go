package d2mapengine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

func TestRecordFlagsOfCell(t *testing.T) {
	tests := []struct {
		name string
		v    uint32
		want uint32
	}{
		{"plain floor (prop1 bit 7 only)", 0x80, 0},
		{"cell bit 17 blocks walking", 0x20000, 0x40},
		{"cell bit 16", 0x10000, 0x80},
		{"cell bit 28", 0x10000000, 0x2},
		{"all three", 0x10030000, 0xC2},
		{"other bits do not count", 0x4000000 | 0x20000000 | 8 | 4, 0},
	}

	for _, tt := range tests {
		if got := recordFlagsOfCell(tt.v); got != tt.want {
			t.Errorf("%s: recordFlagsOfCell(%#x) = %#x, want %#x", tt.name, tt.v, got, tt.want)
		}
	}
}

func TestCellValueRoundTrip(t *testing.T) {
	var tile d2ds1.Tile

	tile.Prop1, tile.Sequence, tile.Unknown1, tile.Style, tile.Unknown2 = 0x81, 3, 8, 5, 4
	// 0x81 | 3<<8 | 8<<14 | 5<<20 | 4<<26
	if got, want := cellValue(&tile), uint32(0x81|3<<8|8<<14|5<<20|4<<26); got != want {
		t.Fatalf("cellValue = %#x, want %#x", got, want)
	}

	if recordFlagsOfCell(cellValue(&tile)) != 0x40|0x2 {
		t.Fatalf("unknown1 bit 3 (cell bit 17) and unknown2 bit 2 (cell bit 28) give 0x42")
	}
}

// A lava floor tile has no sub-tile flag in its DT1, the DS1 cell bit 17 is
// what blocks it (verified in the exe, see map_tile.go).
func TestLavaFloorBlockedByCellBit(t *testing.T) {
	lava := d2dt1.Tile{Style: 2, Sequence: 0, Type: 0}
	lava.MaterialFlags.Lava = true

	m, _ := testEngine(1, 2, func(x, y int) bool { return false })
	m.dt1TileData = []d2dt1.Tile{lava}
	m.dt1Starts = []int{0}

	var blocked, open d2ds1.Tile

	blocked.Prop1, blocked.Style, blocked.Unknown1 = 0x81, 2, 8
	open.Prop1, open.Style = 0x81, 2

	m.tiles[0].Components = d2mapstamp.Tile{Floors: []d2ds1.Tile{blocked}}
	m.tiles[1].Components = d2mapstamp.Tile{Floors: []d2ds1.Tile{open}}
	m.tiles[0].PrepareTile(0, 0, m)
	m.tiles[1].PrepareTile(0, 1, m)

	for s := 0; s < 25; s++ {
		if !m.tiles[0].SubTiles[s].BlockWalk {
			t.Fatalf("sub-tile %d of the lava cell with bit 17 must block walking", s)
		}

		if m.tiles[1].SubTiles[s].BlockWalk {
			t.Fatalf("sub-tile %d of the plain floor must stay walkable", s)
		}
	}
}

// Real data: the Act 4 lava floor DT1 has 80 lava floor tiles of 494 and not
// one walk-blocking sub-tile; lavan.ds1 (a River of Flame room) has 512 floor
// cells of which 339 carry cell bit 17.
func TestRealLavaRoom(t *testing.T) {
	root := os.Getenv("D2_DS1_ROOT")
	if root == "" {
		t.Skip("D2_DS1_ROOT not set")
	}

	tiles := filepath.Join(root, "d2data", "data", "global", "tiles", "act4", "lava")

	dt1Data, err := os.ReadFile(filepath.Join(tiles, "floor.dt1"))
	if err != nil {
		t.Skip(err)
	}

	dt1, err := d2dt1.LoadDT1(dt1Data)
	if err != nil {
		t.Fatal(err)
	}

	lavaTiles, walkBits := 0, 0

	for i := range dt1.Tiles {
		if dt1.Tiles[i].Type != 0 || !dt1.Tiles[i].MaterialFlags.Lava {
			continue
		}

		lavaTiles++

		for _, s := range dt1.Tiles[i].SubTileFlags {
			if s.BlockWalk || s.BlockPlayerWalk {
				walkBits++
			}
		}
	}

	if lavaTiles != 80 || walkBits != 0 {
		t.Fatalf("lava floor tiles = %d (want 80), blocked sub-tiles = %d (want 0)", lavaTiles, walkBits)
	}

	dsData, err := os.ReadFile(filepath.Join(tiles, "lavan.ds1"))
	if err != nil {
		t.Skip(err)
	}

	ds, err := d2ds1.Unmarshal(dsData)
	if err != nil {
		t.Fatal(err)
	}

	w, h := ds.Size()
	floors, blocked := 0, 0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			f := ds.GetFloor(0).Tile(x, y)
			if f == nil || f.Prop1 == 0 || f.Hidden() {
				continue
			}

			floors++

			if recordFlagsOfCell(cellValue(f))&recordFlagBlockWalk != 0 {
				blocked++
			}
		}
	}

	if floors != 512 || blocked != 339 {
		t.Fatalf("lavan.ds1: floor cells %d (want 512), blocked by bit 17 %d (want 339)", floors, blocked)
	}
}
