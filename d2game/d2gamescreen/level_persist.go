package d2gamescreen

import (
	"math"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// Level persistence. In the original a level keeps its units while the game
// lives: monsters the hero killed do not come back, chests he opened stay
// open and what he dropped stays on the ground. Here the map is rebuilt from
// the seed on every level change, so the state of the level the hero leaves is
// kept in a store and applied to the rebuilt map:
//
//   - monsters: the living hostile ones are parked by the director and put back
//     (the DS1 placements of the new map are discarded and the level is not
//     populated again, so a cleared level stays cleared);
//   - objects: the opened ones (chests, barrels, doors left open) are matched
//     by objects.txt row and tile and put into their opened state;
//   - items: the ground items and gold piles are put back as they lay.
//
// Not kept (UNVERIFIED against the original): town portals, shrines' effects,
// objects that were created while playing, followers.
// OD2_NOPERSIST=1 turns the store off.

// savedObject identifies a map object by its objects.txt row and tile.
type savedObject struct {
	row, x, y int
}

// savedLevel is the state of one left level.
type savedLevel struct {
	parked   *d2monsters.ParkedLevel
	opened   map[savedObject]bool
	items    []*d2mapentity.Item
	visits   int
	monsters int
}

// levelStore keeps the state of the levels the hero has left, by level id.
type levelStore struct {
	levels map[int]*savedLevel
}

func (s *levelStore) put(id int, l *savedLevel) {
	if s.levels == nil {
		s.levels = map[int]*savedLevel{}
	}

	if old := s.levels[id]; old != nil {
		l.visits = old.visits
	}

	s.levels[id] = l
}

// take returns the saved state of a level (nil if the hero never left it).
func (s *levelStore) take(id int) *savedLevel {
	if s.levels == nil {
		return nil
	}

	return s.levels[id]
}

func persistEnabled() bool { return os.Getenv("OD2_NOPERSIST") != "1" }

// persistable says whether the level with this id keeps its state: the real
// dungeon and wilderness levels, never a town (its NPCs are fixed anyway).
func (v *Game) persistable(level int) bool {
	return persistEnabled() && v.isRealLevel() && !d2level.IsTown(level)
}

func objectKey(ob *d2mapentity.Object) savedObject {
	x, y := ob.GetPositionF()

	return savedObject{ob.Record().Index, int(math.Floor(x)), int(math.Floor(y))}
}

// saveLevel records the state of the level the hero is about to leave. It must
// run before the map is rebuilt.
func (v *Game) saveLevel(level int) {
	if !v.persistable(level) {
		return
	}

	sl := &savedLevel{opened: map[savedObject]bool{}, visits: 1}

	if v.monsters != nil { // no director yet means nothing was ever spawned
		sl.parked = v.monsters.ParkLevel()
		sl.monsters = sl.parked.Len()
	}

	for _, e := range v.gameClient.MapEngine.Entities() {
		switch ent := e.(type) {
		case *d2mapentity.Object:
			if ent.IsOpened() && ent.PortalDest == 0 {
				sl.opened[objectKey(ent)] = true
			}
		case *d2mapentity.Item:
			sl.items = append(sl.items, ent)
		}
	}

	v.levelStore.put(level, sl)

	v.Infof("PERSIST saved level %d (%s): %d monsters, %d opened objects, %d ground items", level, v.levelName(level),
		sl.monsters, len(sl.opened), len(sl.items))
}

// restoreLevel applies the saved state of a level to its freshly built map.
// Levels the hero has not left before keep what the build and the population
// give them.
func (v *Game) restoreLevel(level int) {
	if !v.persistable(level) {
		return
	}

	sl := v.levelStore.take(level)
	if sl == nil {
		return
	}

	// resetLevelState dropped the director of the old map; this level gets a
	// new one that adopts the parked monsters
	dir := v.monsterDirector()
	if dir == nil {
		return
	}

	sl.visits++

	engine := v.gameClient.MapEngine
	discarded := dir.DiscardPlacements()
	restored := dir.RestoreLevel(sl.parked)

	// the monsters the hero left alive come back as they were; the level must
	// not be populated again
	v.populated = v.levels.changes + 1

	objects := 0

	for _, e := range engine.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok || !sl.opened[objectKey(ob)] {
			continue
		}

		if changed, err := ob.RestoreOpened(); err != nil {
			v.Warningf("PERSIST could not restore %q: %v", ob.Label(), err)
		} else if changed {
			engine.AddEntity(ob) // a door's collision follows its state
			objects++
		}
	}

	for _, it := range sl.items {
		engine.AddEntity(it)
	}

	v.Infof("PERSIST restored level %d (%s) visit %d: %d monsters back (%d placements dropped), %d of %d opened objects restored, %d ground items",
		level, v.levelName(level), sl.visits, restored, discarded, objects, len(sl.opened), len(sl.items))
}
