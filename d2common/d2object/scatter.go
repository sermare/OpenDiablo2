package d2object

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// The random objects of a room: OBJRGN_SpawnRoomObjects (Game.exe 0x550680),
// called by ROOM_ActivateRoomUnits right before the room's monsters. Every
// number below was read from the exe (Ghidra, read-only); what is not
// modelled is listed at the end of this comment.
//
// Per room (ShouldSpawn 0x5505e0 first):
//   - the room's region (per level: rooms seen +4, rooms total +8, shrines
//     made +0x10, large list +0x14, small list +0x18) counts the room;
//   - a level with a Themes mask draws one game-seed step and, with a chance
//     that grows with the rooms seen, a room theme out of the set bits. The
//     exe then only runs the theme handler when the theme number is below the
//     number of set bits and its table flag is set; with the 1.14b tables that
//     never happens (themes 3..6 against masks of 1 to 4 bits, theme 3 is
//     disabled), so the draw only costs random numbers;
//   - each of the 8 ObjGrp slots draws r = room seed step % 100 (r is forced
//     to 100 when the level is sparse, see Region.Sparse, and the objects.txt
//     row indexed by the GROUP id has a SubClass), and fires when group != 0
//     and r <= ObjPrb; a fired slot draws one more room-seed step s and walks
//     the group's members adding Prob: the first member with s%100 < sum is
//     made through its PopulateFn handler (table 0x72f6b8) and the walk ends
//     (also when the handler slot is empty). A density of 0x80 or more is a
//     fatal error of the exe.
//
// Handlers (density, object id, pct = 100): 1 group by class (0x54eae0),
// 2 shrine / well near the room edge (0x550bc0), 3 density scatter
// (0x54f3e0), 6 = 3 followed by a 29% twin at the same place (0x54f620),
// 8 tracked scatter (0x54f650), 9 one scatter (0x54f500). Not modelled (they
// make no object, the stream after them is approximate): 4 (0x54f7e0), 5
// (0x54fbe0) and 7 (0x54f140, its pattern tables are filled at run time);
// the object's own initialisation draws (shrine type, chest rolls) are not
// replayed either, only the one room-seed step the unit creation takes. The
// engine's collision grid is replaced by the Collider interface, and the
// room link test of ShouldSpawn by Room.NoObjects (UNVERIFIED).

// Rect is a room rectangle in sub-tiles.
type Rect struct{ X, Y, W, H int }

// Collision masks of the exe's placement tests.
const (
	MaskPlace = 0x3f11 // footprint test of every placement
	MaskWide  = 0xc01  // the wider (size + 7) test of CanUnitFitInRect
)

// Collider answers COLLISION_GetMaskForUnitSize != 0: whether any sub-tile of
// the w x h box around (x, y) carries one of the mask bits.
type Collider interface {
	Blocked(x, y, w, h, mask int) bool
}

// Region is the per-level object bookkeeping (the record the game keeps at
// game+0x10f0 + level * 4 + 0x48).
type Region struct {
	Seen   int      // +4: rooms that asked for objects
	Total  int      // +8: rooms of the level the game populates
	Shrine int      // +0x10: shrines made on a sparse level
	Large  [][2]int // +0x14 count, +0x40 entries: shrines / wells (max 10)
	Small  [][2]int // +0x18 count, +0x20 entries: tracked objects (max 4)
}

// Sparse is MONREGION_IsKillRatioAbove75Pct (0x545210): the level has been
// walked over more than 75% (96/128) without a single shrine.
func (r *Region) Sparse() bool {
	return r.Total > 0 && (r.Seen<<7)/r.Total > 0x60 && r.Shrine == 0
}

func (r *Region) quota(count, max int) bool { // QuotaCheckA / B: true = full
	return count == max || count > r.Total/8
}

// LevelRow is what the scatter needs of a levels.txt row.
type LevelRow struct {
	ID     int
	Groups [8]int
	Probs  [8]int
	Themes int
}

// RoomCtx is one room being populated.
type RoomCtx struct {
	Rect
	Seed      *d2rand.Seed // Room1 +0x6c
	NoObjects bool         // ShouldSpawn says no (town, linked room); UNVERIFIED mapping
}

// Placed is one created object.
type Placed struct{ ID, X, Y int }

// Scatter holds the tables and the game-wide state of the object scatter.
type Scatter struct {
	Defs   []Def
	Groups map[int]Group
	Game   *d2rand.Seed // game seed (MONREGION_GetGameRegionData)
	Region *Region
	Col    Collider
	// Create is SERVER_CreateUnitAtPosition for an object; the scatter takes the unit's
	// one room-seed step itself.
	Create func(p Placed)

	// Unmodelled counts handler calls of the unmodelled kinds by PopulateFn.
	Unmodelled map[int]int
	Made       []Placed
}

// ErrDensity is the exe's fatal error for an objgroup density byte of 0x80 or more.
var ErrDensity = errors.New("d2object: objgroup density >= 0x80")

func (s *Scatter) def(id int) Def {
	if id < 0 || id >= len(s.Defs) {
		return Def{}
	}

	return s.Defs[id]
}

// Room runs OBJRGN_SpawnRoomObjects for one room and returns the objects made.
func (s *Scatter) Room(lv LevelRow, room *RoomCtx) (int, error) {
	before := len(s.Made)

	if !s.shouldSpawn(lv, room) {
		return 0, nil
	}

	for i := 0; i < 8; i++ {
		gid := lv.Groups[i]
		r := int(room.Seed.Roll(100))

		if s.Region.Sparse() && s.def(gid).SubClass != 0 {
			r = 100
		}

		if gid == 0 || r > lv.Probs[i] {
			continue
		}

		g, ok := s.Groups[gid]
		if !ok {
			return len(s.Made) - before, nil // the exe returns from the whole function
		}

		roll := int(room.Seed.Roll(100))
		cum := 0

		for _, m := range g.Members {
			if m.ID == 0 {
				break
			}

			cum += m.Prob

			if roll >= cum {
				continue
			}

			fn := s.def(m.ID).PopulateFn
			if fn == 0 || fn >= 10 {
				break // empty handler slot: the walk ends without a spawn (>= 9 is fatal in the exe)
			}

			if m.Density >= 0x80 {
				return len(s.Made) - before, fmt.Errorf("%w: group %d member %d", ErrDensity, gid, m.ID)
			}

			s.handle(fn, lv, room, m.Density, m.ID)

			break
		}
	}

	return len(s.Made) - before, nil
}

// shouldSpawn is OBJRGN_ShouldSpawnObjectsForRoom.
func (s *Scatter) shouldSpawn(lv LevelRow, room *RoomCtx) bool {
	if room.NoObjects {
		return false
	}

	r := s.Region
	r.Seen++

	if lv.Themes == 0 {
		return true
	}

	// OBJRGN_SpawnObjectsFromTypeMask (0x550470); the +0xc counter it compares is never raised
	step := int(s.Game.Step() % 100)
	chance := 0

	if r.Total/2 < r.Seen {
		chance = 5
	}

	if r.Total-r.Total/4 < r.Seen {
		chance += 5
	}

	if step < chance+12 {
		n := 0

		for b := 0; b < 7; b++ {
			if lv.Themes&(1<<uint(b)) != 0 {
				n++
			}
		}

		if n > 0 {
			s.Game.Roll(int32(n)) // the theme pick; the handler is never reached (see above)
		}
	}

	return true
}

func (s *Scatter) handle(fn int, lv LevelRow, room *RoomCtx, density, id int) {
	switch fn {
	case 1:
		s.groupByClass(room, density, id)
	case 2:
		s.shrine(lv, room, id)
	case 3:
		s.densityA(room, density, id)
	case 6:
		if s.densityA(room, density, id) != nil {
			if int(s.Game.Step()%100) > 0x46 {
				p := s.Made[len(s.Made)-1]
				s.create(room, p.ID, p.X, p.Y)
			}
		}
	case 8:
		s.tracked(lv, room, id)
	case 9:
		s.densityB(room, density, id)
	default:
		if s.Unmodelled == nil {
			s.Unmodelled = map[int]int{}
		}

		s.Unmodelled[fn]++
	}
}

func count(r Rect, density int) int { return ((r.H * r.W >> 7) * density) >> 8 }

func (s *Scatter) create(room *RoomCtx, id, x, y int) *Placed {
	room.Seed.Step() // UNIT_InitCreatedUnit: the unit's own seed

	p := Placed{id, x, y}
	s.Made = append(s.Made, p)

	if s.Create != nil {
		s.Create(p)
	}

	return &s.Made[len(s.Made)-1]
}

// inside is the bounds test of the placement loops (corner excluded, one sub-tile margin).
func inside(r Rect, x, y int) bool {
	return x != 0 && y != 0 && r.X+1 <= x && r.Y+1 <= y && x < r.X-1+r.W && y < r.Y-1+r.H
}

// atRandomPoint is OBJRGN_SpawnObjectAtRandomPoint (0x54e1a0): five tries on the game seed.
func (s *Scatter) atRandomPoint(room *RoomCtx, id int) *Placed {
	d := s.def(id)
	r := room.Rect

	if r.W < 2 || r.H < 2 {
		return nil
	}

	for try := 0; try < 5; try++ {
		x := int(s.Game.Roll(int32(r.W-d.SizeX-1))) + r.X
		y := int(s.Game.Roll(int32(r.H-d.SizeY-1))) + r.Y

		if inside(r, x, y) && !s.Col.Blocked(x, y, d.SizeX+6, d.SizeY+6, MaskPlace) {
			return s.create(room, id, x, y)
		}
	}

	return nil
}

// densityA is 0x54f3e0 (handler 3): the whole count is tried, the last result is returned.
func (s *Scatter) densityA(room *RoomCtx, density, id int) *Placed {
	if int(s.Game.Step()%100) > 100 {
		return nil
	}

	var last *Placed

	for n := count(room.Rect, density); n > 0; n-- {
		if p := s.atRandomPoint(room, id); p != nil {
			last = p
		}
	}

	return last
}

// densityB is 0x54f500 (handler 9): room-seed gate, then the first success ends it.
func (s *Scatter) densityB(room *RoomCtx, density, id int) *Placed {
	if int(room.Seed.Step()%100) > 100 {
		return nil
	}

	for n := count(room.Rect, density); n > 0; n-- {
		if p := s.atRandomPoint(room, id); p != nil {
			return p
		}
	}

	return nil
}

// farFrom is the callback of the placement loops: no listed point within lim in x or in y.
func farFrom(list [][2]int, x, y, lim int) bool {
	for _, e := range list {
		if abs(e[0]-x) < lim || abs(e[1]-y) < lim {
			return false
		}
	}

	return true
}

func abs(a int) int {
	if a < 0 {
		return -a
	}

	return a
}

// tracked is 0x54f650 (handler 8) with 0x54e370: room-seed placement of a (2s+1) box, kept 100 apart.
func (s *Scatter) tracked(_ LevelRow, room *RoomCtx, id int) {
	g := s.Region
	if g.quota(len(g.Small), 4) {
		return
	}

	d := s.def(id)

	if int(room.Seed.Step()%100) > 100 {
		return
	}

	sx, sy := d.SizeX*2+1, d.SizeY*2+1
	r := room.Rect

	if r.W < 2 || r.H < 2 || r.W <= sx || r.H <= sy {
		return
	}

	for try := 0; try < 5; try++ {
		x := int(room.Seed.Roll(int32(r.W-sx-1))) + r.X
		y := int(room.Seed.Roll(int32(r.H-sy-1))) + r.Y

		if inside(r, x, y) && !s.Col.Blocked(x, y, sx+6, sy+6, MaskPlace) && farFrom(g.Small, x, y, 100) {
			s.create(room, id, x, y)

			if len(g.Small) < 4 {
				g.Small = append(g.Small, [2]int{x, y})
			}

			return
		}
	}
}

// shrine is 0x550bc0 (handler 2) with 0x54e8b0: shrines, wells and the like near the room edge.
func (s *Scatter) shrine(_ LevelRow, room *RoomCtx, id int) {
	g := s.Region
	d := s.def(id)
	step := int(room.Seed.Step() % 100)
	tries := 3
	forced := g.Sparse()

	if forced {
		tries = 30
	} else if 100 < step {
		return
	}

	if g.quota(len(g.Large), 10) {
		return
	}

	for ; tries > 0; tries-- {
		p := s.nearEdge(room, id, d)
		if p == nil {
			continue
		}

		// UNVERIFIED: the exe raises the shrine counter only for a forced shrine (or an object whose
		// data byte is already 2); a forced shrine also becomes a destroyed variant with a new shrine roll.
		if forced {
			g.Shrine++
		}

		if len(g.Large) < 10 {
			g.Large = append(g.Large, [2]int{p.X, p.Y})
		}

		return
	}
}

func (s *Scatter) nearEdge(room *RoomCtx, id int, d Def) *Placed {
	r := room.Rect
	sd := room.Seed

	if r.W < d.SizeX+2 || r.H < d.SizeY+2 {
		return nil
	}

	for try := 0; try < 5; try++ {
		var x, y int

		switch d.Orientation {
		case 1:
			x = int(sd.Roll(int32(r.W/2))) + r.W/4 + r.X
			y = int(sd.Roll(2)) + r.Y
		case 2:
			x = int(sd.Roll(2)) + r.X
			y = int(sd.Roll(int32(r.H/2))) + r.H/4 + r.Y
		default:
			x = int(sd.Roll(int32(r.W-d.SizeX-1))) + r.X
			y = int(sd.Roll(int32(r.H-d.SizeY-1))) + r.Y
		}

		if inside(r, x, y) && !s.Col.Blocked(x, y, d.SizeX+6, d.SizeY+6, MaskPlace) && farFrom(s.Region.Large, x, y, 50) {
			return s.create(room, id, x, y)
		}
	}

	return nil
}

// Ring tables of 0x54eae0 (DAT_0072f538 / 0x72f558).
var (
	ringDX = [8]int{-1, 0, 1, -1, 1, -1, 0, 1}
	ringDY = [8]int{-1, -1, -1, 0, 0, 1, 1, 1}
)

// fits is COLLISION_CanUnitFitInRect (0x54e050).
func (s *Scatter) fits(r Rect, x, y, sx, sy int) bool {
	if sx+2 > r.W || sy+2 > r.H {
		return false
	}

	if !(r.X+1 < x && r.Y+1 < y && x < r.X-sx-2+r.W && y < r.Y-sy-2+r.H) {
		return false
	}

	return !s.Col.Blocked(x, y, sx+7, sy+7, MaskWide) && !s.Col.Blocked(x, y, sx, sy, MaskPlace)
}

// groupByClass is 0x54eae0 (handler 1): clusters of the barrel / urn / casket kinds, picked by object id.
func (s *Scatter) groupByClass(room *RoomCtx, density, id int) {
	d := s.def(id)
	g := s.Game
	r := room.Rect

	if int(g.Step()%100) > 100 {
		return
	}

	tries, pre, ring, base := 12, 3, 5, 5

	switch id {
	case 3:
		tries, pre = 18, 2
	case 0x4f, 1:
		// defaults
	case 4:
		pre, ring, base = 5, 0, 1
	case 0x59:
		pre = 2
	case 0xd0, 0xd1:
		pre, ring, base = 2, 0, 1
	default:
		return
	}

	small := ring == 0
	n := count(r, density)

	if n < 1 {
		return
	}

	canPlace := func(x, y int) bool {
		if small { // OBJRGN_CanPlaceSmallObjectAtPoint (0x54e110)
			if r.W < 2 || r.H < 2 || !(r.X+2 < x && r.Y+2 < y && x < r.X-d.SizeX+r.W && y < r.Y-d.SizeY+r.H) {
				return false
			}

			return !s.Col.Blocked(x, y, d.SizeX+2, d.SizeY+2, MaskPlace)
		}

		return s.fits(r, x, y, d.SizeX, d.SizeY)
	}

	for ; tries > 0 && n > 0; tries-- {
		if pre > 0 {
			g.Step()
		}

		x := int(g.Roll(int32(r.W-d.SizeX-1))) + r.X
		y := int(g.Roll(int32(r.H-d.SizeY-1))) + r.Y

		if !s.fits(r, x, y, d.SizeX, d.SizeY) {
			continue
		}

		s.create(room, id, x, y)

		made, found := 1, true

		for {
			if half := made >> 1; half > 0 && g.Roll(int32(half)) != 0 {
				break
			}

			if !found {
				break
			}

			found = false

			lim := n
			if lim < 4 {
				lim = 4
			}

			for t := 0; t < lim*3 && !found; t++ {
				dir := g.Step() & 7
				ox, oy := 0, 0

				if ring > 0 {
					ox = int(g.Roll(int32(ring)))
				}

				x += (ox + base) * ringDX[dir] * 2

				if ring > 0 {
					oy = int(g.Roll(int32(ring)))
				}

				y += (oy + base) * ringDY[dir] * 2
				found = canPlace(x, y)
			}

			if !found {
				continue // the exe loops back to the halving roll with the same count
			}

			if pre > 0 {
				g.Step()
			}

			s.create(room, id, x, y)

			made++
		}

		n--
	}
}
