package drlgworld

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Acts 4 and 5 of DRLG_InitActLevelLinks (0x67b8f0), cases 3 and 4
// (drlg-act45-outdoor.md section 2, verified against the emulator for 50
// seeds x 2 acts). Both acts run their clusters from a fresh copy of the DRLG
// seed (S0 = Init(gameSeed); Step()), like Act 1.

// Cluster tables (exe 0x6f21b0, 0x6f22a0, 0x6f2390, 0x6f2480, 0x6f20c0).
var (
	a4cluster1 = []entry{{anchor, 103, -1}, {outerSteppes, 104, 0}, {pinwheel, 105, 1}, {pinwheel, 106, 2}}
	a4cluster2 = []entry{{anchor, 108, -1}}
	a5cluster1 = []entry{{anchor, 109, -1}, {anchor, 110, 0}, {frigidHighland, 111, 1}, {arreatPlateau, 112, 2}}
	a5cluster2 = []entry{{frozenTundra, 117, -1}}
	a5cluster3 = []entry{{anchor, 134, -1}, {anchor, 136, -1}}
)

// arreatOffsets is the table at 0x6f352c: (dx, dy) of Arreat Plateau relative
// to Frigid Highlands, indexed d + 2*dirOfFrigid.
var arreatOffsets = [4][2]int{{0, -160}, {-96, -64}, {-64, -96}, {-160, 0}}

// place45 runs the placement function of an Act 4/5 entry.
func (c *ctx) place45(i int, lv d2drlg.Levels) bool {
	e := c.tbl[i]
	r := &c.rect[i]

	var p Rect
	if e.ref >= 0 {
		p = c.rect[e.ref]
	}

	switch e.kind {
	case outerSteppes: // 0x679940, direction fixed to 3 (east of the town)
		c.firs[i], c.dir[i] = 3, 3
		odd := c.seed.Step()&1 != 0
		r.X = p.X + p.W

		if odd {
			r.Y = p.H - r.H + p.Y + 8
			c.glob = 0x800000
		} else {
			r.Y = p.Y - 8
			c.glob = 0x400000
		}
	case frigidHighland: // 0x680840
		bit := int(c.seed.Step() & 1)
		c.dir[i], c.firs[i] = bit, bit
		r.W, r.H = orient(bit)
		r.X = p.X - r.W
		r.Y = p.H - r.H + p.Y - 16
	case arreatPlateau: // 0x680990, two options: bit, then the other
		if c.firs[i] == -1 {
			bit := int(c.seed.Step() & 1)
			c.firs[i], c.dir[i] = bit, bit
		} else {
			nd := 0
			if c.dir[i] == 0 {
				nd = 1
			}

			if nd == c.firs[i] {
				return false
			}

			c.dir[i] = nd
		}

		d := c.dir[i]
		r.W, r.H = orient(d)
		k := d + 2*c.dir[e.ref]
		r.X, r.Y = p.X+arreatOffsets[k][0], p.Y+arreatOffsets[k][1]
	case frozenTundra: // 0x6808f0, position from Levels.txt
		bit := int(c.seed.Step() & 1)
		c.dir[i], c.firs[i] = bit, bit
		r.W, r.H = orient(bit)
		l, _ := lv.Level(e.level)
		r.X, r.Y = l.OffsetX, l.OffsetY
	}

	return true
}

// orient is the size of the Act 5 outdoor levels: bit 0 tall (64x160), 1 wide.
func orient(bit int) (w, h int) {
	if bit == 0 {
		return 64, 160
	}

	return 160, 64
}

// noOverlap is the validator of 0x679de0 / 0x679e50 / 0x679d70: no overlap
// with any earlier entry except the reference.
func (c *ctx) noOverlap(i int) bool {
	for j := 0; j < i; j++ {
		if j != c.tbl[i].ref && overlap(c.rect[i], c.rect[j]) {
			return false
		}
	}

	return true
}

// GenerateAct4 places the Act 4 levels 103-106 and 108 and returns the layout.
// Level 104 carries od.flags 0x400000 or 0x800000 in Placement.Flags.
func GenerateAct4(lv d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty) (*Layout, error) {
	return generate45(lv, gameSeed, diff, [][]entry{a4cluster1, a4cluster2}, []bool{true, true}, 104)
}

// GenerateAct5 places the Act 5 levels 109-112, 117, 134 and 136.
func GenerateAct5(lv d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty) (*Layout, error) {
	return generate45(lv, gameSeed, diff, [][]entry{a5cluster1, a5cluster2, a5cluster3}, []bool{false, false, true}, 0)
}

func generate45(lv d2drlg.Levels, gameSeed uint32, diff d2drlg.Difficulty, tabs [][]entry, validated []bool, flagLevel int) (*Layout, error) {
	_, drlg := d2rand.DrlgBaseSeed(gameSeed)
	out := &Layout{Levels: map[int]*Placement{}, BarracksExitSide: -1}

	for ti, tbl := range tabs {
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

		validate := func(int) bool { return true }
		if validated[ti] {
			validate = c.noOverlap
		}

		if err := c.search(lv, validate); err != nil {
			return nil, err
		}

		for i, e := range tbl {
			pl := &Placement{ID: e.level, Rect: c.rect[i], Dir: c.dir[i], Flip: c.flip[i]}
			if e.level == flagLevel {
				pl.Flags = c.glob
			}

			out.Levels[e.level] = pl
		}
	}

	out.DrlgSeed = *drlg

	return out, nil
}
