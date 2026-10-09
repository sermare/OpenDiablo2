package d2path

import (
	"container/heap"
	"math"
)

// longAStarRef is the original map and container/heap implementation of LongAStar, kept to prove that
// the array based one returns the same routes.
func longAStarRef(g Grid, mask uint16, from, to Point, budget int) (Route, bool) {
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
