package d2object

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"

// WaypointBitOf is the waypoint bit a waypoint object activates when the hero
// touches it: the bit of the level the object stands in. It returns false if
// the object is not a waypoint, is not allowed in that level's act, or the
// level has no waypoint bit.
func WaypointBitOf(def Def, level int, expansion bool) (bit int, ok bool) {
	if !def.IsWaypoint() {
		return 0, false
	}

	if !def.AllowedIn(d2level.ActOfLevel(level), expansion) {
		return 0, false
	}

	return d2level.WaypointBit(level)
}
