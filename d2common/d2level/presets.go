package d2level

// The exits of the single-file preset levels of Act 2: Lut Gholein (40, Act2/Town/LutW.ds1 and LutN.ds1) and the
// Harem (50, Act2/Palace/Harem2.ds1). Their DS1 files carry special tiles with small styles that are not LvlWarp
// ids: Lut Gholein has the styles 2, 3 and 4 (and 31-34 and the start marker 30), the Harem 0, 2 and 3.
//
// Levels.txt lists the exits of these levels in the Vis slots 2, 3 and 4 (town: Sewers 1 twice, the Harem) and 0, 2
// and 3 (Harem: town, Corrupt Harem 1 twice): exactly the styles the files carry, and the slots in between are empty.
// So the tile of style n leads to the level in Vis slot n (see Act2VisDestination).
const (
	LevelLutGholeinTown = 40
	LevelSewers1        = 47
	LevelHarem1         = 50
	LevelHarem2         = 51
)

// Act2VisDestination returns the level in the Vis slot of an Act 2 level (0-7): the special tile of a DS1 file with the style n
// leads there, for every Act 2 level (the town, the harem, the sewers, the palace cellars, the tombs, the lairs and the
// canyon: Vis0 of the canyon is empty and the seven tomb tiles carry the styles 1-7). UNVERIFIED: the executable resolves
// the tiles through a table that is not decoded; every special tile style seen in the Act 2 files fits the slot of
// Levels.txt (TestAct2VisSlotsMatchLevelsTxt compares the table with a real Levels.txt when D2_TABLES is set).
func Act2VisDestination(level, slot int) (int, bool) {
	v, ok := act2VisSlots[level]
	if !ok || slot < 0 || slot >= len(v) || v[slot] == 0 {
		return 0, false
	}

	return v[slot], true
}

// IsActPresetLevel reports the single-file preset levels besides the towns that the engine builds from their DS1.
func IsActPresetLevel(level int) bool { return level == LevelHarem1 }

// PresetTileDestination returns the level a special tile of a single-file preset level (the town and the harem) leads to.
func PresetTileDestination(level, style int) (int, bool) {
	if level != LevelLutGholeinTown && level != LevelHarem1 {
		return 0, false
	}

	return Act2VisDestination(level, style)
}
