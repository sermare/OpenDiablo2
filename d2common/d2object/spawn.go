package d2object

import (
	"fmt"
	"strconv"
	"strings"
)

// GroupMembers is the number of object slots of an objgroup.txt row (the
// extracted 1.14b table has 8 triples; the older OD2 loader assumed 7).
const GroupMembers = 8

// GroupMember is one (object, density, probability) triple.
type GroupMember struct{ ID, Density, Prob int }

// Group is a row of objgroup.txt: the objects a level can scatter over its
// rooms. Levels.txt picks a group with ObjGrp0..7 / ObjPrb0..7.
type Group struct {
	Name    string
	Offset  int
	Members [GroupMembers]GroupMember
	Shrines bool
	Wells   bool
}

// ParseGroups reads objgroup.txt. Columns: name, offset, 8 x (id, density,
// prob), shrines, wells. Header rows (non numeric offset) are skipped.
func ParseGroups(data []byte) ([]Group, error) {
	var out []Group

	for i, ln := range strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}

		f := strings.Split(ln, "\t")
		if len(f) < 2 {
			return nil, fmt.Errorf("objgroup row %d: %d columns", i, len(f))
		}

		if _, err := strconv.Atoi(strings.TrimSpace(f[1])); err != nil {
			continue // header row
		}

		if len(f) < 2+3*GroupMembers+2 {
			return nil, fmt.Errorf("objgroup row %d: %d columns", i, len(f))
		}

		n := func(j int) int { v, _ := strconv.Atoi(strings.TrimSpace(f[j])); return v }

		g := Group{Name: f[0], Offset: n(1)}
		for m := 0; m < GroupMembers; m++ {
			g.Members[m] = GroupMember{n(2 + 3*m), n(3 + 3*m), n(4 + 3*m)}
		}

		g.Shrines = n(2+3*GroupMembers) != 0
		g.Wells = n(3+3*GroupMembers) != 0
		out = append(out, g)
	}

	return out, nil
}

// PickGroups chooses the object groups of one room. VERIFIED against the
// per-room object scatter at 550680 (Game.exe): each of the 8 ObjGrp slots of
// the level (bytes at level record +0xe5..) is tried independently, with
// ObjPrb (bytes at +0xed..) as the threshold: r = rand(100) and the slot
// fires when group != 0 and r <= ObjPrb (so ObjPrb 0 still fires 1% of the
// time, and 99 or more always). It returns the fired group ids in slot order.
// (The 550680 path additionally forces r = 100 in rooms that are over 75% (96
// of 128) covered; that needs the room coverage and is not modelled.)
func PickGroups(groups, probs [8]int, r *Roller) []int {
	var out []int

	for i := range groups {
		roll := r.Roll(100)
		if groups[i] != 0 && roll <= probs[i] {
			out = append(out, groups[i])
		}
	}

	return out
}

// Placement is one object to put into a room.
type Placement struct{ ObjectID int }

// MaxDensity is the largest objgroup density the exe accepts (the populate
// handlers abort for a density byte above 0x80).
const MaxDensity = 0x80

// SpawnCount is how many objects a populate handler tries to place in a room
// of roomTiles = width*height subtiles. VERIFIED (54f3e0, 54f500, 54eae0 and
// others): n = ((w*h >> 7) * density) >> 8, no minimum of one and no use of
// objects.txt SpawnMax. density is the objgroup byte (0..128).
func SpawnCount(density, roomTiles int) int {
	if density <= 0 || roomTiles <= 0 {
		return 0
	}

	if density > MaxDensity {
		density = MaxDensity
	}

	return ((roomTiles >> 7) * density) >> 8
}

// singlePopulateFns are the objects.txt PopulateFn values whose handler
// places a single object instead of a density-derived count: 2 (550bc0, a
// few placement tries), 7 (54f140, a fixed cluster) and 8 (54f650, one
// object). The other handlers (1, 3, 4, 5, 9) compute SpawnCount. Handler
// table: VERIFIED at 72f6b8, entry 0 empty, 1..9 as listed; the clusters and
// retry loops inside 1, 4 and 5 are not modelled.
var singlePopulateFns = map[int]bool{2: true, 7: true, 8: true}

// RollRoom fills one room from a group. VERIFIED against 550680/550960: a
// single roll = rand(100) walks the members in order (stopping at the first
// empty id), adding each member's probability byte; the first member whose
// running total exceeds the roll (and that is allowed, see below) is the one
// member spawned for the group, through the handler its PopulateFn selects. A
// member that fails the allowed test is skipped and the walk goes on; a roll
// past the total spawns nothing. The exe's allowed test is an objects.txt
// byte at +0x172 compared with a global; here it is approximated by the Act
// mask (UNVERIFIED). Shrine and well groups take the same path. defs is the
// objects.txt slice from ParseObjects.
func RollRoom(g Group, defs []Def, act int, expansion bool, roomTiles int, r *Roller) []Placement {
	roll := r.Roll(100)
	cum := 0

	for _, m := range g.Members {
		if m.ID <= 0 {
			break
		}

		cum += m.Prob

		if roll >= cum || m.ID >= len(defs) || !defs[m.ID].AllowedIn(act, expansion) {
			continue
		}

		n := SpawnCount(m.Density, roomTiles)
		if singlePopulateFns[defs[m.ID].PopulateFn] {
			n = 1
		}

		out := make([]Placement, 0, n)
		for ; n > 0; n-- {
			out = append(out, Placement{m.ID})
		}

		return out
	}

	return nil
}

func atLeast1(n int) int {
	if n < 1 {
		return 1
	}

	return n
}
