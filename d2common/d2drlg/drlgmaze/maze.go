// Package drlgmaze ports the Act 1 maze level generator (Levels.txt DrlgType 1:
// Den of Evil/caves, crypts, jail, catacombs) from drlg2.md part A.
//
// Provenance of each step is noted in comments: "verified" means the notes
// traced it in the real binary, "unverified" means it is inferred or was not
// read. Act 1 barracks (level 28, its joint with level 27, A.10) and the
// Act 2+ maze types are not implemented and return ErrUnsupported.
//
// The level-seed draw order is reproduced exactly as documented: room
// allocation steps, FillMazeRooms rolls, finisher counter, theme pass (31
// steps) and the commit pass (file rolls and per-chunk room allocations).
// What is NOT reproduced: the DS1 object RNG gates (A.9) fire at DS1 load
// time on the level seed; they are exposed through Params.GateSteps (nil =
// zero extra steps) because the generator has no DS1 access.
package drlgmaze

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// ErrUnsupported is returned for level types whose algorithm is not covered.
var ErrUnsupported = errors.New("drlgmaze: unsupported maze level")

// Directions (verified, drlg2.md): 0 west, 1 north, 2 east, 3 south.
const (
	West = iota
	North
	East
	South
)

// Door mask bits used to pick the LvlPrest Def (verified): W=1 E=2 S=4 N=8.
var dirMask = [4]int{1, 8, 2, 4}

// LevelType ids (LvlTypes.txt) handled here.
const (
	typeCave      = 3
	typeCrypt     = 4
	typeBarracks  = 7
	typeJail      = 8
	typeCatacombs = 10
)

// Def bases of the 16 door-mask variants per level type (verified for cave
// and crypt against LvlPrest; the others by the same pattern).
var typeBase = map[int]int{typeCave: 0x34, typeCrypt: 0x6c, typeBarracks: 0xa7, typeJail: 0xcd, typeCatacombs: 0x101}

// Tables is what the generator needs from the data tables.
type Tables interface {
	d2drlg.Levels
	d2drlg.LvlMaze
	d2drlg.LvlPrest
}

// Params configures one generation.
type Params struct {
	LevelID    int
	Difficulty d2drlg.Difficulty
	// BaseSeed is d2rand.DrlgBaseSeed(gameSeed): the DRLG base seed.
	BaseSeed uint32
	// TombA / TombB are the Act 2 special tomb level ids (room count x3 / x2
	// in FillMazeRooms, verified); 0 for Act 1.
	TombA, TombB int
	// GateSteps, if set, returns how many level-seed steps the DS1 object
	// RNG gates consume when the given preset file is loaded (unverified
	// effect, see package doc).
	GateSteps func(file string) int
}

// Room is a committed maze room (one LvlPrest preset).
type Room struct {
	X, Y, W, H int // absolute tile rect
	Def        int // LvlPrest Def
	File       int // preset file index (-1 when the Def has no files)
	FileName   string
	Doors      int   // door mask: W=1 E=2 S=4 N=8
	Links      []int // indices (into Result.Rooms) of connected rooms
	Locked     bool  // special (finisher/theme) room
}

// Chunk is a DrlgRoom created from a Room: the whole preset for rooms up to
// 12x12, else 8x8 pieces (rows outer, columns inner).
type Chunk struct {
	Room       int
	X, Y, W, H int
}

// Result is a generated maze level.
type Result struct {
	LevelID    int
	LevelType  int
	Rooms      []Room // in game list order (head first = most recently added)
	Chunks     []Chunk
	MinX, MinY int
	MaxX, MaxY int
	// Notes lists things that are not exact (TODOs, unverified steps).
	Notes []string
}

type link struct {
	r   *room
	dir int
}

type room struct {
	x, y, w, h int
	seed       d2rand.Seed
	def, file  int
	locked     bool
	nb         []link
}

type level struct {
	seed  d2rand.Seed
	rooms []*room // head first
	rec   d2drlg.MazeRec
	typ   int
	id    int
	tbl   Tables
	ctrs  []*defCounter
}

type defCounter struct{ def, n, ctr int }

func overlaps(a, b *room) bool {
	return a.x < b.x+b.w && b.x < a.x+a.w && a.y < b.y+b.h && b.y < a.y+a.h
}

func (l *level) alloc() *room {
	s, _ := d2rand.NewRoomSeed(&l.seed) // AllocRoomEx: one level-seed step (verified)
	r := &room{seed: *s, w: l.rec.SizeX, h: l.rec.SizeY, file: -1}

	return r
}

func (l *level) prepend(r *room) { l.rooms = append([]*room{r}, l.rooms...) }

func (r *room) has(o *room) bool {
	for _, n := range r.nb {
		if n.r == o {
			return true
		}
	}

	return false
}

func (r *room) addNb(o *room, dir int) {
	if !r.has(o) { // idempotent per room (verified)
		r.nb = append(r.nb, link{o, dir})
	}
}

func linkRooms(a, b *room, dir int) {
	a.addNb(b, dir)
	b.addNb(a, (dir+2)&3)
}

// tryPlace positions n next to cur in dir and tests collisions (verified).
// No RNG is consumed.
func (l *level) tryPlace(cur *room, dir int, n *room) bool {
	switch dir {
	case West:
		n.x, n.y = cur.x-cur.w, cur.y
	case North:
		n.x, n.y = cur.x, cur.y-cur.h
	case East:
		n.x, n.y = cur.x+cur.w, cur.y
	case South:
		n.x, n.y = cur.x, cur.y+cur.h
	}

	for _, nb := range cur.nb {
		if overlaps(n, nb.r) {
			return false
		}
	}

	for _, r := range l.rooms {
		if r != cur && r != n && overlaps(n, r) {
			return false
		}
	}

	return true
}

// gap is the signed distance between two intervals (negative = overlap).
func gap(a0, a1, b0, b1 int) int {
	lo, hi := a0, a1

	if b0 > lo {
		lo = b0
	}

	if b1 < hi {
		hi = b1
	}

	return lo - hi
}

// adjacencyDir returns the direction in which b lies from a when they share an
// edge, else -1 (verified semantics, exact edge test is inferred).
func adjacencyDir(a, b *room) int {
	yOv := gap(a.y, a.y+a.h, b.y, b.y+b.h) < 0
	xOv := gap(a.x, a.x+a.w, b.x, b.x+b.w) < 0

	switch {
	case yOv && b.x+b.w == a.x:
		return West
	case yOv && b.x == a.x+a.w:
		return East
	case xOv && b.y+b.h == a.y:
		return North
	case xOv && b.y == a.y+a.h:
		return South
	}

	return -1
}

// selectDef re-picks a room's Def from its door mask (verified for the Act 1
// types) and clears the lock bit (flag=1).
func (l *level) selectDef(r *room) {
	base, ok := typeBase[l.typ]
	if !ok {
		return
	}

	mask := 0
	for _, n := range r.nb {
		mask |= dirMask[n.dir]
	}

	r.def = base + mask
	r.file = -1
	r.locked = false
}

// merge randomly adds extra doors between a just-placed room and touching
// rooms (MergeMazeNeighbors, verified). Draws one room-seed step on the OTHER
// room per candidate pair.
func (l *level) merge(n *room) {
	if n.locked {
		return
	}

	for _, o := range l.rooms {
		if o == n || o.locked {
			continue
		}

		gx := gap(n.x, n.x+n.w, o.x, o.x+o.w)
		gy := gap(n.y, n.y+n.h, o.y, o.y+o.h)

		if !(gx < 1 && gy < 1) || gx == gy || n.has(o) {
			continue
		}

		if o.seed.Step()%1000 < uint32(l.rec.Merge) {
			if d := adjacencyDir(o, n); d != -1 {
				linkRooms(o, n, d)
				l.selectDef(o)
			}
		}
	}
}

// addGrown places a new room next to cur (ring/catacomb/fill growth).
func (l *level) addGrown(cur *room, dir int) *room {
	n := l.alloc()
	if !l.tryPlace(cur, dir, n) {
		return nil // FreeRoomEx; the level-seed step stays consumed
	}

	linkRooms(cur, n, dir)
	l.merge(n)
	l.prepend(n)
	l.selectDef(cur)
	l.selectDef(n)

	return n
}

// buildRing: a closed square ring of side n (BuildMazeRing, verified).
func (l *level) buildRing(n int) {
	start := l.rooms[0]
	cur := start

	chain := func(dir, count int) {
		for i := 0; i < count; i++ {
			if nr := l.addGrown(cur, dir); nr != nil {
				cur = nr
			}
		}
	}

	chain(North, n-1)
	chain(West, n-1)
	chain(South, n-1)
	chain(East, n-2)
	linkRooms(cur, start, East)
	l.selectDef(cur)
	l.selectDef(start)
}

func (l *level) target(diff d2drlg.Difficulty, p Params) int {
	t := l.rec.Rooms[diff]
	if p.TombA != 0 && l.id == p.TombA {
		t *= 3
	}

	if p.TombB != 0 && l.id == p.TombB {
		t *= 2
	}

	return t
}

// fillRooms grows the level until the LvlMaze room count (FillMazeRooms,
// verified).
func (l *level) fillRooms(target int) error {
	for guard := 0; len(l.rooms) < target; guard++ {
		if guard > 1000000 {
			return errors.New("drlgmaze: room growth did not terminate")
		}

		p := l.rooms[int(l.seed.Roll(int32(len(l.rooms))))]
		dir := int(p.seed.Step() & 3) // consumed even when p is locked

		if !p.locked {
			l.addGrown(p, dir)
		}
	}

	return nil
}

type stamp struct{ from, to, dir int }

var stampDirs = [4]int{South, West, North, East} // {3,0,1,2} (verified from the exe tables)

func tbl(from, to [4]int) [4]stamp {
	var t [4]stamp
	for i := range t {
		t[i] = stamp{from[i], to[i], stampDirs[i]}
	}

	return t
}

// Finisher tables (verified: cave, crypt from the exe dump in drlg2.md A.5; jail
// Prev and catacomb Next re-read from the exe for this port). Rows are N,E,S,W.
var (
	caveFrom   = [4]int{60, 54, 56, 53}
	cavePrev   = tbl(caveFrom, [4]int{86, 84, 85, 83})
	caveDOE    = tbl(caveFrom, [4]int{98, 96, 97, 95})
	caveDown   = tbl(caveFrom, [4]int{94, 92, 93, 91})
	caveCrow   = tbl(caveFrom, [4]int{102, 100, 101, 99})
	caveNext   = tbl(caveFrom, [4]int{90, 88, 89, 87})
	cryptFrom  = [4]int{116, 110, 112, 109}
	cryptPrev  = tbl(cryptFrom, [4]int{142, 140, 141, 139})
	cryptBone  = tbl(cryptFrom, [4]int{150, 148, 149, 147})
	cryptChest = tbl(cryptFrom, [4]int{154, 152, 153, 151})
	cryptNext  = tbl(cryptFrom, [4]int{146, 144, 145, 143})
	jailFrom   = [4]int{213, 207, 209, 206}
	jailPrev   = tbl(jailFrom, [4]int{239, 237, 238, 236})
	jailWP     = tbl(jailFrom, [4]int{251, 249, 250, 248})
	jailPit    = tbl(jailFrom, [4]int{255, 253, 254, 252})
	jailCath   = tbl(jailFrom, [4]int{247, 245, 246, 244})
	jailNext   = tbl(jailFrom, [4]int{243, 241, 242, 240})
	catFrom    = [4]int{265, 259, 261, 258}
	catNext    = tbl(catFrom, [4]int{294, 292, 293, 291})
	catWP      = tbl(catFrom, [4]int{298, 296, 297, 295})
)

// addLocked adds a locked room with a fixed Def at dir of the first unlocked
// room that has space (AddLockedMazeRoom, verified). No merge pass.
func (l *level) addLocked(dir, def int) bool {
	for _, r := range append([]*room(nil), l.rooms...) {
		if r.locked {
			continue
		}

		n := l.alloc()
		if l.tryPlace(r, dir, n) {
			linkRooms(r, n, dir)
			l.prepend(n)
			l.selectDef(r)
			n.def, n.file, n.locked = def, -1, true

			return true
		}
	}

	return false
}

// stamp converts a dead-end room into a special Def, or adds one (verified).
func (l *level) stamp(t [4]stamp, ctr *int) bool {
	e := t[*ctr]
	done := false

	for _, r := range l.rooms {
		if !r.locked && r.def == e.from {
			r.locked, r.def, r.file = true, e.to, -1
			done = true

			break
		}
	}

	if !done {
		done = l.addLocked(e.dir, e.to)
	}

	*ctr = (*ctr + 1) & 3

	return done
}

func (l *level) finish(notes *[]string) {
	ctr := int(l.seed.Step() & 3) // one level-seed step (verified)
	run := func(ts ...[4]stamp) {
		for _, t := range ts {
			if !l.stamp(t, &ctr) {
				*notes = append(*notes, "finisher stamp could not be placed")
			}
		}
	}

	switch l.typ {
	case typeCave:
		run(cavePrev)

		if l.id == 8 {
			run(caveDOE)
		} else {
			run(caveDown)
		}

		switch l.id {
		case 9:
			run(caveCrow)
		case 10:
			run(caveNext)
		}
	case typeCrypt:
		run(cryptPrev)

		switch {
		case l.id == 18:
			run(cryptBone)
		case l.id == 19 || l.id == 0x85:
			run(cryptChest)
		case l.id >= 21 && l.id <= 24:
			run(cryptNext)
		}
	case typeJail:
		run(jailPrev)

		switch l.id {
		case 29:
			run(jailWP)
		case 30:
			run(jailPit)
		case 31:
			run(jailCath)
			return
		}

		run(jailNext)
	case typeCatacombs:
		run(catNext)

		if l.id == 35 {
			run(catWP)
		}
	}
}

// catacombsStart is BuildCatacombsStart (verified).
func (l *level) catacombsStart(plus bool) {
	start := l.rooms[0]

	if plus {
		for _, d := range []int{North, East, South, West} {
			l.addGrown(start, d)
		}

		start.def, start.file, start.locked = 0x122, -1, true

		return
	}

	if l.seed.Step()&1 == 0 {
		l.addGrown(start, North)
		l.addGrown(start, South)

		start.def = 0x121
	} else {
		l.addGrown(start, West)
		l.addGrown(start, East)

		start.def = 0x120
	}

	start.file, start.locked = -1, true
}

// themeRooms upgrades some rooms to theme Defs (ApplyMazeThemeRooms, verified;
// 1 + 30 level-seed steps).
func (l *level) themeRooms() {
	base, ok := typeBase[l.typ]
	if !ok || l.id == 8 {
		return
	}

	idx := int(l.seed.Roll(15))

	themes := len(l.rooms)/5 + 1
	if themes < 2 {
		themes = 2
	}

	var arr [15]int
	for i := range arr {
		arr[i] = i
	}

	for i := 0; i < 15; i++ {
		a := l.seed.Roll(15)
		b := l.seed.Roll(15)
		arr[a], arr[b] = arr[b], arr[a]
	}

	for budget := len(l.rooms) * 2; themes != 0 && budget != 0; budget-- {
		m := arr[idx]

		for _, r := range l.rooms {
			if !r.locked && r.def == base+m {
				r.locked = true
				themes--
				r.def += 15
				r.file = -1

				break
			}
		}

		idx = (idx + 1) % 15
	}
}

// chooseFile implements ChoosePresetFile (verified): per-Def round robin that
// starts at a random offset, only for Defs strictly inside (base, base+16).
func (l *level) chooseFile(r *room, rolled int, files int) int {
	if r.file != -1 {
		return r.file
	}

	base, ok := typeBase[l.typ]
	if !ok || r.def <= base || r.def >= base+16 || files <= 0 {
		return rolled
	}

	var c *defCounter

	for _, x := range l.ctrs {
		if x.def == r.def {
			c = x
			break
		}
	}

	if c == nil {
		c = &defCounter{def: r.def, n: files, ctr: int(l.seed.Roll(int32(files)))}
		l.ctrs = append(l.ctrs, c)
	}

	c.ctr = (c.ctr + 1) % c.n

	return c.ctr
}

func (l *level) normalize(ox, oy int) {
	minX, minY := l.rooms[0].x, l.rooms[0].y
	for _, r := range l.rooms {
		if r.x < minX {
			minX = r.x
		}

		if r.y < minY {
			minY = r.y
		}
	}

	for _, r := range l.rooms {
		r.x += ox - minX
		r.y += oy - minY
	}
}

// Generate builds an Act 1 maze level for the given seed.
func Generate(t Tables, p Params) (*Result, error) {
	rec, ok := t.Level(p.LevelID)
	if !ok {
		return nil, fmt.Errorf("drlgmaze: unknown level %d", p.LevelID)
	}

	if rec.DrlgType != 1 {
		return nil, fmt.Errorf("drlgmaze: level %d is not a maze level (DrlgType %d)", p.LevelID, rec.DrlgType)
	}

	mz, ok := t.Maze(p.LevelID)
	if !ok {
		return nil, fmt.Errorf("drlgmaze: level %d has no LvlMaze row", p.LevelID)
	}

	switch rec.LevelType {
	case typeCave, typeCrypt, typeJail, typeCatacombs:
	default:
		return nil, fmt.Errorf("%w: level %d LevelType %d", ErrUnsupported, p.LevelID, rec.LevelType)
	}

	l := &level{rec: mz, typ: rec.LevelType, id: p.LevelID, tbl: t}
	l.seed = *d2rand.LevelSeed(p.BaseSeed, uint32(p.LevelID))

	lw, lh := rec.SizeX[p.Difficulty], rec.SizeY[p.Difficulty]
	ox, oy := rec.OffsetX, rec.OffsetY // Depend/-1 offsets: unverified, not handled

	if ox < 0 || oy < 0 {
		ox, oy = 0, 0
	}

	r0 := l.alloc() // first room: level-seed step #1
	r0.x = (lw-r0.w)/2 + ox
	r0.y = (lh-r0.h)/2 + oy
	l.prepend(r0)

	res := &Result{LevelID: p.LevelID, LevelType: l.typ}
	target := l.target(p.Difficulty, p)

	switch l.typ {
	case typeCave, typeCrypt:
		if err := l.fillRooms(target); err != nil {
			return nil, err
		}
	case typeJail:
		l.buildRing(2)

		if err := l.fillRooms(target); err != nil {
			return nil, err
		}
	case typeCatacombs:
		l.catacombsStart(p.LevelID == 0x22)

		if err := l.fillRooms(target); err != nil {
			return nil, err
		}
	}

	l.finish(&res.Notes)
	l.normalize(ox, oy) // NormalizeMazeRooms (verified as a bbox translate)
	l.themeRooms()
	l.commit(p, res)

	return res, nil
}

// commit turns each placeholder room into preset rooms (CommitMazeRoom).
func (l *level) commit(p Params, res *Result) {
	idx := map[*room]int{}
	for i, r := range l.rooms {
		idx[r] = i
	}

	for i, r := range l.rooms {
		pr, hasDef := l.tbl.PrestByDef(r.def)
		rolled := -1

		if hasDef && pr.Files > 0 {
			rolled = int(l.seed.Roll(int32(pr.Files))) // AllocPresetMap (verified)
		}

		file := l.chooseFile(r, rolled, pr.Files)
		out := Room{X: r.x, Y: r.y, W: r.w, H: r.h, Def: r.def, File: file, Locked: r.locked}

		if file >= 0 && file < len(pr.File) {
			out.FileName = pr.File[file]
		}

		for _, n := range r.nb {
			out.Doors |= dirMask[n.dir]
			out.Links = append(out.Links, idx[n.r])
		}

		if p.GateSteps != nil && out.FileName != "" {
			for k := p.GateSteps(out.FileName); k > 0; k-- {
				l.seed.Step()
			}
		}

		res.Rooms = append(res.Rooms, out)

		// PlacePresetRooms: one DrlgRoom for rooms <= 12x12, else 8x8 chunks
		// (rows outer, columns inner); one level-seed step each (verified).
		if r.w <= 12 && r.h <= 12 {
			l.seed.Step()
			res.Chunks = append(res.Chunks, Chunk{i, r.x, r.y, r.w, r.h})

			continue
		}

		for y := 0; y < r.h; y += 8 {
			for x := 0; x < r.w; x += 8 {
				l.seed.Step()
				res.Chunks = append(res.Chunks, Chunk{i, r.x + x, r.y + y, imin(8, r.w-x), imin(8, r.h-y)})
			}
		}
	}

	res.MinX, res.MinY = res.Rooms[0].X, res.Rooms[0].Y
	res.MaxX, res.MaxY = res.MinX, res.MinY

	for _, r := range res.Rooms {
		res.MinX, res.MinY = imin(res.MinX, r.X), imin(res.MinY, r.Y)
		res.MaxX, res.MaxY = imax(res.MaxX, r.X+r.W), imax(res.MaxY, r.Y+r.H)
	}

	res.Notes = append(res.Notes,
		"unverified: DS1 object RNG gates (A.9) not applied unless Params.GateSteps is set",
		"unverified: chunk size for rooms that are not multiples of 8",
		"unverified: level origin for levels with Depend/negative Offset")
}

func imin(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func imax(a, b int) int {
	if a > b {
		return a
	}

	return b
}
