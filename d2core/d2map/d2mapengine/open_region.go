package d2mapengine

// A hero who arrives through a seamless border lands at the same world position he left from. The generated
// tiles (an approximation of the original, see drlgoutdoor/doc.go) can put a roof, a wall or a column on
// that cell, so the nearest free sub-tile may be a pocket of a few cells that is closed on all sides (Upper
// Kurast: a 5x5 alcove behind a building) and the hero cannot walk anywhere. NearestOpen moves the arrival
// to the nearest free sub-tile that belongs to a region the hero can really walk in.

// pocketCells is the size (sub-tiles) below which a closed free region counts as a pocket.
const pocketCells = 600

// nearestOpenCell is NearestOpen on a plain grid: blocked reports blocked sub-tiles of a w x h grid; the result
// is the nearest free cell (ring search up to radius) whose 4-connected free region has at least minCells cells.
func nearestOpenCell(blocked func(x, y int) bool, w, h, sx, sy, radius, minCells int) (x, y int, ok bool) {
	size := map[int]int{} // region of a cell -> its size, filled lazily

	regionOf := map[[2]int]int{}
	next := 0

	// flood fills the region of a free cell and counts it; stops counting at minCells
	flood := func(fx, fy int) int {
		next++
		id := next
		stack := [][2]int{{fx, fy}}
		regionOf[[2]int{fx, fy}] = id
		count := 0

		for len(stack) > 0 {
			c := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			count++

			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := c[0]+d[0], c[1]+d[1]
				if nx < 0 || ny < 0 || nx >= w || ny >= h || blocked(nx, ny) {
					continue
				}

				if _, seen := regionOf[[2]int{nx, ny}]; seen {
					continue
				}

				regionOf[[2]int{nx, ny}] = id
				stack = append(stack, [2]int{nx, ny})
			}
		}

		size[id] = count

		return id
	}

	for r := 0; r <= radius; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}

				cx, cy := sx+dx, sy+dy
				if cx < 0 || cy < 0 || cx >= w || cy >= h || blocked(cx, cy) {
					continue
				}

				id, seen := regionOf[[2]int{cx, cy}]
				if !seen {
					id = flood(cx, cy)
				}

				if size[id] >= minCells {
					return cx, cy, true
				}
			}
		}
	}

	return 0, 0, false
}

// NearestOpen is NearestWalkable restricted to free sub-tiles of regions with at least pocketCells cells (not
// the closed alcoves between buildings). When no such cell lies within radius it behaves like NearestWalkable.
func (m *MapEngine) NearestOpen(subX, subY, radius int) (x, y int, ok bool) {
	w, h := m.size.Width*subtilesPerTile, m.size.Height*subtilesPerTile

	if x, y, ok = nearestOpenCell(m.WalkBlocked, w, h, subX, subY, radius, pocketCells); ok {
		return x, y, true
	}

	return m.NearestWalkable(subX, subY, radius)
}
