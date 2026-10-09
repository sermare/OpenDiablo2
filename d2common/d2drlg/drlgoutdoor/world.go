package drlgoutdoor

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// link is a per-level vis/warp record of the drlg sub-list (drlg+0x90).
type link struct{ vis, warp [8]int }

// register is DRLG_RegisterLevelVisLink (0x643db0).
func (k *link) register(b int) error {
	for i := range k.vis {
		if k.vis[i] == b {
			k.warp[i] = -1
			return nil
		}
	}

	for i := range k.vis {
		if k.vis[i] == 0 && k.warp[i] == -1 {
			k.vis[i], k.warp[i] = b, -1
			return nil
		}
	}

	return fmt.Errorf("drlgoutdoor: no free vis slot linking %d", b)
}

// act1Links lists (level, ref) of pass 3 of DRLG_PlaceOutdoorLevelsInWorld: each
// level with a reference is registered with it in both directions.
var act1Links = [][2]int{{3, 4}, {2, 3}, {1, 2}, {0x11, 3}, {7, 0x1a}, {6, 7}, {5, 6}}

type linkSet map[int]*link

func (s linkSet) get(t d2drlg.Levels, id int) *link {
	if k, ok := s[id]; ok {
		return k
	}

	rec, _ := t.Level(id)
	k := &link{vis: rec.Vis, warp: rec.Warp}
	s[id] = k

	return k
}

func adjacency(a, b Rect) int {
	if b.X < a.X {
		if a.X == b.X+b.W {
			return 0
		}
	} else if b.X == a.X+a.W {
		return 2
	}

	if b.Y < a.Y {
		if a.Y == b.Y+b.H {
			return 1
		}
	} else if b.Y == a.Y+a.H {
		return 3
	}

	return -1
}

// before is DRLG_CompareNeighborOrder (0x66e3c0): true if n goes before old.
func before(n, old Neighbor) bool {
	if old.Dir > n.Dir {
		return true
	}

	if old.Dir < n.Dir {
		return false
	}

	switch n.Dir {
	case 0:
		return old.Rect.Y > n.Rect.Y
	case 1:
		return old.Rect.X < n.Rect.X
	case 2:
		return old.Rect.Y < n.Rect.Y
	case 3:
		return old.Rect.X > n.Rect.X
	}

	return false
}

// insertSorted is DRLG_InsertNeighborSorted (0x66e440), including its quirk
// that with two or more entries the head is never compared.
func insertSorted(lst []Neighbor, n Neighbor) []Neighbor {
	insert := func(i int) []Neighbor {
		lst = append(lst, Neighbor{})
		copy(lst[i+1:], lst[i:])
		lst[i] = n

		return lst
	}

	switch len(lst) {
	case 0:
		return append(lst, n)
	case 1:
		if before(n, lst[0]) {
			return insert(0)
		}

		return append(lst, n)
	}

	for i := 1; i < len(lst); i++ {
		if before(n, lst[i]) {
			return insert(i)
		}
	}

	return append(lst, n)
}

// ParamsFromLayout derives the inputs of an outdoor level from the Act 1
// world layout: rectangle, exit flags, the registered vis/warp arrays and the
// neighbour list (drlg3.md section 3).
func ParamsFromLayout(t d2drlg.Levels, lay *drlgworld.Layout, id int, gameSeed uint32) (Params, error) {
	pl, ok := lay.Levels[id]
	if !ok {
		return Params{}, fmt.Errorf("drlgoutdoor: level %d not in the layout", id)
	}

	rects := map[int]Rect{}
	for lid, p := range lay.Levels {
		rects[lid] = Rect(p.Rect)
	}

	links := linkSet{}

	for _, l := range act1Links {
		if err := links.get(t, l[0]).register(l[1]); err != nil {
			return Params{}, err
		}

		if err := links.get(t, l[1]).register(l[0]); err != nil {
			return Params{}, err
		}
	}

	rec, _ := t.Level(id)
	p := Params{ID: id, Rect: rects[id], OdFlags: pl.Flags, Vis: rec.Vis, Warp: rec.Warp, Town: rects[1]}
	p.BaseSeed, _ = d2rand.DrlgBaseSeed(gameSeed)

	if k, ok := links[id]; ok {
		p.Vis, p.Warp = k.vis, k.warp

		for i := 0; i < 8; i++ {
			if k.vis[i] == 0 || k.warp[i] != -1 {
				continue
			}

			nb, ok := rects[k.vis[i]]
			if !ok {
				continue
			}

			nrec, _ := t.Level(k.vis[i])
			p.Neighbors = insertSorted(p.Neighbors, Neighbor{Level: k.vis[i], Dir: adjacency(rects[id], nb), F8: nrec.DrlgType == 2, Rect: nb})
		}
	}

	return p, nil
}
