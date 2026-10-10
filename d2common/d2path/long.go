package d2path

import "math"

// LongSearchNodes is the default budget of LongAStar.
const LongSearchNodes = 40000

// LongAStar is NOT part of the original path code. The original click-to-move
// search (FindPath / ShortAStar) only routes around obstacles for goals closer
// than about 18 subtiles and gives up on longer detours, which makes winding
// cave levels unplayable with a mouse in this engine. LongAStar is the same
// search with a large budget (expanded nodes), no limit on corner nodes and
// the stricter rule that a diagonal step needs both orthogonal neighbours to
// be free (no squeezing through wall corners). When the goal is unreachable it
// returns the route to the visited cell nearest to it, marked Partial.
func LongAStar(g Grid, mask uint16, from, to Point, budget int) (Route, bool) {
	if from == to {
		return Route{}, true
	}

	if budget <= 0 {
		budget = LongSearchNodes
	}

	s := newLongSearch()
	start := s.add(from, 0, EstimateDistance(to.X-from.X, to.Y-from.Y), -1, 0)
	s.cells.put(from, start)
	s.push(start)

	closest := start
	expanded := 0
	seq := int32(0)

	for len(s.open) > 0 && expanded < budget {
		cur := s.pop()
		cp := s.nodes[cur].p

		slot := s.cells.find(cp)
		if s.cells.closed[slot] {
			continue
		}

		s.cells.closed[slot] = true
		expanded++

		cn := &s.nodes[cur]
		if cn.h < s.nodes[closest].h || cn.h == s.nodes[closest].h && cn.g < s.nodes[closest].g {
			closest = cur
		}

		if cp == to {
			return finishLimit(s.chain(cur), false, math.MaxInt32)
		}

		cg := cn.g

		for _, nb := range neighbours {
			p := Point{cp.X + nb.dx, cp.Y + nb.dy}

			if Blocked(g, p.X, p.Y, mask) {
				continue
			}

			slot := s.cells.find(p)
			if slot >= 0 && s.cells.closed[slot] {
				continue
			}

			if nb.dx != 0 && nb.dy != 0 &&
				(Blocked(g, cp.X+nb.dx, cp.Y, mask) || Blocked(g, cp.X, cp.Y+nb.dy, mask)) {
				continue
			}

			ng := cg + nb.cost
			if slot >= 0 && s.nodes[s.cells.best[slot]].g <= ng {
				continue
			}

			seq++
			n := s.add(p, ng, EstimateDistance(to.X-p.X, to.Y-p.Y), cur, seq)
			s.cells.put(p, n)
			s.push(n)
		}
	}

	return finishLimit(s.chain(closest), true, math.MaxInt32)
}

// longNode is a search node; parent and the open list refer to nodes by index.
type longNode struct {
	p      Point
	g, h   int
	parent int32
	seq    int32
}

// longSearch holds the arrays of one LongAStar call. The map and pointer based version spent most
// of its time in map lookups, node allocations and interface calls of container/heap; the order in
// which nodes leave the open list is the same, as (f, h, seq) is a strict total order.
type longSearch struct {
	nodes []longNode
	open  []int32
	cells cellTable
}

func newLongSearch() *longSearch {
	return &longSearch{nodes: make([]longNode, 0, 1024), cells: newCellTable(1024)}
}

func (s *longSearch) add(p Point, g, h int, parent, seq int32) int32 {
	s.nodes = append(s.nodes, longNode{p: p, g: g, h: h, parent: parent, seq: seq})

	return int32(len(s.nodes) - 1)
}

// chain turns the parent links of node i into the pointer chain finishLimit walks (start has a nil parent).
func (s *longSearch) chain(i int32) *node {
	var n *node

	var path []int32

	for j := i; j >= 0; j = s.nodes[j].parent {
		path = append(path, j)
	}

	for k := len(path) - 1; k >= 0; k-- {
		ln := s.nodes[path[k]]
		n = &node{p: ln.p, g: ln.g, h: ln.h, parent: n}
	}

	return n
}

func (s *longSearch) less(a, b int32) bool {
	na, nb := &s.nodes[a], &s.nodes[b]

	fa, fb := na.g+na.h, nb.g+nb.h
	if fa != fb {
		return fa < fb
	}

	if na.h != nb.h {
		return na.h < nb.h
	}

	return na.seq < nb.seq
}

func (s *longSearch) push(i int32) {
	s.open = append(s.open, i)

	for c := len(s.open) - 1; c > 0; {
		p := (c - 1) / 2
		if !s.less(s.open[c], s.open[p]) {
			break
		}

		s.open[c], s.open[p] = s.open[p], s.open[c]
		c = p
	}
}

func (s *longSearch) pop() int32 {
	top := s.open[0]
	last := len(s.open) - 1
	s.open[0] = s.open[last]
	s.open = s.open[:last]

	for p := 0; ; {
		c := 2*p + 1
		if c >= last {
			break
		}

		if c+1 < last && s.less(s.open[c+1], s.open[c]) {
			c++
		}

		if !s.less(s.open[c], s.open[p]) {
			break
		}

		s.open[c], s.open[p] = s.open[p], s.open[c]
		p = c
	}

	return top
}

// cellTable maps a Point to its best node and closed flag: open addressing with linear probing.
// A slot is free while its best node is negative.
type cellTable struct {
	keys   []uint64
	best   []int32
	closed []bool
	used   int
}

func newCellTable(n int) cellTable {
	t := cellTable{keys: make([]uint64, n), best: make([]int32, n), closed: make([]bool, n)}
	for i := range t.best {
		t.best[i] = -1
	}

	return t
}

func pointKey(p Point) uint64 { return uint64(uint32(int32(p.X)))<<32 | uint64(uint32(int32(p.Y))) }

func (t *cellTable) hash(k uint64) int {
	return int((k * 0x9E3779B97F4A7C15) >> 33 & uint64(len(t.keys)-1))
}

// find returns the slot of p, or -1.
func (t *cellTable) find(p Point) int {
	k := pointKey(p)

	for i := t.hash(k); ; i = (i + 1) & (len(t.keys) - 1) {
		if t.best[i] < 0 {
			return -1
		}

		if t.keys[i] == k {
			return i
		}
	}
}

// put sets the best node of p (the closed flag of an existing slot is kept).
func (t *cellTable) put(p Point, node int32) {
	if t.used*2 >= len(t.keys) {
		t.grow()
	}

	k := pointKey(p)

	i := t.hash(k)
	for ; t.best[i] >= 0 && t.keys[i] != k; i = (i + 1) & (len(t.keys) - 1) {
	}

	if t.best[i] < 0 {
		t.keys[i] = k
		t.used++
	}

	t.best[i] = node
}

func (t *cellTable) grow() {
	old := *t
	*t = newCellTable(len(old.keys) * 2)
	t.used = old.used

	for i, k := range old.keys {
		if old.best[i] < 0 {
			continue
		}

		j := t.hash(k)
		for ; t.best[j] >= 0; j = (j + 1) & (len(t.keys) - 1) {
		}

		t.keys[j], t.best[j], t.closed[j] = k, old.best[i], old.closed[i]
	}
}
