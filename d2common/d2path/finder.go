package d2path

import "container/heap"

// Limits taken from the notes.
const (
	// MaxNodes is the size of a unit's node array: a path with more than 77
	// corner nodes fails (VERIFIED, `0x4d < n`).
	MaxNodes = 77
	// MaxSearchNodes is the short A* budget ("200-node A*"). Whether it
	// counts expanded or open nodes is UNVERIFIED; expanded nodes here.
	MaxSearchNodes = 200
	// MaxDestinationDelta rejects destinations 100 or more subtiles away on
	// either axis (VERIFIED, PATH_ComputePathForUnit).
	MaxDestinationDelta = 100
	// LineFallbackDistSq: when the straight line is blocked, A* runs only if
	// dx*dx+dy*dy is below this (VERIFIED, 0x145 = 325, ~18 subtiles).
	LineFallbackDistSq = 325

	costStraight = 2
	costDiagonal = 3
)

// Point is a subtile coordinate.
type Point struct{ X, Y int }

// Route is a computed path: the nodes to walk through after the start point.
type Route struct {
	Nodes []Point
	// Partial is set when the goal was not reachable and the route ends at
	// the reachable cell closest to it (VERIFIED behaviour of the short A*).
	Partial bool
}

// EstimateDistance is the A* heuristic: 2*max(|dx|,|dy|)+min(|dx|,|dy|)
// (VERIFIED). The exe consults an exact 8x8 lookup table for deltas below 8
// whose contents are not in the notes; that table is not ported.
func EstimateDistance(dx, dy int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dx < dy {
		dx, dy = dy, dx
	}

	return 2*dx + dy
}

// TraceLine walks the Bresenham line from a to b (exclusive of a) and
// reports whether every cell is free. When blocked it returns the last free
// cell (PATH_TraceLineCollision). a==b is trivially clear.
func TraceLine(g Grid, mask uint16, a, b Point) (clear bool, last Point) {
	dx, dy := abs(b.X-a.X), abs(b.Y-a.Y)
	sx, sy := sgn(b.X-a.X), sgn(b.Y-a.Y)
	err := dx - dy
	cur := a
	last = a

	for cur != b {
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			cur.X += sx
		}

		if e2 < dx {
			err += dx
			cur.Y += sy
		}

		if Blocked(g, cur.X, cur.Y, mask) {
			return false, last
		}

		last = cur
	}

	return true, b
}

type node struct {
	p      Point
	g, h   int
	parent *node
	seq    int
	index  int
}

type openList []*node

func (o openList) Len() int { return len(o) }
func (o openList) Less(i, j int) bool {
	fi, fj := o[i].g+o[i].h, o[j].g+o[j].h
	if fi != fj {
		return fi < fj
	}

	if o[i].h != o[j].h {
		return o[i].h < o[j].h
	}

	return o[i].seq < o[j].seq
}
func (o openList) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
	o[i].index, o[j].index = i, j
}
func (o *openList) Push(x interface{}) {
	n := x.(*node)
	n.index = len(*o)
	*o = append(*o, n)
}
func (o *openList) Pop() interface{} {
	old := *o
	n := old[len(old)-1]
	*o = old[:len(old)-1]

	return n
}

var neighbours = [8]struct{ dx, dy, cost int }{
	{1, 0, costStraight}, {-1, 0, costStraight}, {0, 1, costStraight}, {0, -1, costStraight},
	{1, 1, costDiagonal}, {1, -1, costDiagonal}, {-1, 1, costDiagonal}, {-1, -1, costDiagonal},
}

// ShortAStar is PATH_FindPathShortAStar: 8 neighbours, cost 2 straight and 3
// diagonal, heuristic EstimateDistance, a budget of MaxSearchNodes expanded
// nodes, output compacted to corner nodes, failure when there are more than
// MaxNodes corners. When the goal is unreachable (or the budget runs out) the
// route leads to the visited cell closest to the goal and is Partial.
//
// UNVERIFIED details: ties are broken by lower h and then by insertion order
// (the exe's open list order is only described as "lower h, then +5 later
// nodes"); diagonal steps test only the destination cell, not the two
// orthogonal neighbours.
func ShortAStar(g Grid, mask uint16, from, to Point) (Route, bool) {
	if from == to {
		return Route{}, true
	}

	start := &node{p: from, h: EstimateDistance(to.X-from.X, to.Y-from.Y)}
	open := &openList{}
	best := map[Point]*node{from: start}
	closed := map[Point]bool{}
	seq := 0

	heap.Push(open, start)

	closest := start
	expanded := 0

	for open.Len() > 0 && expanded < MaxSearchNodes {
		cur := heap.Pop(open).(*node)
		if closed[cur.p] {
			continue
		}

		closed[cur.p] = true
		expanded++

		if cur.h < closest.h || cur.h == closest.h && cur.g < closest.g {
			closest = cur
		}

		if cur.p == to {
			return finish(cur, false)
		}

		for _, nb := range neighbours {
			p := Point{cur.p.X + nb.dx, cur.p.Y + nb.dy}
			if closed[p] || Blocked(g, p.X, p.Y, mask) {
				continue
			}

			ng := cur.g + nb.cost
			if old, ok := best[p]; ok && old.g <= ng {
				continue
			}

			seq++
			n := &node{p: p, g: ng, h: EstimateDistance(to.X-p.X, to.Y-p.Y), parent: cur, seq: seq}
			best[p] = n

			heap.Push(open, n)
		}
	}

	return finish(closest, true)
}

// finish rebuilds the node list from goal back to start and keeps corners.
func finish(end *node, partial bool) (Route, bool) {
	var cells []Point

	for n := end; n.parent != nil; n = n.parent {
		cells = append(cells, n.p)
	}

	// cells runs goal -> first step; reverse it.
	for i, j := 0, len(cells)-1; i < j; i, j = i+1, j-1 {
		cells[i], cells[j] = cells[j], cells[i]
	}

	var nodes []Point

	prev := end
	for prev.parent != nil {
		prev = prev.parent
	}

	dir := Point{}

	for i, c := range cells {
		from := prev.p
		if i > 0 {
			from = cells[i-1]
		}

		d := Point{c.X - from.X, c.Y - from.Y}
		if i > 0 && d != dir {
			nodes = append(nodes, cells[i-1])
		}

		dir = d
	}

	if len(cells) > 0 {
		nodes = append(nodes, cells[len(cells)-1])
	}

	if len(nodes) > MaxNodes {
		return Route{}, false
	}

	return Route{Nodes: nodes, Partial: partial}, true
}

// FindPath is the player's click-to-move path (path type 7,
// PATH_FindPathLineThenAStar): reject far destinations, try a straight line,
// and when it is blocked and the goal is closer than LineFallbackDistSq run
// ShortAStar. When the goal is farther and the line is blocked the route is
// the straight line up to the last free cell (UNVERIFIED: the notes only say
// the A* fallback is skipped).
func FindPath(g Grid, mask uint16, from, to Point) (Route, bool) {
	if abs(to.X-from.X) >= MaxDestinationDelta || abs(to.Y-from.Y) >= MaxDestinationDelta {
		return Route{}, false
	}

	if from == to {
		return Route{}, false
	}

	if clear, _ := TraceLine(g, mask, from, to); clear {
		return Route{Nodes: []Point{to}}, true
	}

	dx, dy := to.X-from.X, to.Y-from.Y
	if dx*dx+dy*dy < LineFallbackDistSq {
		return ShortAStar(g, mask, from, to)
	}

	_, last := TraceLine(g, mask, from, to)
	if last == from {
		return Route{}, false
	}

	return Route{Nodes: []Point{last}, Partial: true}, true
}

// NearestFree is COLLISION_FindNearestFreeCell: a square spiral outward from p
// (radius 0..maxRadius) for a cell the mask allows (UNVERIFIED: the exe's
// spiral order and step parameter are not recorded).
func NearestFree(g Grid, mask uint16, p Point, maxRadius int) (Point, bool) {
	for r := 0; r <= maxRadius; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}

				q := Point{p.X + dx, p.Y + dy}
				if !Blocked(g, q.X, q.Y, mask) {
					return q, true
				}
			}
		}
	}

	return Point{}, false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func sgn(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}

	return 0
}
