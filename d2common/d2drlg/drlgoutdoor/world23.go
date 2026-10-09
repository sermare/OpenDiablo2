package drlgoutdoor

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// World23 is the layout of Act 2 or Act 3 as DRLG_InitActLevelLinks
// (0x67b8f0, cases 1 and 2) decides it: level rectangles, registered vis links,
// and the per-act extras (drlg-act23-outdoor.md 3.1 and 4.1/4.2).
type World23 struct {
	Act      int // 2 or 3
	Base     uint32
	Flip     int // Act 3 Kurast layout flip (drlg+0x474)
	TownFile int // Act 2: preset file slot of Lut Gholein (1 LutW, 2 LutN)
	Rects    map[int]Rect
	Jungle   map[int]JungleInfo // Act 3: levels 76..78
	// Seed is the real drlg seed after the placer (Act 3: advanced by the
	// jungle placer; Act 2: untouched, the search runs on a copy).
	Seed d2rand.Seed

	links linkSet
}

var errWorld23 = errors.New("drlgoutdoor: act world search did not converge")

func overlaps(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

type act2Kind int

const (
	act2Anchor act2Kind = iota
	act2NextToTown
	act2Staircase
	act2Compass
)

type act2Entry struct {
	kind  act2Kind
	level int
	ref   int
}

var (
	act2ClusterA = []act2Entry{{act2Anchor, 40, -1}, {act2NextToTown, 41, 0}, {act2Staircase, 42, 1},
		{act2Staircase, 43, 2}, {act2Staircase, 44, 3}, {act2Compass, 45, 4}}
	act2ClusterB = []act2Entry{{act2Anchor, 46, -1}}

	// act2Links are the pass 3 vis registrations (level, ref) in table order.
	act2Links = [][2]int{{41, 40}, {42, 41}, {43, 42}, {44, 43}, {45, 44}}
)

type act2Ctx struct {
	seed       d2rand.Seed
	tbl        []act2Entry
	rect       []Rect
	dir, first []int
}

// place runs the placement function of entry i (false = options exhausted).
func (c *act2Ctx) place(i int, t d2drlg.Levels) bool {
	e := c.tbl[i]
	r := &c.rect[i]

	if e.kind == act2Anchor {
		l, _ := t.Level(e.level)
		r.X, r.Y = l.OffsetX, l.OffsetY

		return true
	}

	step := func(mask, wrap uint32) bool {
		if c.first[i] == -1 {
			c.first[i] = int(c.seed.Step() & mask)
			c.dir[i] = c.first[i]

			return true
		}

		nd := (c.dir[i] + 1) & int(wrap)
		if e.kind == act2NextToTown {
			nd = 2
			if c.dir[i] != 1 {
				nd = 1
			}
		}

		if nd == c.first[i] {
			return false
		}

		c.dir[i] = nd

		return true
	}

	p, w, h := c.rect[e.ref], r.W, r.H

	switch e.kind {
	case act2NextToTown: // 0x6796e0: first = (Step & 1) + 1
		if c.first[i] == -1 {
			c.first[i] = int(c.seed.Step()&1) + 1
			c.dir[i] = c.first[i]
		} else if !step(0, 0) {
			return false
		}

		if c.dir[i] == 1 {
			r.X, r.Y = p.X-w, p.Y
		} else {
			r.X, r.Y = p.X, p.Y-h
		}
	case act2Staircase: // 0x678fc0
		if !step(7, 7) {
			return false
		}

		switch c.dir[i] {
		case 0:
			r.X, r.Y = p.X-8-cdiv(w, 2), p.Y+p.H
		case 1:
			r.X, r.Y = p.X+cdiv(w, 2)+8, p.Y+p.H
		case 2:
			r.X, r.Y = p.X-w, p.Y-cdiv(h, 2)-8
		case 3:
			r.X, r.Y = p.X-w, p.Y+cdiv(h, 2)+8
		case 4:
			r.X, r.Y = p.X-8-cdiv(w, 2), p.Y-h
		case 5:
			r.X, r.Y = p.X+cdiv(w, 2)+8, p.Y-h
		case 6:
			r.X, r.Y = p.X+p.W, p.Y-cdiv(h, 2)-8
		case 7:
			r.X, r.Y = p.X+p.W, p.Y+cdiv(h, 2)+8
		}
	case act2Compass: // 0x679820
		if !step(7, 7) {
			return false
		}

		switch c.dir[i] >> 1 {
		case 0:
			r.X, r.Y = p.X, p.Y+p.H
		case 1:
			r.X, r.Y = p.X-w, p.Y
		case 2:
			r.X, r.Y = p.X, p.Y-h
		case 3:
			r.X, r.Y = p.X+p.W, p.Y
		}
	}

	return true
}

// runAct2Cluster is the randomised depth-first search of the world placer; the
// validators of both Act 2 clusters are the same no-overlap test (touching is
// fine, the reference entry is exempt).
func runAct2Cluster(seed d2rand.Seed, tbl []act2Entry, t d2drlg.Levels, diff d2drlg.Difficulty) (*act2Ctx, error) {
	c := &act2Ctx{seed: seed, tbl: tbl, rect: make([]Rect, len(tbl)), dir: make([]int, len(tbl)), first: make([]int, len(tbl))}

	for i, e := range tbl {
		l, ok := t.Level(e.level)
		if !ok {
			return nil, fmt.Errorf("drlgoutdoor: level %d missing from Levels.txt", e.level)
		}

		c.rect[i] = Rect{l.OffsetX, l.OffsetY, l.SizeX[diff], l.SizeY[diff]}
		c.dir[i], c.first[i] = -1, -1
	}

	i := 0

	for guard := 0; i < len(tbl); guard++ {
		if guard > 200000 {
			return nil, errWorld23
		}

		if !c.place(i, t) {
			c.first[i], c.dir[i] = -1, -1

			if i--; i < 0 {
				return nil, errWorld23
			}

			continue
		}

		ok := true

		for j := 0; j < i; j++ {
			if j != tbl[i].ref && overlaps(c.rect[i], c.rect[j]) {
				ok = false
				break
			}
		}

		if ok {
			i++
		}
	}

	return c, nil
}

// PlaceAct2World is the Act 2 case of DRLG_InitActLevelLinks: levels 40..46.
func PlaceAct2World(t d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty) (*World23, error) {
	base, _ := d2rand.DrlgBaseSeed(gameSeed)
	ex := d2drlg.DrawActExtras(gameSeed, 1)
	w := &World23{Act: 2, Base: base, Rects: map[int]Rect{}, Seed: ex.Seed, links: linkSet{}}

	c, err := runAct2Cluster(ex.Seed, act2ClusterA, t, diff)
	if err != nil {
		return nil, err
	}

	for i, e := range act2ClusterA {
		w.Rects[e.level] = c.rect[i]
	}

	w.TownFile = c.dir[1]

	c2, err := runAct2Cluster(ex.Seed, act2ClusterB, t, diff)
	if err != nil {
		return nil, err
	}

	w.Rects[46] = c2.rect[0]

	for _, l := range act2Links {
		if err := w.links.get(t, l[0]).register(l[1]); err != nil {
			return nil, err
		}

		if err := w.links.get(t, l[1]).register(l[0]); err != nil {
			return nil, err
		}
	}

	return w, nil
}

// rectsAdjacent is DRLG_AreRectsAdjacent (0x66e590) with margin -1: the
// rectangles share a border segment of positive length.
func rectsAdjacent(a, b Rect) bool {
	gx := b.X - a.X - a.W
	if b.X < a.X {
		gx = a.X - b.X - b.W
	}

	gy := b.Y - a.Y - a.H
	if b.Y < a.Y {
		gy = a.Y - b.Y - b.H
	}

	if gx == 0 && gy <= -1 {
		return true
	}

	return gy == 0 && gx <= -1
}

type jungleNode struct {
	r        Rect
	dir      int
	children []*jungleNode
	c0, r1   int
	info     JungleInfo
}

// jungleChild is DRLG_PlaceJungleChildRect (0x67a510).
func jungleChild(par *jungleNode, d, w, h int) *jungleNode {
	var dx, dy int

	switch d {
	case 0:
		dy = -h
	case 1:
		dx, dy = -w, -cdiv(h, 3)
	case 2:
		dx, dy = w, -cdiv(h, 3)
	case 3:
		dx, dy = -w, cdiv(-2*h, 3)
	case 4:
		dx, dy = w, cdiv(-2*h, 3)
	}

	return &jungleNode{r: Rect{par.r.X + dx, par.r.Y + dy, w, h}, dir: d}
}

// placeJungle is DRLG_PlaceJungleLevels (0x67a5c0) on the REAL drlg seed. It
// returns the three jungle levels sorted by descending y (index 0 = level 76).
func placeJungle(seed *d2rand.Seed, town Rect, w, h int) ([]*jungleNode, error) {
	cw, ch := w/32, h/32
	s := []*jungleNode{{r: Rect{town.X, town.Y - h, w, h}}}
	minx, miny, maxx := town.X, town.Y-h, town.X+w

	for guard := 0; len(s) < 3; guard++ {
		if guard > 100000 {
			return nil, errWorld23
		}

		idx := int(seed.Roll(int32(len(s))))
		d := int(seed.Step() % 5)
		nw := jungleChild(s[idx], d, w, h)
		bad := false

		for _, t := range s {
			if overlaps(nw.r, t.r) {
				bad = true
				break
			}
		}

		if bad {
			continue
		}

		s[idx].children = append(s[idx].children, nw)
		minx, miny, maxx = min(minx, nw.r.X), min(miny, nw.r.Y), max(maxx, nw.r.X+nw.r.W)
		s = append(s, nw)
	}

	if (maxx-minx)%32 != 0 {
		return nil, GameError{0x64d}
	}

	if (town.Y-miny)%32 != 0 {
		return nil, GameError{0x64e}
	}

	nc, nr := (maxx-minx)/32+2, (town.Y-miny)/32+2
	A, B, C, D := make([]int, nr*nc), make([]int, nr*nc), make([]int, nr*nc), make([]int, nr*nc)

	g := func(arr []int, i int) int {
		if i >= 0 && i < len(arr) {
			return arr[i]
		}

		return 0
	}

	for restart := true; restart; {
		restart = false

		for _, arr := range [][]int{A, B, C, D} {
			for i := range arr {
				arr[i] = 0
			}
		}

		for k, nd := range s {
			col0, row0 := (nd.r.X-minx)/32+1, (nd.r.Y-miny)/32
			r1 := row0 + 1
			nd.c0, nd.r1 = col0, r1
			bottom := row0 + ch

			var phase int
			if k == 0 {
				phase = int(seed.Step() & 1)
			} else {
				phase = nd.dir & 1
			}

			side := func() int { // +1 when the path runs in the left column
				if phase == 0 {
					return 1
				}

				return -1
			}

			var cnt2 int

			for {
				for rr := 0; rr < ch; rr++ {
					for cc := 0; cc < cw; cc++ {
						i := (r1+rr)*nc + col0 + cc
						A[i], B[i], C[i], D[i] = k+1, 0, 0, 0
					}
				}

				pcol := col0 + phase

				if k != 0 {
					C[bottom*nc+pcol] = 1
					C[bottom*nc+pcol+side()] = 2
				}

				for _, cd := range nd.children {
					switch cd.dir {
					case 0:
						C[r1*nc+col0] = 1
					case 1:
						C[(row0+4)*nc+col0] = 1
					case 2:
						C[(row0+4)*nc+col0+1] = 1
					case 3:
						C[(row0+2)*nc+col0] = 1
					case 4:
						C[(row0+2)*nc+col0+1] = 1
					}
				}

				cnt2 = b2i(k != 0)
				run := 0

				if r1 <= bottom {
					rowoff, tile, row := nc*bottom, (k+1)*100, bottom

					for {
						B[rowoff+pcol] = tile
						tile++
						goUp := true

						if !(run == 0 || row == r1) {
							if seed.Step()%3 == 0 {
								goUp = false
								run = 0

								if phase == 0 {
									pcol++
									phase = 1
								} else {
									pcol--
									phase = 0
								}
							}
						}

						if goUp {
							run++

							if run > 1 && row > 1 {
								j := side() + rowoff + pcol
								if C[j] == 0 {
									cnt2++
									C[j] = 2
								}
							}

							row--
							rowoff -= nc
						}

						if !(r1 <= row) {
							break
						}
					}
				}

				if cnt2 >= 2 {
					break
				}
			}

			for cnt2 > 3 {
				sel := int(seed.Roll(int32(cnt2)))
				counter, found := 0, false

				for rr := 0; rr < ch && !found; rr++ {
					for cc := 0; cc < cw && !found; cc++ {
						i := (r1+rr)*nc + col0 + cc
						if C[i] == 2 {
							if sel == counter {
								C[i] = 0
								found = true
								cnt2--
							}

							counter++
						}
					}
				}
			}
		}

		// stage 4: masks from B adjacency and start markers
		for row := 0; row < nr; row++ {
			for col := 0; col < nc; col++ {
				cell := row*nc + col
				a, m, best, l12 := A[cell], 0, 0x7fffffff, 0
				up, dn, rt, lf := cell-nc, cell+nc, cell+1, cell-1

				if C[cell] == 1 {
					if v := g(B, up); v != 0 && v < best && g(A, up) == a {
						m, l12, best = 8, 8, v
					}

					if v := g(B, dn); v != 0 && v < best && g(A, dn) == a {
						m, l12, best = 4, 4, v
					}

					if v := g(B, rt); v != 0 && v < best {
						m = l12

						if g(A, rt) == a {
							m, best = 2, v
						}
					}

					if v := g(B, lf); v != 0 && v < best && g(A, lf) == a {
						m = 1
					}

					if m&8 != 0 {
						D[up] |= 4
					}

					if m&4 != 0 {
						D[dn] |= 8
					}

					if m&2 != 0 {
						D[rt] |= 1
					}

					if m&1 != 0 {
						D[lf] |= 2
					}

					if g(C, up) == 1 && g(A, up) != a {
						m |= 8
					}

					if g(C, dn) == 1 && g(A, dn) != a {
						m |= 4
					}

					if g(C, rt) == 1 && g(A, rt) != a {
						m |= 2
					}

					if g(C, lf) == 1 && g(A, lf) != a {
						m |= 1
					}
				}

				if b := B[cell]; b != 0 {
					for _, nb := range [4][2]int{{up, 8}, {dn, 4}, {rt, 2}, {lf, 1}} {
						if abs(b-g(B, nb[0])) == 1 {
							m |= nb[1]
						}
					}
				}

				D[cell] |= m
			}
		}

		// stage 5: extra branches at the candidate cells
	stage5:
		for row := 0; row < nr; row++ {
			for col := 0; col < nc; col++ {
				cell := row*nc + col
				m, a := D[cell], A[cell]
				rot := int(seed.Step() & 3)
				was0 := m == 0
				up, dn, rt, lf := cell-nc, cell+nc, cell+1, cell-1

				if C[cell] == 2 {
					nbs := [4][2]int{{up, 0x80}, {dn, 0x40}, {rt, 0x20}, {lf, 0x10}}

					for i := 0; i < 4; i++ {
						nb := nbs[(i+rot)&3]
						if g(A, nb[0]) == a && g(D, nb[0]) != 0 && g(D, nb[0]) < 0xf {
							m |= nb[1]
							break
						}
					}

					if m == 0 {
						restart = true
						break stage5
					}

					if was0 {
						found := false

						for i := 0; i < 4; i++ {
							nb := nbs[(i+rot)&3]
							if g(A, nb[0]) != a && g(C, nb[0]) == 2 {
								m |= nb[1]
								found = true

								break
							}
						}

						none := !found
						r2 := seed.Step()
						none = r2&1 != 0 && none
						r3 := seed.Step()

						for i := 0; none && i < 4; i++ {
							nb := nbs[(int(r3&3)+i)&3]
							if g(D, nb[0]) != 0 && g(D, nb[0]) < 0xf && m&nb[1] == 0 {
								m |= nb[1]
								none = false
							}
						}
					}

					if m&0x80 != 0 {
						D[up] |= 0x40
					}

					if m&0x40 != 0 {
						D[dn] |= 0x80
					}

					if m&0x20 != 0 {
						D[rt] |= 0x10
					}

					if m&0x10 != 0 {
						D[lf] |= 0x20
					}

					C[cell] = 0
				}

				D[cell] |= m
			}
		}
	}

	// stage 6: masks to LvlPrest Defs
	for i := range D {
		m := D[i]
		low := m & 0xf

		switch {
		case low == 0:
			t := 0

			if m > 0xf {
				if t = t296c[(m>>4)&0xf]; t == 0 {
					return nil, GameError{0x799}
				}
			}

			D[i] = t
		case m < 0x10:
			D[i] = low + 0x211
		default:
			t := low

			for col, bit := range [4]int{0x10, 0x20, 0x40, 0x80} {
				if m&bit != 0 {
					t = t29a0[t][col] // rows the exe never exposes read as 0
				}
			}

			if t == 0 {
				return nil, GameError{0x78c}
			}

			D[i] = t
		}
	}

	// stage 7: the 2x6 windows
	for _, nd := range s {
		for rr := 0; rr < ch; rr++ {
			for cc := 0; cc < cw; cc++ {
				v := D[(nd.r1+rr)*nc+nd.c0+cc]
				nd.info.Arr[rr*cw+cc] = v

				if v > 0x23e {
					nd.info.Count++
				}
			}
		}
	}

	// stage 8: bubble sort, largest y first
	for changed := true; changed; {
		changed = false

		for i := 1; i <= 2; i++ {
			if s[i-1].r.Y < s[i].r.Y {
				s[i-1], s[i] = s[i], s[i-1]
				changed = true
			}
		}
	}

	return s, nil
}

// PlaceAct3World is the Act 3 case of DRLG_InitActLevelLinks
// (DRLG_PlaceAct3Levels 0x67b7d0): town, jungle levels 76..78 (real drlg seed),
// the Kurast column 79..83 above Flayer Jungle and the adjacency vis links.
func PlaceAct3World(t d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty) (*World23, error) {
	base, _ := d2rand.DrlgBaseSeed(gameSeed)
	ex := d2drlg.DrawActExtras(gameSeed, 2)
	seed := ex.Seed
	w := &World23{Act: 3, Base: base, Flip: ex.Flip, Rects: map[int]Rect{}, Jungle: map[int]JungleInfo{}, links: linkSet{}}

	rec := func(id int) (d2drlg.LevelRec, error) {
		l, ok := t.Level(id)
		if !ok {
			return l, fmt.Errorf("drlgoutdoor: level %d missing from Levels.txt", id)
		}

		return l, nil
	}

	tl, err := rec(75)
	if err != nil {
		return nil, err
	}

	town := Rect{tl.OffsetX, tl.OffsetY, tl.SizeX[diff], tl.SizeY[diff]}
	w.Rects[75] = town

	j0, err := rec(76)
	if err != nil {
		return nil, err
	}

	nodes, err := placeJungle(&seed, town, j0.SizeX[diff], j0.SizeY[diff])
	if err != nil {
		return nil, err
	}

	for i, nd := range nodes {
		w.Rects[76+i] = nd.r
		w.Jungle[76+i] = nd.info
	}

	l78, yacc := w.Rects[78], 0

	for id := 79; id <= 83; id++ {
		l, err := rec(id)
		if err != nil {
			return nil, err
		}

		sx, sy := l.SizeX[diff], l.SizeY[diff]
		yacc -= sy
		w.Rects[id] = Rect{cdiv(l78.W, 2) - cdiv(sx, 2) + l78.X, l78.Y + yacc, sx, sy}
	}

	for i := 75; i <= 83; i++ {
		w.links.get(t, i)
	}

	for i := 75; i <= 83; i++ {
		for j := 75; j <= 83; j++ {
			if i != j && rectsAdjacent(w.Rects[i], w.Rects[j]) {
				if err := w.links.get(t, i).register(j); err != nil {
					return nil, err
				}
			}
		}
	}

	w.Seed = seed

	return w, nil
}

// Params derives the inputs of one level of the world: rectangle, the
// registered vis/warp arrays and the sorted neighbour list (drlg3.md 3).
func (w *World23) Params(t d2drlg.Levels, id int) (Params, error) {
	r, ok := w.Rects[id]
	if !ok {
		return Params{}, fmt.Errorf("drlgoutdoor: level %d not in the act %d layout", id, w.Act)
	}

	rec, _ := t.Level(id)
	p := Params{ID: id, Rect: r, Vis: rec.Vis, Warp: rec.Warp, BaseSeed: w.Base, Flip: w.Flip, Jungle: w.Jungle[id]}

	if k, ok := w.links[id]; ok {
		p.Vis, p.Warp = k.vis, k.warp

		for i := 0; i < 8; i++ {
			if k.vis[i] == 0 || k.warp[i] != -1 {
				continue
			}

			nb, ok := w.Rects[k.vis[i]]
			if !ok {
				continue
			}

			nrec, _ := t.Level(k.vis[i])
			p.Neighbors = insertSorted(p.Neighbors, Neighbor{Level: k.vis[i], Dir: adjacency(r, nb), F8: nrec.DrlgType == 2, Rect: nb})
		}
	}

	return p, nil
}

// ParamsAct23 is the one-call form for the engine: lay out the world of the
// level's act for a game seed and return the level's Params.
func ParamsAct23(t d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty, id int) (Params, error) {
	var (
		w   *World23
		err error
	)

	switch {
	case id >= 40 && id <= 46:
		w, err = PlaceAct2World(t, gameSeed, diff)
	case id >= 75 && id <= 83:
		w, err = PlaceAct3World(t, gameSeed, diff)
	default:
		return Params{}, fmt.Errorf("drlgoutdoor: level %d is not an Act 2/3 outdoor level", id)
	}

	if err != nil {
		return Params{}, err
	}

	return w.Params(t, id)
}
