package d2s

// Waypoint is the index of a waypoint bit in the 40-bit mask of a save.
// A real save has 39 bits (a fully activated mask is 0x7FFFFFFFFF, verified
// with the NokkaSorc sample). The order is the community layout (act by act
// in game order); only the count was verified, the per-bit names are not.
type Waypoint int

// Waypoint indices.
const (
	WPRogueEncampment Waypoint = iota
	WPColdPlains
	WPStonyField
	WPDarkWood
	WPBlackMarsh
	WPOuterCloister
	WPJailLevel1
	WPInnerCloister
	WPCatacombsLevel2

	WPLutGholein
	WPSewersLevel2
	WPDryHills
	WPHallsOfTheDeadLevel2
	WPFarOasis
	WPLostCity
	WPPalaceCellarLevel1
	WPArcaneSanctuary
	WPCanyonOfTheMagi

	WPKurastDocks
	WPSpiderForest
	WPGreatMarsh
	WPFlayerJungle
	WPLowerKurast
	WPKurastBazaar
	WPUpperKurast
	WPTravincal
	WPDuranceOfHateLevel2

	WPPandemoniumFortress
	WPCityOfTheDamned
	WPRiverOfFlame

	WPHarrogath
	WPFrigidHighlands
	WPArreatPlateau
	WPCrystallinePassage
	WPHallsOfPain
	WPGlacialTrail
	WPFrozenTundra
	WPTheAncientsWay
	WPWorldstoneKeepLevel2

	// NumWaypoints is the number of waypoint bits.
	NumWaypoints
)

// actWaypointStart holds the first waypoint of each act, plus the end.
var actWaypointStart = [NumActs + 1]Waypoint{
	WPRogueEncampment, WPLutGholein, WPKurastDocks, WPPandemoniumFortress, WPHarrogath, NumWaypoints,
}

// WaypointsOfAct returns the waypoints of act 1..5 in order (nil if invalid).
func WaypointsOfAct(act int) []Waypoint {
	if act < 1 || act > NumActs {
		return nil
	}

	var out []Waypoint

	for w := actWaypointStart[act-1]; w < actWaypointStart[act]; w++ {
		out = append(out, w)
	}

	return out
}

// Act returns the act (1..5) a waypoint belongs to, or 0 if invalid.
func (w Waypoint) Act() int {
	if w < 0 || w >= NumWaypoints {
		return 0
	}

	for act := NumActs; act >= 1; act-- {
		if w >= actWaypointStart[act-1] {
			return act
		}
	}

	return 0
}

// Has reports whether a waypoint is activated in a difficulty (0..2).
func (w Waypoints) Has(difficulty int, wp Waypoint) bool {
	if difficulty < 0 || difficulty >= numDifficulties || wp < 0 || wp >= NumWaypoints {
		return false
	}

	return w[difficulty]>>uint(wp)&1 != 0
}

// Set activates or deactivates a waypoint.
func (w *Waypoints) Set(difficulty int, wp Waypoint, on bool) {
	if difficulty < 0 || difficulty >= numDifficulties || wp < 0 || wp >= NumWaypoints {
		return
	}

	if on {
		w[difficulty] |= 1 << uint(wp)
	} else {
		w[difficulty] &^= 1 << uint(wp)
	}
}

// ActiveInAct lists the activated waypoints of an act.
func (w Waypoints) ActiveInAct(difficulty, act int) []Waypoint {
	var out []Waypoint

	for _, wp := range WaypointsOfAct(act) {
		if w.Has(difficulty, wp) {
			out = append(out, wp)
		}
	}

	return out
}
