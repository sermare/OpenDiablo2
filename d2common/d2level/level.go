package d2level

// Act numbers are 1..5 in this package (the original stores 0..4).
const (
	NumActs = 5
)

// Town levels, one per act. They are also the start levels of the acts
// (DRLG_GetActStartLevel 0x61a8b0, table {1,40,75,103,109}; verified).
const (
	RogueEncampment     = 1
	LutGholein          = 40
	KurastDocks         = 75
	PandemoniumFortress = 103
	Harrogath           = 109
)

// actStart holds the first level id of each act (verified, drlg.md), plus a
// sentinel after the last level of act 5.
var actStart = [NumActs + 1]int{1, 40, 75, 103, 109, 133}

// ActStartLevel returns the start level (the town) of act 1..5, or 0.
func ActStartLevel(act int) int {
	if act < 1 || act > NumActs {
		return 0
	}

	return actStart[act-1]
}

// ActOfLevel returns the act (1..5) a level id belongs to, or 0 for level 0
// or ids past the last act (DRLG_GetActFromLevelId compares against the same
// thresholds; ids from 133 are the Pandemonium levels, which Levels.txt puts
// in act 5).
func ActOfLevel(id int) int {
	if id < 1 {
		return 0
	}

	for act := NumActs; act >= 1; act-- {
		if id >= actStart[act-1] {
			return act
		}
	}

	return 0
}

// IsTown reports whether a level is the town of its act. These are exactly
// the levels SERVER_HandleTownTransition (0x534d40) looks at: 1, 0x28, 0x4b,
// 0x67 and 0x6d.
func IsTown(id int) bool {
	for _, t := range actStart[:NumActs] {
		if id == t {
			return true
		}
	}

	return false
}

// Difficulty is 0 normal, 1 nightmare, 2 hell.
type Difficulty int

// StartType selects the arrival rule used by the position finder
// (0x61ad90). Only the values named in the notes are listed.
type StartType int

// Start types.
const (
	// StartDefault is the default arrival (startType 0).
	StartDefault StartType = 0
	// StartPortal is used when arriving through a portal (startType 3).
	StartPortal StartType = 3
	// StartActChange (5) is the arrival type of an act change (session-core.md).
	StartActChange StartType = 5
	// StartSpecial (0xD) is used for waypoint travel to town levels and to
	// levels 0x2e, 0x4a and 0x85-0x88 (special arrival spots).
	StartSpecial StartType = 0xD
)

// WaypointStartType returns the start type SERVER_HandleC2STakeWaypoint uses
// for a destination level.
func WaypointStartType(dest int) StartType {
	if IsTown(dest) || dest == 0x2e || dest == 0x4a || (dest >= 0x85 && dest <= 0x88) {
		return StartSpecial
	}

	return StartDefault
}
