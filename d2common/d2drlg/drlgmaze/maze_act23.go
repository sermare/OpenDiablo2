package drlgmaze

import "errors"

// Stamp tables of the Act 2, Act 3 and barracks finishers, read from the exe
// (0x6f1298..0x6f1a08, one 4-row table per special room kind, rows N,E,S,W).
var (
	sewerFrom   = [4]int{309, 303, 305, 302}
	sewerPrev   = tbl(sewerFrom, [4]int{335, 333, 334, 332})
	sewerNext   = tbl(sewerFrom, [4]int{340, 338, 339, 337})
	sewerWP     = tbl(sewerFrom, [4]int{348, 346, 347, 345})
	sewerRadam  = tbl(sewerFrom, [4]int{344, 342, 343, 341})
	sewerChest  = tbl(sewerFrom, [4]int{352, 350, 351, 349})
	tombFrom    = [4]int{421, 415, 417, 414}
	tombNext    = tbl(tombFrom, [4]int{451, 449, 450, 448})
	tombWP      = tbl(tombFrom, [4]int{479, 477, 478, 476})
	tombChest   = tbl(tombFrom, [4]int{475, 473, 474, 472})
	tombLeather = tbl(tombFrom, [4]int{467, 465, 466, 464})
	tombCube    = tbl(tombFrom, [4]int{459, 457, 458, 456})
	tombTreas   = tbl(tombFrom, [4]int{455, 453, 454, 452})
	tombTal     = tbl(tombFrom, [4]int{463, 461, 462, 460})
	tombKaa     = tbl(tombFrom, [4]int{471, 469, 470, 468})
	lairFrom    = [4]int{489, 483, 485, 482}
	lairPrev    = tbl(lairFrom, [4]int{500, 498, 499, 497})
	lairNext    = tbl(lairFrom, [4]int{504, 502, 503, 501})
	lairTreas   = tbl(lairFrom, [4]int{508, 506, 507, 505})
	lairTight   = stamp{485, 509, 1} // single entry 0x6f1698
	arcFrom     = [4]int{517, 511, 513, 510}
	arcSummoner = tbl(arcFrom, [4]int{528, 526, 527, 525})
	dunFrom     = [4]int{672, 666, 668, 665}
	dunPrev     = tbl(dunFrom, [4]int{698, 696, 697, 695})
	dunNext     = tbl(dunFrom, [4]int{702, 700, 701, 699})
	sw3From     = [4]int{712, 706, 708, 705}
	sw3Drain    = tbl(sw3From, [4]int{742, 740, 741, 739})
	sw3Chest    = tbl(sw3From, [4]int{746, 744, 745, 743})
	mephFrom    = [4]int{761, 755, 757, 754}
	mephPrev    = tbl(mephFrom, [4]int{787, 785, 786, 784})
	mephWP      = tbl(mephFrom, [4]int{795, 793, 794, 792})
	mephNext    = tbl(mephFrom, [4]int{791, 789, 790, 788})
	barFrom     = [4]int{175, 169, 171, 168}
	barNext     = tbl(barFrom, [4]int{201, 199, 200, 198})
	barForge    = tbl(barFrom, [4]int{205, 203, 204, 202})
	// 0x6f0e38: tomb Prev Defs indexed by the start counter (stride 8).
	tombPrevDef = [4]int{447, 444, 446, 445}
)

var errPlace = errors.New("drlgmaze: special room placement failed")

// addWith is AddMazeRoomWithDef (0x673b50): a locked room with a fixed Def and
// file next to cur. No merge pass; cur is re-picked from its door mask.
func (l *level) addWith(cur *room, dir, def, file int) *room {
	n := l.alloc()
	if !l.tryPlace(cur, dir, n) {
		return nil
	}

	linkRooms(cur, n, dir)
	l.prepend(n)
	l.selectDef(cur)
	n.def, n.file, n.locked = def, file, true

	return n
}

// probe is FUN_006750d0: can a room be added at dir of r? It draws one
// level-seed step (the trial room) and leaves no trace.
func (l *level) probe(r *room, dir int) bool {
	if r.locked {
		return false
	}

	for _, n := range r.nb {
		if n.dir == dir {
			return false
		}
	}

	n := l.alloc()

	return l.tryPlace(r, dir, n)
}

// extreme is 0x675170/0x6751b0/0x6751f0/0x675230: the room that reaches
// furthest in dir and can still grow there.
func (l *level) extreme(dir int) *room {
	var best *room

	for _, r := range l.rooms {
		var beats bool

		switch {
		case best == nil:
			beats = true
		case dir == North:
			beats = r.y < best.y
		case dir == West:
			beats = r.x < best.x
		case dir == East:
			beats = r.x > best.x
		default:
			beats = r.y > best.y
		}

		if beats && l.probe(r, dir) {
			best = r
		}
	}

	return best
}

func (l *level) ringFill(target int) error {
	l.buildRing(2)

	return l.fillRooms(target)
}

func (l *level) stampAll(ctr *int, ts ...[4]stamp) {
	for _, t := range ts {
		l.stamp(t, ctr)
	}
}

// buildAct23 runs the LevelType switch of DRLG_GenerateMazeLevel for the
// remaining types. done reports that it also replaced the normalize step.
func (l *level) buildAct23(target int, res *Result) (done bool, err error) {
	switch l.typ {
	case typeBarracks:
		if err = l.ringFill(target); err != nil {
			return false, err
		}

		if l.id == 28 {
			return true, l.barracks(res)
		}
	case typeSewer2:
		if err = l.ringFill(target); err != nil {
			return false, err
		}

		return false, l.sewer2()
	case typeHarem, typeBasement, typeSpider:
		l.buildRing(2)
	case typeTomb:
		if l.id == 0x3d {
			r := l.rooms[0]
			r.locked, r.def, r.file = true, 0x1e0, -1

			return false, nil
		}

		l.tombStart()

		if err = l.fillRooms(target); err != nil {
			return false, err
		}

		l.tombFinish()
	case typeLair:
		if err = l.ringFill(target); err != nil {
			return false, err
		}

		ctr := int(l.seed.Step() & 3)

		if l.id == 0x40 {
			l.stampOne(lairTight, &ctr)

			ctr = 3
			l.stampOne(lairTreas[3], &ctr)
		} else {
			l.stamp(lairNext, &ctr)
		}

		l.stamp(lairPrev, &ctr)
	case typeMephisto:
		if err = l.ringFill(target); err != nil {
			return false, err
		}

		ctr := int(l.seed.Step() & 3)
		l.stamp(mephPrev, &ctr)

		switch l.id {
		case 0x65:
			l.stampAll(&ctr, mephWP, mephNext)
		case 100:
			l.stampAll(&ctr, mephNext)
		}
	case typeDungeon:
		if err = l.ringFill(target); err != nil {
			return false, err
		}

		ctr := int(l.seed.Step() & 3)
		l.stampAll(&ctr, dunPrev, dunNext)
	case typeSewer3:
		return false, l.sewer3(target)
	case typeArcane:
		return false, l.arcane()
	}

	return false, nil
}

func (l *level) sewer2() error {
	dirA := int(l.seed.Step()&1)*2 | 1
	ctr := int(l.seed.Step() & 3)

	switch l.id {
	case 0x2f:
		top := l.extreme(North)
		if top == nil {
			return errPlace
		}

		n1 := l.addGrown(top, North)
		if n1 == nil {
			return errPlace
		}

		n2 := l.addGrown(n1, North)
		if n2 == nil {
			return errPlace
		}

		l.addWith(n2, West, 0x14d, 0)

		east := l.extreme(East)
		if east == nil {
			return errPlace
		}

		e1 := l.addGrown(east, East)
		if e1 == nil {
			return errPlace
		}

		e2 := l.addGrown(e1, East)
		if e2 == nil {
			return errPlace
		}

		w := l.addWith(e2, dirA, 0x150, 0)
		if w == nil {
			return errPlace
		}

		// last room: placed and linked without a merge pass; only the new
		// room's Def is re-picked.
		n := l.alloc()
		if l.tryPlace(w, dirA, n) {
			linkRooms(w, n, dirA)
			l.prepend(n)
			l.selectDef(n)
		}

		l.stamp(sewerNext, &ctr)
	case 0x30:
		l.stampAll(&ctr, sewerPrev, sewerWP, sewerNext)
	case 0x31:
		l.stampAll(&ctr, sewerPrev, sewerRadam)
	case 0x41:
		l.stampAll(&ctr, sewerPrev, sewerChest)
	}

	return nil
}

func (l *level) sewer3(target int) error {
	if l.id == 0x5c {
		l.buildRing(5)

		for _, sw := range [][2]int{{0x2c5, 0x2df}, {0x2c6, 0x2e0}, {0x2c9, 0x2e1}, {0x2ca, 0x2e2}} {
			found := false

			for _, r := range l.rooms {
				if !r.locked && r.def == sw[0] {
					r.locked, r.def, r.file = true, sw[1], -1
					found = true

					break
				}
			}

			if !found {
				return errPlace
			}
		}

		if err := l.fillRooms(target); err != nil {
			return err
		}
	} else if err := l.ringFill(target); err != nil {
		return err
	}

	ctr := int(l.seed.Step() & 3)
	l.stampAll(&ctr, sw3Drain, sw3Chest)

	return nil
}

// tombStart is FUN_00674620 for the tomb type: three rooms around the start
// in consecutive directions from a random one, then the Prev Def.
func (l *level) tombStart() {
	start := l.rooms[0]
	ctr := int(l.seed.Step() & 3)

	for i := 0; i < 3; i++ {
		l.addGrown(start, ctr)

		ctr = (ctr + 1) & 3
	}

	start.def, start.file, start.locked = tombPrevDef[ctr], -1, true
}

// tombFinish is FUN_00675940.
func (l *level) tombFinish() {
	var prev *room

	for _, r := range l.rooms {
		prev = r
		if r.def > 0x1ac {
			break
		}
	}

	ctr := 0

	switch prev.def {
	case 0x1bc:
		ctr = 3
	case 0x1bd:
		ctr = 1
	case 0x1be:
		ctr = 0
	case 0x1bf:
		ctr = 2
	}

	id, a, b := l.id, l.p.TombA, l.p.TombB

	if id >= 55 && id <= 58 {
		l.stamp(tombNext, &ctr)
	}

	if id == 57 {
		l.stamp(tombWP, &ctr)
	}

	if id == 59 || id == 61 {
		l.stamp(tombChest, &ctr)
	}

	if id >= 66 && id <= 72 && id != a {
		l.stamp(tombChest, &ctr)
	}

	if id == 59 {
		l.stamp(tombLeather, &ctr)
	}

	if id == 60 {
		l.stamp(tombCube, &ctr)
	}

	if id == 59 || id == 61 {
		l.stamp(tombTreas, &ctr)
	}

	if id == a {
		l.stamp(tombTal, &ctr)
	}

	if id == b {
		l.stamp(tombKaa, &ctr)
	}
}

// barracks is DRLG_FinishBarracksLevel (0x675e80): hang the barracks off
// level 27's exit side and recompute the rect.
func (l *level) barracks(res *Result) error {
	s27 := l.p.L27
	side := s27.Side
	ctr := side

	var r *room

	switch side {
	case 0:
		r = l.extreme(East)
	case 1:
		r = l.extreme(South)
	case 2:
		r = l.extreme(West)
	default:
		return errPlace
	}

	if r == nil {
		return errPlace
	}

	dx, dy := s27.X, s27.Y

	var h *room

	switch side {
	case 0:
		h = l.addWith(r, East, 0xa7, 0)
		if h == nil {
			return errPlace
		}

		dx -= h.x + l.rec.SizeX
		dy += s27.H/2 - h.y
	case 1:
		h = l.addWith(r, South, 0xa7, 1)
		if h == nil {
			return errPlace
		}

		dx += s27.W/2 - h.x - 6
		dy -= h.y + l.rec.SizeY
	default:
		h = l.addWith(r, West, 0xa7, 2)
		if h == nil {
			return errPlace
		}

		dx += s27.W - h.x
		dy += s27.H/2 - h.y + 1
	}

	if l.seed.Step()&1 != 0 {
		l.stampAll(&ctr, barNext, barForge)
	} else {
		l.stampAll(&ctr, barForge, barNext)
	}

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

// Arcane Sanctuary (FUN_00674730 + FUN_00675bb0): four arms of 15 rooms around
// the start room. The placement script (parent index within the arm, 0 = start,
// and direction offset) was read from the emulated game by tracing
// TryPlaceMazeRoom and is identical for every seed and difficulty; arm k
// rotates every direction by k. The arm of a room sets its preset file
// ((slot/15 + ctr) & 3); the start room gets file 4.
var (
	arcaneParent = [15]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 8, 10, 11, 12, 12, 14}
	arcaneDir    = [15]int{0, 0, 3, 0, 0, 0, 0, 1, 0, 1, 2, 2, 3, 2, 2}
)

func (l *level) arcane() error {
	ctr := int(l.seed.Step() & 3)
	start := l.rooms[0]

	var slots [60]*room

	for k := 0; k < 4; k++ {
		made := []*room{start}

		for p := 0; p < 15; p++ {
			n := l.addGrown(made[arcaneParent[p]], (arcaneDir[p]+k)&3)
			if n == nil {
				return errPlace
			}

			made = append(made, n)

			if p != 8 && p != 12 {
				slots[15*k+p] = n
			}
		}
	}

	for i, r := range slots {
		if r != nil {
			r.file = (i/15 + ctr) & 3
		}
	}

	start.file = 4

	ctr = int(l.seed.Step() & 3)
	l.stamp(arcSummoner, &ctr)

	return nil
}
