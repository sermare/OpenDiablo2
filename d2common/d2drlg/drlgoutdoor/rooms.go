package drlgoutdoor

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// ltFlags is the initial Dt1Mask of a plain room by LevelType (0x677dd0).
func ltFlags(lt int) uint32 {
	switch lt {
	case 2:
		return 0x44103
	case 0x10, 0x16, 0x1b, 0x1c:
		return 1
	case 0x15:
		return 4
	case 0x1e, 0x1f:
		return 0x11
	}

	return 0
}

// allocRoom is DRLG_AllocRoomEx: one level-seed step seeds the room seed, one
// room-seed step gives room+4.
func (l *Level) allocRoom(typ int) *Room {
	lo := l.Seed.Step()
	r := &Room{Type: typ}
	r.Seed.Init(lo)
	r.S4 = r.Seed.Step()

	return r
}

// rollSubThemeMask is DRLG_RollSubThemeMask (0x6733e0).
func (l *Level) rollSubThemeMask(r *Room, typ, theme int) {
	if typ == -1 || theme == -1 {
		return
	}

	for b, row := range l.env.Tables.SubRows(typ) {
		if r.Seed.Step()%100 < uint32(row.Prob[theme]) {
			r.Mask |= 1 << uint(b)
			r.R50 |= uint32(row.Dt1Mask)
		}
	}
}

// createRooms is DRLG_CreateOutdoorRooms (0x677dd0). The golden lists rooms in
// creation order.
func (l *Level) createRooms() {
	rec, _ := l.env.Tables.Level(l.Params.ID)
	ltf := ltFlags(l.LType)

	for yc := 0; yc < l.H; yc++ {
		for xc := 0; xc < l.W; xc++ {
			tx, ty := l.Params.Rect.X+8*xc, l.Params.Rect.Y+8*yc
			fl, gB, gD := l.Flag.Get(xc, yc), l.GridB.Get(xc, yc), l.GridD.Get(xc, yc)

			switch {
			case fl&cellPreset != 0:
				if d := int(l.Def.Get(xc, yc)); d != 0 {
					_, _, files := l.prest(d)
					l.Seed.Roll(int32(files)) // DRLG_AllocPresetMap draw, value discarded
					l.placePresetRooms(d, int(fl>>16)&0xf, tx, ty, gB)
				}
			case fl&cellNoRoom == 0:
				r := l.allocRoom(1)
				r.X, r.Y, r.W, r.H = tx, ty, 8, 8
				r.Flags = gB | 0x80000
				r.R50, r.Info54, r.Info58 = ltf, fl, gD
				r.SubType, r.SubTheme = rec.SubType, rec.SubTheme
				l.rollSubThemeMask(r, rec.SubType, rec.SubTheme)
				l.Rooms = append(l.Rooms, r)
			}
		}
	}
}

// presetChunkBits are the room flags the DS1 wall cells of orientation 10/11
// contribute per 8x8 chunk (drlg3.md section 10, DRLG_LoadPresetDs1ForRooms).
func presetChunkBits(d *Pattern, cx, cy int) uint32 {
	var bits uint32

	for k := range d.Wall {
		for y := cy * 8; y < min(d.H, cy*8+8); y++ {
			for x := cx * 8; x < min(d.W, cx*8+8); x++ {
				o := d.Orient[k][y*d.W+x]
				if o != 10 && o != 11 {
					continue
				}

				w := d.Wall[k][y*d.W+x]
				seq, st := w>>8&0xff, w>>20&0x3f

				if st < 8 && (seq == 0 || seq == 4 || w&0x80000000 != 0) {
					bits |= 1 << (st + 4)
				}
			}
		}
	}

	return bits
}

// NormalizePrestFile turns a LvlPrest file column into the form the loader
// expects ("Act1/Outdoors/x.ds1"): the compiled table stores the full
// "data\global\tiles\" path.
func NormalizePrestFile(f string) string {
	f = strings.ReplaceAll(f, "\\", "/")
	if i := strings.Index(strings.ToLower(f), "data/global/tiles/"); i >= 0 {
		f = f[i+len("data/global/tiles/"):]
	}

	return f
}

// placePresetRooms is DRLG_PlacePresetRooms (0x66ab50) for outdoor presets: one
// room per 8x8 chunk of the preset, each costing one level-seed step.
func (l *Level) placePresetRooms(def, file, tx, ty int, gB uint32) {
	rec, _ := l.env.Tables.PrestByDef(def)

	flags := gB
	if rec.Outdoors != 0 {
		flags |= 0x80000
	}

	for k := 0; k < 8; k++ {
		if l.Params.Vis[k] != 0 && l.Params.Warp[k] == -1 {
			flags |= 0x10 << uint(k)
		}
	}

	if rec.Populate == 0 {
		flags |= 0x800000
	}

	var ds1 *Pattern

	gateSeed, gateSteps := *l.Seed, 0

	if file >= 0 && file < len(rec.File) {
		// DS1 object RNG gate (0x66a230): some files cost level-seed steps
		// when they are loaded, before the chunk rooms are allocated.
		gateSteps = ds1GateSteps(rec.File[file])

		for i := gateSteps; i > 0; i-- {
			l.Seed.Step()
		}
	}

	if file >= 0 && file < len(rec.File) && rec.File[file] != "" {
		if p, err := l.env.Pattern(NormalizePrestFile(rec.File[file])); err == nil {
			ds1 = p
		}
	}

	w, h := 0, 0
	if rec.SizeX != 0 && rec.SizeY != 0 {
		w, h = rec.SizeX, rec.SizeY
	}

	if l.presetSize != [2]int{} { // town levels tile their own rectangle
		w, h = l.presetSize[0], l.presetSize[1]
	}

	remY := h

	for cy := ty; cy < ty+h; cy += 8 {
		remX := w

		for cx := tx; cx < tx+w; cx += 8 {
			r := l.allocRoom(2)
			r.X, r.Y, r.W, r.H = cx, cy, min(8, remX), min(8, remY)
			r.PrestDef, r.File = def, file
			r.PrestX, r.PrestY, r.PrestW, r.PrestH = tx, ty, w, h
			r.GateSeed, r.GateSteps = gateSeed, gateSteps
			r.Flags = flags
			r.R50 = uint32(rec.Dt1Mask)

			if ds1 != nil {
				r.Flags |= presetChunkBits(ds1, (cx-tx)/8, (cy-ty)/8)

				if waypointChunk(ds1, (cx-tx)/8, (cy-ty)/8) {
					r.Flags |= 0x30000 // a waypoint object in this chunk
				}
			}

			l.Rooms = append(l.Rooms, r)
			remX -= 8
		}

		remY -= 8
	}
}

// ---- plain room tile grids (DRLG_BuildOutdoorRoomGrids, 0x6802b0) ----

// RoomGrids are the A (orientation), B (wall) and C (floor) layers of a room,
// each (w+1) x (h+1), plus the room seed after the build.
type RoomGrids struct {
	A, B, C *Grid
	Seed    d2rand.Seed
	// Tags counts the random tile markers met (each costs one PickTile).
	Tags int
}

// RoomBuildOptions customise the room build.
type RoomBuildOptions struct {
	// PickTile stands in for DRLG_PickRandomTile at a random-tile marker of a
	// sub-theme pattern. The default consumes exactly one room-seed step,
	// which is what the golden assumes (the DT1 library is not emulated).
	// TODO: implement with the DT1 tile library once available.
	PickTile func(rs *d2rand.Seed, x, y int, dword uint32)

	// tiles, when set, makes the build create the real tile records (the
	// random tile markers pick from the room's DT1 library); see BuildTiles.
	tiles *tileBuilder
}

type roomBuilder struct {
	l    *Level
	r    *Room
	g    *RoomGrids
	opts RoomBuildOptions
}

// BuildRoomGrids builds the grids of a plain (type 1) room.
func (l *Level) BuildRoomGrids(r *Room, opts *RoomBuildOptions) (grids *RoomGrids, err error) {
	defer func() {
		if x := recover(); x != nil {
			if e, ok := x.(error); ok {
				grids, err = nil, e
			} else {
				panic(x)
			}
		}
	}()

	b := &roomBuilder{l: l, r: r, g: &RoomGrids{A: NewGrid(r.W+1, r.H+1), B: NewGrid(r.W+1, r.H+1), C: NewGrid(r.W+1, r.H+1)}}
	if opts != nil {
		b.opts = *opts
	}

	if b.opts.tiles != nil {
		b.opts.tiles.rs = &b.g.Seed
	}

	return b.build(), nil
}

func (b *roomBuilder) build() *RoomGrids {
	g, r, l := b.g, b.r, b.l
	g.Seed.Init(r.S4)

	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			g.C.Set(x, y, 0x40002)
		}
	}

	b.roadTiles()

	rec, _ := l.env.Tables.Level(l.Params.ID)

	if wp := int(r.Flags>>16) & 3; wp != 0 {
		b.placeSubType(rec.SubWaypoint, 0, uint32(wp))
	}

	if sh := int(r.Flags>>12) & 0xf; sh != 0 {
		b.placeSubType(rec.SubShrine, 0, uint32(sh))
	}

	b.placeSubType(r.SubType, r.SubTheme, r.Mask)

	for _, gr := range []*Grid{g.B, g.C} {
		for x := 0; x < gr.W; x++ {
			gr.Op(x, 0, 4, opOr)
			gr.Op(x, gr.H-1, 4, opOr)
		}

		for y := 1; y < gr.H-1; y++ {
			gr.Op(0, y, 4, opOr)
			gr.Op(gr.W-1, y, 4, opOr)
		}
	}

	b.levelTypeFill()

	return g
}

// drawThickLine is DRLG_DrawThickLine (0x67f880) with op OR.
func drawThickLine(g *Grid, a, b [2]int, rect Rect, val uint32, brush int) {
	x, y := a[0], a[1]
	dx, dy := b[0]-a[0], b[1]-a[1]
	sx, sy := 1, 1

	if dx < 0 {
		sx = -1
	}

	if dy < 0 {
		sy = -1
	}

	adx, ady := abs(dx), abs(dy)

	put := func(px, py int) {
		if rect.X <= px && px < rect.X+rect.W && rect.Y <= py && py < rect.Y+rect.H {
			g.Op(px-rect.X, py-rect.Y, val, opOr)
		}
	}

	err := 0

	if adx < ady {
		for k := 0; k < brush; k++ {
			put(x+k, y)
		}

		for i := 0; i < ady; i++ {
			err += adx
			y += sy

			if err > ady {
				x += sx
				err -= ady
			}

			for k := 0; k < brush; k++ {
				put(x+k, y)
			}
		}
	} else {
		for k := 0; k < brush; k++ {
			put(x, y+k)
		}

		for i := 0; i < adx; i++ {
			x += sx
			err += ady

			if err > adx {
				y += sy
				err -= adx
			}

			for k := 0; k < brush; k++ {
				put(x, y+k)
			}
		}
	}
}

// roadTiles is DRLG_BuildRoomRiverTiles (0x683c90): the dirt road crossing a room.
func (b *roomBuilder) roadTiles() {
	r, g := b.r, b.g
	rg := NewGrid(r.W+3, r.H+3)
	rect := Rect{r.X - 1, r.Y - 1, r.W + 3, r.H + 3}

	for _, path := range b.l.River.Paths {
		for i := 0; i+1 < len(path); i++ {
			drawThickLine(rg, path[i], path[i+1], rect, 1, 2)
		}
	}

	for X := 1; X < r.W+2; X++ {
		for Y := r.H + 1; Y >= 1; Y-- {
			if rg.Get(X, Y) == 0 {
				continue
			}

			nb := [8]uint32{rg.Get(X+1, Y-1), rg.Get(X+1, Y), rg.Get(X+1, Y+1), rg.Get(X, Y-1), rg.Get(X, Y+1),
				rg.Get(X-1, Y-1), rg.Get(X-1, Y), rg.Get(X-1, Y+1)}

			m := 0
			for _, v := range nb {
				m <<= 1
				if v != 0 {
					m |= 1
				}
			}

			if m != 0 && t3cb0[m] != 0 {
				g.C.Set(X-1, Y-1, uint32(t3cb0[m])<<8|0x82)
			}
		}
	}
}

func (b *roomBuilder) placeSubType(typ, theme int, mask uint32) {
	if typ == -1 {
		return
	}

	rows := b.l.env.Tables.SubRows(typ)

	for i := 0; mask != 0; i, mask = i+1, mask>>1 {
		if mask&1 != 0 {
			b.placeSubRow(rows[i], theme)
		}
	}
}

func (b *roomBuilder) placeSubRow(row d2drlg.SubRec, theme int) {
	d, err := b.l.env.Pattern(row.File)
	if err != nil {
		panic(err)
	}

	maxN := row.Max[theme]
	n := len(d.Groups)

	if n == 0 || maxN < 1 {
		return
	}

	rs, room := &b.g.Seed, b.r

	for i := 0; i < maxN; i++ {
		g := d.Groups[rs.Roll(int32(n))]
		dh, dw := room.H-g.H, room.W-g.W

		if !(dw >= 1 && dh >= 1) {
			continue
		}

		trials := row.Trials[theme]

		if trials == -1 {
			cnt := dh * dw
			if cnt == 0 {
				continue
			}

			lst := make([]pt, cnt)
			for k := range lst {
				lst[k] = pt{k % dw, k / dw}
			}

			for k := 0; k < cnt; k++ {
				a := int(rs.Roll(int32(cnt)))
				c := int(rs.Roll(int32(cnt)))
				lst[a], lst[c] = lst[c], lst[a]
			}

			for _, p := range lst {
				if b.checkSub(p.x+1, p.y+1, g, d) {
					b.stampSub(p.x+1, p.y+1, g, d, 0)
					break
				}
			}

			continue
		}

		for t := 0; t < trials; t++ {
			x := int(rs.Roll(int32(dw))) + 1
			y := int(rs.Roll(int32(dh))) + 1

			if b.checkSub(x, y, g, d) {
				b.stampSub(x, y, g, d, 0)
				break
			}
		}
	}
}

func (b *roomBuilder) checkSub(x, y int, g Group, d *Pattern) bool {
	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			pf := d.floor(g.X+gx, g.Y+gy)
			pw := d.wall(g.X+gx, g.Y+gy)

			if pf&2 != 0 || (len(d.Wall) > 0 && pw&1 != 0) {
				v := b.g.C.Get(x+gx, y+gy)
				if v&0x3f0ff00 != 0 || v&2 == 0 {
					return false
				}

				if b.g.B.Get(x+gx, y+gy)&1 != 0 {
					return false
				}
			}
		}
	}

	return true
}

func (b *roomBuilder) stampSub(x, y int, g Group, d *Pattern, val int) {
	for gy := 0; gy < g.H; gy++ {
		for gx := 0; gx < g.W; gx++ {
			sx, sy := g.X+val+gx, g.Y+gy
			pf := d.floor(sx, sy)

			if pf&2 != 0 {
				b.g.C.Set(x+gx, y+gy, pf|0x80)
			}

			if len(d.Wall) > 0 {
				if wv := d.Wall[0][sy*d.W+sx]; wv&1 != 0 {
					b.g.B.Set(x+gx, y+gy, wv)
				}

				if ov := d.Orient[0][sy*d.W+sx]; ov != 0 {
					b.g.A.Set(x+gx, y+gy, ov)
				}
			}

			if tv := d.Shadow[sy*d.W+sx]; tv&0x8000000 != 0 {
				b.g.Tags++

				if b.opts.tiles != nil {
					b.opts.tiles.addRandom(b.r.X+x+gx, b.r.Y+y+gy, tv)
				} else if b.opts.PickTile != nil {
					b.opts.PickTile(&b.g.Seed, b.r.X+x+gx, b.r.Y+y+gy, tv)
				} else {
					b.g.Seed.Step() // DRLG_PickRandomTile: one room-seed step (model, DT1 library not available)
				}
			}
		}
	}
}
