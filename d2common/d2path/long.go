package d2path

import (
	"container/heap"
	"math"
)

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

	start := &node{p: from, h: EstimateDistance(to.X-from.X, to.Y-from.Y)}
	open := &openList{}
	best := map[Point]*node{from: start}
	closed := map[Point]bool{}
	seq := 0

	heap.Push(open, start)

	closest := start
	expanded := 0

	for open.Len() > 0 && expanded < budget {
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
			return finishLimit(cur, false, math.MaxInt32)
		}

		for _, nb := range neighbours {
			p := Point{cur.p.X + nb.dx, cur.p.Y + nb.dy}
			if closed[p] || Blocked(g, p.X, p.Y, mask) {
				continue
			}

			if nb.dx != 0 && nb.dy != 0 &&
				(Blocked(g, cur.p.X+nb.dx, cur.p.Y, mask) || Blocked(g, cur.p.X, cur.p.Y+nb.dy, mask)) {
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

	return finishLimit(closest, true, math.MaxInt32)
}
