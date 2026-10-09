package d2inventory

// This file ports the original game's inventory auto-placement search
// (INV_FindFreeSlotForItem 0x63c950 and the helpers at 0x63c410..0x63c910 in
// Game.exe 1.14b). It is pure: it works on an occupancy grid only.

// PerfectScore is the neighbour score that ends a scored search early.
const PerfectScore = 0xff

// OccupancyGrid is a width x height cell map of an inventory page.
type OccupancyGrid struct {
	Width, Height int
	cells         []bool
}

// NewOccupancyGrid creates an empty grid.
func NewOccupancyGrid(width, height int) *OccupancyGrid {
	return &OccupancyGrid{Width: width, Height: height, cells: make([]bool, width*height)}
}

// Occupied reports whether a cell is taken; cells outside the grid are not.
func (g *OccupancyGrid) Occupied(x, y int) bool {
	if x < 0 || y < 0 || x >= g.Width || y >= g.Height {
		return false
	}

	return g.cells[y*g.Width+x]
}

// Fill marks a rectangle occupied (or free). The rectangle is clipped to the grid.
func (g *OccupancyGrid) Fill(x, y, w, h int, occupied bool) {
	for cy := y; cy < y+h; cy++ {
		for cx := x; cx < x+w; cx++ {
			if cx >= 0 && cy >= 0 && cx < g.Width && cy < g.Height {
				g.cells[cy*g.Width+cx] = occupied
			}
		}
	}
}

// CellsFree reports whether the w x h rectangle at (x,y) lies inside the grid
// and contains no occupied cell. Touching neighbours do not matter.
func (g *OccupancyGrid) CellsFree(x, y, w, h int) bool {
	if x < 0 || y < 0 || w < 1 || h < 1 || x+w > g.Width || y+h > g.Height {
		return false
	}

	for cy := y; cy < y+h; cy++ {
		for cx := x; cx < x+w; cx++ {
			if g.cells[cy*g.Width+cx] {
				return false
			}
		}
	}

	return true
}

// ScoreSlotNeighbours is INV_ScoreSlotNeighbours: it counts occupied cells
// (grid edges count as occupied) on the four sides of the w x h rectangle at
// (x,y), excluding the corners. A score of at least 2*(w+h), meaning every
// side cell is blocked, is reported as PerfectScore.
func (g *OccupancyGrid) ScoreSlotNeighbours(x, y, w, h int) int {
	score := 0

	// left side
	if x < 1 {
		score += h
	} else {
		score += g.countColumn(x-1, y, h)
	}

	// right side
	if x+w < g.Width {
		score += g.countColumn(x+w, y, h)
	} else {
		score += h
	}

	// top side
	if y < 1 {
		score += w
	} else {
		score += g.countRow(y-1, x, w)
	}

	// bottom side
	if y+h < g.Height {
		score += g.countRow(y+h, x, w)
	} else {
		score += w
	}

	if score >= 2*(w+h) {
		return PerfectScore
	}

	return score
}

func (g *OccupancyGrid) countColumn(x, y, h int) int {
	n := 0

	for cy := y; cy < y+h; cy++ {
		if g.Occupied(x, cy) {
			n++
		}
	}

	return n
}

func (g *OccupancyGrid) countRow(y, x, w int) int {
	n := 0

	for cx := x; cx < x+w; cx++ {
		if g.Occupied(cx, y) {
			n++
		}
	}

	return n
}

// SearchStrategy names one of the original's four search routines.
type SearchStrategy int

// The four strategies, named after their original functions.
const (
	BestScoreColumnsDesc SearchStrategy = iota // 0x63c570
	FirstFitColumnsDesc                        // 0x63c640
	BestScoreRowMajor                          // 0x63c720
	FirstFitColumnMajor                        // 0x63c7f0
)

// ChooseStrategy mirrors the dispatch in INV_FindFreeSlotForItem. Items one
// cell high use the right-to-left column scan; every other size (the 2x2,
// 3-high and generic cases all lead to the same routines in the binary) uses
// the row-major scan. Scored variants are only used when the owner is a
// player; monsters and vendors take the first-fit variants.
func ChooseStrategy(w, h int, playerOwner bool) SearchStrategy {
	if h == 1 {
		if playerOwner {
			return BestScoreColumnsDesc
		}

		return FirstFitColumnsDesc
	}

	if playerOwner {
		return BestScoreRowMajor
	}

	return FirstFitColumnMajor
}

// FindFreeSlot finds a position for a w x h item and reports whether one
// exists. It reproduces the original, including that scored searches only
// accept a candidate whose score is strictly greater than the best so far,
// starting from 0: a free spot with no occupied or edge neighbour at all
// (score 0) is never chosen.
func (g *OccupancyGrid) FindFreeSlot(w, h int, playerOwner bool) (x, y int, ok bool) {
	if w < 1 || h < 1 || w > g.Width || h > g.Height {
		return 0, 0, false
	}

	switch ChooseStrategy(w, h, playerOwner) {
	case BestScoreColumnsDesc:
		return g.bestScore(w, h, true)
	case BestScoreRowMajor:
		return g.bestScore(w, h, false)
	case FirstFitColumnsDesc:
		return g.firstFit(w, h, true)
	default:
		return g.firstFit(w, h, false)
	}
}

// order calls visit for each anchor in scan order until it returns true.
// columnsDesc: x from the right edge to 0, y from the bottom up (the scored
// variant) or top down (the first-fit variant, see firstFit). Otherwise
// row-major: y outer, x inner, both ascending.
func (g *OccupancyGrid) bestScore(w, h int, columnsDesc bool) (bx, by int, ok bool) {
	best := 0

	try := func(x, y int) bool {
		if !g.CellsFree(x, y, w, h) {
			return false
		}

		if s := g.ScoreSlotNeighbours(x, y, w, h); s > best {
			best, bx, by, ok = s, x, y, true

			return s == PerfectScore
		}

		return false
	}

	if columnsDesc {
		for x := g.Width - 1; x >= 0; x-- {
			for y := g.Height - 1; y >= 0; y-- {
				if try(x, y) {
					return bx, by, true
				}
			}
		}

		return bx, by, ok
	}

	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			if try(x, y) {
				return bx, by, true
			}
		}
	}

	return bx, by, ok
}

// firstFit: columnsDesc scans x from the right edge down to 0 with y
// ascending; otherwise x ascends with y ascending (column-major).
func (g *OccupancyGrid) firstFit(w, h int, columnsDesc bool) (int, int, bool) {
	for i := 0; i < g.Width; i++ {
		x := i
		if columnsDesc {
			x = g.Width - 1 - i
		}

		for y := 0; y < g.Height; y++ {
			if g.CellsFree(x, y, w, h) {
				return x, y, true
			}
		}
	}

	return 0, 0, false
}
