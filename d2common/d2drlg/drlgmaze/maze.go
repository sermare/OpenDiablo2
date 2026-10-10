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
	typeSewer2    = 13 // Act 2 sewers
	typeHarem     = 14
	typeBasement  = 15
	typeTomb      = 17
	typeLair      = 18
	typeArcane    = 19
	typeMephisto  = 22
	typeSpider    = 23
	typeDungeon   = 24
	typeSewer3    = 25
)

// Act 4/5 level types are in maze_act45.go.

// Def bases of the 16 door-mask variants per level type (verified for cave
// and crypt against LvlPrest; the others by the same pattern).
var typeBase = map[int]int{typeCave: 0x34, typeCrypt: 0x6c, typeBarracks: 0xa7, typeJail: 0xcd, typeCatacombs: 0x101,
	typeSewer2: 0x12d, typeTomb: 0x19d, typeLair: 0x1e1, typeArcane: 0x1fd, typeMephisto: 0x2f1, typeDungeon: 0x298, typeSewer3: 0x2c0,
	typeLava: 0x344, typeIce: 0x3ea, typeBaal: 0x422}

// fileBase lists the level types whose Defs in (base, base+16) get the
// round-robin file choice (ChoosePresetFile 0x676640, verified from the
// jump table: arcane, harem, basement and spider are not in it).
var fileBase = map[int]int{typeCave: 0x34, typeCrypt: 0x6c, typeBarracks: 0xa7, typeJail: 0xcd, typeCatacombs: 0x101,
	typeSewer2: 0x12d, typeTomb: 0x19d, typeLair: 0x1e1, typeMephisto: 0x2f1, typeDungeon: 0x298, typeSewer3: 0x2c0,
	typeLava: 0x344, typeIce: 0x3ea, typeBaal: 0x422, typeHell: 0x41c}

// themeBase lists the types ApplyMazeThemeRooms upgrades (0x676360).
var themeBase = map[int]int{typeCave: 0x34, typeCrypt: 0x6c, typeBarracks: 0xa7, typeJail: 0xcd, typeCatacombs: 0x101,
	typeSewer2: 0x12d, typeTomb: 0x19d, typeMephisto: 0x2f1, typeDungeon: 0x298, typeSewer3: 0x2c0}

// Door-mask lookup tables for the types that do not use base+mask
// (0x6f0d08 stride 12; zero = leave the Def alone). Indexed by mask.
var (
	haremDef    = [16]int{5: 356, 6: 355, 9: 357, 10: 354}
	basementDef = [16]int{5: 360, 6: 359, 9: 361, 10: 358}
	spiderDef   = [16]int{5: 659, 6: 660, 9: 661, 10: 662}
)

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
	// L27 is the Act 1 level 27 (Courtyard) rect and exit side (0 west,
	// 1 north, 2 east in the game's data+4), needed only for the barracks
	// (level 28), which is placed relative to it.
	L27 Level27
	// L108 is the rect of level 108 (Chaos Sanctuary) that River of Flame
	// (level 107) is placed against.
	L108 Level27
}

// Level27 is what the barracks joint needs from level 27.
type Level27 struct {
	X, Y, W, H, Side int
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
	// Lo is the level-seed step that seeded the chunk's room (AllocRoomEx);
	// the room seed is Init(Lo) and the tile builder's S4 its first step.
	Lo uint32
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
	// Seed is the level seed after generation (used to compare against the oracle).
	Seed d2rand.Seed
	// RectX..RectH is the final level rect when the generator recomputes it
	// (barracks only, from the room bounding box).
	RectX, RectY, RectW, RectH int
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
	p     Params
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
	case 4:
		n.x, n.y = cur.x-cur.w, cur.y-cur.h
	case 5:
		n.x, n.y = cur.x+cur.w, cur.y-cur.h
	case 6:
		n.x, n.y = cur.x+cur.w, cur.y+cur.h
	case 7:
		n.x, n.y = cur.x-cur.w, cur.y+cur.h
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
	mask := 0
	for _, n := range r.nb {
		mask |= maskOf(n.dir)
	}

	def, file := 0, -1

	switch l.typ {
	case typeTemple:
		def = templeDef[mask]
	case typeHell:
		def = hellDef[mask]
	case typeHarem:
		def = haremDef[mask]
	case typeBasement: // 0x673710 case 7
		def = basementDef[mask]

		if l.id == 0x34 && def == 0x169 {
			file = 2
		}

		if l.id == 0x36 && (def == 0x169 || def == 0x167) {
			file = 3
		}
	case typeSpider:
		def = spiderDef[mask]

		if l.id == 0x54 && def == 0x296 {
			def = 0x298
		}

		if l.id == 0x55 && def == 0x295 {
			def = 0x297
		}
	default:
		base, ok := typeBase[l.typ]
		if !ok {
			return
		}

		def = base + mask
	}

	if def == 0 {
		return // table miss: the real code leaves the room untouched
	}

	r.def = def
	r.file = file
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
func (l *level) buildAct1(target int, res *Result) error {
	switch l.typ {
	case typeJail:
		l.buildRing(2)
	case typeCatacombs:
		l.catacombsStart(l.id == 0x22)
	}

	if err := l.fillRooms(target); err != nil {
		return err
	}

	l.finish(&res.Notes)

	return nil
}

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
func (l *level) stamp(t [4]stamp, ctr *int) bool { return l.stampOne(t[*ctr], ctr) }

// stampOne is StampMazeSpecialRoom (0x675270) for one table entry.
func (l *level) stampOne(e stamp, ctr *int) bool {
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
	base, ok := themeBase[l.typ]
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

	base, ok := fileBase[l.typ]
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
	case typeCave, typeCrypt, typeJail, typeCatacombs, typeBarracks, typeSewer2, typeHarem, typeBasement,
		typeTomb, typeLair, typeArcane, typeMephisto, typeSpider, typeDungeon, typeSewer3,
		typeLava, typeTemple, typeIce, typeBaal, typeHell:
	default:
		return nil, fmt.Errorf("%w: level %d LevelType %d", ErrUnsupported, p.LevelID, rec.LevelType)
	}

	l := &level{rec: mz, typ: rec.LevelType, id: p.LevelID, tbl: t, p: p}
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
	case typeCave, typeCrypt, typeJail, typeCatacombs:
		if err := l.buildAct1(target, res); err != nil {
			return nil, err
		}

		l.normalize(ox, oy) // NormalizeMazeRooms (verified as a bbox translate)
	case typeLava, typeTemple, typeIce, typeBaal, typeHell:
		done, err := l.buildAct45(target, res)
		if err != nil {
			return nil, err
		}

		if !done {
			l.normalize(ox, oy)
		}
	default:
		done, err := l.buildAct23(target, res)
		if err != nil {
			return nil, err
		}

		if !done {
			l.normalize(ox, oy)
		}
	}

	l.themeRooms()
	l.commit(p, res)
	res.Seed = l.seed

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
		if file < 0 && hasDef {
			file = 0 // Files==0 rows: the preset map keeps file index 0 (verified on the arcane rooms)
		}

		out := Room{X: r.x, Y: r.y, W: r.w, H: r.h, Def: r.def, File: file, Locked: r.locked}

		if file >= 0 && file < len(pr.File) {
			out.FileName = pr.File[file]
		}

		for _, n := range r.nb {
			out.Doors |= maskOf(n.dir)
			out.Links = append(out.Links, idx[n.r])
		}

		// DS1 object RNG gates run once per preset room, not per 8x8 chunk
		// (verified by the emulator: constant steps per file; only a few
		// warp-room files have any).
		gate := 0

		if out.FileName != "" {
			if p.GateSteps != nil {
				gate = p.GateSteps(out.FileName)
			} else {
				gate = defaultGateSteps(out.FileName)
			}
		}

		res.Rooms = append(res.Rooms, out)

		// PlacePresetRooms: one DrlgRoom for rooms <= 12x12, else 8x8 chunks
		// (rows outer, columns inner); one level-seed step each (verified).
		if r.w <= 12 && r.h <= 12 {
			lo := l.seed.Step()
			l.gate(gate)
			res.Chunks = append(res.Chunks, Chunk{i, r.x, r.y, r.w, r.h, lo})

			continue
		}

		gate1 := gate

		for y := 0; y < r.h; y += 8 {
			for x := 0; x < r.w; x += 8 {
				lo := l.seed.Step()
				l.gate(gate1)
				gate1 = 0
				res.Chunks = append(res.Chunks, Chunk{i, r.x + x, r.y + y, imin(8, r.w-x), imin(8, r.h-y), lo})
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
		"unverified: chunk size for rooms that are not multiples of 8",
		"unverified: level origin for levels with Depend/negative Offset")
}

func (l *level) gate(n int) {
	for ; n > 0; n-- {
		l.seed.Step()
	}
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
