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

// CanWalkTo reports whether a hero can walk from one sub-tile to another with
// the engine's own search (the same one a click uses, long detours included).
func (m *MapEngine) CanWalkTo(fromX, fromY, toX, toY int) bool {
	from, to := d2path.Point{X: fromX, Y: fromY}, d2path.Point{X: toX, Y: toY}

	if from == to {
		return true
	}

	route, ok := d2path.LongAStar(collisionGrid{m}, d2path.MaskPlayer, from, to, 0)

	return ok && !route.Partial
}

// Reachable is the set of sub-tiles connected to a start sub-tile by walking
// (4-neighbour flood fill over the player collision mask).
type Reachable struct {
	w, h int
	ok   []bool
}

// At reports whether the sub-tile can be reached.
func (r *Reachable) At(x, y int) bool {
	if r == nil || x < 0 || y < 0 || x >= r.w || y >= r.h {
		return false
	}

	return r.ok[y*r.w+x]
}

// ReachableFrom flood fills the sub-tiles a hero standing on (subX, subY) can
// walk to. It returns nil when the start itself is blocked. Closed doors are
// not obstacles (like CanWalkTo, only the tile flags count).
func (m *MapEngine) ReachableFrom(subX, subY int) *Reachable {
	w, h := m.size.Width*subtilesPerTile, m.size.Height*subtilesPerTile
	grid := collisionGrid{m}

	open := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < w && y < h && grid.Flags(x, y)&d2path.MaskPlayer == 0
	}

	if !open(subX, subY) {
		return nil
	}

	r := &Reachable{w: w, h: h, ok: make([]bool, w*h)}
	r.ok[subY*w+subX] = true
	queue := []int{subY*w + subX}

	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w, i/w

		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if open(nx, ny) && !r.ok[ny*w+nx] {
				r.ok[ny*w+nx] = true
				queue = append(queue, ny*w+nx)
			}
		}
	}

	return r
}
