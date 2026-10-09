package drlgoutdoor

import "fmt"

// Preset rooms: DRLG_BuildPresetRoomGrids (0x6693e0) windows the DS1 layers of
// the preset to the room (w+1 by h+1 cells), marks the window borders and the
// layer index in the cell values, and DRLG_BuildPresetRoomTiles (0x6696c0) runs
// the cell function (0x671680) over the floor layers, then the wall layers with
// their orientation layers, then the shadow layer.

// window copies the (w+1) x (h+1) cells of a DS1 layer that belong to a room.
func (p *Pattern) window(layer []uint32, ox, oy, w, h int) []uint32 {
	out := make([]uint32, w*h)

	if layer == nil {
		return out
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sx, sy := ox+x, oy+y
			if sx >= 0 && sy >= 0 && sx < p.W && sy < p.H && sy*p.W+sx < len(layer) {
				out[y*w+x] = layer[sy*p.W+sx]
			}
		}
	}

	return out
}

func orAll(c []uint32, v uint32) {
	for i := range c {
		c[i] |= v
	}
}

// orBorder is DRLG_ApplyOpToGridBorder with OR.
func orBorder(c []uint32, w, h int, v uint32) {
	for x := 0; x < w; x++ {
		c[x] |= v
		c[(h-1)*w+x] |= v
	}

	for y := 1; y < h-1; y++ {
		c[y*w] |= v
		c[y*w+w-1] |= v
	}
}

func (t *tileBuilder) buildPreset() error {
	l, r := t.l, t.r

	rec, ok := l.env.Tables.PrestByDef(r.PrestDef)
	if !ok || r.File < 0 || r.File >= len(rec.File) || rec.File[r.File] == "" {
		return fmt.Errorf("preset Def %d file %d unknown", r.PrestDef, r.File)
	}

	// Def 1 and 0x6c also run the cell passes over two scratch grids the game
	// clears and never fills: no records come from them.
	if rec.Animate != 0 {
		return unported("animated preset tiles (0x66ff50 / 0x6703e0)")
	}

	pat, err := l.env.Pattern(NormalizePrestFile(rec.File[r.File]))
	if err != nil {
		return err
	}

	ww, wh := r.W+1, r.H+1
	ox, oy := r.X-r.PrestX, r.Y-r.PrestY

	nWall, nFloor := len(pat.Wall), len(pat.Floor)
	walls := make([][]uint32, nWall)
	oris := make([][]uint32, nWall)
	floors := make([][]uint32, nFloor)

	for k := 0; k < nWall; k++ {
		walls[k] = pat.window(pat.Wall[k], ox, oy, ww, wh)
		oris[k] = pat.window(pat.Orient[k], ox, oy, ww, wh)
	}

	for k := 0; k < nFloor; k++ {
		floors[k] = pat.window(pat.Floor[k], ox, oy, ww, wh)
	}

	shadow := pat.window(pat.Shadow, ox, oy, ww, wh)

	if nWall != 0 {
		orBorder(walls[0], ww, wh, 0x84)
	}

	for k := 1; k < nWall; k++ {
		orAll(walls[k], uint32(k)<<18)
	}

	for k := 0; k < nFloor; k++ {
		orAll(floors[k], uint32(k)<<18)
	}

	for k := 0; k < nFloor; k++ {
		orBorder(floors[k], ww, wh, 0x84)
	}

	orBorder(shadow, ww, wh, 0x84)

	// the window is one cell wider than the room unless the room ends at the
	// edge of the preset and the preset kills that edge (KillEdge)
	cw, ch := ww, wh

	if rec.KillEdge != 0 {
		if r.W+r.X == r.PrestX+r.PrestW {
			cw--
		}

		if r.H+r.Y == r.PrestY+r.PrestH {
			ch--
		}
	}

	pass := func(ori, val []uint32, fill bool) {
		for y := 0; y < ch; y++ {
			for x := 0; x < cw; x++ {
				o := 0
				if ori != nil {
					o = int(ori[y*ww+x])
				}

				t.cell(o, val[y*ww+x], r.X+x, r.Y+y, fill)
			}
		}
	}

	for k := 0; k < nFloor; k++ {
		pass(nil, floors[k], k == 0 && rec.FillBlanks != 0)
	}

	for k := 0; k < nWall; k++ {
		pass(oris[k], walls[k], false)
	}

	pass(nil, shadow, false)

	return nil
}
