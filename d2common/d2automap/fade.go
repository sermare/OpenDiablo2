package d2automap

// Transparency is the engine's per-cell draw mode (the last argument of
// GFX_DrvSlot98 in AUTOMAP_DrawCellNodeTree 0x454b00). 5 is the normal automap
// draw; 0, 1 and 2 are passed by the fade option. VERIFIED that these values
// are passed; that 0/1/2 mean about 25/50/75 percent opaque is UNVERIFIED.
type Transparency int

// Transparency values.
const (
	TransVeryLight Transparency = 0
	TransLight     Transparency = 1
	TransMedium    Transparency = 2
	TransNormal    Transparency = 5
)

// Alpha returns the opacity (0..1) used for the transparency.
func (t Transparency) Alpha() float64 {
	switch t {
	case TransVeryLight:
		return 0.25
	case TransLight:
		return 0.5
	case TransMedium:
		return 0.75
	}

	return 1
}

// CellTransparency returns the draw mode of a cell whose frame is drawn at
// window position (x, y) (bottom-left anchor) on a w x h window. shiftPx is the
// horizontal panel shift of the origin (-200, +200 or 0: PanelShift.Pixels).
//
// With the fade option off every cell is normal (AutoMapFade, 0x452d80). With it
// on, the mini map is drawn flat at mode 1; the full map fades the cells near the
// hero so that the world shows through: inside a box of 0x8c either side of the
// centre and -0x96..+0x82 vertically, within 0x96 on the isometric metric
// (min*2+max)/2, mode 0 below 50, 1 below 100, else 2. The box and 0x96 are read
// from the code; the 50/100 steps compare a register the decompiler lost and
// are UNVERIFIED.
func CellTransparency(fade bool, size Size, x, y, w, h, shiftPx int) Transparency {
	if !fade {
		return TransNormal
	}

	if size == SizeMini {
		return TransLight
	}

	cx, cy := w/2, h/2
	if x < cx-0x8c+shiftPx || x > cx+0x8c+shiftPx || y < cy-0x96 || y > cy+0x82 {
		return TransNormal
	}

	dx := (cx - x) + shiftPx
	if dx < 0 {
		dx = -dx
	}

	dy := (cy - 10) - y
	if dy < 0 {
		dy = -dy
	}

	d := dx + dy*2
	if dy < dx {
		d = dy + dx*2
	}

	d /= 2

	switch {
	case d >= 0x96:
		return TransNormal
	case d < 50:
		return TransVeryLight
	case d < 100:
		return TransLight
	}

	return TransMedium
}
