package drlgoutdoor

// Logic regions of a room (DrlgLogic.cpp): the rectangles the monster population rolls its density in.
//
// MONREGION_SpawnRoomMonsters (0x54cad0) does not roll over the room rectangle: it walks the room's list of logic
// regions (Room+0x64 -> +0x30, 0x30 byte nodes, newest first) and rolls ((w / 3) * (h / 3)) times for the clipped
// rectangle (node +0x10..+0x1c, in subtiles) of every node that has an id (+0x28 != 0) and is not skipped
// (+0x20 == 0). Plain (outdoor) rooms and the presets of a LvlPrest row with Logicals = 0 have one node, the whole
// room (DRLG_AllocRoomLogicData 0x66f960). A preset with Logicals = 1 (maze rooms, the Act 1 cloisters, Act 2 tombs,
// Act 3 sewers and Travincal, Act 4 lava and mesa) runs DRLG_BuildPresetRoomLogicGrid (0x66fdb0):
//
//  1. block grid ((w+1) x (h+1) cells): every wall tile record of the room, and of the already built neighbouring
//     rooms (their non-floor groups), with layer bits 14-16 == 1 (wall layer 0), orientation != 0xf and flag 0x800
//     clear, marks its cell (DRLG_PresetLogicGridPassB 0x66f4f0);
//  2. labelling (DRLG_LabelLogicRegions 0x66f210 + DRLG_FloodFillLogicRegion 0x66f090): cells are scanned row by row; an
//     unlabelled one starts a new label (counter + 1), flagged "skip" when the floor layer 0 cell is the hidden/lava
//     tile ((v & 0x1e0ff00) == 0x1e00000 or bit 31), and floods: an open cell takes the label and spreads to its four
//     neighbours, a block cell (a wall record) spreads only as far as the table 0x6f03ec says for the orientation of
//     wall layer 0 (selector table 0x6f0450) and the direction it was entered from;
//  3. rectangles (DRLG_BuildLogicGridFromTiles 0x66f6d0): the label grid is cut into rectangles, scanning row by row:
//     the first unclaimed cell starts a node that grows right while the label is the same and the cells are
//     unclaimed, then down while the whole row segment matches; every node is clipped to the room.
//
// Verified against the emulated game (testdata/logic_*.json). The id remap between overlapping rooms
// (DRLG_RemapOverlappingRoomRegions) changes ids only and is not modelled; ids are counted per level as the original
// does (level +0x1dc).

// LogicRegion is one node of a room's logic region list.
type LogicRegion struct {
	// X0, Y0, X1, Y1 are the clipped rectangle (node +0x10..+0x1c) in level tiles, right and bottom exclusive; all
	// zero when the node lies outside the room (the population skips it).
	X0, Y0, X1, Y1 int
	// Skip is node +0x20: the region starts on a lava / hidden floor tile and gets no density rolls.
	Skip bool
	// ID is node +0x28 (0: unlabelled cells, never populated).
	ID int
}

// logicTbl is the data at 0x6f03c4..0x6f049c: the flood direction deltas (word 0, 1, 3, 4, ...: dx at +0, dy at +4
// of each 8 byte entry), the pass-through masks 0x6f03ec (word 10 + dir + 5 * selector) and the selector table 0x6f0450
// (word 35 + orientation).
var logicTbl = [55]int32{1, 0, 0, 1, -1, 0, 0, -1, 0, 23, 0, 5, 21, 17, 15, 3, 0, 9, 7, 39, 0, 0, 5, 3, 31, 31, 31, 31, 31, 31,
	31, 31, 31, 31, 0, -1, 0, 1, 2, 2, 0, 1, 3, 0, 1, 0, 1, 4, -1, 4, 0, 0, 0, 0, 0}

// logicSelector is the table 0x6f0450 entry of a wall layer 0 orientation (values past the table, whose end is the
// string that follows it, never occur in the DS1 files).
func logicSelector(ori uint32) int {
	if ori >= 20 {
		return 0
	}

	return int(logicTbl[35+int(ori)])
}

// logicRegionsWhole is DRLG_AllocRoomLogicData: one node, the whole room.
func (t *tileBuilder) logicRegionsWhole() []LogicRegion {
	r := t.r
	t.l.logicCtr = 1

	return []LogicRegion{{X0: r.X, Y0: r.Y, X1: r.X + r.W, Y1: r.Y + r.H, ID: 1}}
}

// logicRegions is DRLG_BuildPresetRoomLogicGrid for a preset room: ori0 / floor0 are the (w+1) x (h+1) windows of the
// first wall orientation layer and the first floor layer (nil when the DS1 has none), ww x wh their size.
func (t *tileBuilder) logicRegions(ori0, floor0 []uint32, ww, wh int) []LogicRegion {
	r := t.r
	block := make([]bool, ww*wh)

	mark := func(x, y int) {
		if x >= 0 && y >= 0 && x < ww && y < wh {
			block[y*ww+x] = true
		}
	}

	blocks := func(rec *TileRecord) bool {
		return rec.Flags&0x1c000 == 0x4000 && rec.Ori != 0xf && rec.Flags&0x800 == 0
	}

	for _, rec := range t.rt.Walls {
		if blocks(rec) {
			mark(rec.X, rec.Y)
		}
	}

	for _, nb := range t.rt.nbrs {
		nrt := t.built[nb]
		if nb == r || nrt == nil {
			continue
		}

		for _, g := range nrt.groups {
			if g.floor {
				continue
			}

			for rec := g.head; rec != nil; rec = rec.next {
				if !blocks(rec) {
					continue
				}

				// RECT_ContainsPoint: the room rectangle with its right and bottom edge
				if ax, ay := nb.X+rec.X, nb.Y+rec.Y; ax >= r.X && ay >= r.Y && ax <= r.X+r.W && ay <= r.Y+r.H {
					mark(ax-r.X, ay-r.Y)
				}
			}
		}
	}

	const (
		labelled = 0x10000000
		skipBit  = 0x20000000
	)

	label := make([]uint32, ww*wh)
	counter := t.l.logicCtr

	if counter == 0 {
		counter = 1
	}

	cur := uint32(0)

	var flood func(x, y, dir int)

	flood = func(x, y, dir int) {
		for x >= 0 && y >= 0 && x < ww && y < wh {
			if label[y*ww+x]&labelled != 0 {
				return
			}

			if !block[y*ww+x] {
				label[y*ww+x] |= cur

				for i := 0; i < 4; i++ {
					flood(x+int(logicTbl[2*i]), y+int(logicTbl[2*i+1]), i)
				}

				return
			}

			sel := 0
			if ori0 != nil {
				sel = logicSelector(ori0[y*ww+x])
			}

			mask := logicTbl[10+dir+5*sel]

			if mask&1 != 0 {
				label[y*ww+x] |= cur
			}

			if mask&2 != 0 && dir != 2 {
				flood(x+1, y, 0)
			}

			if mask&4 != 0 && dir != 3 {
				flood(x, y+1, 1)
			}

			if mask&8 != 0 && dir != 0 {
				flood(x-1, y, 2)
			}

			if mask&0x10 != 0 && dir != 1 {
				flood(x, y-1, 3)
			}

			if mask&0x20 == 0 {
				return
			}

			x, y, dir = x+1, y+1, -1
		}
	}

	for y := 0; y < wh; y++ {
		for x := 0; x < ww; x++ {
			if label[y*ww+x]&labelled != 0 {
				continue
			}

			counter++
			cur = uint32(counter)&0xfffffff | labelled

			var fv uint32
			if floor0 != nil {
				fv = floor0[y*ww+x]
			}

			if fv&0x1e0ff00 == 0x1e00000 || int32(fv) < 0 {
				cur |= skipBit
			}

			flood(x, y, -1)
		}
	}

	// DRLG_BuildLogicGridFromTiles: rectangles of equal label, newest node first
	claimed := make([]bool, ww*wh)

	same := func(x, y int, id uint32) bool {
		return label[y*ww+x]&0xfffffff == id && !claimed[y*ww+x]
	}

	var out []LogicRegion

	for y := 0; y < wh; y++ {
		for x := 0; x < ww; x++ {
			if claimed[y*ww+x] {
				continue
			}

			v := label[y*ww+x]
			id := v & 0xfffffff
			x1 := x

			for x1 < ww && same(x1, y, id) {
				x1++
			}

			y1 := y

			for y1 < wh {
				ok := true

				for xx := x; xx < x1; xx++ {
					if !same(xx, y1, id) {
						ok = false
						break
					}
				}

				if !ok {
					break
				}

				y1++
			}

			for yy := y; yy < y1; yy++ {
				for xx := x; xx < x1; xx++ {
					claimed[yy*ww+xx] = true
				}
			}

			n := LogicRegion{X0: x + r.X, Y0: y + r.Y, X1: x1 + r.X, Y1: y1 + r.Y, Skip: v&skipBit != 0, ID: int(id)}

			if n.X1 > r.X+r.W {
				n.X1 = r.X + r.W
			}

			if n.Y1 > r.Y+r.H {
				n.Y1 = r.Y + r.H
			}

			if n.X0 >= r.X+r.W || n.Y0 >= r.Y+r.H {
				n.X0, n.Y0, n.X1, n.Y1 = 0, 0, 0, 0
			}

			out = append([]LogicRegion{n}, out...)
		}
	}

	t.l.logicCtr = counter + 1 // level +0x1dc += (labels + 1)

	return out
}

// isTownLevel is LEVEL_IsTownLevelId (0x643b50).
func isTownLevel(id int) bool {
	switch id {
	case 1, 0x28, 0x4b, 0x67, 0x6d:
		return true
	}

	return false
}

// markTownAdjacency is ROOM_MarkTownAdjacency (0x66ea10, called by DRLG_BuildRoomNeighborLinks 0x66f030 when a room is
// first built): a room of a level that is not a town gets the no-population flag 0x800000 when the neighbour list
// it collected contains a room of a town. The list holds the rooms of the adjacent level that DRLG_LinkNeighborAndWarp
// (0x66eb40) adds for each exit bit (0x10 << k, vis slot k) of the room: the rooms of that level with the matching
// back exit bit that are closer than 6 tiles on both axes. The adjacent level's rooms are not generated here; the
// rectangle of the town stands in for them (a town is tiled with rooms and its exit rooms touch the neighbour), which
// matched the emulator's flags for the levels next to a town (TestOracleLogicRegions). An exit through a level warp
// (Palace / Sewers from Lut Gholein) links the destination's first room with the back exit bit whatever the distance.
// UNVERIFIED where a town's exit does not touch the whole length of the neighbour's edge.
func (t *tileBuilder) markTownAdjacency() {
	l, r := t.l, t.r

	if isTownLevel(l.Params.ID) || l.Params.ID == 0x85 {
		return
	}

	for k := 0; k < 8; k++ {
		if r.Flags&(0x10<<uint(k)) == 0 || !isTownLevel(l.Params.Vis[k]) {
			continue
		}

		nb := l.Params.Vis[k]

		// an exit through a level warp (a stair or portal tile rather than an open edge): the first room of the
		// destination with the matching exit bit is linked whatever its distance, so the town marks the room
		if l.Params.Warp[k] != -1 {
			r.Flags |= 0x800000

			return
		}

		rects := map[int]Rect{}
		for _, n := range l.Params.Neighbors {
			rects[n.Level] = n.Rect
		}

		if a, ok := l.Params.Adjacent[nb]; ok {
			rects[nb] = a
		}

		if tr, ok := rects[nb]; ok && gapX(r.X, r.W, tr.X, tr.W) < 6 && gapX(r.Y, r.H, tr.Y, tr.H) < 6 {
			r.Flags |= 0x800000

			return
		}
	}
}

// gapX is the distance measure of DRLG_CollectNeighborRooms: the gap between two intervals (negative when they
// overlap).
func gapX(a, aw, b, bw int) int {
	if a < b {
		return b - aw - a
	}

	return a - bw - b
}
