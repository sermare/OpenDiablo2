// Package drlgoutdoor ports the verified part of the Act 1 outdoor generator
// (drlg2.md B.2.1-B.2.3, B.4, B.5): the cell grids, preset placement with the
// per-Def file round robin, the shuffled/near-exit/farthest free-cell
// searches, cave entrance and town transition placement, the per-level
// special features and the marker scatterers.
//
// This is a FEATURE-PLACEMENT SKELETON, not a complete level. The notes mark
// these stages as unread and they are left as Hooks (nil = skipped, with a
// note in the result): polygon exit notches / MarkExitCells, DrawBoundaryEdges,
// the LvlSub rule matcher (ApplyLvlSubType, 4 passes), the river finder, the
// concave-corner cliff marking, and sub-theme stamping. They run between the
// stages implemented here and draw from the same level seed, so positions
// produced here WILL differ from the real game wherever those stages would
// have consumed random numbers or occupied cells.
package drlgoutdoor

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Flag grid bits (drlg2.md B.2; partly inferred).
const (
	cellOccupied = 0x1
	cellExitSite = 0x80 // "adjacent to an exit" marker tested by NearExit
	cellBlocked  = 0x1b81
	cellPreset   = 0x200
	cellFileMask = 0xf0000
)

// od.flags bits.
const (
	FlagRiverEdge1  = 0x4
	FlagRiverEdge2  = 0x8
	FlagRiverEdge3  = 0x10
	FlagCliffCave   = 0x20
	FlagCavePlaced  = 0x40
	FlagTownTransS0 = 0x80
	FlagTownTransS1 = 0x100
	FlagTownTransE0 = 0x200
	FlagTownTransE1 = 0x400
)

// Rect is a rectangle in tiles.
type Rect struct{ X, Y, W, H int }

// Placement is one preset written into the cell grid.
type Placement struct {
	X, Y     int // cell coordinates (8 tiles per cell)
	Def      int
	File     int
	FileName string
	Name     string
	W, H     int // size in cells
}

// Hooks are the stages the notes mark as unread. Each receives the grid and
// the level seed and may modify both. Nil hooks are skipped.
type Hooks struct {
	// TODO(unread): FillConcaveCornersMarkCliffCave (0x6830b0), levels not in {2,3,17}.
	FillConcaveCorners func(g *Grid, seed *d2rand.Seed)
	// TODO(unread): MarkExitCells (0x678480) + polygon notches (0x67fdf0).
	MarkExitCells func(g *Grid, seed *d2rand.Seed)
	// TODO(unread): DrawBoundaryEdges (0x678560).
	DrawBoundaryEdges func(g *Grid, seed *d2rand.Seed)
	// TODO(unread): ApplyLvlSubType(type 0..3, base def 4) (LvlSub matcher family).
	ApplyLvlSub func(g *Grid, seed *d2rand.Seed, subType int)
	// TODO(unread): river generator (0x684420) and DrawRiverBorderPresets (0x682ed0).
	PlaceRivers          func(g *Grid, seed *d2rand.Seed)
	DrawRiverBorderCol   func(g *Grid, seed *d2rand.Seed, col int)
	PlaceCliffCaveAtCell func(g *Grid, x, y int) bool // (0x6831e0) needs boundary defs 0x10/0x11
}

// Grid is the outdoor cell state of a level (od.defGrid and od.flagGrid).
type Grid struct {
	W, H  int
	Flags [][]uint32 // [x][y]
	Defs  [][]int
	// OdFlags is od.flags.
	OdFlags int

	level  Rect
	id     int
	prest  d2drlg.LvlPrest
	seed   *d2rand.Seed
	ctrs   map[int]*counter
	Placed []Placement
	err    error
}

type counter struct{ n, ctr int }

func newGrid(id int, r Rect, p d2drlg.LvlPrest, seed *d2rand.Seed) *Grid {
	g := &Grid{W: r.W >> 3, H: r.H >> 3, level: r, id: id, prest: p, seed: seed, ctrs: map[int]*counter{}}
	g.Flags = make([][]uint32, g.W)
	g.Defs = make([][]int, g.W)

	for x := range g.Flags {
		g.Flags[x] = make([]uint32, g.H)
		g.Defs[x] = make([]int, g.H)
	}

	return g
}

func (g *Grid) in(x, y int) bool { return x >= 0 && y >= 0 && x < g.W && y < g.H }

// cells returns the preset size in cells: (v + (v>>31 & 7)) >> 3 (verified).
func cells(v int) int { return (v + ((v >> 31) & 7)) >> 3 }

// nextFile is GetNextOutdoorPresetFile (verified): per-Def counter starting at
// a random offset (Roll(Files) on first use), then ++ mod Files every call.
func (g *Grid) nextFile(def int) int {
	rec, _ := g.prest.PrestByDef(def)
	if rec.Files <= 0 {
		return 0 // "never used" in the game (it would divide by zero)
	}

	c := g.ctrs[def]
	if c == nil {
		c = &counter{n: rec.Files, ctr: int(g.seed.Roll(int32(rec.Files)))}
		g.ctrs[def] = c
	}

	c.ctr = (c.ctr + 1) % c.n

	return c.ctr
}

// PlaceOutdoorPreset writes a Def into the grids (0x677100, verified).
func (g *Grid) PlaceOutdoorPreset(x, y, def, file, flag int) {
	rec, ok := g.prest.PrestByDef(def)
	if !ok {
		g.err = fmt.Errorf("drlgoutdoor: unknown Def %d", def)
		return
	}

	w, h := cells(rec.SizeX), cells(rec.SizeY)
	if file == -1 {
		file = g.nextFile(def)
	}

	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !g.in(xx, yy) {
				continue
			}

			g.Flags[xx][yy] &^= cellFileMask
			g.Flags[xx][yy] |= uint32(file)<<16 | cellPreset

			if flag != 0 && (def >= 4 && def <= 15 || def >= 0x16c && def <= 0x177) {
				g.Flags[xx][yy] |= cellOccupied
			}

			g.Defs[xx][yy] = 0
		}
	}

	if g.in(x, y) {
		g.Defs[x][y] = def
	}

	p := Placement{X: x, Y: y, Def: def, File: file, Name: rec.Name, W: w, H: h}
	if file >= 0 && file < len(rec.File) {
		p.FileName = rec.File[file]
	}

	g.Placed = append(g.Placed, p)
}

func (g *Grid) fits(x, y, def, pad, mask int) bool {
	w, h := 1, 1

	if def != 0 {
		rec, _ := g.prest.PrestByDef(def)
		w, h = cells(rec.SizeX), cells(rec.SizeY)
	}

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
			x -= pad
			w += pad
		}
	}

	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if !g.in(xx, yy) || g.Flags[xx][yy]&cellBlocked != 0 {
				return false
			}
		}
	}

	return true
}

// shuffle is ShuffleCells (verified): 2n level-seed draws over the interior.
func (g *Grid) shuffle() [][2]int {
	iw, ih := g.W-2, g.H-2
	n := iw * ih

	if n <= 0 {
		return nil
	}

	l := make([][2]int, n)
	for k := range l {
		l[k] = [2]int{k % iw, k / iw}
	}

	for i := 0; i < n; i++ {
		a := g.seed.Roll(int32(n))
		b := g.seed.Roll(int32(n))
		l[a], l[b] = l[b], l[a]
	}

	return l
}

// PlaceRandom is PlaceOutdoorPresetRandom (0x6774a0, verified).
func (g *Grid) PlaceRandom(def, file, pad, mask int) bool {
	for _, c := range g.shuffle() {
		x, y := c[0]+1, c[1]+1
		if g.fits(x, y, def, pad, mask) {
			g.PlaceOutdoorPreset(x, y, def, file, 0)
			return true
		}
	}

	return false
}

var (
	nearDX = [8]int{-1, 0, 0, 1, -1, 1, 1, -1}
	nearDY = [8]int{0, -1, 1, 0, -1, 1, -1, 1}
)

// PlaceNearExit is PlaceOutdoorPresetNearExit (0x677670, verified).
func (g *Grid) PlaceNearExit(def, file int) bool {
	for _, c := range g.shuffle() {
		x1, y1 := c[0]+1, c[1]+1
		if g.Flags[x1][y1]&cellExitSite == 0 {
			continue
		}

		for j := 0; j < 8; j++ {
			x, y := x1+nearDX[j], y1+nearDY[j]
			if g.fits(x, y, def, 0, 0xf) {
				g.PlaceOutdoorPreset(x, y, def, file, 0)
				return true
			}
		}
	}

	return g.PlaceRandom(def, file, 0, 0xf)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

// PlaceFarthest is PlaceOutdoorPresetFarthest (0x677230, verified incl. the
// inclusive loop bounds). The initial "best" value is not stated in the notes;
// -1 is assumed.
func (g *Grid) PlaceFarthest(rect Rect, def, file, pad, mask int) bool {
	iw, ih := g.W-2, g.H-2
	ox := int(g.seed.Roll(int32(iw)))
	oy := int(g.seed.Roll(int32(ih)))
	best, bx, by := -1, 0, 0

	if iw > 0 && ih > 0 {
		for dy := 0; dy <= ih; dy++ {
			for dx := 0; dx <= iw; dx++ {
				y := (dy+oy)%ih + 1
				x := (dx+ox)%iw + 1

				if !g.fits(x, y, def, pad, mask) {
					continue
				}

				dxp := abs(x*8 - (rect.W/2 + rect.X) + 4 + g.level.X)
				dyp := abs(y*8 - (rect.H/2 + rect.Y) + 4 + g.level.Y)

				var m int
				if dyp < dxp {
					m = dyp + 2*dxp
				} else {
					m = dxp + 2*dyp
				}

				if m /= 2; m > best {
					best, bx, by = m, x, y
				}
			}
		}
	}

	if best < 0 {
		return false
	}

	g.PlaceOutdoorPreset(bx, by, def, file, 0)

	return true
}

func (g *Grid) placeCounted(def, flag int) {
	r := g.seed.Step()
	if r&3 == 0 {
		g.PlaceNearExit(def, -1)
		g.PlaceNearExit(def, -1)

		return
	}

	g.PlaceNearExit(def, -1)

	if flag != 0 {
		if g.seed.Step()&1 != 0 {
			g.PlaceNearExit(0x31, -1)
		}
	}
}

// placeSpecials is DRLG_PlaceLevelSpecials (0x683590, verified by disassembly).
func (g *Grid) placeSpecials() {
	P := func(d int) { g.PlaceRandom(d, -1, 0, 0xf) }
	N := func(d int) { g.PlaceNearExit(d, -1) }
	C := g.placeCounted
	tail := func() { P(0x1d); P(0x1e) }

	switch g.id {
	case 2:
		N(0x2e)
		C(0x2f, 0)
		tail()
	case 3:
		C(0x30, 1)
		P(0x2c)
		tail()
	case 4:
		N(0xa0)
		N(0x2d)
		P(0xa2)
		C(0x2f, 1)
		C(0x2a, 0)
		P(0x1f)
	case 5:
		P(0xa1)
		P(0x29)
		P(0x28)
		C(0x30, 1)
		C(0x2b, 0)
		tail()
	case 6:
		P(0xa3)
		P(0x26)
		P(0x27)
		C(0x2f, 1)
		C(0x2a, 0)
		tail()
	case 7:
		C(0x30, 1)
		C(0x2b, 0)
		P(0x1f)
	case 17:
		g.PlaceOutdoorPreset(1, 1, 0x6c, -1, 0)
	case 39:
		P(0x32)
		P(0x2e)
		P(0x1f)
		P(0x26)
		P(0x27)
		tail()
	}
}

// markWaypoint is 0x6778a0 (levels 3..6). Only the shuffle fallback path is
// ported; the level-3 exit-slot path needs exit marks that the unread
// MarkExitCells stage would write (TODO).
func (g *Grid) markWaypoint() {
	for _, c := range g.shuffle() {
		x, y := c[0]+1, c[1]+1
		if g.Flags[x][y]&cellBlocked == 0 {
			g.Flags[x][y] |= 0x10000 | 0x800

			return
		}
	}
}

// scatterMarkers is 0x677b50(count 5): shrine/object site markers (verified code).
func (g *Grid) scatterMarkers(count int) {
	rot := int(g.seed.Step() & 3)
	bits := [4]uint32{0x1000, 0x2000, 0x4000, 0x8000}

	for _, c := range g.shuffle() {
		if count <= 0 {
			break
		}

		x, y := c[0]+1, c[1]+1
		if g.Flags[x][y]&cellBlocked == 0 {
			g.Flags[x][y] |= bits[rot] | 0x1000
			rot = (rot + 1) & 3
			count--
		}
	}
}

func noRiverInColumns(g *Grid, col int) bool {
	for y := 0; y < g.H; y++ {
		for _, c := range []int{col, col + 1} {
			if g.in(c, y) && g.Flags[c][y]&2 != 0 {
				return false
			}
		}
	}

	return true
}

// Fatal errors of the original (error codes 0x194 and 0x1e5).
var (
	ErrNoCliffCave  = errors.New("drlgoutdoor: no cliff cave placed (game error 0x194)")
	ErrNoCaveEntry  = errors.New("drlgoutdoor: no cave entrance placed (game error 0x1e5)")
	ErrUnknownLevel = errors.New("drlgoutdoor: not an Act 1 outdoor level")
)

func (g *Grid) caveEntrances(h *Hooks) error {
	if g.id == 0x27 {
		return nil
	}

	if g.OdFlags&0xc != 0 && noRiverInColumns(g, g.W-2) && h.DrawRiverBorderCol != nil {
		h.DrawRiverBorderCol(g, g.seed, g.W-2)
	}

	if g.OdFlags&FlagCliffCave != 0 && g.OdFlags&FlagCavePlaced == 0 && h.PlaceCliffCaveAtCell != nil {
		done := false
		r := g.seed.Step()

		if r&1 != 0 { // transposed bounds: an original quirk (verified)
			for xc := 0; xc < g.H && !done; xc++ {
				for yc := 0; yc < g.W && !done; yc++ {
					done = h.PlaceCliffCaveAtCell(g, xc, yc)
				}
			}
		} else {
			for y := 0; y < g.H && !done; y++ {
				for x := 0; x < g.W && !done; x++ {
					done = h.PlaceCliffCaveAtCell(g, x, y)
				}
			}
		}

		if !done {
			return ErrNoCliffCave
		}

		g.OdFlags |= FlagCavePlaced
	}

	if g.OdFlags&0x1c != 0 && g.OdFlags&FlagCavePlaced == 0 {
		k := 4 | (^(g.OdFlags >> 4) & 1)
		xr, yr := g.W-k, g.H-4
		r := g.seed.Step()

		x, y := xr, yr
		if r&1 != 0 {
			x = 3
		}

		if (r>>1)&1 != 0 {
			y = 3
		}

		def := 0x33
		if g.id == 2 {
			def = 0x34
		}

		g.PlaceOutdoorPreset(x, y, def, -1, 0)
		g.OdFlags |= FlagCavePlaced
	}

	return nil
}

func (g *Grid) townTransitions(h *Hooks, town Rect) error {
	if g.id == 0x27 {
		return nil
	}

	if g.OdFlags&FlagRiverEdge3 != 0 && noRiverInColumns(g, g.W/2-1) && h.DrawRiverBorderCol != nil {
		h.DrawRiverBorderCol(g, g.seed, g.W/2-1)
	}

	if g.OdFlags&FlagTownTransS0 != 0 {
		g.PlaceOutdoorPreset(0, 0, 3, 1, 0)
	}

	if g.OdFlags&FlagTownTransS1 != 0 {
		g.PlaceOutdoorPreset(g.W-7, 0, 3, 2, 0)
	}

	if g.OdFlags&FlagTownTransE0 != 0 {
		g.PlaceOutdoorPreset(0, 1, 2, 1, 0)
	}

	if g.OdFlags&FlagTownTransE1 != 0 {
		g.PlaceOutdoorPreset(0, g.H-6, 2, 1, 0)
	}

	if g.OdFlags&FlagCavePlaced == 0 {
		var ok bool
		if g.id == 2 {
			ok = g.PlaceFarthest(town, 0x34, -1, 1, 0xf)
		} else {
			ok = g.PlaceRandom(0x33, -1, 1, 0xf)
		}

		if !ok {
			return ErrNoCaveEntry
		}

		g.OdFlags |= FlagCavePlaced
	}

	return nil
}

// Params configures GenerateAct1.
type Params struct {
	LevelID  int
	BaseSeed uint32 // d2rand.DrlgBaseSeed(gameSeed)
	Rect     Rect   // level rect from the world layout
	OdFlags  int    // exit flags from the world layout
	Town     Rect   // town level rect (Blood Moor's PlaceFarthest reference)
	Hooks    Hooks
}

// Result is the placement skeleton of one outdoor level.
type Result struct {
	LevelID   int
	W, H      int // cells
	Placed    []Placement
	OdFlags   int
	Grid      *Grid
	Notes     []string
	SeedAfter d2rand.Seed
}

// GenerateAct1 runs the Act 1 per-level flow (0x683800) with the implemented
// stages and skips (with notes) the unread ones.
func GenerateAct1(src d2drlg.LvlPrest, p Params) (*Result, error) {
	id := p.LevelID
	if !(id >= 2 && id <= 7 || id == 0x11 || id == 0x27) {
		return nil, ErrUnknownLevel
	}

	seed := d2rand.LevelSeed(p.BaseSeed, uint32(id))
	g := newGrid(id, p.Rect, src, seed)
	g.OdFlags = p.OdFlags
	h := &p.Hooks
	res := &Result{LevelID: id, W: g.W, H: g.H, Grid: g}

	skip := func(name string, set bool) {
		if !set {
			res.Notes = append(res.Notes, "TODO unread, skipped: "+name)
		}
	}

	if id != 2 && id != 3 && id != 0x11 {
		if h.FillConcaveCorners != nil {
			h.FillConcaveCorners(g, seed)
		}

		skip("FillConcaveCornersMarkCliffCave", h.FillConcaveCorners != nil)
	}

	if h.MarkExitCells != nil {
		h.MarkExitCells(g, seed)
	}

	skip("MarkExitCells", h.MarkExitCells != nil)

	if h.DrawBoundaryEdges != nil {
		h.DrawBoundaryEdges(g, seed)
	}

	skip("DrawBoundaryEdges", h.DrawBoundaryEdges != nil)

	sub := func(t int) {
		if h.ApplyLvlSub != nil {
			h.ApplyLvlSub(g, seed, t)
		}
	}

	skip("ApplyLvlSubType (LvlSub matcher)", h.ApplyLvlSub != nil)

	if id >= 2 && id <= 7 {
		sub(0)

		if err := g.caveEntrances(h); err != nil {
			return res, err
		}

		sub(1)
		sub(2)

		if err := g.townTransitions(h, p.Town); err != nil {
			return res, err
		}

		sub(3)

		if h.PlaceRivers != nil {
			h.PlaceRivers(g, seed)
		}

		skip("PlaceRivers", h.PlaceRivers != nil)
	}

	if id == 0x27 {
		for t := 0; t <= 3; t++ {
			sub(t)
		}
	}

	if id >= 3 && id <= 6 {
		g.markWaypoint()
	}

	if id >= 2 && id <= 7 {
		g.scatterMarkers(5)
	}

	g.placeSpecials()

	if g.err != nil {
		return res, g.err
	}

	res.Placed, res.OdFlags, res.SeedAfter = g.Placed, g.OdFlags, *seed
	res.Notes = append(res.Notes, "TODO unread: sub-theme stamping and final room tile fill (CreateOutdoorRooms) not implemented")

	return res, nil
}
