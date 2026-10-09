package d2level

import (
	"errors"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// waypointLevels maps the 39 waypoint bits of a save to level ids. It is the
// Waypoint column of Levels.txt (patch_d2, 1.14b): the level with Waypoint=N
// uses bit N. The game maps a level to its bit with 0x6634d0 (notes). This
// order differs from the names in the d2s package in act 5: bit 34 is the
// Glacial Trail (level 115) and bit 35 the Halls of Pain (123); the d2s
// package names were never verified.
var waypointLevels = [d2s.NumWaypoints]int{
	1, 3, 4, 5, 6, 27, 29, 32, 35, // act 1
	40, 48, 42, 57, 43, 44, 52, 74, 46, // act 2
	75, 76, 77, 78, 79, 80, 81, 83, 101, // act 3
	103, 106, 107, // act 4
	109, 111, 112, 113, 115, 123, 117, 118, 129, // act 5
}

// WaypointLevel returns the level id of waypoint bit bit, or 0.
func WaypointLevel(bit int) int {
	if bit < 0 || bit >= len(waypointLevels) {
		return 0
	}

	return waypointLevels[bit]
}

// WaypointBit returns the waypoint bit of a level, if the level has one.
func WaypointBit(level int) (int, bool) {
	for bit, l := range waypointLevels {
		if l == level {
			return bit, true
		}
	}

	return 0, false
}

// WaypointBitsOfAct returns the waypoint bits whose level is in act 1..5, in
// menu order.
func WaypointBitsOfAct(act int) []int {
	var out []int

	for bit, l := range waypointLevels {
		if ActOfLevel(l) == act {
			out = append(out, bit)
		}
	}

	return out
}

// DefaultLevelNames are English names of the waypoint levels, used when the
// string tables are not available (tests, tools). The game shows the
// Levels.txt LevelName string instead.
var DefaultLevelNames = map[int]string{
	1: "Rogue Encampment", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 6: "Black Marsh",
	27: "Outer Cloister", 29: "Jail Level 1", 32: "Inner Cloister", 35: "Catacombs Level 2",
	40: "Lut Gholein", 48: "Sewers Level 2", 42: "Dry Hills", 57: "Halls of the Dead Level 2",
	43: "Far Oasis", 44: "Lost City", 52: "Palace Cellar Level 1", 74: "Arcane Sanctuary", 46: "Canyon of the Magi",
	75: "Kurast Docks", 76: "Spider Forest", 77: "Great Marsh", 78: "Flayer Jungle", 79: "Lower Kurast",
	80: "Kurast Bazaar", 81: "Upper Kurast", 83: "Travincal", 101: "Durance of Hate Level 2",
	103: "The Pandemonium Fortress", 106: "City of the Damned", 107: "River of Flame",
	109: "Harrogath", 111: "Frigid Highlands", 112: "Arreat Plateau", 113: "Crystalline Passage",
	115: "Glacial Trail", 123: "Halls of Pain", 117: "Frozen Tundra", 118: "The Ancients' Way",
	129: "Worldstone Keep Level 2",
}

// WaypointEntry is one row of the waypoint panel.
type WaypointEntry struct {
	Bit    int
	Level  int
	Active bool // the character has activated it
	// Loadable is whether the engine can build the level right now. Active but
	// not loadable entries are listed greyed out.
	Loadable bool
}

// Enabled reports whether the entry can be chosen.
func (e WaypointEntry) Enabled() bool { return e.Active && e.Loadable }

// WaypointList builds the panel rows for an act: every waypoint of the act in
// menu order. Rows the character has not activated are included with
// Active=false (the real panel shows them dimmed too); canLoad may be nil.
func WaypointList(wp d2s.Waypoints, diff Difficulty, act int, canLoad func(level int) bool) []WaypointEntry {
	var out []WaypointEntry

	for _, bit := range WaypointBitsOfAct(act) {
		lvl := waypointLevels[bit]
		out = append(out, WaypointEntry{
			Bit:      bit,
			Level:    lvl,
			Active:   wp.Has(int(diff), d2s.Waypoint(bit)),
			Loadable: canLoad == nil || canLoad(lvl),
		})
	}

	return out
}

// Waypoint travel errors. The original returns small integers
// (SERVER_ValidateWaypointTravel 0x547450: 0 ok, 1..3 errors); their exact
// numbering is UNVERIFIED, so they are named here instead.
var (
	ErrWaypointCooldown     = errors.New("waypoint: less than 10 s since the last level change")
	ErrWaypointWrongAct     = errors.New("waypoint: object is in another act than the player")
	ErrWaypointOutOfRange   = errors.New("waypoint: player is too far from the waypoint")
	ErrWaypointInvalidLevel = errors.New("waypoint: target level has no waypoint")
	ErrWaypointNotActive    = errors.New("waypoint: waypoint is not activated for this difficulty")
	ErrWaypointNotInAct     = errors.New("waypoint: target level is in another act")
)

// WaypointRange is the distance (tiles) within which the waypoint object can
// be used. The original reads the range from objects.txt OperateRange via
// 0x546de0 (value in a register, UNVERIFIED); 2 matches the column for the
// waypoint objects.
const WaypointRange = 2.0

// WaypointRequest is a C2S 0x49 (take waypoint) as the server sees it.
type WaypointRequest struct {
	PlayerAct int // act of the player, 1..5
	ObjectAct int // act of the waypoint object, 1..5
	Distance  float64
	Range     float64 // 0 = WaypointRange
	Target    int     // destination level id
	Diff      Difficulty
	Waypoints d2s.Waypoints
}

// ValidateWaypoint applies the checks of SERVER_ValidateWaypointTravel. The
// cooldown is checked separately with Cooldown.Ready, as in the original.
// Target 0 means "just close the menu" and is not an error.
func ValidateWaypoint(r WaypointRequest) error {
	if r.Target == 0 {
		return nil
	}

	if r.ObjectAct != r.PlayerAct {
		return ErrWaypointWrongAct
	}

	rng := r.Range
	if rng <= 0 {
		rng = WaypointRange
	}

	if r.Distance > rng {
		return ErrWaypointOutOfRange
	}

	bit, ok := WaypointBit(r.Target)
	if !ok {
		return ErrWaypointInvalidLevel
	}

	// The notes say travel is limited to the current act ("same-act check").
	if ActOfLevel(r.Target) != r.PlayerAct {
		return ErrWaypointNotInAct
	}

	if !r.Waypoints.Has(int(r.Diff), d2s.Waypoint(bit)) {
		return ErrWaypointNotActive
	}

	return nil
}

// ActivateWaypoint sets the bit of a level's waypoint for a difficulty and
// reports whether it was newly set (false if already active or the level has
// no waypoint).
func ActivateWaypoint(wp *d2s.Waypoints, diff Difficulty, level int) bool {
	bit, ok := WaypointBit(level)
	if !ok || wp.Has(int(diff), d2s.Waypoint(bit)) {
		return false
	}

	wp.Set(int(diff), d2s.Waypoint(bit), true)

	return true
}
