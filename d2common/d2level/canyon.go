package d2level

// The Canyon of the Magi (level 46) has no seamless neighbour. Its seven tomb entrances are the special tiles of the
// seven King Tomb presets (Act2/BigCliff/TalCor.ds1, TalLeft1-3.ds1, TalRght1-3.ds1), which carry the styles 1 to 7.
// Levels.txt lists the seven tombs as the Vis slots 0-6 of level 46 (levels 66-72, LvlWarp ids 38-44 "Desert to
// Tomb Tal 1-7").
const (
	LevelCanyonOfTheMagi = 46
	FirstTalRashaTomb    = 66
	LastTalRashaTomb     = 72
)

// CanyonTombDestination returns the tomb level that the entrance tile of the given style leads to: style n is
// "Tal n", the n-th Vis slot of the canyon. UNVERIFIED: the executable resolves the tile through a table that is
// not decoded; the numbering by style is the simplest rule that gives every entrance a different tomb (the old
// dungeon-stair rule resolved only the styles 4 to 7, to the tombs 66 to 69). Which of the seven tombs is the real
// one is decided by the game seed (d2drlg.RealTomb), not by the entrance.
func CanyonTombDestination(style int) (int, bool) {
	if style < 1 || style > LastTalRashaTomb-FirstTalRashaTomb+1 {
		return 0, false
	}

	to := FirstTalRashaTomb + style - 1

	// the level must really be one of the canyon's links
	if _, ok := TileLinkTo(LevelCanyonOfTheMagi, to); !ok {
		return 0, false
	}

	return to, true
}
