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

	if rec.Animate != 0 {
		t.animate(walls, ww, t.rt.Walls, false)
		t.animate(floors, ww, t.rt.Floors, false)
		t.animate([][]uint32{shadow}, ww, t.rt.Shadows, true)
	}

	return nil
}

// animate is 0x6703e0 / 0x670120 for one record list of an Animate preset.
// In DT1 an animated tile (material bit 0x100) is a set of frame tiles under
// one (orientation, style, sequence) key whose Rarity column is the frame
// number. The record found by the pick becomes frame 0 and one more record at
// the same cell is made for every other frame (flag 8, not chained into a
// border group). The cell value is read back from the layer grid that the
// record's layer bits (flags 14-16) name (the shadow list has one grid).
// Nothing is rolled. The loop covers the records that existed when it started.
// The counting pass 0x66ff50 only sizes buffers and sets room flag 0x8000000;
// it is not needed here.
func (t *tileBuilder) animate(grids [][]uint32, ww int, list []*TileRecord, shadow bool) {
	n := len(list)

	for i := 0; i < n; i++ {
		rec := list[i]
		if rec.Tile.DT == nil || rec.Tile.tile().material&0x100 == 0 {
			continue
		}

		layer := 0
		if !shadow {
			layer = int((rec.Flags>>14)&7) - 1
		}

		if layer < 0 || layer >= len(grids) {
			panic(unported("animated record of an unknown layer"))
		}

		var v uint32
		if g := grids[layer]; rec.Y*ww+rec.X < len(g) {
			v = g[rec.Y*ww+rec.X]
		}

		var style, seq int32
		if v != 0 {
			style, seq = int32(v>>20)&0x3f, int32(v>>8)&0xff
		}

		refs := t.rt.Lib.query(int32(rec.Ori), style, seq, 40)
		frame := func(k int) TileRef {
			for _, r := range refs {
				if r.Rarity() == k {
					return r
				}
			}

			panic(GameError{0xb6}) // fatal in the original too
		}

		rec.Tile = frame(0)
		wx, wy := t.r.X+rec.X, t.r.Y+rec.Y

		for k := 1; k < len(refs); k++ {
			tile := frame(k)

			var nr *TileRecord

			switch rec.Ori {
			case 0:
				nr = floorRec(t.rt, nil, wx, wy, v, tile)
			case 0xd:
				shadowRec(t.rt, nil, wx, wy, v, tile)
				nr = t.rt.Shadows[len(t.rt.Shadows)-1]
			default:
				nr = t.wallRec(t.rt, nil, wx, wy, v, tile, rec.Ori)
			}

			nr.Flags |= 8
		}
	}
}
