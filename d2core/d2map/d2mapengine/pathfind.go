package d2mapengine

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// UseCollisionPaths makes PathFind route around obstacles with the ported
// click-to-move search (d2path.FindPath: straight line, else A* for nearby
// goals) instead of walking a straight line up to the first blocked subtile.
// Level generators for winding dungeon levels switch it on; the town keeps the
// old behaviour.
func (m *MapEngine) UseCollisionPaths(on bool) { m.gridPaths = on }

// collisionGrid adapts the sub-tile flags of the map to a d2path.Grid.
type collisionGrid struct{ m *MapEngine }

func (g collisionGrid) Flags(x, y int) uint16 {
	if x < 0 || y < 0 || x/subtilesPerTile >= g.m.size.Width || y/subtilesPerTile >= g.m.size.Height {
		return d2path.OutOfGrid
	}

	s := g.m.SubTileAt(x, y)

	var f uint16

	if s.BlockWalk {
		f |= d2path.FlagWalk
	}

	if s.BlockPlayerWalk {
		f |= d2path.FlagPlayerOnly
	}

	if s.BlockLOS {
		f |= d2path.FlagWall
	}

	return f
}

// PathFind finds a path between given start and dest positions and returns the positions of the path
func (m *MapEngine) PathFind(start, dest d2vector.Position) []d2vector.Position {
	if m.gridPaths {
		from := d2path.Point{X: int(start.X()), Y: int(start.Y())}
		to := d2path.Point{X: int(dest.X()), Y: int(dest.Y())}

		grid := collisionGrid{m}

		route, ok := d2path.FindPath(grid, d2path.MaskPlayer, from, to)
		if !ok || route.Partial {
			// the original search gives up on long detours; this engine does not (see d2path.LongAStar)
			if long, lok := d2path.LongAStar(grid, d2path.MaskPlayer, from, to, 0); lok && len(long.Nodes) > 0 {
				route, ok = long, true
			}
		}

		if ok && len(route.Nodes) > 0 {
			path := make([]d2vector.Position, len(route.Nodes))
			for i, n := range route.Nodes {
				path[i] = d2vector.NewPosition(float64(n.X), float64(n.Y))
			}

			return path
		}
	}

	points := make([]d2vector.Position, 0)
	_, point := m.checkLos(start, dest)
	points = append(points, point)

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

		if m.SubTileAt(int(math.Floor(x)), int(math.Floor(y))).BlockWalk {
			return false, d2vector.NewPosition(x-xstep, y-ystep)
		}
	}

	return true, end
}
