package drlgoutdoor

func (l *Level) prest(def int) (sx, sy, files int) {
	rec, ok := l.env.Tables.PrestByDef(def)
	if !ok {
		panic(GameError{0x1000 + def})
	}

	return rec.SizeX, rec.SizeY, rec.Files
}

// nextFile is DRLG_GetNextOutdoorPresetFile (0x677070): a per-Def counter that
// starts at Roll(Files) on first use and advances by one on every call.
func (l *Level) nextFile(def int) int {
	_, _, files := l.prest(def)
	if files <= 0 {
		return 0 // never reached in the game (it would divide by zero)
	}

	c := l.ctr[def]
	if c == nil {
		c = &counter{n: files, ctr: int(l.Seed.Roll(int32(files)))}
		l.ctr[def] = c
	}

	c.ctr = (c.ctr + 1) % c.n

	return c.ctr
}

// placePreset is DRLG_PlaceOutdoorPreset (0x677100). file -1 draws the next
// file of the Def. Cells outside the grid are skipped (the original writes
// anyway; valid layouts never get there).
func (l *Level) placePreset(x, y, def, file, flag int) {
	sx, sy, _ := l.prest(def)
	w, h := sar3(sx), sar3(sy)

	if file == -1 {
		file = l.nextFile(def)
	}

	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !l.Flag.In(xx, yy) {
				continue
			}

			l.Flag.Op(xx, yy, cellFileMask, opAndNot)
			l.Flag.Op(xx, yy, uint32(file)<<16|cellPreset, opOr)

			if flag != 0 && ((def >= 4 && def <= 15) || (def >= 0x16c && def <= 0x177)) {
				l.Flag.Op(xx, yy, cellOccupied, opOr)
			}

			l.Def.Set(xx, yy, 0)
		}
	}

	if l.Def.In(x, y) {
		l.Def.Set(x, y, uint32(def))
	}
}

func (l *Level) kind(b int) int {
	switch l.LType {
	case 2:
		if b == 0 {
			return 1
		}

		return 0
	case 0x10:
		return 2
	case 0x1b:
		return 3
	case 0x1f:
		if l.Params.ID == 0x75 {
			return 5
		}

		return 4
	}

	return -1
}

func widen(v int) int {
	switch {
	case v < 0:
		return v - 2
	case v > 0:
		return v + 2
	}

	return 0
}

func styleOf(idx int) int {
	switch idx {
	case -3:
		return 1
	case -1:
		return 0
	case 1:
		return 2
	case 3:
		return 3
	}

	return -1
}

// drawBoundaryEdges is DRLG_DrawBoundaryEdges (0x678560), the Act 1 branch.
func (l *Level) drawBoundaryEdges() {
	vs := l.verts()

	for i, a := range vs {
		b := vs[(i+1)%len(vs)]
		c := b.Next
		dx1, dy1 := sgn(b.X-a.X), sgn(b.Y-a.Y)
		dx2, dy2 := sgn(c.X-b.X), sgn(c.Y-b.Y)

		var length int
		if dx1 != 0 {
			length = abs(a.X - b.X)
		} else {
			length = abs(a.Y - b.Y)
		}

		val := uint32(1)
		if a.B != 0 {
			val = 3
		}

		k := l.kind(a.B)
		s := styleOf(dx1 + 3*dy1)

		var d int
		if k >= 4 {
			d = t1ca0[(k-4)+2*s]
		} else {
			d = t1be0[k+4*s]
		}

		if a.F&2 == 0 {
			x, y := a.X, a.Y
			for x != b.X || y != b.Y {
				y += dy1
				x += dx1
				l.placePreset(x, y, d, -1, 0)
				l.Flag.Op(x, y, val, opOr)
			}
		}

		if a.F&1 != 0 && a.F&2 == 0 {
			mx := min(a.X, b.X) + cdiv(length*abs(dx1), 2)
			my := min(a.Y, b.Y) + cdiv(length*abs(dy1), 2)
			if l.LType == 0x10 { // Act 2: two door pieces instead of exit markers
				l.act2Notch(mx, my, dx1, dy1)
			} else {
				l.Flag.Op(mx, my, cellFileMask, opAndNot)

				if l.Params.ID == 0x11 {
					l.Flag.Op(mx, my, 0x40400, opOr)
				} else {
					l.Flag.Op(mx, my, 0x30400, opOr)
				}
			}
		}

		b12 := a.B
		if b12 == 0 {
			b12 = b.B
		}

		if a.B != 0 || b.B != 0 {
			val |= 2
		}

		k2 := l.kind(b12)

		var p, r, q int

		if a.F&2 != 0 {
			p, r = widen(dx1), dy1
		} else {
			p, r = widen(2*dx1), 2*dy1
		}

		if b.F&2 != 0 {
			q = dy2 + widen(dx2)
		} else {
			q = widen(2*dx2) + 2*dy2
		}

		cc := t2638[p+9*q+r+45]
		if cc == -1 {
			continue
		}

		var dc int
		if k2 < 4 {
			dc = t1bd0[k2+4*cc]
		} else {
			dc = t1ca0[(k2-6)+2*cc]
		}

		switch dc {
		case 0x13:
			if a.B == 1 {
				dc = 0x13
				if b.B != 1 {
					dc = 0x14
				}
			} else {
				dc = 0x15
			}
		case 0:
			continue
		}

		l.placePreset(b.X, b.Y, dc, -1, 0)
		l.Flag.Op(b.X, b.Y, val, opOr)
	}

	l.floodNoRoom()
}

// floodNoRoom is DRLG_FlagOutsideCellsNoRoom (0x678360).
func (l *Level) floodNoRoom() {
	for _, c := range [4][4]int{{0, 0, 1, 1}, {1, 0, -1, 1}, {0, 1, 1, -1}, {1, 1, -1, -1}} {
		x0, y := 0, 0
		if c[0] != 0 {
			x0 = l.W - 1
		}

		if c[1] != 0 {
			y = l.H - 1
		}

		for l.Flag.Get(x0, y)&1 == 0 {
			x := x0
			for l.Flag.Get(x, y)&1 == 0 {
				l.Flag.Op(x, y, cellNoRoom, opOr)
				x += c[2]
			}

			y += c[3]
		}
	}
}
