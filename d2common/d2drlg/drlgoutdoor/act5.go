package drlgoutdoor

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"

// Act 5 outdoor levels 110 Bloody Foothills, 111 Frigid Highlands, 112 Arreat
// Plateau and 117 Frozen Tundra: DRLG_GenerateAct5Outdoors (0x681640),
// drlg-act45-outdoor.md section 4.2.

// act5Dir is the table at 0x6f3588: the walk direction of each border piece.
var act5Dir = [12][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}, {0, -1}, {1, 0}, {0, 1}, {-1, 0}, {-1, 0}, {0, -1}, {1, 0}, {0, 1}}

// act5Side is the table at 0x6f354c: {level, file, flag, defTall, defWide}.
var act5Side = [3][5]int{{112, 0, 0, 913, 914}, {117, 0, 1, 983, 984}, {117, 0, 0, 985, 986}}

// act5Random is the table at 0x6f36b0: {level, defTall, defWide, file, -, count, fatal}.
var act5Random = [15][7]int{
	{111, 955, 956, 0, 25, 1, 1},
	{112, 955, 956, 0, 25, 1, 1},
	{117, 955, 956, 1, 25, 1, 1},
	{112, 953, 953, -1, 25, 1, 1},
	{117, 954, 954, -1, 25, 1, 1},
	{111, 944, 947, -1, 10, 1, 0},
	{111, 942, 945, -1, 20, 4, 0},
	{111, 943, 946, -1, 20, 4, 0},
	{112, 941, 941, -1, 10, 1, 0},
	{112, 939, 939, -1, 8, 1, 0},
	{112, 940, 940, -1, 15, 5, 0},
	{117, 948, 948, -1, 9, 4, 0},
	{117, 949, 949, -1, 9, 4, 0},
	{117, 950, 950, -1, 9, 4, 0},
	{117, 951, 951, -1, 5, 3, 0},
}

// act5SubMap is the table at 0x6f35e8: {layer, seqLo, seqHi, defNormal, defSnow}.
var act5SubMap = [10][5]int{
	{0x31, 1, 16, 915, 987}, {0x31, 31, 46, 915, 987}, {0x30, 1, 1, 883, 959}, {0x30, 2, 3, 881, 957},
	{0x30, 4, 4, 884, 960}, {0x30, 5, 5, 895, 971}, {0x30, 6, 7, 893, 969}, {0x30, 8, 8, 896, 972},
	{0x30, 30, 30, 0, 0}, {0x30, 31, 31, -5, -5},
}

func (l *Level) generateAct5() {
	id := l.Params.ID

	if id == 110 {
		l.placeBloodyFoothills()
		return
	}

	l.markExits()
	l.drawAct5Edges()
	l.drawAct5InnerBorder()
	l.placeAct5Exits()
	l.placeAct5Side()

	if id == 111 {
		l.placeFrigidGate()
	}

	l.applyAct5Barricades()

	if id == 111 {
		l.placeFrigidWaypointPieces()
	}

	l.placeAct5Random()
}

// placeBloodyFoothills is DRLG_PlaceBloodyFoothillsRow (0x681590): fifteen
// strip presets right to left, explicit file 0.
func (l *Level) placeBloodyFoothills() {
	sx, _, _ := l.prest(0x361)
	w := sar3(sx)
	x := l.W - w

	for i := 0; i < 15; i++ {
		if x < 0 {
			panic(GameError{0x294})
		}

		l.placePreset(x, 0, 0x361+i, 0, 0)
		x -= w
	}
}

// drawAct5Edges is DRLG_DrawAct5BoundaryEdges (0x680cf0).
func (l *Level) drawAct5Edges() {
	id := l.Params.ID
	kind := 4

	if id == 0x75 {
		kind = 5
	}

	head := l.Poly
	a, b := head, head.Next

	for {
		dx1, dy1 := sgn(b.X-a.X), sgn(b.Y-a.Y)
		c := b.Next
		dx2, dy2 := sgn(c.X-b.X), sgn(c.Y-b.Y)
		tx, ty := b.X&^1, b.Y&^1
		x, y := a.X&^1, a.Y&^1
		d := t1ca0[(kind-4)+2*styleOf(dx1+3*dy1)]

		if a.F&2 == 0 {
			for n := 0; x != tx || y != ty; n++ {
				if n > 1000 {
					panic(GameError{0x1001})
				}

				x += dx1 * 2
				y += dy1 * 2
				l.placePreset(x, y, d, -1, 0)
				l.Flag.Op(x, y, 1, opOr)
			}
		}

		if a.F&1 != 0 {
			mx, my := max(a.X, b.X), max(a.Y, b.Y)
			x1, y1 := (mx-4*abs(dx1))&^1, (my-4*abs(dy1))&^1
			l.Flag.Op(x1, y1, cellExit, opOr)
			l.Flag.Op(x1+2*abs(dx1), y1+2*abs(dy1), cellExit, opOr)
		}

		idx := widen(2*dx1) + 9*(widen(2*dx2)+2*dy2) + 2*dy1

		if cc := t2638[idx+45]; cc != -1 {
			if dc := t1ca0[kind+2*cc-6]; dc != 0 {
				l.placePreset(tx, ty, dc, -1, 0)
				l.Flag.Op(tx, ty, 1, opOr)
			}
		}

		if b == head {
			break
		}

		a, b = b, b.Next
	}

	if id == 0x6f {
		l.Flag.Op(l.W-2, l.H-4, cellExit, opOr)
		l.Flag.Op(l.W-2, l.H-3, cellExit, opOr)
	}
}

// drawAct5InnerBorder is DRLG_DrawAct5InnerBorder (0x680f20): it walks the
// border ring and swaps every piece for its inner variant.
func (l *Level) drawAct5InnerBorder() {
	off := 0
	if l.Params.ID == 0x75 {
		off = 0x4c
	}

	base, base2 := 0x371+off, 0x37d+off
	x, y := l.W-2, 0

	for n := 0; x != 0 || y != l.H-2; n++ {
		idx := int(l.Def.Get(x, y)) - base
		if n > 2000 || idx < 0 || idx >= len(act5Dir) {
			panic(GameError{0x1002}) // the exe would wander off; no valid layout gets here
		}

		l.placePreset(x, y, idx+base2, -1, 0)
		x += act5Dir[idx][0] * 2
		y += act5Dir[idx][1] * 2
	}

	l.placePreset(l.W-2, 0, 0x38a+off, -1, 0)
	l.placePreset(0, l.H-2, 0x389+off, -1, 0)
}

// placeAct5Exits is DRLG_PlaceAct5ExitPresets (0x680b50): the first exit-marked
// cell of each of the four scans gets an Entrance/Exit preset.
func (l *Level) placeAct5Exits() {
	W, H := l.W, l.H

	for x := 0; x < W; x++ {
		if l.Flag.Get(x, 0)&cellExit != 0 {
			l.placePreset(x, 0, 0x38d, -1, 0)
			break
		}
	}

	for x := 0; x < W; x++ {
		if l.Flag.Get(x, H-2)&cellExit != 0 {
			l.placePreset(x, H-2, 0x38c, -1, 0)
			break
		}
	}

	for y := 0; y < H; y++ {
		if l.Flag.Get(0, y)&cellExit != 0 {
			l.placePreset(0, y, 0x38e, -1, 0)
			break
		}
	}

	for y := 0; y < H; y++ {
		if l.Flag.Get(W-2, y)&cellExit != 0 {
			l.placePreset(W-2, y, 0x38b, -1, 0)
			break
		}
	}
}

// placeAct5Side is DRLG_PlaceAct5SideFeatures (0x680a80): the cave connectors.
func (l *Level) placeAct5Side() {
	r := l.Params.Rect

	for _, row := range act5Side {
		if row[0] != l.Params.ID {
			continue
		}

		if r.W > r.H {
			x := 0
			if row[2] != 0 {
				x = l.W - 2
			}

			l.placePreset(x, 2, row[4], row[1], 0)
		} else {
			y := 0
			if row[2] != 0 {
				y = l.H - 2
			}

			l.placePreset(2, y, row[3], row[1], 0)
		}
	}
}

// placeFrigidGate is DRLG_PlaceFrigidHighlandsGate (0x6814d0).
func (l *Level) placeFrigidGate() {
	sx, sy, _ := l.prest(0x370)
	x, y := l.W-sar3(sx), l.H-sar3(sy)
	l.placePreset(x, y, 0x370, -1, 0)
	l.placePreset(x, y-2, 0x380, -1, 0)
}

// mapAct5Sub is DRLG_MapAct5SubPieceToDef (0x681030).
func (l *Level) mapAct5Sub(layer, seq int) int {
	if layer != 0x30 && layer != 0x31 {
		panic(GameError{0x184})
	}

	for _, r := range act5SubMap {
		if r[0] == layer && r[1] <= seq && seq <= r[2] {
			def := r[3]
			if l.Params.ID == 0x75 {
				def = r[4]
			}

			return def - r[1] + seq
		}
	}

	panic(GameError{0x1a5})
}

// subCallbacks replace the Act 1 per-cell rules of the LvlSub matcher.
type subCallbacks struct {
	check func(row d2drlg.SubRec, d *Pattern, g Group, x, y int) bool
	stamp func(row d2drlg.SubRec, d *Pattern, g Group, x, y, val int)
}

// checkAct5Group is DRLG_CheckAct5SubPieceCell (0x6810d0) over a group.
func (l *Level) checkAct5Group(row d2drlg.SubRec, d *Pattern, g Group, x, y int) bool {
	sc := row.GridSize
	x0, y0 := x-x%sc, y-y%sc

	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			wl := d.wall(g.X+gx, g.Y+gy)
			tx, ty := sc*gx+x0, sc*gy+y0
			v := int(l.Def.Get(tx, ty))
			r := l.mapAct5Sub(int(wl>>20&0x3f), int(wl>>8&0xff))

			if r == -5 {
				continue
			}

			if r != v || l.Flag.Get(tx, ty)>>10&1 != 0 {
				return false
			}
		}
	}

	return true
}

func (l *Level) stampAct5Group(row d2drlg.SubRec, d *Pattern, g Group, x, y, val int) {
	sc := row.GridSize
	x0, y0 := x-x%sc, y-y%sc

	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			wl := d.wall(g.X+val+gx, g.Y+gy)
			fl := d.floor(g.X+val+gx, g.Y+gy)
			tx, ty := sc*gx+x0, sc*gy+y0

			switch {
			case wl&1 != 0:
				seq := int(wl >> 8 & 0xff)
				if r := l.mapAct5Sub(int(wl>>20&0x3f), seq); r != -5 && seq-1 != subSentinel {
					l.placePreset(tx, ty, r, 0, 1)
				}
			case fl&2 != 0:
				l.Def.Set(tx, ty, 0)
				l.Flag.Set(tx, ty, 0)
			default:
				l.Def.Set(tx, ty, 0)
				l.Flag.Set(tx, ty, cellNoRoom)
			}
		}
	}
}

// applyAct5Barricades is DRLG_ApplyAct5BarricadeSubs (0x681120): LvlSub type 12
// with the Act 5 callbacks.
func (l *Level) applyAct5Barricades() {
	l.sub = &subCallbacks{check: l.checkAct5Group, stamp: l.stampAct5Group}
	defer func() { l.sub = nil }()

	l.applyLvlSub(12, 0)
}

// placeFrigidWaypointPieces is DRLG_PlaceFrigidHighlandsWaypointPieces (0x681270).
func (l *Level) placeFrigidWaypointPieces() {
	W, H := l.W, l.H
	n := 0

	for k := 0; k < 0x5a && n < 3; k++ {
		x := 2 * int(l.Seed.Roll(int32(cdiv(W, 2))))
		y := 2 * int(l.Seed.Roll(int32(cdiv(H, 2))))

		if v := l.presetCellDef(x, y); v >= 0x393 && v < 0x393+8 {
			l.placePreset(x, y, v+0x10, -1, 0)
			n++
		}
	}

	ox := int(l.Seed.Roll(int32(cdiv(W, 2))))
	oy := int(l.Seed.Roll(int32(cdiv(H, 2))))

	for yy := 0; yy < H && n < 3; yy++ {
		for xx := 0; xx < W && n < 3; xx++ {
			X, Y := (xx+ox*2)%W, (yy+oy*2)%H

			if v := l.presetCellDef(X, Y); v >= 0x393 && v < 0x393+8 {
				l.placePreset(X, Y, v+0x10, -1, 0)
				n++
			}
		}
	}

	if n < 3 {
		panic(GameError{0x259})
	}
}

// presetCellDef is DRLG_GetPresetCellDef (0x676e80).
func (l *Level) presetCellDef(x, y int) int {
	if l.Flag.Get(x, y)&cellPreset == 0 {
		return 0
	}

	return int(l.Def.Get(x, y))
}

// placeAct5Random is DRLG_PlaceAct5RandomFeatures (0x6811a0).
func (l *Level) placeAct5Random() {
	r := l.Params.Rect
	tall := r.W < r.H

	for _, row := range act5Random {
		if row[0] != l.Params.ID {
			continue
		}

		def := row[2]
		if tall {
			def = row[1]
		}

		ok := false

		for i := 0; i < row[5]; i++ {
			ok = l.placeRandom(def, row[3], 0, 0xf)
		}

		if row[5] > 0 && ok {
			continue
		}

		if row[6] != 0 {
			panic(GameError{0x219})
		}
	}
}
