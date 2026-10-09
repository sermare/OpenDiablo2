package d2missile

import "math"

// Serial is optionally implemented by a Target: its unit id (unit+0xc in the
// exe). Guided Arrow re-targeting prefers the lowest (verified 0x569a40).
type Serial interface{ Serial() int }

// Sized is optionally implemented by a Target: UNIT_GetSizeX, subtracted
// (halved) from distances (0x642b10).
type Sized interface{ Size() int }

// PlayerAligned is optionally implemented by a Target: a monster that is in
// state 105 (alignment) with stat 172 equal to 2 counts as a player for
// CollideType 1 missiles (predicate 0x5a6210 via 0x625c10, verified; what the
// value 2 means in game terms is UNVERIFIED).
type PlayerAligned interface{ PlayerAligned() bool }

// lowestSerial picks the candidate with the lowest unit id (the exe's
// callback 0x569a40); targets without a Serial keep the listing order.
func lowestSerial(ts []Target) Target {
	var best Target

	bestID := 0

	for _, t := range ts {
		if t == nil || !t.Alive() {
			continue
		}

		id := 0
		if sr, ok := t.(Serial); ok {
			id = sr.Serial()
		}

		if best == nil || (id != 0 && (bestID == 0 || id < bestID)) {
			best, bestID = t, id
		}
	}

	return best
}

// Distance is UNIT_GetDistanceToUnit (0x642b10, verified): the absolute
// differences of the integer subtile positions, each reduced by half of both
// unit sizes (floored at 0), combined as max + min/2 (integer division).
func Distance(x1, y1 float64, size1 int, x2, y2 float64, size2 int) int {
	dx := int(math.Abs(math.Floor(x2) - math.Floor(x1)))
	dy := int(math.Abs(math.Floor(y2) - math.Floor(y1)))
	adj := -(size2 / 2) - size1/2
	dx, dy = dx+adj, dy+adj

	if dx < 0 {
		dx = 0
	}

	if dy < 0 {
		dy = 0
	}

	if dx <= dy {
		return (dx + dy*2) / 2
	}

	return (dy + dx*2) / 2
}

func targetSize(t Target) int {
	if sz, ok := t.(Sized); ok {
		return sz.Size()
	}

	return 0
}
