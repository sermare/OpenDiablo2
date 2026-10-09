// Package drlgworld ports the Act 1 world layout: DRLG_PlaceOutdoorLevelsInWorld
// (0x679ff0), the randomised depth-first search that decides where the Act 1
// wilderness levels, the Rogue Encampment, the Burial Grounds and the
// Monastery cluster sit relative to each other (drlg2.md B.1).
//
// Verified in the notes: the search loop, the five placement functions, the
// cluster tables and the compat table, the post-pass writes (town file index,
// level 27 exit side) and the exit flag table. UNVERIFIED (kept as noted):
// which rectangle ValidateCluster2 inflates, and whether each cluster call
// copies the DRLG seed afresh (assumed: yes, both clusters start from the same
// copy, because the real seed is only touched in pass 3).
package drlgworld

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Rect is a level rectangle in tiles.
type Rect struct{ X, Y, W, H int }

// Placement is where one level ended up.
type Placement struct {
	ID    int
	Rect  Rect
	Dir   int // search direction state (-1 for anchors)
	Flip  int
	Flags int // outdoor "od.flags" exit bits (0x4/0x8/0x10 river edge, 0x80-0x400 town transitions)
}

// Layout is the result of the Act 1 world search.
type Layout struct {
	Levels map[int]*Placement
	// TownFile is the Rogue Encampment preset file index (0 TownN1, 1 TownE1,
	// 2 TownS1, 3 TownW1), forced to the search direction (verified).
	TownFile int
	// BarracksExitSide is level 27's exit side field (2 - (lo&1) or ~lo&1),
	// -1 when the Black Marsh direction did not set it.
	BarracksExitSide int
	// DrlgSeed is the real DRLG seed after the pass-3 draws.
	DrlgSeed d2rand.Seed
}

type placeKind int

const (
	anchor placeKind = iota
	pinwheel
	below
	town
	bloodMoor
	outerSteppes   // Act 4: Outer Steppes next to the Fortress (world45.go)
	frigidHighland // Act 5: Frigid Highlands relative to Bloody Foothills
	arreatPlateau  // Act 5: Arreat Plateau relative to Frigid Highlands
	frozenTundra   // Act 5: Frozen Tundra at its Levels.txt offset
)

type entry struct {
	kind  placeKind
	level int
	ref   int
}

// Cluster tables (verified, exe 0x6f1d00 and 0x6f1df0).
var (
	cluster1 = []entry{{anchor, 4, -1}, {pinwheel, 3, 0}, {bloodMoor, 2, 1}, {town, 1, 2}, {pinwheel, 0x11, 1}}
	cluster2 = []entry{{anchor, 0x27, -1}, {anchor, 0x1a, -1}, {below, 7, 1}, {pinwheel, 6, 2}, {pinwheel, 5, 3}}
)

// compat is the town compatibility table 0x6f2708 (verified dump), indexed
// dir_i + 4*(flip_i + 2*(dir_ref + 4*flip_ref)).
var compat = [64]int{
	1, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 1, 0, 1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1,
	0, 1, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 0, 0, 1, 1,
}

type exitRow struct{ lvl, notA, notB, dirI, dirNext, bit int }

// Exit flag conditions (verified dump of the first rows of 0x6f2808).
var exitRows = []exitRow{
	{0, 2, 3, 1, 0, 0x4}, {0, 2, 3, 2, 3, 0x4}, {0, 3, 17, 2, 1, 0x8}, {0, 3, 17, 3, 0, 0x8},
	{0, 3, 17, 1, 1, 0x10}, {0, 3, 17, 3, 3, 0x10},
	{2, 0, 0, 0, 0, 0x8}, {2, 0, 0, 2, 2, 0x8}, {2, 0, 0, 3, 0, 0x8}, {2, 0, 0, 3, 2, 0x8},
	{2, 0, 0, 0, 1, 0x400}, {2, 0, 0, 1, 1, 0x400}, {2, 0, 0, 2, 1, 0x200}, {2, 0, 0, 2, 2, 0x80}, {2, 0, 0, 3, 2, 0x100},
}

type ctx struct {
	seed      d2rand.Seed
	tbl       []entry
	rect      []Rect
	dir, firs []int
	flip, ff  []int
	// glob is the exe's process-global DAT_009656ac written by the Outer
	// Steppes placer (od.flags 0x400000 / 0x800000 of level 104), here per world.
	glob int
}

func (c *ctx) reset(i int) { c.dir[i], c.firs[i], c.flip[i], c.ff[i] = -1, -1, -1, -1 }

func overlap(a, b Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// place runs the placement function of entry i; false = options exhausted.
func (c *ctx) place(i int, lv d2drlg.Levels) bool {
	e := c.tbl[i]
	r := &c.rect[i]

	var p Rect
	if e.ref >= 0 {
		p = c.rect[e.ref]
	}

	if e.kind >= outerSteppes {
		return c.place45(i, lv)
	}

	switch e.kind {
	case anchor:
		l, _ := lv.Level(e.level)
		r.X, r.Y = l.OffsetX, l.OffsetY

		return true
	case below:
		c.dir[i], c.flip[i] = 0, 0
		r.X, r.Y = p.X, p.Y+p.H

		return true
	case pinwheel:
		if c.firs[i] == -1 {
			c.dir[i] = int(c.seed.Step() & 3)
			c.firs[i] = c.dir[i]
		} else {
			c.dir[i] = (c.dir[i] + 1) & 3
			if c.dir[i] == c.firs[i] {
				return false
			}
		}

		switch c.dir[i] {
		case 0:
			r.X, r.Y = p.X-16, p.Y+p.H
		case 1:
			r.X, r.Y = p.X-r.W, p.Y-16
		case 2:
			r.X, r.Y = p.X+p.W-r.W+16, p.Y-r.H
		case 3:
			r.X, r.Y = p.X+p.W, p.Y+p.H-r.H+16
		}

		return true
	}

	// town and Blood Moor share the (dir, flip) state machine
	if c.firs[i] == -1 {
		c.dir[i] = int(c.seed.Step() & 3)
		c.firs[i] = c.dir[i]
		c.flip[i] = int(c.seed.Step() & 1)
		c.ff[i] = c.flip[i]
	} else {
		nd := (c.dir[i] + c.flip[i]) & 3
		nf := (c.flip[i] + 1) & 1

		if nd == c.firs[i] && nf == c.ff[i] {
			return false
		}

		c.dir[i], c.flip[i] = nd, nf
	}

	d, f := c.dir[i], c.flip[i]

	if e.kind == bloodMoor {
		if d&1 == 1 {
			r.W, r.H = 96, 56
		} else {
			r.W, r.H = 56, 96
		}

		w, h := r.W, r.H
		if f == 1 {
			switch d {
			case 0:
				r.X, r.Y = p.X-16, p.Y+p.H
			case 1:
				r.X, r.Y = p.X-w, p.Y-16
			case 2:
				r.X, r.Y = p.X+p.W-w+16, p.Y-h
			case 3:
				r.X, r.Y = p.X+p.W, p.Y+p.H-h+16
			}
		} else {
			switch d {
			case 0:
				r.X, r.Y = p.X+p.W-w+16, p.Y+p.H
			case 1:
				r.X, r.Y = p.X-w, p.Y+p.H-h+16
			case 2:
				r.X, r.Y = p.X-16, p.Y-h
			case 3:
				r.X, r.Y = p.X+p.W, p.Y-16
			}
		}

		return true
	}

	w, h := r.W, r.H
	if f == 1 {
		switch d {
		case 0:
			r.X, r.Y = p.X, p.Y+p.H
		case 1:
			r.X, r.Y = p.X-w, p.Y+8
		case 2:
			r.X, r.Y = p.X+p.W-w, p.Y-h
		case 3:
			r.X, r.Y = p.X+p.W, p.Y+p.H-h-8
		}
	} else {
		switch d {
		case 0:
			r.X, r.Y = p.X+p.W-w, p.Y+p.H
		case 1:
			r.X, r.Y = p.X-w, p.Y+p.H-h-8
		case 2:
			r.X, r.Y = p.X, p.Y-h
		case 3:
			r.X, r.Y = p.X+p.W, p.Y+8
		}
	}

	return true
}

func (c *ctx) validate1(i int) bool {
	e := c.tbl[i]

	for j := 0; j < i; j++ {
		if j != e.ref && overlap(c.rect[i], c.rect[j]) {
			return false
		}
	}

	switch e.level {
	case 1:
		r := e.ref

		return compat[c.dir[i]+4*(c.flip[i]+2*(c.dir[r]+4*c.flip[r]))] != 0
	case 0x11:
		for j := range c.tbl {
			if j != i && c.tbl[j].ref == e.ref && c.dir[j] == c.dir[i] {
				return false
			}
		}
	}

	return true
}

func (c *ctx) validate2(i int) bool {
	e := c.tbl[i]

	for j := 0; j < i; j++ {
		if j != e.ref && overlap(c.rect[i], c.rect[j]) {
			return false
		}
	}

	// UNVERIFIED which rect is inflated (register args lost in the notes):
	// entry 0 (Moo Moo Farm) grown by 200 upwards, as the decompile suggests.
	g := c.rect[0]
	g.Y -= 200
	g.H += 200

	return i == 0 || !overlap(g, c.rect[i])
}

var errSearch = errors.New("drlgworld: world search did not converge")

// search runs pass 2 (verified).
func (c *ctx) search(lv d2drlg.Levels, validate func(int) bool) error {
	i := 0

	for guard := 0; i < len(c.tbl); guard++ {
		if guard > 100000 {
			return errSearch
		}

		if !c.place(i, lv) {
			c.reset(i)

			if i--; i < 0 {
				return errSearch
			}

			continue
		}

		if validate(i) {
			i++
		}
	}

	return nil
}

// Generate runs the Act 1 world placement for a game seed.
func Generate(lv d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty) (*Layout, error) {
	_, drlg := d2rand.DrlgBaseSeed(gameSeed)
	out := &Layout{Levels: map[int]*Placement{}, BarracksExitSide: -1}

	for ci, tbl := range [][]entry{cluster1, cluster2} {
		c := &ctx{seed: *drlg, tbl: tbl}
		n := len(tbl)
		c.rect, c.dir, c.firs, c.flip, c.ff = make([]Rect, n), make([]int, n), make([]int, n), make([]int, n), make([]int, n)

		for i, e := range tbl {
			l, ok := lv.Level(e.level)
			if !ok {
				return nil, fmt.Errorf("drlgworld: level %d missing from Levels.txt", e.level)
			}

			c.rect[i].W, c.rect[i].H = l.SizeX[diff], l.SizeY[diff]
			c.reset(i)
		}

		v := c.validate1
		if ci == 1 {
			v = c.validate2
		}

		if err := c.search(lv, v); err != nil {
			return nil, err
		}

		// pass 3
		for i, e := range tbl {
			pl := &Placement{ID: e.level, Rect: c.rect[i], Dir: c.dir[i], Flip: c.flip[i]}
			out.Levels[e.level] = pl

			switch {
			case e.level == 1:
				out.TownFile = c.dir[i]
			case e.level == 6 && c.dir[i] == 1:
				out.BarracksExitSide = 2 - int(drlg.Step()&1)
			case e.level == 6 && c.dir[i] == 3:
				out.BarracksExitSide = int(^drlg.Step() & 1)
			}

			l, _ := lv.Level(e.level)
			if l.DrlgType != 3 {
				continue
			}

			next := -1
			if i+1 < n {
				next = c.dir[i+1]
			}

			for _, r := range exitRows {
				if (r.lvl == e.level || r.lvl == 0) && e.level != r.notA && e.level != r.notB && c.dir[i] == r.dirI && next == r.dirNext {
					pl.Flags |= r.bit
				}
			}
		}
	}

	out.DrlgSeed = *drlg

	return out, nil
}
