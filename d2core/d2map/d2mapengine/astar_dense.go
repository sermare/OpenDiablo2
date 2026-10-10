package d2mapengine

import "sync"

// The dense A*: the same search as findPathSparse (same expansion order, same
// tie breaks, same route) with flat tables over the search box instead of
// maps, a node slab instead of one allocation per node, and the blocked test
// of every cell asked once. Clicking on a building (an unreachable goal) makes
// the search expand its whole box (up to astarMaxNodes cells); with maps that
// cost 24 ms and 15 MB of garbage per click on the main thread.

const (
	walkUnknown int8 = iota
	walkOpen
	walkShut
)

type denseCell struct {
	node   int32 // slab index of the newest node of this cell + 1, 0 = none
	parent int32 // cell index of the predecessor
	walk   int8
	closed bool
}

type denseNode struct {
	cell int32
	g, f float64
}

type denseSearch struct {
	cells []denseCell
	nodes []denseNode
	heap  []denseItem // ordered by f (container/heap's algorithm)
}

type denseItem struct {
	f    float64
	node int32 // slab index
}

var densePool = sync.Pool{New: func() interface{} { return &denseSearch{} }}

func (s *denseSearch) less(i, j int) bool { return s.heap[i].f < s.heap[j].f }

func (s *denseSearch) push(n denseItem) {
	s.heap = append(s.heap, n)

	for j := len(s.heap) - 1; ; {
		i := (j - 1) / 2 // parent
		if i == j || !s.less(j, i) {
			break
		}

		s.heap[i], s.heap[j] = s.heap[j], s.heap[i]
		j = i
	}
}

func (s *denseSearch) pop() int32 {
	n := len(s.heap) - 1
	s.heap[0], s.heap[n] = s.heap[n], s.heap[0]

	for i := 0; ; {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 {
			break
		}

		j := j1
		if j2 := j1 + 1; j2 < n && s.less(j2, j1) {
			j = j2
		}

		if !s.less(j, i) {
			break
		}

		s.heap[i], s.heap[j] = s.heap[j], s.heap[i]
		i = j
	}

	top := s.heap[n].node
	s.heap = s.heap[:n]

	return top
}

func findPathDense(blocked func(x, y int) bool, start, goal pt) (path []pt, reached bool) {
	lox, hix := minInt(start.x, goal.x)-astarMargin, maxInt(start.x, goal.x)+astarMargin
	loy, hiy := minInt(start.y, goal.y)-astarMargin, maxInt(start.y, goal.y)+astarMargin
	w, h := hix-lox+1, hiy-loy+1

	s := densePool.Get().(*denseSearch)
	defer densePool.Put(s)

	if cap(s.cells) < w*h {
		s.cells = make([]denseCell, w*h)
	} else {
		s.cells = s.cells[:w*h]

		for i := range s.cells {
			s.cells[i] = denseCell{}
		}
	}

	s.nodes, s.heap = s.nodes[:0], s.heap[:0]
	cells := s.cells

	at := func(p pt) int { return (p.y-loy)*w + (p.x - lox) }
	pointOf := func(c int32) pt { return pt{lox + int(c)%w, loy + int(c)/w} }

	// walk is the memoised "inside the box and not blocked"
	walk := func(x, y int) bool {
		if x < lox || x > hix || y < loy || y > hiy {
			return false
		}

		c := &cells[(y-loy)*w+(x-lox)]
		if c.walk == walkUnknown {
			c.walk = walkOpen
			if blocked(x, y) {
				c.walk = walkShut
			}
		}

		return c.walk == walkOpen
	}

	add := func(c int32, g, f float64) {
		s.nodes = append(s.nodes, denseNode{cell: c, g: g, f: f})
		cells[c].node = int32(len(s.nodes))
		s.push(denseItem{f: f, node: int32(len(s.nodes) - 1)})
	}

	sc, gc := int32(at(start)), int32(at(goal))
	add(sc, 0, octile(start, goal))

	best, bestH := start, octile(start, goal)

	for expanded := 0; len(s.heap) > 0 && expanded < astarMaxNodes; expanded++ {
		cur := s.nodes[s.pop()]
		if cells[cur.cell].closed {
			continue
		}

		cells[cur.cell].closed = true
		cp := pointOf(cur.cell)

		if hh := octile(cp, goal); hh < bestH {
			best, bestH = cp, hh
		}

		if cur.cell == gc {
			best, reached = goal, true
			break
		}

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}

				np := pt{cp.x + dx, cp.y + dy}
				if np.x < lox || np.x > hix || np.y < loy || np.y > hiy {
					continue
				}

				nc := int32(at(np))
				if cells[nc].closed || !walk(np.x, np.y) {
					continue
				}

				cost := astarStraight

				if dx != 0 && dy != 0 {
					// no cutting corners
					if !walk(cp.x+dx, cp.y) || !walk(cp.x, cp.y+dy) {
						continue
					}

					cost = astarDiagonal
				}

				g := cur.g + cost
				if old := cells[nc].node; old != 0 && s.nodes[old-1].g <= g+astarGoalSlack {
					continue
				}

				cells[nc].parent = cur.cell
				add(nc, g, g+octile(np, goal))
			}
		}
	}

	// walk back from best to start
	var rev []pt

	for p := best; p != start; p = pointOf(cells[at(p)].parent) {
		rev = append(rev, p)
	}

	for i := len(rev) - 1; i >= 0; i-- {
		path = append(path, rev[i])
	}

	return smooth(blocked, start, path), reached
}
