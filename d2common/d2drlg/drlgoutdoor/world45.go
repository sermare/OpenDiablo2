package drlgoutdoor

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// edgeAdjacent is DRLG_AreRectsEdgeAdjacent (0x66e590) with margin -1: the
// rectangles share an edge segment.
func edgeAdjacent(a, b Rect) bool {
	gap := func(a0, aw, b0, bw int) int {
		if a0 <= b0 {
			return b0 - aw - a0
		}

		return a0 - bw - b0
	}

	dx, dy := gap(a.X, a.W, b.X, b.W), gap(a.Y, a.H, b.Y, b.H)

	return (dx == 0 && dy <= -1) || (dy == 0 && dx <= -1)
}

// act4Links are the (level, reference) pairs of pass 3 of the Act 4 world
// search, in table order (exe tables 0x6f21b0/0x6f22a0).
var act4Links = [][2]int{{104, 103}, {105, 104}, {106, 105}}

// ParamsFromLayout45 derives the inputs of an Act 4 (act = 3) or Act 5
// (act = 4) level from the world layout produced by drlgworld.GenerateAct4 /
// GenerateAct5: rectangle, od.flags, the registered vis/warp arrays and the
// neighbour list as it exists when the level is generated. Levels that no
// cluster places (120, 121, 124, 131, 132) keep their Levels.txt rectangle.
//
// Act 4 builds all neighbour lists after the vis registration. Act 5 builds
// them between the explicit link ranges (111/112 first, then 110/111 and
// 109/110), so 110 gets no list and 111 never learns about 110 in its list
// (drlg-act45-outdoor.md section 2).
func ParamsFromLayout45(t d2drlg.Levels, lay *drlgworld.Layout, act, id int, gameSeed uint32, diff d2drlg.Difficulty) (Params, error) {
	rec, ok := t.Level(id)
	if !ok {
		return Params{}, fmt.Errorf("drlgoutdoor: level %d unknown", id)
	}

	rects := map[int]Rect{}
	for lid, p := range lay.Levels {
		rects[lid] = Rect(p.Rect)
	}

	p := Params{ID: id, Vis: rec.Vis, Warp: rec.Warp}
	p.BaseSeed, _ = d2rand.DrlgBaseSeed(gameSeed)

	if pl, ok := lay.Levels[id]; ok {
		p.Rect, p.OdFlags = rects[id], pl.Flags
	} else {
		p.Rect = Rect{rec.OffsetX, rec.OffsetY, rec.SizeX[diff], rec.SizeY[diff]}
	}

	links := linkSet{}
	reg := func(a, b int) error { return links.get(t, a).register(b) }
	lists := map[int][]Neighbor{}

	buildLists := func(hi int) {
		for lid := 1; lid <= hi; lid++ {
			r, placed := rects[lid]
			l, known := t.Level(lid)
			k, reged := links[lid]

			if !placed || !known || l.DrlgType != 3 || !reged {
				continue
			}

			var out []Neighbor

			for i := 0; i < 8; i++ {
				if k.vis[i] == 0 || k.warp[i] != -1 {
					continue
				}

				nb, ok := rects[k.vis[i]]
				if !ok {
					continue
				}

				nrec, _ := t.Level(k.vis[i])
				out = insertSorted(out, Neighbor{Level: k.vis[i], Dir: adjacency(r, nb), F8: nrec.DrlgType == 2, Rect: nb})
			}

			lists[lid] = out
		}
	}

	link := func(lo, hi int) error {
		for i := lo; i <= hi; i++ {
			links.get(t, i)

			for j := lo; j <= hi; j++ {
				ri, oki := rects[i]
				rj, okj := rects[j]

				if i != j && oki && okj && edgeAdjacent(ri, rj) {
					if err := reg(i, j); err != nil {
						return err
					}
				}
			}
		}

		return nil
	}

	var err error

	switch act {
	case 3:
		for _, l := range act4Links {
			if err = reg(l[0], l[1]); err == nil {
				err = reg(l[1], l[0])
			}

			if err != nil {
				return Params{}, err
			}
		}

		buildLists(0x6a)
	case 4:
		if err = link(0x6f, 0x70); err == nil {
			buildLists(0x70)

			if err = link(0x6e, 0x6f); err == nil {
				err = link(0x6d, 0x6e)
			}
		}

		if err != nil {
			return Params{}, err
		}
	default:
		return Params{}, fmt.Errorf("drlgoutdoor: act index %d is not 3 or 4", act)
	}

	if k, ok := links[id]; ok {
		p.Vis, p.Warp = k.vis, k.warp
	}

	p.Neighbors = lists[id]

	return p, nil
}
