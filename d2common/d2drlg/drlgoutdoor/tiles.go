package drlgoutdoor

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Tile records of an outdoor room: DRLG_BuildOutdoorRoomTiles (0x680720) for
// plain rooms and DRLG_BuildPresetRoomTiles (0x6696c0) for preset rooms, with
// their callees, ported from Game.exe 1.14b and checked against the real code
// running in the unicorn oracle (~/git/drlg-oracle/gen_tiles.py).
//
// The layers of a room (A orientation, B wall, C floor for plain rooms; the
// DS1 layers of a preset) are turned into three record lists, in the order the
// original appends them:
//
//	walls   (room+0x54 info +0x14)
//	floors  (+0x1c)
//	shadows (+0x24): the random tile markers of the sub-theme patterns (made
//	        while the grids are built) followed by cells with bit 0x8000000
//
// Every record is created by DRLG_PickRandomTile (0x6704f0): the tiles of the
// room's DT1 library matching (orientation, style, sequence), weighted by
// rarity, one room-seed Roll per pick. Border cells (flag 4 of the cell) first
// look for a record another, already built room made at the same world cell
// (0x671250 / 0x671190) and merge with it (0x671420). The result therefore
// depends on the order the rooms are built, which in the game is the order the
// player's movement streams them in. BuildTiles uses one fixed order: plain
// rooms in creation order, then preset rooms in creation order. The golden was
// made the same way, each level in a fresh game so that rooms of neighbouring
// levels are never built.

// TileRecord is one 0x30 byte tile record of a room.
type TileRecord struct {
	X, Y  int // cell relative to the room
	Ori   int // orientation layer value (record +0x1c)
	Flags uint32
	Tile  TileRef
	next  *TileRecord
}

type tileGroup struct {
	floor bool
	head  *TileRecord
}

// RoomTiles are the finished tile records of one room.
type RoomTiles struct {
	Room                   *Room
	Lib                    *Library
	Walls, Floors, Shadows []*TileRecord
	// Logic is the room's logic region list (logic.go), newest node first.
	Logic []LogicRegion
	// Seed is the room seed after the whole build (later builds of
	// neighbouring rooms can still advance it).
	Seed d2rand.Seed

	groups   []*tileGroup // newest first
	nbrs     []*Room
	warpHead map[int]*TileRecord // exit entry (by cell style) -> newest linked warp record
}

// unported marks a branch of the original that no tested room reaches.
type unported string

func (u unported) Error() string { return "drlgoutdoor: unported tile code path: " + string(u) }

// tileBuilder is the state of one room build.
type tileBuilder struct {
	l     *Level
	r     *Room
	rt    *RoomTiles
	rs    *d2rand.Seed
	built map[*Room]*RoomTiles
}

// pick is DRLG_PickRandomTile (0x6704f0) on the library and seed of a room.
func pick(rt *RoomTiles, rs *d2rand.Seed, ori int, dword uint32) TileRef {
	var style, seq int32

	if dword != 0 {
		style, seq = int32(dword>>20)&0x3f, int32(dword>>8)&0xff
	}

	refs := rt.Lib.query(int32(ori), style, seq, 40)
	if len(refs) == 0 {
		refs = rt.Lib.query(10, 0, 0, 40)
		if len(refs) == 0 {
			panic(GameError{0x73})
		}

		return refs[0]
	}

	var sum uint32

	for _, r := range refs {
		sum += uint32(r.Rarity())
	}

	var roll uint32
	if int32(sum) >= 1 {
		roll = rs.Roll(int32(sum))
	}

	i, rem := 0, int32(roll)+1

	if sum != 0 {
		for ; len(refs) > 1 && rem > 0; rem -= int32(refs[i-1].Rarity()) {
			i++
		}

		if i != 0 {
			i--
		}
	}

	return refs[i]
}

func (t *tileBuilder) pick(ori int, dword uint32) TileRef { return pick(t.rt, t.rs, ori, dword) }

// seedOf is the live seed of a room: the builder's own while it is built.
func (t *tileBuilder) seedOf(rt *RoomTiles) *d2rand.Seed {
	if rt == t.rt {
		return t.rs
	}

	return &rt.Seed
}

// flagsCommon are the record flag bits all record kinds derive the same way
// from the cell value v (0x670aa0, 0x6707d0, 0x670c00).
func flagsCommon(f uint32, v uint32, mat uint16) uint32 {
	if int8(v) < 0 {
		f |= 1
	}

	if v&0x10000000 != 0 {
		f |= 0x102
	}

	if v&0x20000 != 0 {
		f |= 0x40
	}

	if v&0x10000 != 0 {
		f |= 0x80
	}

	if v&8 != 0 {
		f |= 4
	}

	if int32(v) < 0 {
		f |= 8
	} else {
		f &^= 8
	}

	if v&0x4000000 != 0 {
		f |= 0x20c
	}

	if v&0x20000000 != 0 {
		f |= 0x800
	}

	if v&4 != 0 {
		f |= 0x2000
	}

	if mat&1 != 0 {
		f |= 4
	}

	if mat&4 != 0 {
		f |= 0x800
	}

	return f
}

func newRec(rt *RoomTiles, x, y, ori int, tile TileRef) *TileRecord {
	return &TileRecord{X: x - rt.Room.X, Y: y - rt.Room.Y, Ori: ori, Tile: tile}
}

func chain(g *tileGroup, rec *TileRecord) {
	if g != nil {
		rec.next, g.head = g.head, rec
	}
}

// floorRec is the floor record of 0x670aa0 (made by 0x670bb0 / 0x671822).
func floorRec(rt *RoomTiles, g *tileGroup, x, y int, v uint32, tile TileRef) *TileRecord {
	rec := newRec(rt, x, y, 0, tile)
	rec.Flags = flagsCommon(((v>>0x12)&3)*0x4000+0x4000, v, tile.tile().material)
	chain(g, rec)
	rt.Floors = append(rt.Floors, rec)

	return rec
}

// wallFlags is 0x6707d0 on an existing flag word.
func wallFlags(f uint32, ori int, v uint32, mat uint16) uint32 {
	if ori != 0xd {
		f |= ((v>>0x12)&3)*0x4000 + 0x4000
	}

	switch ori {
	case 0xe:
		f |= 4
	case 8, 9, 10, 11:
		f |= 2
	}

	return flagsCommon(f, v, mat)
}

// wallRec is 0x670900 (+ 0x6707d0). Orientation 3 makes a second record of
// orientation 4 right behind it.
func (t *tileBuilder) wallRec(rt *RoomTiles, g *tileGroup, x, y int, v uint32, tile TileRef, ori int) *TileRecord {
	rec := newRec(rt, x, y, ori, tile)
	if ori == 8 || ori == 9 {
		t.tileObject(rec, rt.Room, v, ori == 9, x, y)
	}

	rec.Flags = wallFlags(rec.Flags, ori, v, tile.tile().material)
	chain(g, rec)
	rt.Walls = append(rt.Walls, rec)

	if ori == 3 {
		t.wallRec(rt, g, x, y, v, pick(rt, t.seedOf(rt), 4, v), 4)
	}

	return rec
}

// shadowRec is DRLG_CreateRandomTileRecord (0x670c00).
func shadowRec(rt *RoomTiles, g *tileGroup, x, y int, v uint32, tile TileRef) {
	rec := newRec(rt, x, y, 0xd, tile)
	rec.Flags = flagsCommon(0, v, tile.tile().material)
	chain(g, rec)
	rt.Shadows = append(rt.Shadows, rec)
}

// addRandom is DRLG_AddRandomTileRecord (0x670d20): a random tile marker of a
// stamped sub-theme pattern.
func (t *tileBuilder) addRandom(x, y int, v uint32) {
	shadowRec(t.rt, nil, x, y, v, t.pick(0xd, v))
}

func (rt *RoomTiles) group(floor bool) *tileGroup {
	for _, g := range rt.groups {
		if g.floor == floor {
			return g
		}
	}

	g := &tileGroup{floor: floor}
	rt.groups = append([]*tileGroup{g}, rt.groups...)

	return g
}

func inRect(r *Room, x, y int) bool {
	return x >= r.X && y >= r.Y && x <= r.X+r.W && y <= r.Y+r.H
}

// lookup is 0x671250 / 0x671190: the first record in a built neighbour room
// that covers the world cell.
func (t *tileBuilder) lookup(floor bool, x, y int, v uint32) (*TileRecord, *Room) {
	for _, n := range t.rt.nbrs {
		if n == t.r {
			continue
		}

		nt := t.built[n]
		if nt == nil || !inRect(n, x, y) {
			continue
		}

		for _, g := range nt.groups {
			if g.floor != floor {
				continue
			}

			for rec := g.head; rec != nil; rec = rec.next {
				if rec.X+n.X != x || n.Y+rec.Y != y || rec.Ori == 4 {
					continue
				}

				if rec.Ori != 0xd && v&0x8000000 != 0 {
					continue
				}

				if rec.Flags&0x1c000 == 0 || int((rec.Flags>>14)&7)-1 == int((v>>0x12)&3) {
					return rec, n
				}
			}
		}
	}

	return nil, nil
}

// orientation merge tables of 0x671420 (0x6f0bd0 and 0x6f0b24, rows of 7).
var (
	oriRow = [16]int{-1, 0, 1, 2, -1, 3, 4, 5, -2, -2, -1, -1, -1, -2, -1, -1}
	oriMix = [56]int{
		0, 1, 3, 3, 4, 1, 3,
		1, 1, 2, 3, 4, 3, 2,
		2, 3, 3, 3, 4, 3, 3,
		3, 1, 3, 3, 4, 5, 6,
		1, 3, 2, 3, 4, 3, 6,
		2, 1, 2, 3, 4, 1, 2,
		7, -1, 0, 1, 2, -1, 3,
		4, 5, -2, -2, -1, -1, -1,
	}
)

// merge is 0x671420: the cell of the current room at (x, y) with orientation
// ori finds the record rec of the built room nbr. The record may change its
// orientation and tile (the tile is picked with the neighbour's seed).
func (t *tileBuilder) merge(rec *TileRecord, nbr *Room, ori int, v uint32, x, y int) {
	nrt := t.built[nbr]
	row := oriRow[ori&15]

	var nori int

	switch {
	case rec.Flags&1 != 0:
		if rec.Ori != 9 && rec.Ori != 8 {
			return
		}

		t.touch(rec, rec.Ori, v, x, y)

		return
	case v&0x80 != 0:
		nori = ori
	default:
		keep := false

		if ori == 9 || ori == 8 {
			keep = x-t.r.X == 0 || y-t.r.Y == 0
		} else if rec.Ori == 9 || rec.Ori == 8 {
			if x-nbr.X == 0 || y-nbr.Y == 0 {
				return
			}
		}

		switch {
		case keep:
			nori = ori
		case row >= 0 && rec.Ori <= 7:
			nori = oriMix[row*7+rec.Ori]
		case row == -1:
			nori = ori
		default:
			return
		}
	}

	if rec.Ori == 3 {
		if nori != 3 {
			if rec.next == nil {
				panic(unported("orientation 3 record without a partner"))
			}

			rec.next.Flags |= 8
		}
	} else if nori == 3 {
		rec.Flags |= 0xc008
		t.wallRec(t.rt, t.rt.group(false), x, y, v, t.pick(3, v), 3)
	}

	special := rec.Ori == 0 && rec.Tile.Style() == 0x1e && rec.Tile.Sequence() == 0
	if nori != rec.Ori || special {
		rec.Tile = pick(nrt, t.seedOf(nrt), nori, v)
		rec.Ori = nori
	}

	t.touch(rec, rec.Ori, v, x, y)
}

// touch is the tail of 0x671420 (0x6707d0 on the merged record).
func (t *tileBuilder) touch(rec *TileRecord, ori int, v uint32, x, y int) {
	if ori == 8 || ori == 9 {
		t.tileObject(rec, t.r, v, ori == 9, x, y)
	}

	rec.Flags = wallFlags(rec.Flags, ori, v, rec.Tile.tile().material)
}

// tileObjRow is a row of the exe's tile object table (0x6f0738, 28 bytes): a
// wall tile (style, sequence, orientation 9 or not) of a level spawns an object
// (typ 2) or monster (typ 1) with id at a subtile offset. The rows a level
// owns are the inclusive range of the level table at 0x6f0578 (tileObjLevels).
// A typ 2 row with id 0x5b/0x5c rolls the room seed before the spawn (rows 19
// and 20, the Act 2 mazes; not ported). The spawn itself touches no RNG.
type tileObjRow struct {
	style, seq int
	ori9       bool
	typ, id    int
	dx, dy     int
}

var tileObjRows = [...]tileObjRow{
	{7, 0, true, 2, 14, 5, 0}, {7, 0, false, 2, 13, 0, 5}, {5, 0, true, 2, 16, 0, 0}, {5, 0, false, 2, 15, 0, 0},
	{6, 0, true, 2, 27, 5, -2}, {4, 0, true, 2, 24, 1, 2}, {4, 0, false, 2, 23, 0, 0}, {4, 3, true, 2, 25, 1, 0},
	{1, 2, false, 2, 62, 0, 3}, {1, 2, true, 2, 63, 3, 0}, {0, 0, true, 2, 16, 0, 0}, {0, 0, false, 2, 64, 0, 0},
	{2, 0, true, 2, 47, 5, 0}, {0, 1, true, 2, 291, 2, 0}, {0, 1, false, 2, 290, 0, 2}, {5, 0, true, 2, 293, 2, 0},
	{4, 0, false, 2, 292, 0, 2}, {0, 0, true, 2, 295, 2, 0}, {0, 0, false, 2, 294, 0, 2}, {2, 4, true, 2, 92, 1, 0},
	{2, 1, false, 2, 91, 0, 2}, {0, 1, true, 2, 229, 0, 0}, {0, 1, false, 2, 230, 0, 0}, {3, 3, false, 2, 449, -2, 4},
	{2, 1, false, 1, 435, 1, 2}, {2, 1, true, 1, 435, 2, 1}, {2, 6, false, 1, 435, 1, 1}, {2, 2, false, 1, 433, 0, 1},
	{2, 3, true, 1, 432, 1, 0}, {26, 0, false, 1, 434, 0, 1}, {2, 4, true, 1, 524, 0, 0}, {2, 4, false, 1, 525, 0, 0},
	{29, 0, true, 2, 60, 2, 0}, {29, 0, false, 2, 60, 0, 2},
}

// tileObjLevels is the level table at 0x6f0578 (level, first row, last row),
// searched in order; read from the exe.
var tileObjLevels = [...][3]int{
	{28, 0, 3}, {29, 0, 3}, {30, 0, 3}, {31, 0, 3}, {26, 4, 6}, {27, 4, 6}, {32, 5, 9}, {33, 5, 9},
	{34, 10, 11}, {35, 10, 11}, {36, 10, 11}, {37, 10, 12}, {51, 13, 14}, {52, 15, 18}, {53, 15, 18},
	{54, 15, 18}, {55, 19, 20}, {56, 19, 20}, {57, 19, 20}, {58, 19, 20}, {59, 19, 20}, {60, 19, 20},
	{61, 19, 20}, {66, 19, 20}, {67, 19, 20}, {68, 19, 20}, {69, 19, 20}, {70, 19, 20}, {71, 19, 20},
	{72, 19, 20}, {62, 21, 22}, {63, 21, 22}, {64, 21, 22}, {109, 23, 24}, {111, 24, 33}, {112, 24, 33},
	{117, 24, 33},
}

// tileObject is DRLG_CreateTileObject (0x6706a0) as far as it shows in the
// tile records: it spawns an object into the room's object list (not modelled)
// and marks the record with flag 0x20 so that it does not spawn twice.
func (t *tileBuilder) tileObject(rec *TileRecord, room *Room, v uint32, ori9 bool, x, y int) {
	if rec != nil && rec.Flags&0x20 != 0 {
		return
	}

	style, seq := int(v>>20)&0x3f, int(v>>8)&0xff

	for _, lv := range tileObjLevels {
		if lv[0] != t.l.Params.ID {
			continue
		}

		t.tileObjectRows(rec, room, lv[1], lv[2], style, seq, ori9, x, y)

		return
	}
}

func (t *tileBuilder) tileObjectRows(rec *TileRecord, room *Room, lo, hi, style, seq int, ori9 bool, x, y int) {
	for k := lo; k <= hi; k++ {
		r := tileObjRows[k]
		if r.style != style || r.seq != seq || r.ori9 != ori9 {
			continue
		}

		sx, sy := (x-room.X)*5+r.dx, (y-room.Y)*5+r.dy
		if sx < 0 || sy < 0 || sx >= room.W*5 || sy >= room.H*5 {
			return
		}

		if r.typ == 2 && (r.id == 0x5b || r.id == 0x5c) {
			panic(unported("tile object row that rolls the room seed"))
		}

		if rec != nil {
			rec.Flags |= 0x20
		}

		return
	}
}

// border is 0x671620: reuse the neighbour's record (0x671420) or make a new
// one in the border group (0x671300).
func (t *tileBuilder) border(ori int, v uint32, x, y int) {
	floor := ori == 0

	if rec, nbr := t.lookup(floor, x, y, v); rec != nil {
		t.merge(rec, nbr, ori, v, x, y)
		return
	}

	if (ori == 0xb || ori == 0xa) && !inRectEx(t.r, x, y) {
		return
	}

	g := t.rt.group(floor)
	tile := t.pick(ori, v)

	switch ori {
	case 0:
		floorRec(t.rt, g, x, y, v, tile)
	case 0xd:
		shadowRec(t.rt, g, x, y, v, tile)
	default:
		rec := t.wallRec(t.rt, g, x, y, v, tile, ori)

		if ori == 10 || ori == 11 {
			t.warpWall(rec, v, ori)
		}
	}
}

// warpRec finds the LvlWarp description of the exit slot of a hidden warp cell
// (the link entry of room+0x4c that 0x66eed0 makes for every exit bit of the
// room whose level has a warp).
func (t *tileBuilder) warpRec(style int, ori int) (d2drlg.WarpRec, bool) {
	if style < 0 || style > 7 || t.l.Params.Warp[style] == -1 || t.r.Flags&(0x10<<uint(style)) == 0 {
		return d2drlg.WarpRec{}, false
	}

	src, ok := t.l.env.Tables.(d2drlg.LvlWarps)
	if !ok {
		panic(unported("no LvlWarp table"))
	}

	w, ok := src.WarpRec(t.l.Params.Warp[style])
	if !ok {
		return w, false
	}

	if w.Dir != 'b' && ((ori != 0xb && w.Dir != 'l') || (ori == 0xb && w.Dir != 'r')) {
		// 0x670e20 swaps the link's LvlWarp row for the first row with the same
		// Id whose Direction is 'b' or the wanted side (0x61f4a0). In the
		// expansion LvlWarp.txt those sibling rows (Ids 71, 73, 74, 81, 82: an
		// 'l' and an 'r' row) differ only in the Direction column, so the
		// LitVersion and Tiles read here are the same.
		w.Dir = 'b'
	}

	return w, true
}

// warpCell is a hidden cell of orientation 10/11 (0x670e80 + 0x671010): the
// cave entrance floor, four records picked by the cell's sequence.
func (t *tileBuilder) warpCell(v uint32, ori, x, y int) {
	t.r.Flags |= 0x800000

	w, ok := t.warpRec(int(v>>20)&0x3f, ori)
	if !ok || w.LitVersion == 0 {
		return
	}

	seq := (v >> 8) & 0xff
	dx := [4]int{0, 1, 0, 1}
	dy := [4]int{0, 0, 1, 1}

	for k := 0; k < 4; k++ {
		dw := (seq<<12 | uint32(k) | 4) << 8
		rec := floorRec(t.rt, nil, x-1+dx[k], y-1+dy[k], dw, t.pick(0, dw))
		rec.Flags |= 8
	}
}

// warpWall is 0x670f20: a wall record of orientation 10/11 (a cave entrance or
// exit piece) is linked to the room's exit entry; the first and fifth piece
// (sequence 0 / 4) also make the warp object, and a lit warp gets a second wall
// record picked with the warp's tile count.
func (t *tileBuilder) warpWall(rec *TileRecord, v uint32, ori int) {
	style := int(v>>20) & 0x3f
	if style > 7 {
		return
	}

	w, ok := t.warpRec(style, ori)
	if !ok {
		panic(GameError{0x2b1})
	}

	if seq := (v >> 8) & 0xff; seq == 0 || seq == 4 {
		if rec.X == t.r.W || rec.Y == t.r.H {
			return
		}
	}

	// 0x670fae: the record leaves its group chain: its next pointer is reused
	// for the exit entry's own list (rec.next = link.head; link.head = rec).
	// Every older record of the same group becomes unreachable for the
	// neighbour lookup (0x671190), so a room next to a cave-entrance piece does
	// not merge with the cells built before it (levels 94 and 97).
	if t.rt.warpHead == nil {
		t.rt.warpHead = map[int]*TileRecord{}
	}

	rec.next = t.rt.warpHead[style]
	t.rt.warpHead[style] = rec

	if w.LitVersion != 0 {
		dw := uint32(w.Tiles)<<8 | v
		n := t.wallRec(t.rt, nil, t.r.X+rec.X, t.r.Y+rec.Y, dw, t.pick(ori, dw), ori)
		n.Flags |= 8
	}
}

// cell is 0x671680: the records of one cell of a layer pass. fill is the
// FillBlanks flag of the first floor layer of a preset.
func (t *tileBuilder) cell(ori int, v uint32, x, y int, fill bool) {
	style := int(v>>20) & 0x3f
	seq := int(v>>8) & 0xff

	if (ori == 0xb || ori == 10) && style >= 8 {
		return
	}

	if ori == 0 && style == 0x1e && (seq == 0 || seq == 1) {
		v |= 0x80000000
	}

	hidden := v&0x80000000 != 0

	if hidden {
		switch ori {
		case 8, 9:
			if id := t.l.Params.ID; id < 0x6f || (id > 0x70 && id != 0x75) {
				// object only (no record flag; the room's object list is not modelled)
				return
			}
		case 10, 11:
			t.warpCell(v, ori, x, y)
			return
		}
	}

	if v&4 != 0 {
		switch {
		case v&2 != 0:
			vv := v
			if style == 0x1e && (seq == 0 || seq == 1) {
				vv &^= 0x80
			}

			t.border(0, vv, x, y)

			return
		case v&1 != 0:
			t.border(ori, v, x, y)
			return
		case v&0x8000000 != 0 && !hidden:
			t.border(0xd, v, x, y)
			return
		}
	}

	if v&2 != 0 {
		floorRec(t.rt, nil, x, y, v, t.pick(0, v))
	} else if fill && inRectEx(t.r, x, y) {
		d := uint32(0x1e00000)
		if t.l.Params.ID == 0x4a {
			d = 0x1e00100
		}

		floorRec(t.rt, nil, x, y, v&0xffffff7f|0x80000000, t.pick(0, d))
	}

	if v&1 != 0 {
		rec := t.wallRec(t.rt, nil, x, y, v, t.pick(ori, v), ori)

		if (ori == 10 || ori == 11) && t.l.Params.ID != 0x85 {
			t.warpWall(rec, v, ori)
		}
	}

	if v&0x8000000 != 0 {
		shadowRec(t.rt, nil, x, y, v, t.pick(0xd, v))
	}
}

// buildCells runs the two passes of 0x680720.
func (t *tileBuilder) buildCells(g *RoomGrids) {
	r := t.r

	for y := 0; y <= r.H; y++ {
		for x := 0; x <= r.W; x++ {
			t.cell(int(g.A.Get(x, y)), g.B.Get(x, y), r.X+x, r.Y+y, false)
		}
	}

	for y := 0; y <= r.H; y++ {
		for x := 0; x <= r.W; x++ {
			t.cell(0, g.C.Get(x, y), r.X+x, r.Y+y, false)
		}
	}

	t.rt.Logic = t.logicRegionsWhole()
}

// neighbourRooms is 0x66e8f0 for the rooms of the same level: every room whose
// rectangle is closer than 6 tiles on both axes (the room itself included), in
// the original's list order (newest first), then the 0x66e8a0 bubble sort.
func (l *Level) neighbourRooms(r *Room) []*Room {
	var out []*Room

	for i := len(l.Rooms) - 1; i >= 0; i-- {
		n := l.Rooms[i]

		dx := r.X - n.W - n.X
		if r.X < n.X {
			dx = n.X - r.W - r.X
		}

		dy := r.Y - n.H - n.Y
		if r.Y < n.Y {
			dy = n.Y - r.H - r.Y
		}

		if dx < 6 && dy < 6 {
			out = append(out, n)
		}
	}

	for k := len(out) - 1; k > 0; k-- {
		for j := 0; j < len(out)-1; j++ {
			a, b := out[j], out[j+1]
			if b.W+b.X <= a.X || b.H+b.Y <= a.Y {
				out[j], out[j+1] = b, a
			}
		}
	}

	return out
}

// BuildTiles builds the tile records of every room of the level: the plain
// rooms in creation order, then the preset rooms in creation order. The result
// is parallel to l.Rooms.
func (l *Level) BuildTiles() ([]*RoomTiles, error) { return l.buildTiles(false) }

// BuildPlainTiles is BuildTiles without the preset rooms (their entries are nil).
func (l *Level) BuildPlainTiles() ([]*RoomTiles, error) { return l.buildTiles(true) }

func (l *Level) buildTiles(plainOnly bool) (res []*RoomTiles, err error) {
	defer func() {
		if x := recover(); x != nil {
			switch e := x.(type) {
			case unported:
				res, err = nil, e
			case GameError:
				res, err = nil, e
			case error:
				res, err = nil, e
			default:
				panic(x)
			}
		}
	}()

	built := map[*Room]*RoomTiles{}
	res = make([]*RoomTiles, len(l.Rooms))

	for _, typ := range []int{1, 2} {
		if typ == 2 && plainOnly {
			break
		}

		for i, r := range l.Rooms {
			if r.Type != typ {
				continue
			}

			lib, e := l.RoomLibrary(r)
			if e != nil {
				return nil, e
			}

			rt := &RoomTiles{Room: r, Lib: lib, nbrs: l.neighbourRooms(r)}
			tb := &tileBuilder{l: l, r: r, rt: rt, built: built}
			tb.markTownAdjacency()

			if typ == 1 {
				g, e := l.BuildRoomGrids(r, &RoomBuildOptions{tiles: tb})
				if e != nil {
					return nil, fmt.Errorf("room %d,%d: %w", r.X, r.Y, e)
				}

				rt.Seed = g.Seed
				tb.rs = &rt.Seed
				tb.buildCells(g)
			} else {
				rt.Seed.Init(r.S4)
				tb.rs = &rt.Seed

				if e := tb.buildPreset(); e != nil {
					return nil, fmt.Errorf("preset room %d,%d: %w", r.X, r.Y, e)
				}
			}

			built[r] = rt
			res[i] = rt
		}
	}

	return res, nil
}

// inRectEx is DRLG_IsPointInRect (0x66e690): the room rectangle without its
// right and bottom edge.
func inRectEx(r *Room, x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}
