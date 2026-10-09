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

// PickGroup chooses one of the 8 ObjGrp slots of a level: the slot probabilities
// (ObjPrb, percent) are walked cumulatively against a roll in [0,100).
// Returns the group id (0 = none). UNVERIFIED against objrgn.cpp
// (0x544b50..0x5451b8, Ghidra was busy): the cumulative walk is the documented
// modding behaviour.
func PickGroup(groups, probs [8]int, r *Roller) int {
	total := 0
	for _, p := range probs {
		total += p
	}

	if total <= 0 {
		return 0
	}

	roll := r.Roll(100)
	acc := 0

	for i, p := range probs {
		acc += p
		if roll < acc {
			return groups[i]
		}
	}

	return 0
}

// Placement is one object to put into a room.
type Placement struct{ ObjectID int }

// SpawnCount is how many copies of a member fit a room. UNVERIFIED model:
// density is per thousand tiles of room area, at least 1 when density > 0,
// capped by the object's SpawnMax per room when that is above 0.
func SpawnCount(density, roomTiles, spawnMax int) int {
	if density <= 0 || roomTiles <= 0 {
		return 0
	}

	n := density * roomTiles / 1000
	if n < 1 {
		n = 1
	}

	if spawnMax > 0 && n > spawnMax {
		n = spawnMax
	}

	return n
}

// RollRoom fills one room from a group: every member passes its probability
// roll, must be allowed in the act, and yields SpawnCount placements. Shrine
// and well groups (density 0 by definition) yield one placement for a single
// member chosen by weight; the shrine type is then rolled with RollShrine.
// defs is the objects.txt slice from ParseObjects.
func RollRoom(g Group, defs []Def, act int, expansion bool, roomTiles int, r *Roller) []Placement {
	var out []Placement

	if g.Shrines || g.Wells {
		total := 0

		for _, m := range g.Members {
			if m.ID > 0 {
				total += atLeast1(m.Prob)
			}
		}

		if total == 0 {
			return nil
		}

		roll := r.Roll(total)

		for _, m := range g.Members {
			if m.ID <= 0 {
				continue
			}

			roll -= atLeast1(m.Prob)
			if roll < 0 {
				if m.ID < len(defs) && defs[m.ID].AllowedIn(act, expansion) {
					out = append(out, Placement{m.ID})
				}

				break
			}
		}

		return out
	}

	for _, m := range g.Members {
		if m.ID <= 0 || m.ID >= len(defs) || !defs[m.ID].AllowedIn(act, expansion) {
			continue
		}

		if m.Prob < 100 && r.Roll(100) >= m.Prob {
			continue
		}

		for i := SpawnCount(m.Density, roomTiles, defs[m.ID].SpawnMax); i > 0; i-- {
			out = append(out, Placement{m.ID})
		}
	}

	return out
}

func atLeast1(n int) int {
	if n < 1 {
		return 1
	}

	return n
}
