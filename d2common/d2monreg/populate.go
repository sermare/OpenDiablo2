package d2monreg

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// Game is the state the natural population reads and changes.
type Game struct {
	Tables  *Tables
	Regions []*Region
	// Seed is the game seed (game+0xd0): every cell of every room draws its
	// density roll from it, so the result depends on the order in which rooms
	// are populated.
	Seed       d2rand.Seed
	Difficulty int
	Expansion  bool
	// ChaosQuiet is FUN_005b2e40 != 0: level 108 is not populated.
	ChaosQuiet bool
	// RoomCount returns the number of rooms of a level the game populates
	// (FUN_00644070); called once per level.
	RoomCount func(level int) int
}

// NewGame builds the regions of a game (MONREGION_InitForGame).
func NewGame(t *Tables, seed uint32, diff int, expansion bool) *Game {
	g := &Game{Tables: t, Difficulty: diff, Expansion: expansion, Seed: *d2rand.New(seed)}
	g.Regions, _ = BuildRegions(t, &g.Seed, diff, expansion)

	return g
}

// Unit is a monster the population created (the call of FUN_005532e0 plus the
// bookkeeping the creation does).
type Unit struct {
	Class    int
	X, Y     int
	Seed     d2rand.Seed // the unit's own seed (a room-seed step at creation)
	Unique   bool
	Champion bool
	Minion   bool
	Leader   *Unit
}

// Event is one entry of the creation log: "c" a unit was created, "champ" and
// "uniq" the modifier pick that followed for a pack leader.
type Event struct {
	Kind  string
	Class int
	X, Y  int
}

// Population collects what a population created, in creation order.
type Population struct {
	Units []*Unit
	Log   []Event
}

func (g *Game) region(room *Room) *Region {
	if room.Level <= 0 || room.Level >= len(g.Regions) {
		return nil
	}

	return g.Regions[room.Level]
}

// create makes the unit of a successful placement: UNIT_InitCreatedUnit takes
// one step of the room seed for the unit's own seed.
func (g *Game) create(pop *Population, room *Room, class, x, y int) *Unit {
	u := &Unit{Class: class, X: x, Y: y}
	lo := room.Seed.Step()
	u.Seed = *d2rand.New(lo)
	pop.Units = append(pop.Units, u)
	pop.Log = append(pop.Log, Event{Kind: "c", Class: class, X: x, Y: y})

	return u
}

// eligible is FUN_0054ca00.
func (g *Game) eligible(room *Room) bool {
	r := g.region(room)
	if r == nil {
		return false
	}

	r.RoomsSeen++

	if room.Level == 0 {
		return false
	}

	if r.TotalRooms < 0 {
		r.TotalRooms = g.RoomCount(room.Level)
	}

	if r.Density != 0 && r.TotalRooms != 0 {
		if room.Level == 0x6c && g.ChaosQuiet {
			return false
		}

		return true
	}

	return false
}

// pickType is FUN_005bba00: the class of a natural monster. mode1 draws from
// the level's umon list in normal difficulty.
func (g *Game) pickType(r *Region, room *Room, thresh int, mode1 bool) (class int, ok bool) {
	t := g.Tables
	s := &room.Seed

	if !mode1 || g.Difficulty != 0 {
		if len(r.Types) == 0 {
			return -1, false
		}

		v := int(s.Roll(int32(r.TotalRarity))) + 1
		i := 0

		for i < len(r.Types) {
			v -= r.Types[i].Rarity
			if v <= 0 {
				break
			}

			i++
		}

		if i >= len(r.Types) {
			return -1, false // the game reads one entry past the end
		}

		class = r.Types[i].Class

		if class >= 0 && class < len(t.Mons) {
			m := &t.Mons[class]
			if m.Spawn >= 0 && m.Has(FlagPlaceSpawn) {
				if int(s.Step()%100) > thresh {
					class = m.Spawn
				}
			}
		}
	} else {
		lv := t.Level(r.Level)
		if lv == nil || lv.CountUMon == 0 {
			return -1, false
		}

		class = lv.UMon[int(s.Roll(int32(lv.CountUMon)))]
	}

	if class < 0 || class >= len(t.Mons) {
		return -1, false
	}

	return class, true
}

// packKind is FUN_005bbba0: 0 makes a unique / champion pack, anything else a
// plain group.
func (g *Game) packKind(r *Region, room *Room) int {
	s := &room.Seed

	if r.Uniques < r.UMin && r.TotalRooms != 0 {
		if int(s.Step()%100) < (r.RoomsSeen*100)/r.TotalRooms {
			return 0
		}
	}

	if r.Uniques < r.UMax {
		if s.Step()%100 < 6 {
			return 0
		}
	}

	if s.Step()%100 > 0x23 {
		return 2
	}

	return 1
}

// PopulateNatural is FUN_0054cad0: the natural monsters of one room. The
// density rolls use the game seed, everything else the room seed.
func (g *Game) PopulateNatural(w World, room *Room, pop *Population) {
	if !g.eligible(room) {
		return
	}

	r := g.region(room)
	if r.Density > 10000 {
		r.Density = 10000
	}

	placed := false

	for ci := range room.Cells {
		c := &room.Cells[ci]
		if c.Flag == 0 || c.Skip {
			continue
		}

		l, t, rr, b := c.subRect()
		if l == 0 && rr == 0 {
			continue
		}

		n := ((rr - l) / 3) * ((b - t) / 3)

		for ; n > 0; n-- {
			if int(g.Seed.Step()%100000) > r.Density {
				continue
			}

			class, ok := g.pickType(r, room, 0x14, false)
			if !ok {
				return
			}

			kind := g.packKind(r, room)

			if kind == 0 {
				if g.makeUnique(w, room, r, c, pop) {
					placed = true
				}

				continue
			}

			mon := &g.Tables.Mons[class]
			lo, hi := 1, 1

			if !g.noGroup(class) {
				lo, hi = mon.MinGrp, mon.MaxGrp
			}

			if mon.Sparse != 0 {
				if int(g.Seed.Step()%100) > mon.Sparse {
					continue
				}
			}

			if g.spawnGroup(w, room, c, class, lo, hi, pop) != nil {
				placed = true
			}
		}
	}

	if placed {
		r.Placed++
	}
}

// noGroup is FUN_0054ca80: classes whose base is 0x13 or 0x5b always spawn
// alone-sized groups (min = max = 1).
func (g *Game) noGroup(class int) bool {
	if class < 0 || class >= len(g.Tables.Mons) {
		return false
	}

	b := g.Tables.Mons[class].BaseID

	return b == 0x13 || b == 0x5b
}

// pickPos is FUN_0054ba70: up to 20 random spots of the cell, each tested
// with a placement that does not create anything.
func (g *Game) pickPos(w World, room *Room, c *Cell, class int, nearExit bool) (x, y int, ok bool) {
	l, t, rr, b := c.subRect()
	x0, y0 := l+1, t+1
	wd, ht := rr-x0, b-y0

	for try := 0; try < 20; try++ {
		x = x0 + int(room.Seed.Roll(int32(wd)))
		y = y0 + int(room.Seed.Roll(int32(ht)))

		if nearExit && w.NearExit(room, x, y) {
			continue
		}

		if w.CellAt(room, x, y) != c.Flag {
			continue
		}

		if _, _, ok := g.Tables.Place(w, Placement{Room: room, Cell: c, Class: class, X: x, Y: y, Ring: -1, Flags: flagTestOnly, Param4: 1}); ok {
			return x, y, true
		}
	}

	return 0, 0, false
}

// spawnGroup is FUN_0054bdf0: the leader at a random spot, then
// min-1 .. max-1 followers around it (rolled with the leader's own seed).
func (g *Game) spawnGroup(w World, room *Room, c *Cell, class, lo, hi int, pop *Population) *Unit {
	if lo == 0 || hi == 0 || hi < lo {
		return nil
	}

	x, y, ok := g.pickPos(w, room, c, class, true)
	if !ok {
		return nil
	}

	px, py, ok := g.Tables.Place(w, Placement{Room: room, Cell: c, Class: class, X: x, Y: y, Ring: -1, Param4: 1})
	if !ok {
		return nil
	}

	leader := g.create(pop, room, class, px, py)
	lo--
	hi--

	n := int(leader.Seed.Roll(int32(hi-lo+1))) + lo

	for ; n > 0; n-- {
		g.follower(w, room, c, leader, class, 3, pop)
	}

	return leader
}

// follower is FUN_005b0bc0: a unit of the class placed on the ring of the
// given radius around the leader (it becomes a follower of the leader).
func (g *Game) follower(w World, room *Room, c *Cell, leader *Unit, class, ring int, pop *Population) *Unit {
	x, y, ok := g.Tables.Place(w, Placement{Room: room, Cell: c, Class: class, X: leader.X, Y: leader.Y, Ring: ring, Param4: 1})
	if !ok {
		return nil
	}

	u := g.create(pop, room, class, x, y)
	u.Minion = true
	u.Leader = leader

	return u
}

// makeUnique is the unique / champion pack branch (FUN_005bba00 mode 1,
// FUN_005a1ec0, FUN_0054c050); see unique.go.
func (g *Game) makeUnique(w World, room *Room, r *Region, c *Cell, pop *Population) bool {
	return g.uniquePack(w, room, r, c, pop)
}
