package drlgoutdoor

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"

const subSentinel = 0x3e

// applyLvlSub is DRLG_ApplyLvlSubType (0x677f90): run every LvlSub row of the
// type through the border/cliff matcher.
func (l *Level) applyLvlSub(typ, base int) {
	for _, row := range l.env.Tables.SubRows(typ) {
		d, err := l.env.Pattern(row.File)
		if err != nil {
			panic(err)
		}

		l.matchRow(row, d, base)
	}
}

func (l *Level) matchRow(row d2drlg.SubRec, d *Pattern, base int) {
	n := len(d.Groups)
	if n == 0 {
		return
	}

	start := 0
	if row.BordType == 0 {
		start = int(l.Seed.Roll(int32(n)))
	}

	for i := 0; i < n; i++ {
		placed := l.matchGroup(row, d, d.Groups[(start+i)%n], base)
		if placed && row.BordType == 0 {
			return
		}
	}
}

func (l *Level) matchGroup(row d2drlg.SubRec, d *Pattern, g Group, base int) bool {
	id := l.Params.ID
	wild := id >= 2 && id <= 7
	ebx := row.Type == 1

	delta := 1
	if ebx && l.OdFlags&0xc != 0 {
		delta = -1
	}

	sc := row.GridSize
	nX := l.W - g.W*sc + delta
	nY := l.H - g.H*sc + 1
	n := nX * nY

	if n == 0 {
		return false
	}

	skip := ebx && wild && nX < 6 && nY < 6

	type pt struct{ x, y int }

	lst := make([]pt, max(n, 0))
	for k := range lst {
		lst[k] = pt{k % nX, k / nX}
	}

	for i := 0; i < n; i++ {
		a := int(l.Seed.Roll(int32(n)))
		b := int(l.Seed.Roll(int32(n)))
		lst[a], lst[b] = lst[b], lst[a]
	}

	for _, p := range lst {
		if skip && p.x == 2 && p.y == 2 {
			continue
		}

		if l.checkGroup(row, d, g, base, p.x, p.y) {
			r := 0
			if g.NVar >= 1 {
				r = int(l.Seed.Roll(int32(g.NVar)))
			}

			l.stampGroup(row, d, g, base, p.x, p.y, (g.W+1)*(r+1))

			if row.BordType == 0 || row.BordType == 1 {
				return true
			}
		}
	}

	return false
}

func (l *Level) checkGroup(row d2drlg.SubRec, d *Pattern, g Group, base, x, y int) bool {
	sc := row.GridSize
	x0, y0 := x-x%sc, y-y%sc

	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			fl := d.floor(g.X+gx, g.Y+gy)
			wl := d.wall(g.X+gx, g.Y+gy)
			tx, ty := sc*gx+x0, sc*gy+y0
			v := int(l.Def.Get(tx, ty))

			switch {
			case wl&1 != 0:
				idx := int(wl>>8&0xff) - 1
				if idx != subSentinel && base+idx != v {
					return false
				}

				if l.Flag.Get(tx, ty)&cellExit != 0 {
					return false
				}
			case fl&2 != 0:
				if !l.Flag.In(tx, ty) || l.Flag.Get(tx, ty)&cellBlocked != 0 {
					return false
				}
			}
		}
	}

	return true
}

func (l *Level) stampGroup(row d2drlg.SubRec, d *Pattern, g Group, base, x, y, val int) {
	sc := row.GridSize
	x0, y0 := x-x%sc, y-y%sc

	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			wl := d.wall(g.X+val+gx, g.Y+gy)
			fl := d.floor(g.X+val+gx, g.Y+gy)
			tx, ty := sc*gx+x0, sc*gy+y0

			switch {
			case wl&1 != 0:
				idx := int(wl>>8&0xff) - 1
				if idx != subSentinel {
					l.placePreset(tx, ty, base+idx, 0, 1)
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
