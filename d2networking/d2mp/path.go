package d2mp

import (
	"container/heap"
	"math"
	"sort"
)

type pnode struct {
	idx  int
	f, g float64
}

type pq []pnode

func (q pq) Len() int            { return len(q) }
func (q pq) Less(i, j int) bool  { return q[i].f < q[j].f }
func (q pq) Swap(i, j int)       { q[i], q[j] = q[j], q[i] }
func (q *pq) Push(x interface{}) { *q = append(*q, x.(pnode)) }
func (q *pq) Pop() interface{} {
	o := *q
	n := len(o)
	x := o[n-1]
	*q = o[:n-1]

	return x
}

// reaches reports whether a straight walk from a to b stays on free cells.
func reaches(lv *LevelDef, ax, ay, bx, by float64) bool {
	x, y := March(lv, ax, ay, bx, by)

	return dist(x, y, bx, by) < 0.05
}

// FindPath returns waypoints (wire-aligned, last one is the goal) from a to b
// around blocked cells, or nil when there is no path. Both ends must be free.
func FindPath(lv *LevelDef, ax, ay, bx, by float64) [][2]float64 {
	if reaches(lv, ax, ay, bx, by) {
		return [][2]float64{{bx, by}}
	}

	sx, sy := int(ax), int(ay)
	gx, gy := int(bx), int(by)

	if !lv.Walkable(ax, ay) || !lv.Walkable(bx, by) {
		return nil
	}

	w := lv.W
	start, goal := sy*w+sx, gy*w+gx
	g := map[int]float64{start: 0}
	from := map[int]int{}
	open := &pq{}
	h := func(i int) float64 {
		dx, dy := math.Abs(float64(i%w-gx)), math.Abs(float64(i/w-gy))

		return math.Max(dx, dy) + (math.Sqrt2-1)*math.Min(dx, dy)
	}

	heap.Push(open, pnode{start, h(start), 0})

	for n := 0; open.Len() > 0 && n < 6000; n++ {
		cur := heap.Pop(open).(pnode)
		if cur.idx == goal {
			break
		}

		if cur.g > g[cur.idx] {
			continue
		}

		cx, cy := cur.idx%w, cur.idx/w

		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := cx+dx, cy+dy
				if (dx == 0 && dy == 0) || nx < 0 || ny < 0 || nx >= w || ny >= lv.H || lv.Blocked[ny*w+nx] {
					continue
				}

				step := 1.0

				if dx != 0 && dy != 0 {
					if lv.Blocked[cy*w+nx] || lv.Blocked[ny*w+cx] {
						continue // no corner cutting
					}

					step = math.Sqrt2
				}

				ni := ny*w + nx
				ng := cur.g + step

				if old, ok := g[ni]; !ok || ng < old {
					g[ni] = ng
					from[ni] = cur.idx
					heap.Push(open, pnode{ni, ng + h(ni), ng})
				}
			}
		}
	}

	if _, ok := from[goal]; !ok && goal != start {
		return nil
	}

	var cells []int

	for i := goal; i != start; i = from[i] {
		cells = append(cells, i)
	}

	// reverse into centres
	pts := make([][2]float64, 0, len(cells))

	for i := len(cells) - 1; i >= 0; i-- {
		c := cells[i]
		pts = append(pts, [2]float64{Snap(float64(c%w) + 0.5), Snap(float64(c/w) + 0.5)})
	}

	pts[len(pts)-1] = [2]float64{bx, by}

	// string pulling: skip waypoints that are visible from the current one
	out := make([][2]float64, 0, len(pts))
	cx, cy := ax, ay

	for i := 0; i < len(pts); {
		far := i

		for j := len(pts) - 1; j > i; j-- {
			if reaches(lv, cx, cy, pts[j][0], pts[j][1]) {
				far = j

				break
			}
		}

		out = append(out, pts[far])
		cx, cy = pts[far][0], pts[far][1]
		i = far + 1
	}

	return out
}

// walkTo is moveUnit with path finding: it walks around obstacles by chaining
// segments (the following legs are announced when the previous one ends).
func (s *Sim) walkTo(u *Unit, x, y, speed float64) {
	delete(s.paths, u.ID)

	lv := s.levelDef(u.Level)
	cx, cy := s.pos(u)
	x0, y0 := Snap(cx), Snap(cy)
	gx, gy := Snap(x), Snap(y)

	if reaches(lv, x0, y0, gx, gy) || !lv.Walkable(gx, gy) {
		s.moveUnit(u, x, y, speed)

		return
	}

	p := FindPath(lv, x0, y0, gx, gy)
	if len(p) == 0 {
		s.moveUnit(u, x, y, speed)

		return
	}

	s.moveUnit(u, p[0][0], p[0][1], speed)

	if len(p) > 1 {
		s.paths[u.ID] = &pathState{pts: p[1:], speed: speed}
	}
}

type pathState struct {
	pts   [][2]float64
	speed float64
}

// advancePaths starts the next leg of every unit whose current leg ends
// within this tick.
func (s *Sim) advancePaths() {
	ids := make([]uint32, 0, len(s.paths))
	for id := range s.paths {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		ps := s.paths[id]
		u := s.units[id]

		if u == nil || len(u.Segs) == 0 || u.Dead {
			delete(s.paths, id)

			continue
		}

		cur := u.Segs[len(u.Segs)-1]
		end := float64(cur.T0) + cur.Duration()

		if end > float64(s.now+s.cfg.TickMs) {
			continue
		}

		t0 := uint32(math.Ceil(end))
		next := ps.pts[0]
		ps.pts = ps.pts[1:]

		seg := Seg{X0: cur.X1, Y0: cur.Y1, X1: next[0], Y1: next[1], Speed: math.Round(ps.speed*100) / 100, T0: t0}
		u.Segs = []Seg{cur, seg}
		s.toLevel(u.Level, Event{Type: EvSeg, ID: id, Seg: seg}, 0)

		if len(ps.pts) == 0 {
			delete(s.paths, id)
		}
	}
}
