package d2level

// The special tiles of the Lut Gholein presets (Act2/Town/LutW.ds1 and LutN.ds1, the same map turned) and where
// they lead. Levels.txt lists the town's exits as Vis0 = Sewers 1 (LvlWarp 19 "Town to Sewer Trap"), Vis1 = Sewers 1
// (20 "Town to Sewer Dock") and Vis2 = Harem 1 (24 "Town to Harem"), but the tiles of the presets carry styles that
// are not LvlWarp ids (tile styles 2, 3, 4, 31, 32, 33 and 34 beside the start marker 30), so they could not be
// resolved: the town had no way into the sewers or the palace.
//
// UNVERIFIED: the executable resolves the tiles through a table that is not decoded. The assignment below was made
// from the layout of the preset (the palace gate is the tile at the north east of the town, the sewer ladder and
// the dock hatch are the two southern/western tiles), and it is only used by the engine to give the three exits a
// place. The other special tiles stay unresolved.
const (
	LevelLutGholeinTown = 40
	LevelSewers1        = 47
	LevelHarem1         = 50
)

var lutGholeinTileStyles = map[int]int{
	31: LevelHarem1,  // north east, next to the palace gate
	4:  LevelSewers1, // west side: the trap door
	3:  LevelSewers1, // south east, at the harbour: the dock hatch
}

// TownTileDestination returns the level a special tile of a town preset leads to, by its style.
func TownTileDestination(level, style int) (int, bool) {
	if level != LevelLutGholeinTown {
		return 0, false
	}

	to, ok := lutGholeinTileStyles[style]
	if !ok {
		return 0, false
	}

	// only the exits Levels.txt lists
	if _, listed := TileLinkTo(level, to); !listed {
		return 0, false
	}

	return to, true
}
