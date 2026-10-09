package d2monsters

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"

// footprints keeps the collision flags of unit footprints on top of the
// static map grid: monsters set 0x100, players 0x80 and dead monsters 0x8000
// (VERIFIED flag values, d2path). The footprint bits are in no block mask
// (VERIFIED), so what blocks movement between units is an engine choice
// (UNVERIFIED): d2path.MaskUnits is added to the monster mask for path
// planning and for the per-step test.
type footprints struct {
	grid  *d2path.LayeredGrid
	cells map[d2path.Point][]uint32
	where map[uint32]placed
}

type placed struct {
	at   d2path.Point
	bits uint16
}

func newFootprints(base d2path.Grid) *footprints {
	return &footprints{
		grid:  d2path.NewLayeredGrid(base),
		cells: map[d2path.Point][]uint32{},
		where: map[uint32]placed{},
	}
}

// Keys: monsters use their brain id, players use playerKeyBase+index.
const playerKeyBase = 1 << 30

// Move puts (or moves) a unit's footprint.
func (f *footprints) Move(key uint32, x, y int, bits uint16) {
	to := d2path.Point{X: x, Y: y}

	if old, ok := f.where[key]; ok {
		if old.at == to && old.bits == bits {
			return
		}

		f.drop(key, old)
	}

	f.where[key] = placed{to, bits}
	f.cells[to] = append(f.cells[to], key)
	f.grid.Set(x, y, bits)
}

// Remove deletes a unit's footprint.
func (f *footprints) Remove(key uint32) {
	if old, ok := f.where[key]; ok {
		f.drop(key, old)
		delete(f.where, key)
	}
}

func (f *footprints) drop(key uint32, old placed) {
	list := f.cells[old.at]

	for i, k := range list {
		if k == key {
			list = append(list[:i], list[i+1:]...)
			break
		}
	}

	if len(list) == 0 {
		delete(f.cells, old.at)
	} else {
		f.cells[old.at] = list
	}

	// recompute the bits left in the cell from its remaining occupants
	f.grid.Clear(old.at.X, old.at.Y, old.bits)

	for _, k := range list {
		f.grid.Set(old.at.X, old.at.Y, f.where[k].bits)
	}
}

// BlockedFor reports whether a cell holds a blocking unit other than key
// (monsters and players block, corpses do not).
func (f *footprints) BlockedFor(key uint32, x, y int) bool {
	for _, k := range f.cells[d2path.Point{X: x, Y: y}] {
		if k != key && f.where[k].bits&d2path.MaskUnits != 0 {
			return true
		}
	}

	return false
}

// Flags are the static flags plus the footprints of a cell.
func (f *footprints) Flags(x, y int) uint16 { return f.grid.Flags(x, y) }

// Count is the number of tracked footprints.
func (f *footprints) Count() int { return len(f.where) }

// ignoring returns a grid view in which the unit footprints of the given cells
// do not count: a path request must not be blocked by the unit's own cell nor
// by the target it walks to.
func (f *footprints) ignoring(cells ...d2path.Point) d2path.Grid {
	return ignoreView{f.grid, cells}
}

type ignoreView struct {
	g     d2path.Grid
	cells []d2path.Point
}

func (v ignoreView) Flags(x, y int) uint16 {
	fl := v.g.Flags(x, y)

	for _, c := range v.cells {
		if c.X == x && c.Y == y {
			return fl &^ d2path.MaskUnits
		}
	}

	return fl
}
