package d2path

// PlaceCluster finds n distinct cells for a group of units around center: the
// cells the mask allows, at least spacing subtiles (Chebyshev) from each
// other, searched outward from the center in square rings up to maxRadius.
// It returns fewer than n cells when the area is too cramped. Placement of
// spawned groups is not recorded in the notes (UNVERIFIED engine choice); the
// ring order makes a pack stand together like a real one.
func PlaceCluster(g Grid, mask uint16, center Point, n, spacing, maxRadius int) []Point {
	if spacing < 1 {
		spacing = 1
	}

	taken := map[Point]bool{}
	out := make([]Point, 0, n)

	for r := 0; r <= maxRadius && len(out) < n; r++ {
		for dy := -r; dy <= r && len(out) < n; dy++ {
			for dx := -r; dx <= r && len(out) < n; dx++ {
				if abs(dx) != r && abs(dy) != r {
					continue
				}

				p := Point{center.X + dx, center.Y + dy}
				if Blocked(g, p.X, p.Y, mask) || tooClose(taken, p, spacing) {
					continue
				}

				taken[p] = true
				out = append(out, p)
			}
		}
	}

	return out
}

func tooClose(taken map[Point]bool, p Point, spacing int) bool {
	for q := range taken {
		if abs(q.X-p.X) < spacing && abs(q.Y-p.Y) < spacing {
			return true
		}
	}

	return false
}

// LayeredGrid adds unit footprints on top of a static grid so that path
// finding and placement see monsters (flag 0x100) and players (flag 0x80).
// The footprint bits are in none of the block masks (VERIFIED), so a caller
// that wants units to block adds them to the mask (see MaskUnits).
type LayeredGrid struct {
	Base Grid
	over map[Point]uint16
}

// MaskUnits are the footprint bits of players, monsters and corpses.
const MaskUnits = FlagPlayer | FlagMonster

// NewLayeredGrid wraps a base grid.
func NewLayeredGrid(base Grid) *LayeredGrid {
	return &LayeredGrid{Base: base, over: map[Point]uint16{}}
}

// Flags implements Grid.
func (l *LayeredGrid) Flags(x, y int) uint16 {
	return l.Base.Flags(x, y) | l.over[Point{x, y}]
}

// Set ORs footprint bits into a cell.
func (l *LayeredGrid) Set(x, y int, bits uint16) { l.over[Point{x, y}] |= bits }

// Clear removes footprint bits from a cell.
func (l *LayeredGrid) Clear(x, y int, bits uint16) {
	p := Point{x, y}
	if v := l.over[p] &^ bits; v != 0 {
		l.over[p] = v
	} else {
		delete(l.over, p)
	}
}

// Overlay returns only the footprint bits of a cell.
func (l *LayeredGrid) Overlay(x, y int) uint16 { return l.over[Point{x, y}] }
