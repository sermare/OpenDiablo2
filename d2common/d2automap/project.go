package d2automap

// Projection constants of UI\automap.cpp (VERIFIED in Game.exe 1.14b).
//
// Every cell position is stored at "scale 10": a tile (tx,ty) is at
// ((tx-ty)*8, (tx+ty)*4) (D2Common FUN_00644770 gives ((tx-ty)*0x50,
// (tx+ty)*0x50>>1) and the automap divides by 10). Units are projected with
// ((sx-sy)*16, (sx+sy)*8) on sub-tile coordinates and divided by the same
// scale, which is the same space (5 sub-tiles per tile). The full-screen
// automap draws at scale 10 (1 cell pixel = 1 screen pixel, MaxiMap.dc6 16x32
// frames), the mini map at scale 20 (MaxiMapS.dc6 8x16 frames, half size).
const (
	// ScaleFull and ScaleMini are DAT_00710f30 for the two sizes.
	ScaleFull = 10
	ScaleMini = 20

	// CellW and CellH are the size of a MaxiMap.dc6 frame.
	CellW, CellH = 16, 32

	// UpperWallShift is added to the y of tiles of orientation >= 16 (lower
	// wall equivalents); the engine does y += 0x18 when the tile's field at +0x1c
	// is >= 16. What exactly that field is: UNVERIFIED (assumed the orientation).
	UpperWallShift = 24

	// originDX and originDY are 0x28 and 0xf: the engine adds them to the map
	// origin, so the hero stands at (W/2-40, H/2-15) on the full-screen map and
	// a cross is drawn at +8,-8 from there. (VERIFIED in FUN_00454de0 and
	// AUTOMAP_DrawUnitMarker.)
	originDX, originDY = 0x28, 0xf
)

// TileCell returns the cell position of a tile in cell space (scale 10).
func TileCell(tx, ty int) (x, y int) {
	return (tx - ty) * 8, (tx + ty) * 4
}

// WorldCell converts a position in (fractional) tiles, e.g. the hero's, to cell
// space (scale 10).
func WorldCell(tx, ty float64) (x, y float64) {
	return (tx - ty) * 8, (tx + ty) * 4
}

// MovedFar implements the reveal trigger of FUN_004546b0: the automap is
// updated when the hero's projected position has moved more than 79 "iso
// units" from where it last revealed. The metric is (min*2+max)/2 on the
// absolute differences of the unit projections ((sx-sy)*16, (sx+sy)*8), which
// are 10 times the cell coordinates. (VERIFIED)
func MovedFar(lastCellX, lastCellY, cellX, cellY float64) bool {
	dx := abs(cellX-lastCellX) * 10
	dy := abs(cellY-lastCellY) * 10

	var m float64
	if dy < dx {
		m = dy + 2*dx
	} else {
		m = dx + 2*dy
	}

	return m/2 > 79
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}

	return v
}

// Size selects the full-screen or mini map.
type Size int

// Sizes.
const (
	SizeFull Size = iota
	SizeMini
)

// Scale returns the divisor of the size.
func (s Size) Scale() int {
	if s == SizeMini {
		return ScaleMini
	}

	return ScaleFull
}

// Rect is a screen rectangle (inclusive-exclusive).
type Rect struct{ X0, Y0, X1, Y1 int }

// Contains reports whether the point is inside.
func (r Rect) Contains(x, y int) bool { return x >= r.X0 && x < r.X1 && y >= r.Y0 && y < r.Y1 }

// Layout is the screen mapping of one frame.
type Layout struct {
	Size Size
	// OriginX/OriginY are subtracted from a cell position (after scaling) to
	// get its screen position; the cell's frame is anchored at its bottom left.
	OriginX, OriginY int
	// Clip is where cells may be drawn (the whole screen, or the mini map box).
	Clip Rect
}

// PanelShift says which side panels are open: the real game moves the hero a
// quarter of the screen away from the open panel (full-screen map only).
type PanelShift int

// Panel shifts.
const (
	PanelNone  PanelShift = iota
	PanelRight            // a panel on the right side is open: the map moves left
	PanelLeft             // a panel on the left side is open: the map moves right
)

// ComputeLayout reproduces FUN_00452bb0 / FUN_00452cd0 / FUN_00454de0 for a
// screen of w x h and the hero at (heroCellX, heroCellY) in cell space.
// With the original 800x600: full map hero at (360,285) on screen; mini map
// box x 519..798, y 57..282 at the right. (VERIFIED formulas; the box offsets
// 0x119, 0x39, 0x117, 0xe1 are read from the code.)
func ComputeLayout(size Size, w, h int, heroCellX, heroCellY float64, shift PanelShift, miniLeft bool) Layout {
	d210, d214 := MiniBoxOffsets(w, h, miniLeft)

	return ComputeLayoutOffsets(size, w, h, heroCellX, heroCellY, shift, miniLeft, d210, d214)
}

// MiniBoxOffsets are the mini map's DAT_0079d210/214 as AUTOMAP_RecalcOffsets
// (0x452cd0) computes them for the box at the right (miniLeft false) or left.
// The engine recomputes them when the map size changes and, only if the
// "AutoMap Centers" option is on, whenever the panel layout changes; with the
// option off they stay stale, so the hero is no longer centred in the box.
func MiniBoxOffsets(w, h int, miniLeft bool) (d210, d214 int) {
	var d264, d260 int
	if !miniLeft {
		d264, d260 = (w*2)/3, 0x4e
	}

	return (w/3 - d264) - 0x10, (h/3 - d260) - 0x10
}

// ComputeLayoutOffsets is ComputeLayout with the mini map offsets (see
// MiniBoxOffsets) given by the caller, which keeps them between recalculations.
func ComputeLayoutOffsets(size Size, w, h int, heroCellX, heroCellY float64, shift PanelShift, miniLeft bool, d210, d214 int) Layout {
	scale := size.Scale()
	hx := int(heroCellX*10) / scale
	hy := int(heroCellY*10) / scale

	l := Layout{Size: size}

	var offX, offY int

	if size == SizeFull {
		switch shift {
		case PanelRight:
			offX = -(w / 4)
		case PanelLeft:
			offX = w / 4
		}

		l.Clip = Rect{0, 0, w, h}
		offX = (hx - w/2) - offX
		offY = hy - h/2
		l.OriginX = originDX + offX
		l.OriginY = originDY + offY

		return l
	}

	// mini map: a box of 0x117 x 0xe1 pixels; the map is centred in it
	offX = hx - w/2
	offY = hy - h/2
	l.OriginX = d210 + originDX + offX
	l.OriginY = d214 + originDY + offY

	if miniLeft {
		l.Clip = Rect{0, 0, 0x117, 0xe1 - 0x15}
	} else {
		l.Clip = Rect{w - 0x119, 0x39, w - 0x119 + 0x117, 0x39 + 0xe1}
	}

	return l
}

// ToScreen returns the screen position (frame bottom-left anchor) of a cell.
func (l Layout) ToScreen(cellX, cellY int) (x, y int) {
	s := l.Size.Scale()

	return cellX*10/s - l.OriginX, cellY*10/s - l.OriginY
}

// HeroScreen returns where the hero's own position projects on screen
// (before the +8,-8 marker offset the engine applies).
func (l Layout) HeroScreen(heroCellX, heroCellY float64) (x, y int) {
	s := l.Size.Scale()

	return int(heroCellX*10)/s - l.OriginX, int(heroCellY*10)/s - l.OriginY
}
