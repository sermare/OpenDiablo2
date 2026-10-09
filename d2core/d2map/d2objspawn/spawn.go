// Package d2objspawn is the thin adapter between the engine's records and the
// pure rules of d2common/d2object: it turns the loaded objects.txt and
// objgroup.txt records into d2object tables, rolls the random objects of a
// room, opens chests through the engine's loot hooks and finds waypoint bits.
// The engine side is reached only through the small Room interface, so tests
// use fakes.
package d2objspawn

import (
	"errors"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// ErrUnknownObject is returned for an object id outside objects.txt.
var ErrUnknownObject = errors.New("d2objspawn: unknown object id")

// Tables are the d2object views of the loaded records.
type Tables struct {
	Defs   []d2object.Def // index = object id
	Groups map[int]d2object.Group
}

// Room is what the engine offers for one room being populated.
type Room interface {
	// Tiles is the room rectangle area w*h in subtiles (the exe's density
	// rule uses ((w*h>>7)*density)>>8).
	Tiles() int
	// Place puts an object of the given objects.txt row into the room.
	Place(objectID int)
}

// Level carries the ObjGrp/ObjPrb slots of a levels.txt row.
type Level struct {
	Act    int
	Groups [8]int
	Probs  [8]int
}

// FromRecords converts the loaded records. Missing object ids stay zero Defs
// (Act 0 = never allowed), so ids index the slice safely.
func FromRecords(objs d2records.ObjectDetails, groups d2records.ObjectGroups) Tables {
	t := Tables{Groups: make(map[int]d2object.Group, len(groups))}

	top := -1

	for id := range objs {
		if id > top {
			top = id
		}
	}

	t.Defs = make([]d2object.Def, top+1)

	for id, o := range objs {
		t.Defs[id] = d2object.Def{
			ID: id, Name: o.Name, SpawnMax: o.SpawnMax, TrapProb: o.TrapProbability, Act: o.Act,
			SubClass: o.SubClass, OperateFn: o.OperateFn, PopulateFn: o.PopulateFn, InitFn: o.InitFn,
			OperateRange: o.OperateRange, Lockable: o.Lockable, RestoreVirgins: o.RestoreVirgins,
			Parm0: o.Parm[0], Damage: o.Damage, Selectable0: o.Selectable[0],
		}
	}

	for off, g := range groups {
		out := d2object.Group{Name: g.GroupName, Offset: g.Offset, Shrines: g.Shrines, Wells: g.Wells}

		if g.Members != nil {
			for i, m := range g.Members {
				if i < d2object.GroupMembers {
					out.Members[i] = d2object.GroupMember{ID: m.ID, Density: m.Density, Prob: m.Probability}
				}
			}
		}

		t.Groups[off] = out
	}

	return t
}

// LevelOf extracts the object group slots of a levels.txt record.
func LevelOf(l *d2records.LevelDetailRecord, act int) Level {
	return Level{
		Act: act,
		Groups: [8]int{l.ObjectGroupID0, l.ObjectGroupID1, l.ObjectGroupID2, l.ObjectGroupID3,
			l.ObjectGroupID4, l.ObjectGroupID5, l.ObjectGroupID6, l.ObjectGroupID7},
		Probs: [8]int{l.ObjectGroupSpawnChance0, l.ObjectGroupSpawnChance1, l.ObjectGroupSpawnChance2,
			l.ObjectGroupSpawnChance3, l.ObjectGroupSpawnChance4, l.ObjectGroupSpawnChance5,
			l.ObjectGroupSpawnChance6, l.ObjectGroupSpawnChance7},
	}
}

// PopulateRoom rolls every ObjGrp slot of the level independently
// (d2object.PickGroups) and, for each fired group, places the one member
// RollRoom picks. It returns the number placed. Deterministic for a given seed.
func PopulateRoom(t Tables, lv Level, expansion bool, room Room, seed uint32) int {
	r := d2object.NewRoller(seed)
	total := 0

	for _, id := range d2object.PickGroups(lv.Groups, lv.Probs, r) {
		g, ok := t.Groups[id]
		if !ok {
			continue
		}

		for _, p := range d2object.RollRoom(g, t.Defs, lv.Act, expansion, room.Tiles(), r) {
			room.Place(p.ObjectID)

			total++
		}
	}

	return total
}

// Def returns the object row, or false for an unknown id.
func (t Tables) Def(id int) (d2object.Def, bool) {
	if id < 0 || id >= len(t.Defs) {
		return d2object.Def{}, false
	}

	return t.Defs[id], true
}

// WaypointBit is the waypoint bit object id activates in level, if it is one.
func (t Tables) WaypointBit(id, level int, expansion bool) (int, bool) {
	d, ok := t.Def(id)
	if !ok {
		return 0, false
	}

	return d2object.WaypointBitOf(d, level, expansion)
}

// OpenChest operates a container through d2object.Open with the engine hooks.
func (t Tables) OpenChest(id int, st *d2object.ChestState, h d2object.ChestHooks,
	act, diff, areaLevel int, seed uint32) (d2object.OpenResult, error) {
	d, ok := t.Def(id)
	if !ok {
		return d2object.OpenResult{}, ErrUnknownObject
	}

	return d2object.Open(d, st, h, act, diff, areaLevel, seed)
}

// FromText builds Tables straight from objects.txt and objgroup.txt bytes
// (d2object.ParseObjects / ParseGroups), for tools and tests that bypass the
// record manager.
func FromText(objects, objgroup []byte) (Tables, error) {
	defs, err := d2object.ParseObjects(objects)
	if err != nil {
		return Tables{}, err
	}

	groups, err := d2object.ParseGroups(objgroup)
	if err != nil {
		return Tables{}, err
	}

	t := Tables{Defs: defs, Groups: make(map[int]d2object.Group, len(groups))}
	for _, g := range groups {
		t.Groups[g.Offset] = g
	}

	return t, nil
}
