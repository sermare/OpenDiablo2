package drlgoutdoor

func (l *Level) free(x, y int) bool { return l.Flag.Get(x, y)&cellBlocked == 0 }

// fits is DRLG_CheckOutdoorPresetFits.
func (l *Level) fits(x, y, def, pad, mask int) bool {
	w, h := 1, 1

	if def != 0 {
		sx, sy, _ := l.prest(def)
		w, h = sar3(sx), sar3(sy)
	}

	lx := x

	if pad != 0 {
		if mask&1 != 0 {
			y -= pad
			h += pad
		}

		if mask&2 != 0 {
			w += pad
		}

		if mask&4 != 0 {
			h += pad
		}

		if mask&8 != 0 {
			lx = x - pad
			w += pad
		}
	}

	for yy := y; yy < y+h; yy++ {
		for xx := lx; xx < lx+w; xx++ {
			if !l.Flag.In(xx, yy) || l.Flag.Get(xx, yy)&cellBlocked != 0 {
				return false
			}
		}
	}

	return true
}

type pt struct{ x, y int }

// shuffle is DRLG_ShuffleCells: the interior cells in random order.
func (l *Level) shuffle() []pt {
	iw, ih := l.W-2, l.H-2
	n := iw * ih
	lst := make([]pt, max(n, 0))

	for k := range lst {
		lst[k] = pt{k % iw, k / iw}
	}

	for i := 0; i < n; i++ {
		a := int(l.Seed.Roll(int32(n)))
		b := int(l.Seed.Roll(int32(n)))
		lst[a], lst[b] = lst[b], lst[a]
	}

	return lst
}

func (l *Level) placeRandom(def, file, pad, mask int) bool {
	for _, c := range l.shuffle() {
		x, y := c.x+1, c.y+1
		if l.fits(x, y, def, pad, mask) {
			l.placePreset(x, y, def, file, 0)
			return true
		}
	}

	return false
}

var (
	ndx = [8]int{-1, 0, 0, 1, -1, 1, 1, -1}
	ndy = [8]int{0, -1, 1, 0, -1, 1, -1, 1}
)

func (l *Level) placeNearExit(def, file int) bool {
	for _, c := range l.shuffle() {
		x1, y1 := c.x+1, c.y+1
		if l.Flag.Get(x1, y1)&cellRoad == 0 {
			continue
		}

		for j := 0; j < 8; j++ {
			x, y := x1+ndx[j], y1+ndy[j]
			if l.fits(x, y, def, 0, 0xf) {
				l.placePreset(x, y, def, file, 0)
				return true
			}
		}
	}

	return l.placeRandom(def, file, 0, 0xf)
}

func (l *Level) placeFarthest(rect Rect, def, file, pad, mask int) bool {
	iw, ih := l.W-2, l.H-2
	ox := int(l.Seed.Roll(int32(iw)))
	oy := int(l.Seed.Roll(int32(ih)))
	best, bx, by := -1, 0, 0

	for dy := 0; dy <= ih; dy++ {
		for dx := 0; dx <= iw; dx++ {
			y := (dy+oy)%ih + 1
			x := (dx+ox)%iw + 1

			if !l.fits(x, y, def, pad, mask) {
				continue
			}

			dxp := abs(x*8 - (cdiv(rect.W, 2) + rect.X) + 4 + l.Params.Rect.X)
			dyp := abs(y*8 - (cdiv(rect.H, 2) + rect.Y) + 4 + l.Params.Rect.Y)

			var m int
			if dyp < dxp {
				m = dyp + 2*dxp
			} else {
				m = dxp + 2*dyp
			}

			m = cdiv(m, 2)
			if m > best {
				best, bx, by = m, x, y
			}
		}
	}

	if best >= 0 {
		l.placePreset(bx, by, def, file, 0)
		return true
	}

	return false
}

// placeCounted is DRLG_PlaceCountedFeature (0x6834f0).
func (l *Level) placeCounted(def int, flag bool) {
	r := l.Seed.Step()
	if r&3 == 0 {
		l.placeNearExit(def, -1)
		l.placeNearExit(def, -1)

		return
	}

	l.placeNearExit(def, -1)

	if flag {
		if l.Seed.Step()&1 != 0 {
			l.placeNearExit(0x31, -1)
		}
	}
}

// placeSpecials is DRLG_PlaceLevelSpecials (0x683590).
func (l *Level) placeSpecials() {
	p := func(def int) { l.placeRandom(def, -1, 0, 0xf) }
	n := func(def int) { l.placeNearExit(def, -1) }
	c := l.placeCounted
	tail := func() { p(0x1d); p(0x1e) }

	switch l.Params.ID {
	case 2:
		n(0x2e)
		c(0x2f, false)
		tail()
	case 3:
		c(0x30, true)
		p(0x2c)
		tail()
	case 4:
		n(0xa0)
		n(0x2d)
		p(0xa2)
		c(0x2f, true)
		c(0x2a, false)
		p(0x1f)
	case 5:
		p(0xa1)
		p(0x29)
		p(0x28)
		c(0x30, true)
		c(0x2b, false)
		tail()
	case 6:
		p(0xa3)
		p(0x26)
		p(0x27)
		c(0x2f, true)
		c(0x2a, false)
		tail()
	case 7:
		c(0x30, true)
		c(0x2b, false)
		p(0x1f)
	case 0x11:
		l.placePreset(1, 1, 0x6c, -1, 0)
	case 0x27:
		p(0x32)
		p(0x2e)
		p(0x1f)
		p(0x26)
		p(0x27)
		tail()
	}
}

// scatterShrines is DRLG_ScatterShrineSites (0x677b50).
func (l *Level) scatterShrines(count int) {
	rot := l.Seed.Step() & 3
	bits := [4]uint32{0x1000, 0x2000, 0x4000, 0x8000}

	for _, c := range l.shuffle() {
		if count <= 0 {
			break
		}

		x, y := c.x+1, c.y+1
		if l.Flag.Get(x, y)&cellBlocked == 0 {
			l.GridB.Op(x, y, bits[rot], opOr)
			l.Flag.Op(x, y, cellShrine, opOr)
			rot = (rot + 1) & 3
			count--
		}
	}
}

// markWaypoint is DRLG_MarkWaypointSite (0x6778a0).
func (l *Level) markWaypoint() {
	if l.Params.ID == 3 {
		k := 8

		for i, id := range l.Params.Vis {
			if id == 2 {
				k = i
				break
			}
		}

		mask := uint32(1) << uint(4+k)

		for y := 0; y < l.H; y++ {
			for x := 0; x < l.W; x++ {
				if l.GridB.Get(x, y)&mask != 0 && l.Flag.Get(x, y)&cellExit != 0 {
					xx, yy := x, y
					if x == 0 {
						xx = 1
					}

					if y == 0 {
						yy = 1
					}

					if xx == l.W-1 {
						xx--
					}

					if yy == l.H-1 {
						yy--
					}

					l.GridB.Op(xx, yy, 0x20000, opOr)
					l.Flag.Op(xx, yy, cellWaypoint, opOr)

					return
				}
			}
		}
	}

	for _, c := range l.shuffle() {
		x, y := c.x+1, c.y+1
		if l.Flag.Get(x, y)&cellBlocked == 0 {
			l.GridB.Op(x, y, 0x10000, opOr)
			l.Flag.Op(x, y, cellWaypoint, opOr)

			return
		}
	}
}

// riverColumnFree is DRLG_IsRiverColumnFree (0x682cd0).
func (l *Level) riverColumnFree(col int) bool {
	for y := 0; y < l.H; y++ {
		if l.Flag.Get(col, y)&2 != 0 || l.Flag.Get(col+1, y)&2 != 0 {
			return false
		}
	}

	return true
}

// drawRiverBorder is DRLG_DrawRiverBorderPresets (0x682ed0).
func (l *Level) drawRiverBorder(col int) {
	side := func(c, y, which int) int {
		f := l.Flag.Get(c, y)
		d := int(l.Def.Get(c, y))

		switch {
		case d == 0:
			if f&cellNoRoom != 0 {
				return 0
			}

			return 3
		case d == 7 && f&cellFileMask == 0x30000:
			return 3
		}

		return t3c30[2*d+which]
	}

	for y := 0; y < l.H; y++ {
		l.placePreset(col, y, 0x1a, side(col, y, 0), 0)
		l.placePreset(col+1, y, 0x1b, side(col+1, y, 1), 0)
	}

	if l.OdFlags&0x14 != 0 {
		l.placeRiverBridge(col)
	}
}

// placeRiverBridge is DRLG_PlaceRiverBridge (0x682d70).
func (l *Level) placeRiverBridge(col int) {
	n := l.H - 2
	r := 0

	if n > 0 {
		r = int(l.Seed.Roll(int32(n)))
	}

	for k := 0; k < n; k++ {
		yy := (r+k)%n + 1

		if l.OdFlags&4 != 0 {
			if !l.free(col-1, yy) {
				continue
			}
		} else if !l.free(col-1, yy) || !l.free(col+2, yy) {
			continue
		}

		if l.Flag.Get(col, yy)&cellFileMask != 0x30000 || l.Flag.Get(col+1, yy)&cellFileMask != 0x30000 {
			continue
		}

		l.placePreset(col, yy, 0x1c, 1, 0)

		f := 2
		if l.OdFlags&4 != 0 {
			f = 3
		}

		l.placePreset(col+1, yy, 0x1c, f, 0)

		return
	}
}

func (l *Level) cliffCaveAt(x, y int) bool {
	var d int

	switch l.Def.Get(x, y) {
	case 0x10:
		d = 0x19
	case 0x11:
		d = 0x18
	default:
		return false
	}

	l.placePreset(x, y, d, -1, 0)
	l.OdFlags |= FlagCavePlaced

	return true
}

// caveEntrances is DRLG_PlaceAct1CaveEntrances (0x683240).
func (l *Level) caveEntrances() error {
	if l.Params.ID == 0x27 {
		return nil
	}

	if l.OdFlags&0xc != 0 && l.riverColumnFree(l.W-2) {
		l.drawRiverBorder(l.W - 2)
	}

	if l.OdFlags&FlagCliffCorner != 0 && l.OdFlags&FlagCavePlaced == 0 {
		r := l.Seed.Step()
		done := false

		if r&1 != 0 {
			// transposed walk: x runs to H, y to W; Get(x, y) aliases into the next row
			for xc := 0; xc < l.H && !done; xc++ {
				for yc := 0; yc < l.W && !done; yc++ {
					done = l.cliffCaveAt(xc, yc)
				}
			}
		} else {
			for y := 0; y < l.H && !done; y++ {
				for x := 0; x < l.W && !done; x++ {
					done = l.cliffCaveAt(x, y)
				}
			}
		}

		if !done {
			return GameError{0x194}
		}
	}

	if l.OdFlags&0x1c != 0 && l.OdFlags&FlagCavePlaced == 0 {
		k := 4 | (^(l.OdFlags >> 4) & 1)
		xr, yr := l.W-k, l.H-4
		r := l.Seed.Step()
		x, y := xr, yr

		if r&1 != 0 {
			x = 3
		}

		if (r>>1)&1 != 0 {
			y = 3
		}

		def := 0x33
		if l.Params.ID == 2 {
			def = 0x34
		}

		l.placePreset(x, y, def, -1, 0)
		l.OdFlags |= FlagCavePlaced
	}

	return nil
}

// townTransitions is DRLG_PlaceAct1TownTransitions (0x6833e0).
func (l *Level) townTransitions() error {
	if l.Params.ID == 0x27 {
		return nil
	}

	if l.OdFlags&0x10 != 0 && l.riverColumnFree(l.W/2-1) {
		l.drawRiverBorder(l.W/2 - 1)
	}

	if l.OdFlags&FlagTownTransS0 != 0 {
		l.placePreset(0, 0, 3, 1, 0)
	}

	if l.OdFlags&FlagTownTransS1 != 0 {
		l.placePreset(l.W-7, 0, 3, 2, 0)
	}

	if l.OdFlags&FlagTownTransE0 != 0 {
		l.placePreset(0, 1, 2, 1, 0)
	}

	if l.OdFlags&FlagTownTransE1 != 0 {
		l.placePreset(0, l.H-6, 2, 1, 0)
	}

	if l.OdFlags&FlagCavePlaced == 0 {
		var ok bool
		if l.Params.ID == 2 {
			ok = l.placeFarthest(l.town, 0x34, -1, 1, 0xf)
		} else {
			ok = l.placeRandom(0x33, -1, 1, 0xf)
		}

		if !ok {
			return GameError{0x1e5}
		}

		l.OdFlags |= FlagCavePlaced
	}

	return nil
}
