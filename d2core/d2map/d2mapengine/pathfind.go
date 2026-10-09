package d2mapengine

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
)

// PathFind finds a path between given start and dest positions and returns the positions of the path
func (m *MapEngine) PathFind(start, dest d2vector.Position) []d2vector.Position {
	points := make([]d2vector.Position, 0)
	clear, point := m.checkLos(start, dest)

	if clear {
		return append(points, point)
	}

	// the straight line is blocked: plan around the obstacle
	from := pt{int(math.Floor(start.X())), int(math.Floor(start.Y()))}
	to := pt{int(math.Floor(dest.X())), int(math.Floor(dest.Y()))}

	route, reached := findPath(m.WalkBlocked, from, to)
	if len(route) == 0 {
		return append(points, point) // nothing better: walk up to the obstacle
	}

	for i, p := range route {
		if reached && i == len(route)-1 {
			points = append(points, dest) // the exact point that was clicked
			break
		}

		points = append(points, d2vector.NewPosition(float64(p.x)+0.5, float64(p.y)+0.5))
	}

	return points
}

// checkLos finds out if there is a clear line of sight between two points
func (m *MapEngine) checkLos(start, end d2vector.Position) (bool, d2vector.Position) {
	dv := d2vector.Position{Vector: *end.Clone()}
	dv.Subtract(&start.Vector)
	dx := dv.X()
	dy := dv.Y()
	N := math.Max(math.Abs(dx), math.Abs(dy))

	var divN float64
	if N == 0 {
		divN = 0.0
	} else {
		divN = 1.0 / N // nolint:gomnd // we're just taking inverse...
	}

	xstep := dx * divN
	ystep := dy * divN
	x := start.X()
	y := start.Y()

	for i := 0; i <= int(N); i++ {
		x += xstep
		y += ystep

		if m.WalkBlocked(int(math.Floor(x)), int(math.Floor(y))) {
			return false, d2vector.NewPosition(x-xstep, y-ystep)
		}
	}

	return true, end
}
