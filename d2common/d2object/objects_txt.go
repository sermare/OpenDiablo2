package d2object

import (
	"fmt"
	"strconv"
	"strings"
)

// Def is the subset of an objects.txt row that spawning, chests and
// waypoints need. The row index is the object id (VERIFIED in Game.exe: the
// table at DAT_009654b0 has a 0x1c0-byte stride and is indexed by the unit's
// class id; see notes units.md). The column meanings of PopulateFn and Parm0
// are UNVERIFIED.
type Def struct {
	ID             int
	Name           string
	SpawnMax       int
	TrapProb       int // percent chance a Lockable/trappable container is trapped (UNVERIFIED use)
	Act            int // bit mask: 1..8 = acts 1..4 classic, 16 = act 5 (expansion); 15 = acts 1-4
	SubClass       int // 1 shrine, 2 obelisk, 4 portal, 8 container, 16 sanctuary gate, 32 well, 64 waypoint, 128 jail door
	OperateFn      int
	PopulateFn     int
	InitFn         int
	OperateRange   int
	Lockable       bool
	RestoreVirgins bool
	Parm0          int
	Damage         int
	Selectable0    bool
}

// SubClass bits of objects.txt.
const (
	SubShrine   = 1
	SubObelisk  = 2
	SubPortal   = 4
	SubWaypoint = 64
	SubWell     = 32
	SubChest    = 8 // "container" bit: chests, caskets, urns, corpses, stashes
)

// ParseObjects reads objects.txt (tab separated, header row first). Rows are
// returned in file order, so the slice index is the object id.
func ParseObjects(data []byte) ([]Def, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("objects.txt: no rows")
	}

	col := map[string]int{}
	for i, h := range strings.Split(lines[0], "\t") {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for _, need := range []string{"name", "operatefn", "populatefn", "act", "subclass", "spawnmax"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("objects.txt: missing column %q", need)
		}
	}

	var out []Def

	for _, ln := range lines[1:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}

		f := strings.Split(ln, "\t")
		num := func(name string) int {
			i, ok := col[name]
			if !ok || i >= len(f) {
				return 0
			}

			n, _ := strconv.Atoi(strings.TrimSpace(f[i]))

			return n
		}

		out = append(out, Def{
			ID: len(out), Name: strings.TrimSpace(f[col["name"]]),
			SpawnMax: num("spawnmax"), TrapProb: num("trapprob"), Act: num("act"), SubClass: num("subclass"),
			OperateFn: num("operatefn"), PopulateFn: num("populatefn"), InitFn: num("initfn"),
			OperateRange: num("operaterange"), Lockable: num("lockable") != 0,
			RestoreVirgins: num("restorevirgins") != 0, Parm0: num("parm0"), Damage: num("damage"),
			Selectable0: num("selectable0") != 0,
		})
	}

	return out, nil
}

// ActBit returns the objects.txt Act mask bit of act 1..5 (act 4 = 8, act 5 =
// 16 as in the extracted tables, where 16 marks expansion-only rows).
func ActBit(act int) int {
	if act < 1 || act > 5 {
		return 0
	}

	return 1 << uint(act-1)
}

// AllowedIn reports whether the object may appear in an act. Act 5 objects
// (bit 16) need the expansion; classic acts need the matching bit.
func (d Def) AllowedIn(act int, expansion bool) bool {
	if act == 5 && !expansion {
		return false
	}

	return d.Act&ActBit(act) != 0
}

// IsWaypoint is true for the waypoint objects (SubClass 64 and OperateFn 23).
func (d Def) IsWaypoint() bool { return d.SubClass&SubWaypoint != 0 && d.OperateFn == 23 }

// IsChest is true for lockable treasure chests operated through OperateFn 4.
func (d Def) IsChest() bool { return d.OperateFn == 4 }
