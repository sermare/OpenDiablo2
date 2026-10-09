package d2level

// act3Slots are the Vis slots (Levels.txt Vis0..Vis7, in order, empty slots omitted) of the Act 3 outdoor
// levels that lead into a dungeon. The entrance presets of Kurast and the jungle carry a special tile whose
// style is the number of the slot: Upper Kurast (81) has the tiles of style 0..3 at its two sewer
// entrances (slots 0 and 1, both Sewers 1) and its two temples (slots 2 and 3, Ruined Temple ... ).
// OBSERVED on the levels generated for the playtest (the styles of the special tiles of 76..83 are exactly
// the used slot numbers); that the exe uses the slot number is UNVERIFIED.
var act3Slots = map[int]map[int]int{
	76: {0: 84, 1: 85},
	78: {0: 86, 1: 88},
	80: {0: 92, 1: 92, 2: 94, 3: 95},
	81: {0: 92, 1: 92, 2: 96, 3: 97},
	82: {2: 98, 3: 99},
	83: {0: 100},
}

// Act3SlotDestination returns the dungeon a special tile of style style leads to in an Act 3 outdoor level
// (Spider Forest, Flayer Jungle, the Kurast levels and Travincal).
func Act3SlotDestination(level, style int) (int, bool) {
	to, ok := act3Slots[level][style]

	return to, ok
}
