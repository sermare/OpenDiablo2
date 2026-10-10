package d2missile

import (
	"math"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

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

		// the callback keeps the lowest id and, on a tie, the later one
		if best == nil || (id != 0 && (bestID == 0 || id <= bestID)) {
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

// Candidate is a unit offered to Scan: a living enemy the caller already
// filtered for type, town and targetable flags.
type Candidate struct {
	Target Target
	X, Y   int // integer subtile position
}

// Scan is the geometric part of the exe's unit scan (0x569510 with the
// filter 0x569100, verified): candidates whose subtile position is within
// radius subtiles (euclidean, squared compare) of the integer cell of
// (x, y) and in line of sight of the owner (no wall bit 4 on the line from
// the owner's cell to the candidate), ordered by unit id (lowest first).
// Dead targets are dropped. owner is the owner's subtile.
func Scan(g d2path.Grid, owner d2path.Point, x, y float64, radius int, cs []Candidate) []Target {
	cx, cy := int(math.Floor(x)), int(math.Floor(y))
	r2 := radius * radius

	var picked []Candidate

	for _, c := range cs {
		if c.Target == nil || !c.Target.Alive() {
			continue
		}

		if (c.X-cx)*(c.X-cx)+(c.Y-cy)*(c.Y-cy) > r2 {
			continue
		}

		if clear, _ := d2path.TraceLine(g, d2path.FlagWall, owner, d2path.Point{X: c.X, Y: c.Y}); !clear {
			continue
		}

		picked = append(picked, c)
	}

	serial := func(t Target) int {
		if sr, ok := t.(Serial); ok {
			return sr.Serial()
		}

		return 0
	}

	sort.SliceStable(picked, func(i, j int) bool { return serial(picked[i].Target) < serial(picked[j].Target) })

	out := make([]Target, len(picked))
	for i, c := range picked {
		out[i] = c.Target
	}

	return out
}

// ChainNext is the target pick of SKILL_ScanRadiusForTargets (callback
// 0x569a40 with the excluded id set to the unit just hit), used by Chain
// Lightning's hit function 12 (0x5a81c0). VERIFIED against the exe in the
// emulator (golden chain_golden.json, 4000 cases): of the scanned candidates it
// returns the one with the smallest unit id GREATER than the id of the unit just
// hit; when there is none it wraps to the smallest id of all (ties: the later
// candidate). The unit just hit is never returned (the exe does not spawn a
// bolt then). Guided Arrow passes no hit unit (id -1), which gives the plain
// lowest id (see lowestSerial). Targets without a Serial count as id 0.
func ChainNext(cs []Target, hit Target) Target {
	hitID := uint32(0xffffffff)
	if hit != nil {
		hitID = uint32(targetSerial(hit))
	}

	var best1, best2 Target

	best1ID, best2ID := uint32(0xffffffff), hitID

	for _, t := range cs {
		if t == nil || !t.Alive() {
			continue
		}

		id := uint32(targetSerial(t))

		if id > hitID {
			if id < best1ID {
				best1, best1ID = t, id
			}

			continue
		}

		if id <= best2ID {
			best2, best2ID = t, id
		}
	}

	pick := best1
	if pick == nil {
		pick = best2
	}

	if pick == nil || (hit != nil && sameUnit(pick, hit)) {
		return nil
	}

	return pick
}

func sameUnit(a, b Target) bool {
	if a == b {
		return true
	}

	sa, oka := a.(Serial)
	sb, okb := b.(Serial)

	return oka && okb && sa.Serial() != 0 && sa.Serial() == sb.Serial()
}

func targetSerial(t Target) int {
	if sr, ok := t.(Serial); ok {
		return sr.Serial()
	}

	return 0
}

// ChainSpawn is the jump rule of hit function 12: the missile's data field 0x28
// holds the bolts still allowed (calc1 at the cast); with less than 2 no further
// bolt is spawned, otherwise the child gets one less.
func ChainSpawn(remaining int) (spawn bool, child int) {
	if remaining < 2 {
		return false, remaining
	}

	return true, remaining - 1
}
