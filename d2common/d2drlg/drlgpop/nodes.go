package drlgpop

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// Unit kinds of a DS1 object record and of a preset node.
const (
	KindMonster = 1
	KindObject  = 2
	KindItem    = 4
)

// Subtiles per tile.
const Subtile = 5

// Raw is one DS1 object record as stored in the file.
type Raw struct{ Type, ID, X, Y, Flags int }

// Node is a preset unit as the game keeps it after DRLG_ParseDS1Data
// (0x668530): the node record is 0x20 bytes {+0 Class0, +4 ID, +8 X,
// +0x18 Y, +0x14 Kind, +0x1c Flags}.
type Node struct {
	// Class0 is the word at +0: 1 for monsters, 3 for items, 0 otherwise.
	Class0 int
	// ID is the in-memory id: a monstats class, a monstats superunique /
	// place id (>= monstatsCount), an objects.txt id, or the raw id for the
	// other kinds.
	ID int
	// X, Y are subtile coordinates (relative to the DS1 origin when parsed,
	// absolute in the level after FilterPresetObjects, relative to the room
	// after the hand-over to a room).
	X, Y  int
	Kind  int
	Flags int
	// Src is the index of the DS1 object record the node was made from (not
	// part of the game's node; lets callers match nodes to the stamp's
	// object list).
	Src int
}

// ObjectClass maps a DS1 object id (kind 2, id < 0x96) of the given 0-based
// act to the objects.txt id (the exe table at 0x744af8, 5 x 150 entries). A
// negative result drops the object.
type ObjectClass func(act, id int) int

// ResolveDS1 turns the DS1 object records into preset nodes the way
// DRLG_ParseDS1Data does for version >= 5 files. act is the act byte of the
// DS1 (0 when the file has none). The result is in file order. objClass may
// be nil only for files without objects.
//
// Monster ids go through the monpreset row of the act (the id is an index
// into the rows of that act), then the Act 3 and Act 5 special cases that
// turn a few NPC classes into objects.
func ResolveDS1(version, act int, raws []Raw, nm *Names, objClass ObjectClass) []Node {
	if act > 4 {
		act = 4
	}

	if version <= 7 {
		act = 0
	}

	var out []Node

	for si, r := range raws {
		kind, id, class0 := r.Type, r.ID, 0

		switch r.Type {
		case 1:
			class0 = 1

			if version > 4 {
				id, kind, class0 = resolveMonster(act, r.ID, nm)
			}
		case 2:
			switch {
			case version < 6:
				if id == 0x23d {
					id = -1
				}
			case id < 0x96:
				if objClass != nil {
					id = objClass(act, id)
				} else {
					id = -1
				}
			default:
				id -= 0x96
			}
		case 4:
			class0 = 3 // items: the base item lookup is not modelled (no DS1 of the game has any)
		}

		if id < 0 || (r.Type == 1 && version <= 4) {
			continue
		}

		fl := 0
		if version > 5 {
			fl = r.Flags
		}

		out = append(out, Node{Class0: class0, ID: id, X: r.X, Y: r.Y, Kind: kind, Flags: fl, Src: si})
	}

	return out
}

func resolveMonster(act, id int, nm *Names) (newID, kind, class0 int) {
	kind, class0 = KindMonster, 1

	if rows := nm.Presets[act]; id >= 0 && id < len(rows) {
		e := rows[id]
		id = e.ID

		switch e.Cat {
		case CatPlace:
			id += nm.MonstatsCount + nm.SuperCount
		case CatMonster:
		case CatSuper:
			id += nm.MonstatsCount
		default:
			id = -1
		}
	}

	switch act {
	case 2: // Act 3 NPC classes that are objects in the level
		if id >= 0 && id < nm.MonstatsCount {
			switch id {
			case 0x129:
				id, kind, class0 = 0x17e, KindObject, 0
			case 0x16e:
				id, kind, class0 = 0x194, KindObject, 0
			}
		}
	case 4: // Act 5
		if id >= 0 && id < nm.MonstatsCount {
			switch {
			case id == 0x202:
				id, kind, class0 = 0x1cd, KindObject, 0
			case id >= 0x219 && id <= 0x21b:
				id, kind, class0 = 0x3f5-id, KindObject, 0
			}
		}
	}

	return id, kind, class0
}

// Gated ids of DRLG_FilterPresetObjects (0x66a230).
func gated(n Node, nm *Names, seed *d2rand.Seed) (keep bool) {
	if n.Kind == KindMonster {
		if n.ID < nm.MonstatsCount {
			switch n.ID {
			case 0xcc, 0xcd, 0x173, 0x174:
				// kept when the step is a multiple of 3
				v := seed.Step()
				return v%3 == 0
			}

			return true
		}

		rel := n.ID - nm.MonstatsCount - nm.SuperCount
		if n.ID-nm.MonstatsCount >= nm.SuperCount {
			switch rel {
			case 0x21: // place_group25: dropped when (step & 3) == 0
				return seed.Step()&3 != 0
			case 0x22: // place_group50: dropped when the step is even
				return seed.Step()&1 != 0
			case 0x23: // place_group75: kept when (step & 3) == 0
				return seed.Step()&3 == 0
			}
		}

		return true
	}

	if n.Kind != KindObject {
		return true
	}

	switch n.ID {
	case 0xc4, 0x105: // kept when the step is even
		return seed.Step()&1 == 0
	case 0x245: // dropped when (step & 3) == 0
		return seed.Step()&3 != 0
	}

	return true
}

// Filter is DRLG_FilterPresetObjects: it walks the nodes from the LAST file
// object to the first (the parser prepends to its list), draws from seed for
// the gated ids and returns the kept nodes in file order.
func Filter(file []Node, nm *Names, seed *d2rand.Seed) []Node {
	var kept []Node

	for i := len(file) - 1; i >= 0; i-- {
		if gated(file[i], nm, seed) {
			kept = append(kept, file[i])
		}
	}

	// the walk prepends each kept node to the preset list: file order again
	for i, j := 0, len(kept)-1; i < j; i, j = i+1, j-1 {
		kept[i], kept[j] = kept[j], kept[i]
	}

	return kept
}

// Rect is a room rectangle in tiles.
type Rect struct{ X, Y, W, H int }

// TakeForRoom is DRLG_BuildPresetRoomGrids' hand-over (0x669320): the nodes
// of the preset whose subtile position lies inside the room are moved to the
// room, made room relative, and the rest stays in the preset. The room list
// is built by prepending, so it is in reverse file order. nodes must carry
// absolute subtile positions.
func TakeForRoom(nodes []Node, r Rect) (room, rest []Node) {
	x0, y0, x1, y1 := r.X*Subtile, r.Y*Subtile, (r.X+r.W)*Subtile, (r.Y+r.H)*Subtile

	for _, n := range nodes {
		if n.X >= x0 && n.X < x1 && n.Y >= y0 && n.Y < y1 {
			n.X -= r.X * Subtile
			n.Y -= r.Y * Subtile
			room = append([]Node{n}, room...)
		} else {
			rest = append(rest, n)
		}
	}

	return room, rest
}
