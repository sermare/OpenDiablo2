package d2mapgen

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgpop"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monreg"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// The monster and object population of the real-map levels, following the
// original: the DS1 preset units that survive FilterPresetObjects
// (drlgpop), the monsters the presets turn into (drlgpop.ResolveMonster and
// the placement ring walk of d2monreg), and the natural monsters of the
// Levels.txt density / unique rules (d2monreg.Game.PopulateNatural), all on
// the room seeds of the DRLG port. Each piece is proven equal to the emulated
// game in its package; what the engine has to stand in for is listed here:
//
//   - rooms are populated in room-list order at level build (the original does
//     it when a hero comes near, and its game seed depends on that order);
//   - the collision tests of the placement use the engine's walkability, the
//     room "cells" are the whole room rectangle, and "near an exit" is the
//     distance to a special (warp) tile;
//   - the object spawning stage of the room population is not modelled, so
//     the room seed stream is exact only up to it.
const (
	roomNoPopulate = 0x800000 // room flag of a LvlPrest row with Populate=0

	// nearExitSubtiles stands in for the exit distance test.
	nearExitSubtiles = 10
)

// popEnv holds the tables of the population code, loaded once per generator.
type popEnv struct {
	nm    *drlgpop.Names
	tb    *d2monreg.Tables
	stats map[string]*d2records.MonStatRecord // lower case key
}

func (g *MapGenerator) popEnv() (*popEnv, error) {
	if g.pop != nil {
		return g.pop, nil
	}

	get := func(p string) ([]byte, error) {
		b, err := g.asset.LoadFile(p)
		if err != nil || len(b) == 0 {
			return nil, fmt.Errorf("%s: %v", p, err)
		}

		return b, nil
	}

	var raw [6][]byte

	for i, p := range []string{
		"/data/global/excel/monstats.txt", "/data/global/excel/monstats2.txt", "/data/global/excel/Levels.txt",
		"/data/global/excel/SuperUniques.txt", "/data/global/excel/MonPlace.txt", "/data/global/excel/monpreset.txt",
	} {
		b, err := get(p)
		if err != nil {
			return nil, err
		}

		raw[i] = b
	}

	nm, err := drlgpop.ParseNames(raw[0], raw[3], raw[4], raw[5], 0)
	if err != nil {
		return nil, err
	}

	tb, err := d2monreg.ParseTables(raw[0], raw[1], raw[2])
	if err != nil {
		return nil, err
	}

	e := &popEnv{nm: nm, tb: tb, stats: map[string]*d2records.MonStatRecord{}}
	for k, st := range g.asset.Records.Monster.Stats {
		e.stats[strings.ToLower(k)] = st
	}

	g.pop = e

	return e, nil
}

func (e *popEnv) stat(class int) *d2records.MonStatRecord {
	if class < 0 || class >= len(e.tb.Mons) {
		return nil
	}

	return e.stats[strings.ToLower(e.tb.Mons[class].Key)]
}

// popRoom is a populated server room (the engine's rectangle is in map tiles).
type popRoom struct {
	x, y, w, h int
	seed       d2rand.Seed
	noPop      bool
	preset     *popPreset
}

// popPreset is the preset units of one stamped DS1 that are still to be
// handed to rooms.
type popPreset struct {
	nodes []drlgpop.Node // kept nodes, absolute map subtiles
	keep  map[int]bool   // DS1 object records that survived the filter
}

// popLevel collects the rooms and presets of a level being generated.
type popLevel struct {
	g     *MapGenerator
	env   *popEnv
	level int
	diff  d2drlg.Difficulty
	seed  uint32
	rooms []*popRoom
	stats popStats
}

type popStats struct{ preset, super, natural, skipped, unique int }

// newPopLevel prepares the population of a level; nil (after a warning) when
// the tables are unavailable, in which case the caller keeps the old placement.
func (g *MapGenerator) newPopLevel(levelID int, seed uint32, diff d2drlg.Difficulty) *popLevel {
	env, err := g.popEnv()
	if err != nil {
		g.Warningf("real population: tables unavailable, DS1 monsters use the approximation: %v", err)
		return nil
	}

	return &popLevel{g: g, env: env, level: levelID, diff: diff, seed: seed}
}

// addRoom registers a room (tile rectangle in map coordinates and its room
// seed, as the DRLG port produced it). Rooms are populated in call order.
func (p *popLevel) addRoom(x, y, w, h int, seed d2rand.Seed, noPop bool) *popRoom {
	r := &popRoom{x: x, y: y, w: w, h: h, seed: seed, noPop: noPop}
	p.rooms = append(p.rooms, r)

	return r
}

// addPreset resolves the units of a stamped DS1 placed at tile (ox, oy),
// filters them like FilterPresetObjects and returns the preset plus the
// predicate of the surviving DS1 object records. The filter draws from gate
// (the level seed at DS1 load, for the files that are loaded while the level
// is generated), else lazily from the seed of first, the first room that
// touches the preset. first.seed advances accordingly.
func (p *popLevel) addPreset(stamp *d2mapstamp.Stamp, ox, oy int, gate *d2rand.Seed, first *popRoom) *popPreset {
	objs := stamp.Objects()
	raws := make([]drlgpop.Raw, len(objs))

	for i, o := range objs {
		raws[i] = drlgpop.Raw{Type: o.Type, ID: o.ID, X: o.X, Y: o.Y, Flags: o.Flags}
	}

	act := stamp.Act() - 1
	if act < 0 {
		act = 0
	}

	nodes := drlgpop.ResolveDS1(stamp.Version(), act, raws, p.env.nm, d2records.DS1ObjectClass)
	for i := range nodes {
		nodes[i].X += ox * drlgpop.Subtile
		nodes[i].Y += oy * drlgpop.Subtile
	}

	var seed *d2rand.Seed

	switch {
	case gate != nil:
		cp := *gate
		seed = &cp
	case first != nil:
		seed = &first.seed
	default:
		cp := *d2rand.New(p.seed)
		seed = &cp
	}

	kept := drlgpop.Filter(nodes, p.env.nm, seed)
	ps := &popPreset{nodes: kept, keep: map[int]bool{}}

	for _, n := range kept {
		ps.keep[n.Src] = true
	}

	return ps
}

// keepFunc is the entity filter for PlaceStampClippedWhere.
func (ps *popPreset) keepFunc() func(int) bool {
	if ps == nil {
		return nil
	}

	return func(i int) bool { return ps.keep[i] }
}

// run populates the rooms (call after the map is final: walkability is read).
// Preset monsters become NPC entities the monster director adopts; the natural
// monsters are stored as the level's population plan.
func (p *popLevel) run() {
	g := p.g
	w := newEngineWorld(g, p.env.tb.Level(p.level))

	game := d2monreg.NewGame(p.env.tb, p.seed, int(p.diff), true)
	// every level gets its own density stream (the original has one game seed
	// that the order of the visited rooms advances)
	game.Seed = *d2rand.New(p.seed ^ uint32(p.level)*0x9E3779B1)
	game.RoomCount = func(int) int {
		n := 0

		for _, r := range p.rooms {
			if !r.noPop {
				n++
			}
		}

		return n
	}

	ctx := &drlgpop.Ctx{LevelID: p.level, Difficulty: int(p.diff), Counter: new(int)}
	superDone := map[int]bool{}

	var plan []d2mapengine.PlannedMonster

	for _, r := range p.rooms {
		room := &d2monreg.Room{Level: p.level, NoPopulate: r.noPop, Seed: r.seed,
			X: r.x * drlgpop.Subtile, Y: r.y * drlgpop.Subtile, W: r.w * drlgpop.Subtile, H: r.h * drlgpop.Subtile}
		room.Cells = []d2monreg.Cell{{X0: r.x, Y0: r.y, X1: r.x + r.w, Y1: r.y + r.h, Flag: 1}}

		game.Wanderer(room)

		if r.preset != nil {
			var nodes []drlgpop.Node

			nodes, r.preset.nodes = drlgpop.TakeForRoom(r.preset.nodes, drlgpop.Rect{X: r.x, Y: r.y, W: r.w, H: r.h})
			p.createPreset(game, w, room, drlgpop.Requests(nodes, r.x, r.y, ctx, p.env.nm), superDone)
		}

		var pop d2monreg.Population

		game.PopulateNatural(w, room, &pop)

		idx := map[*d2monreg.Unit]int{}

		for _, u := range pop.Units {
			st := p.env.stat(u.Class)
			if st == nil {
				p.stats.skipped++
				continue
			}

			pm := d2mapengine.PlannedMonster{Key: st.Key, X: u.X, Y: u.Y, Leader: -1, Unique: u.Unique, Champion: u.Champion}
			if li, ok := idx[u.Leader]; ok && u.Leader != nil {
				pm.Leader = li
			}

			idx[u] = len(plan)
			plan = append(plan, pm)
			p.stats.natural++

			if u.Unique || u.Champion {
				p.stats.unique++
			}
		}
	}

	g.engine.SetPopulation(plan)
}

// createPreset makes the monsters of one room's preset requests.
func (p *popLevel) createPreset(game *d2monreg.Game, w d2monreg.World, room *d2monreg.Room, reqs []drlgpop.Request, superDone map[int]bool) {
	for _, rq := range reqs {
		if rq.Node.Kind != drlgpop.KindMonster {
			continue
		}

		class := -1

		switch rq.Monster.Kind {
		case drlgpop.MonClass:
			class = rq.Monster.Class
		case drlgpop.MonSuper:
			if superDone[rq.Monster.Super] { // once per game (FUN_005a2480)
				continue
			}

			superDone[rq.Monster.Super] = true

			if rq.Monster.Super < len(p.env.nm.SuperKeys) {
				if su := p.g.asset.Records.Monster.Unique.Super[p.env.nm.SuperKeys[rq.Monster.Super]]; su != nil {
					class = p.env.tb.MonByKey(su.Class)
				}
			}
		default:
			p.stats.skipped++ // nests, champions, unique packs, groups: not made yet
			continue
		}

		var pop d2monreg.Population

		u := game.PlacePreset(w, room, class, rq.X, rq.Y, &pop)
		st := p.env.stat(class)

		if u == nil || st == nil {
			p.stats.skipped++
			continue
		}

		npc, err := p.g.safeNPC(u.X, u.Y, st)
		if err != nil {
			p.g.Warningf("real population: could not place %s: %v", st.Key, err)
			continue
		}

		p.g.engine.AddEntity(npc)

		if rq.Monster.Kind == drlgpop.MonSuper {
			p.stats.super++
		} else {
			p.stats.preset++
		}
	}
}

func (p *popLevel) logSummary(kind string) {
	p.g.Infof("%s: population: %d rooms, %d preset monsters, %d super uniques, %d natural monsters planned (%d packs with modifiers), %d skipped",
		kind, len(p.rooms), p.stats.preset, p.stats.super, p.stats.natural, p.stats.unique, p.stats.skipped)
}

// engineWorld answers the placement's map questions from the engine.
type engineWorld struct {
	g        *MapGenerator
	warps    [][2]int // tiles of special (exit) markers
	warpDist int
}

func newEngineWorld(g *MapGenerator, lv *d2monreg.Level) *engineWorld {
	w := &engineWorld{g: g, warpDist: nearExitSubtiles}
	size := g.engine.Size()

	for y := 0; y < size.Height; y++ {
		for x := 0; x < size.Width; x++ {
			t := g.engine.TileAt(x, y)
			if t == nil {
				continue
			}

			for i := range t.Components.Walls {
				wl := &t.Components.Walls[i]
				if wl.Prop1 != 0 && wl.Type.Special() && wl.Style != startMarkerStyle {
					w.warps = append(w.warps, [2]int{x, y})
					break
				}
			}
		}
	}

	return w
}

func (w *engineWorld) Blocked(_ *d2monreg.Room, x, y, radius, mask int) bool {
	if mask == 0 {
		return false
	}

	if radius < 0 {
		radius = 0
	}

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if w.g.engine.WalkBlocked(x+dx, y+dy) {
				return true
			}
		}
	}

	return false
}

func (w *engineWorld) Open(_ *d2monreg.Room, x, y int) bool {
	return x >= 0 && y >= 0 && w.g.engine.TileExists(x/subtilesPerTile, y/subtilesPerTile)
}

func (w *engineWorld) CellAt(room *d2monreg.Room, x, y int) int {
	for i := range room.Cells {
		c := &room.Cells[i]
		if x >= c.X0*subtilesPerTile && x < c.X1*subtilesPerTile && y >= c.Y0*subtilesPerTile && y < c.Y1*subtilesPerTile {
			return c.Flag
		}
	}

	return 0
}

func (w *engineWorld) NearExit(_ *d2monreg.Room, x, y int) bool {
	for _, p := range w.warps {
		dx, dy := x-(p[0]*subtilesPerTile+2), y-(p[1]*subtilesPerTile+2)
		if dx < 0 {
			dx = -dx
		}

		if dy < 0 {
			dy = -dy
		}

		if dx <= w.warpDist && dy <= w.warpDist {
			return true
		}
	}

	return false
}
