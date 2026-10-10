package d2mapengine

import (
	"container/heap"
	"math"
)

// A bounded A* over sub-tiles. The original game plans with a purpose-built
// pathing system (missiles-pathing.md); the engine only had a straight line
// that stops at the first obstacle, so heroes got stuck behind any wall. This
// planner is used when the straight line is blocked. It finds a route to the
// goal sub-tile or, if the goal is unreachable or blocked, to the reachable
// sub-tile closest to it.

const (
	astarMargin    = 80    // sub-tiles searched around the start/goal box
	astarMaxNodes  = 60000 // expansion cap
	astarDiagonal  = 1.41421356
	astarStraight  = 1.0
	astarGoalSlack = 1e-9
	astarMaxCells  = 4 << 20 // largest search box (sub-tiles) given dense tables
)

// pt is a sub-tile coordinate.
type pt struct{ x, y int }

type astarNode struct {
	p    pt
	g, f float64
	idx  int
}

type nodeHeap []*astarNode

func (h nodeHeap) Len() int            { return len(h) }
func (h nodeHeap) Less(i, j int) bool  { return h[i].f < h[j].f }
func (h nodeHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i]; h[i].idx = i; h[j].idx = j }
func (h *nodeHeap) Push(x interface{}) { n := x.(*astarNode); n.idx = len(*h); *h = append(*h, n) }
func (h *nodeHeap) Pop() interface{} {
	old := *h
	n := old[len(old)-1]
	*h = old[:len(old)-1]

	return n
}

func octile(a, b pt) float64 {
	dx, dy := math.Abs(float64(a.x-b.x)), math.Abs(float64(a.y-b.y))

	return astarStraight*(dx+dy) + (astarDiagonal-2*astarStraight)*math.Min(dx, dy)
}

// lineClear reports whether the straight sub-tile line from a to b crosses no
// blocked sub-tile (all cells the line passes through are tested).
func lineClear(blocked func(x, y int) bool, a, b pt) bool {
	dx, dy := b.x-a.x, b.y-a.y
	n := int(math.Max(math.Abs(float64(dx)), math.Abs(float64(dy))))

	if n == 0 {
		return !blocked(a.x, a.y)
	}

	// step in half-cells so a diagonal cannot squeeze between two blocked cells
	steps := n * 2

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Floor(float64(a.x) + 0.5 + float64(dx)*t))
		y := int(math.Floor(float64(a.y) + 0.5 + float64(dy)*t))

		if blocked(x, y) {
			return false
		}
	}

	return true
}

// findPath plans a route from start to goal over sub-tiles for which blocked
// is false. It returns the waypoints after the start (smoothed), whether the
// goal cell itself was reached, and whether any route was found. If the goal
// cannot be reached the route leads to the reachable cell nearest to it.
func findPath(blocked func(x, y int) bool, start, goal pt) (path []pt, reached bool) {
	w := maxInt(start.x, goal.x) - minInt(start.x, goal.x) + 2*astarMargin + 1
	h := maxInt(start.y, goal.y) - minInt(start.y, goal.y) + 2*astarMargin + 1

	if w*h > astarMaxCells {
		return findPathSparse(blocked, start, goal)
	}

	return findPathDense(blocked, start, goal)
}

// findPathSparse is the map based search: the reference for findPathDense (same
// results, see TestFindPathDenseMatchesSparse) and the fallback for searches
// whose bounding box is too large for a dense table.
func findPathSparse(blocked func(x, y int) bool, start, goal pt) (path []pt, reached bool) {
	lox, hix := minInt(start.x, goal.x)-astarMargin, maxInt(start.x, goal.x)+astarMargin
	loy, hiy := minInt(start.y, goal.y)-astarMargin, maxInt(start.y, goal.y)+astarMargin

	walk := func(x, y int) bool {
		return x >= lox && x <= hix && y >= loy && y <= hiy && !blocked(x, y)
	}

	nodes := map[pt]*astarNode{}
	parent := map[pt]pt{}
	closed := map[pt]bool{}
	open := &nodeHeap{}

	s := &astarNode{p: start, g: 0, f: octile(start, goal)}
	nodes[start] = s

	heap.Push(open, s)

	best, bestH := start, octile(start, goal)

	for expanded := 0; open.Len() > 0 && expanded < astarMaxNodes; expanded++ {
		cur := heap.Pop(open).(*astarNode)
		if closed[cur.p] {
			continue
		}

		closed[cur.p] = true

		if h := octile(cur.p, goal); h < bestH {
			best, bestH = cur.p, h
		}

		if cur.p == goal {
			best, reached = goal, true
			break
		}

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}

				np := pt{cur.p.x + dx, cur.p.y + dy}
				if closed[np] || !walk(np.x, np.y) {
					continue
				}

				cost := astarStraight

				if dx != 0 && dy != 0 {
					// no cutting corners
					if !walk(cur.p.x+dx, cur.p.y) || !walk(cur.p.x, cur.p.y+dy) {
						continue
					}

					cost = astarDiagonal
				}

				g := cur.g + cost
				if old, ok := nodes[np]; ok && old.g <= g+astarGoalSlack {
					continue
				}

				n := &astarNode{p: np, g: g, f: g + octile(np, goal)}
				nodes[np] = n
				parent[np] = cur.p

				heap.Push(open, n)
			}
		}
	}

	// walk back from best to start
	var rev []pt

	for p := best; p != start; p = parent[p] {
		rev = append(rev, p)
	}

	for i := len(rev) - 1; i >= 0; i-- {
		path = append(path, rev[i])
	}

	return smooth(blocked, start, path), reached
}

// smooth drops intermediate points that a straight line can skip.
func smooth(blocked func(x, y int) bool, start pt, path []pt) []pt {
	var out []pt

	anchor, i := start, 0

	for i < len(path) {
		far := i
		for j := len(path) - 1; j > i; j-- {
			if lineClear(blocked, anchor, path[j]) {
				far = j
				break
			}
		}

		out = append(out, path[far])
		anchor, i = path[far], far+1
	}

	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
