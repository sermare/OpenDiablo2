package d2mapengine

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
)

const blankFloorStyle = 30

// MapTile is a tile placed on the map
type MapTile struct {
	Components d2mapstamp.Tile
	RegionType d2enum.RegionIdType
	SubTiles   [25]d2dt1.SubTileFlags
}

// GetSubTileFlags returns the tile flags for the given subtile
func (t *MapTile) GetSubTileFlags(x, y int) *d2dt1.SubTileFlags {
	var subtileLookup = [5][5]int{
		{20, 21, 22, 23, 24},
		{15, 16, 17, 18, 19},
		{10, 11, 12, 13, 14},
		{5, 6, 7, 8, 9},
		{0, 1, 2, 3, 4},
	}

	return &t.SubTiles[subtileLookup[y][x]]
}

// PrepareTile selects which graphic to use and updates the tiles subtileflags
func (t *MapTile) PrepareTile(x, y int, me *MapEngine) {
	for wIdx := range t.Components.Walls {
		wall := &t.Components.Walls[wIdx]
		// empty cells (prop1 == 0) and hidden cells have no graphic; looking them up only logs warnings
		if wall.Hidden() || wall.Prop1 == 0 {
			continue
		}

		options := me.GetTiles(int(wall.Style), int(wall.Sequence), wall.Type)

		if options == nil {
			// e.g. graphic-less special marker tiles; keep preparing the remaining walls
			continue
		}

		wall.RandomIndex = getRandomTile(options, x, y, me.seed)

		for i := range t.SubTiles {
			t.SubTiles[i].Combine(options[wall.RandomIndex].SubTileFlags[i])
		}

		applyRecordFlags(&t.SubTiles, recordFlagsOfCell(cellValue(wall)))
	}

	for fIdx := range t.Components.Floors {
		floor := &t.Components.Floors[fIdx]
		if floor.Hidden() || floor.Prop1 == 0 {
			continue
		}

		// floor style 30 / sequence 0 is the DS1 convention for "no floor" (blank filler
		// cells next to rooms); no DT1 has a graphic for it. Hide it so that it is neither
		// rendered nor counted as ground by BlockEmptyTiles (observed in the Act 1 cave
		// and town DS1 files; whether the original treats it exactly so is unverified).
		if floor.Style == blankFloorStyle && floor.Sequence == 0 {
			floor.HiddenBytes = 1
			continue
		}

		options := me.GetTiles(int(floor.Style), int(floor.Sequence), 0)

		if options == nil {
			continue
		}

		if options[0].MaterialFlags.Lava {
			floor.Animated = true
			floor.RandomIndex = 0
		} else {
			floor.RandomIndex = getRandomTile(options, x, y, me.seed)
		}

		for i := range t.SubTiles {
			t.SubTiles[i].Combine(options[floor.RandomIndex].SubTileFlags[i])
		}

		applyRecordFlags(&t.SubTiles, recordFlagsOfCell(cellValue(floor)))
	}

	for sIdx := range t.Components.Shadows {
		shadow := &t.Components.Shadows[sIdx]
		if shadow.Hidden() || shadow.Prop1 == 0 {
			continue
		}

		options := me.GetTiles(int(shadow.Style), int(shadow.Sequence), 13)

		if options == nil {
			continue
		}

		shadow.RandomIndex = getRandomTile(options, x, y, me.seed)

		for i := range t.SubTiles {
			t.SubTiles[i].Combine(options[shadow.RandomIndex].SubTileFlags[i])
		}

		applyRecordFlags(&t.SubTiles, recordFlagsOfCell(cellValue(shadow)))
	}
}

// Walker's Alias Method for weighted random selection with xorshifting for random numbers
// Selects a random tile from the slice, rest of args just used for seeding
func getRandomTile(tiles []d2dt1.Tile, x, y int, seed int64) byte {
	var tileSeed uint64
	tileSeed = uint64(seed) + uint64(x)
	tileSeed *= uint64(y)

	const (
		xorshiftA = 13
		xorshiftB = 17
		xorshiftC = 5
	)

	tileSeed ^= tileSeed << xorshiftA
	tileSeed ^= tileSeed >> xorshiftB
	tileSeed ^= tileSeed << xorshiftC

	weightSum := 0

	for i := range tiles {
		weightSum += int(tiles[i].RarityFrameIndex)
	}

	if weightSum == 0 {
		return 0
	}

	random := tileSeed % uint64(weightSum)

	sum := 0

	for i := range tiles {
		sum += int(tiles[i].RarityFrameIndex)
		if sum >= int(random) {
			return byte(i)
		}
	}

	// This return shouldn't be hit
	return 0
}

// Record flags of a tile (floor, wall or shadow record of a room) that the
// room collision builder turns into whole-tile collision bits. VERIFIED in the
// exe (COLLISION_ApplyTileFlagsToGrid, 0x64d9f0): record flag 0x40 ORs 0x1
// (walk blocked) into all 25 sub-tiles, 0x80 ORs 0x4 and 0x2 ORs 0x10, on top
// of the 5x5 pattern of the DT1 tile. The record flags come from the DS1 cell
// value (DRLG_FillFloorTileRecord, 0x670aa0): cell bit 17 gives 0x40, bit 16
// gives 0x80 and bit 28 gives 0x2 (0x102). Lava floors carry no flag in their
// DT1 (all 25 sub-tile bytes are 0 in Act4/Lava/Floor.dt1); the DS1 floor
// cells with bit 17 set are what keeps the hero out of the lava.
const (
	recordFlagBlockWalk = 0x40
	recordFlagBit4      = 0x80
	recordFlagBit10     = 0x2
)

// cellValue rebuilds the DS1 cell dword of a tile record.
func cellValue(t *d2ds1.Tile) uint32 {
	v := uint32(t.Prop1) | uint32(t.Sequence)<<8 | uint32(t.Unknown1)<<14 | uint32(t.Style)<<20 | uint32(t.Unknown2)<<26

	if t.Hidden() {
		v |= 0x80000000
	}

	return v
}

// recordFlagsOfCell is the collision-relevant part of the record flags of a
// DS1 cell value (see the constants above).
func recordFlagsOfCell(v uint32) uint32 {
	var f uint32

	if v&0x20000 != 0 {
		f |= recordFlagBlockWalk
	}

	if v&0x10000 != 0 {
		f |= recordFlagBit4
	}

	if v&0x10000000 != 0 {
		f |= recordFlagBit10
	}

	return f
}

// applyRecordFlags ORs the collision bits of a tile record into all sub-tiles.
func applyRecordFlags(subs *[25]d2dt1.SubTileFlags, rf uint32) {
	if rf&(recordFlagBlockWalk|recordFlagBit4|recordFlagBit10) == 0 {
		return
	}

	for i := range subs {
		if rf&recordFlagBlockWalk != 0 {
			subs[i].BlockWalk = true
		}

		if rf&recordFlagBit4 != 0 {
			subs[i].BlockJump = true
		}

		if rf&recordFlagBit10 != 0 {
			subs[i].Unknown1 = true
		}
	}
}
