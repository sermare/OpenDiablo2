package drlgmaze

// Act 4 and Act 5 maze types (LevelType 28, 32, 33, 34, 35): River of Flame,
// the Halls of Anguish/Death's Calling, the ice caves, the Worldstone Keep
// levels and the Hell/Pandemonium levels. Read from DRLG_GenerateMazeLevel
// (0x6768b0) and its helpers; compared against the emulated game in
// oracle_test.go (maze_act45.json).

// LevelType ids (LvlTypes.txt) of Acts 4 and 5.
const (
	typeLava   = 28 // River of Flame (level 107)
	typeTemple = 32 // Halls of Anguish / Death's Calling (122, 123)
	typeIce    = 33 // ice caves (113-119)
	typeBaal   = 34 // Worldstone Keep (128-130)
	typeHell   = 35 // Hell 1-3 and Pandemonium 3 (125-127, 135)
)

// Stamp tables (4 rows, dir column 3,0,1,2) from the exe, 0x6f18a8..0x6f1b88.
var (
	// 0x6f18a8 / 0x6f18d8: rows are not the usual N,E,S,W order (the first
	// table is indexed by a counter in 0..2); read verbatim.
	templeA = [4]stamp{{0x412, 0x416, 1}, {0x413, 0x417, 0}, {0x415, 0x418, 3}, {0x412, 0x419, 1}}
	templeB = [4]stamp{{0x412, 0x419, 1}, {0x413, 0x41a, 0}, {0x415, 0x41c, 3}, {0x414, 0x41b, 2}}

	iceFrom = [4]int{0x3f2, 0x3ec, 0x3ee, 0x3eb}
	iceT1   = tbl(iceFrom, [4]int{0x3fd, 0x3fb, 0x3fc, 0x3fa})
	iceT2   = tbl(iceFrom, [4]int{0x401, 0x3ff, 0x400, 0x3fe})
	iceT3   = tbl(iceFrom, [4]int{0x405, 0x403, 0x404, 0x402})
	iceT4   = tbl(iceFrom, [4]int{0x409, 0x407, 0x408, 0x406})
	iceT5   = tbl(iceFrom, [4]int{0x40a, 0x40c, 0x40b, 0x40d})

	baalFrom = [4]int{0x42a, 0x424, 0x426, 0x423}
	baalA    = tbl(baalFrom, [4]int{0x436, 0x438, 0x437, 0x439})
	baalB    = tbl(baalFrom, [4]int{0x43a, 0x43c, 0x43b, 0x43d})
	// 0x6f0e3c: Worldstone Keep Prev Defs by start counter.
	baalPrevDef = [4]int{0x433, 0x435, 0x434, 0x432}

	// 0x6f1a24: River of Flame stamp, chosen by one level-seed step & 1.
	lavaStamp = [2]stamp{{0x345, 0x355, 2}, {0x346, 0x356, 0}}

	// 0x6f0c88 (stride 8): Def by door mask for types 32 (A) and 35 (B).
	templeDef = [16]int{5: 0x415, 6: 0x414, 9: 0x413, 10: 0x412}
	hellDef   = [16]int{1: 0x420, 2: 0x41f, 3: 0x421, 4: 0x41e, 8: 0x41d, 12: 0x422}

	// 0x6f0dd8 (stride 0x18): the two rooms added next to the start room of
	// the Hell levels, indexed by a room-seed draw & 3.
	hellAdds = [4][2]struct{ def, dir, file int }{
		{{0x41e, North, 0}, {0x41d, South, 1}},
		{{0x41e, North, 1}, {0x41d, South, 0}},
		{{0x41f, West, 1}, {0x420, East, 0}},
		{{0x41f, West, 0}, {0x420, East, 1}},
	}
)

// linkDiag links a to b in dir 0..7 (DRLG_LinkRoomsBidirectional: the far
// side gets (dir-2)&3).
func linkDiag(a, b *room, dir int) {
	a.addNb(b, dir)
	b.addNb(a, (dir-2)&3)
}

// fillAround is FUN_00674090: for every room of a snapshot of the list
// (except skip) try the 8 neighbouring positions and add a locked room with
// the given Def. One level-seed step per attempt, no merge, no Def re-pick.
func (l *level) fillAround(def int, skip *room) {
	snap := append([]*room(nil), l.rooms...)

	for _, c := range snap {
		if c == skip {
			continue
		}

		for d := 0; d < 8; d++ {
			n := l.alloc()
			if !l.tryPlace(c, d, n) {
				continue
			}

			linkDiag(c, n, d)
			l.prepend(n)
			n.def, n.file, n.locked = def, -1, true
		}
	}
}

// buildAct45 runs the Act 4/5 part of the LevelType switch. done reports
// that it replaced the normalize step.
func (l *level) buildAct45(target int, res *Result) (done bool, err error) {
	switch l.typ {
	case typeLava:
		if err = l.fillRooms(target); err != nil {
			return false, err
		}

		return true, l.lavaJoint(res)
	case typeTemple:
		l.buildRing(2)

		if l.id != 0x7c {
			ctr := int(l.seed.Step() % 3)
			first := ctr

			l.stamp(templeA, &ctr)

			if l.id == 0x7b {
				if ctr == first {
					return false, errPlace
				}

				l.stamp(templeB, &ctr)
			}
		}
	case typeIce:
		switch l.id {
		case 0x72:
			def := 0x40f
			if l.seed.Roll(2) != 0 {
				def--
			}

			l.single(def)
		case 0x74:
			l.single(0x410)
		case 0x77:
			l.single(0x411)
		default:
			l.buildRing(2)

			if err = l.fillRooms(target); err != nil {
				return false, err
			}

			ctr := int(l.seed.Step() & 3)
			l.stampAll(&ctr, iceT1, iceT2, iceT3)

			switch l.id {
			case 0x73:
				l.stamp(iceT4, &ctr)
				l.stamp(iceT5, &ctr)
			case 0x71, 0x76:
				l.stamp(iceT5, &ctr)
			}
		}
	case typeBaal:
		l.tombStart()

		if err = l.fillRooms(target); err != nil {
			return false, err
		}

		ctr := int(l.seed.Step() & 3)
		l.stamp(baalA, &ctr)

		if l.id == 0x81 {
			l.stamp(baalB, &ctr)
		}
	case typeHell:
		start := l.rooms[0]
		k := int(start.seed.Step() & 3)

		for _, a := range hellAdds[k] {
			if l.addWith(start, a.dir, a.def, a.file) == nil {
				return false, errPlace
			}
		}

		l.fillAround(0x344, nil)
	}

	return false, nil
}

// single turns the only room into a fixed locked Def.
func (l *level) single(def int) {
	r := l.rooms[0]
	r.locked, r.def, r.file = true, def, -1
}

// lavaJoint is FUN_00676090: hang River of Flame off Chaos Sanctuary (level
// 108) and recompute the level rect from the room bounding box.
func (l *level) lavaJoint(res *Result) error {
	s := l.p.L108

	a := l.extreme(South)
	if a == nil || l.addWith(a, South, 0x354, -1) == nil {
		return errPlace
	}

	b := l.extreme(North)
	if b == nil {
		return errPlace
	}

	cur := l.addWith(b, North, 0x357, -1)
	if cur == nil {
		return errPlace
	}

	for i := 0; i < 2; i++ {
		n := l.alloc()
		if !l.tryPlace(cur, North, n) {
			return errPlace
		}

		linkRooms(cur, n, North)
		l.prepend(n)
		n.def, n.file, n.locked = 0x358, -1, true
		cur = n
	}

	dx := cur.w*2 - cur.x + s.X
	dy := s.H - cur.y + s.Y

	var dummy int

	l.stampOne(lavaStamp[l.seed.Step()&1], &dummy)
	l.fillAround(0x344, cur)

	for _, rm := range l.rooms {
		rm.x += dx
		rm.y += dy
	}

	minX, minY, maxX, maxY := l.rooms[0].x, l.rooms[0].y, l.rooms[0].x+l.rooms[0].w, l.rooms[0].y+l.rooms[0].h

	for _, rm := range l.rooms {
		minX, minY = imin(minX, rm.x), imin(minY, rm.y)
		maxX, maxY = imax(maxX, rm.x+rm.w), imax(maxY, rm.y+rm.h)
	}

	res.RectX, res.RectY, res.RectW, res.RectH = minX, minY, maxX-minX, maxY-minY

	return nil
}

// maskOf is the door-mask bit of a neighbour direction; diagonal links
// (dir 4..7, only the Act 4/5 filler rooms) have none.
func maskOf(dir int) int {
	if dir < 0 || dir > 3 {
		return 0
	}

	return dirMask[dir]
}
