package d2gamescreen

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapgen"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// Populating a level with the natural monsters of its Levels.txt row.
//
// The DS1 presets of the outdoor levels carry few monster markers (Blood
// Moor's 82 rooms yield none), and the caves only mark a handful of places;
// the original fills the rooms of every level with groups drawn from the
// level's monster list (MONREGION_PickLevelMonsterTypes, density from the
// level record). The director's PopulateRoom implements that draw; this file
// feeds it the rooms of the freshly built map: blocks of 8x8 tiles (the size
// of an outdoor DRLG room) that have walkable ground and no monster yet.

const (
	populateBlockTiles = 16
	// populateSafeTiles is the distance around the hero's arrival that stays
	// empty, so a level does not open with a fight on top of the hero.
	populateSafeTiles = 8.0
	// populateMinWalkable is the share of walkable sub-tiles a block needs.
	populateMinWalkableShare = 0.4
	// populateSparseWalkableShare is the share used when no block reaches the
	// usual one (engine choice, UNVERIFIED against the original's rooms).
	populateSparseWalkableShare = 0.15
	// populateRetries bounds the extra draws per room for a level that got no group.
	populateRetries = 12
)

// populateLevel fills the level the hero stands in. It runs once per level
// build (levels.changes counts the builds) and never in towns.
func (v *Game) populateLevel() {
	if v.populated == v.levels.changes+1 || v.monsters == nil || v.localPlayer == nil {
		return
	}

	v.populated = v.levels.changes + 1

	level := v.currentLevel()
	if d2level.IsTown(level) || !autoPlayPopulate() || !v.isRealLevel() {
		return
	}

	m := v.gameClient.MapEngine

	if plan := m.Population(); plan != nil {
		v.spawnPlannedPopulation(level, plan)
		return
	}

	size := m.Size()
	hx, hy := v.heroTilePos()
	groups, blocks := 0, 0
	natural := map[*d2mapentity.Monster]bool{}

	// the ground the hero can reach from where he arrives: packs are drawn there only (a lava sea or the
	// top of a cliff inside a block is no ground), and a block is judged by its reachable share
	reach := m.ReachableFrom(int(hx*subtiles), int(hy*subtiles))

	// a block that already holds a monster (DS1 markers) is left alone
	occupied := map[[2]int]bool{}

	for _, mon := range v.monsters.Monsters() {
		x, y := mon.GetPositionF()
		occupied[[2]int{int(x) / populateBlockTiles, int(y) / populateBlockTiles}] = true
	}

	var rooms []d2monsters.Room

	fill := func(room d2monsters.Room) bool {
		res, err := v.monsters.PopulateRoom(room, level)
		if err != nil {
			v.Warningf("POPULATE level %d room %+v: %v", level, room, err)
			return false
		}

		groups += len(res)

		for _, g := range res {
			for _, m := range g.Monsters {
				natural[m] = true
			}
		}

		return true
	}

	// a level with no block of the usual walkable share (the Kurast Causeway is a
	// bridge between canals) is tried again with the sparse share, so it is not
	// left empty
	for _, minShare := range []float64{populateMinWalkableShare, populateSparseWalkableShare} {
		for by := 0; by*populateBlockTiles < size.Height; by++ {
			for bx := 0; bx*populateBlockTiles < size.Width; bx++ {
				x0, y0 := bx*populateBlockTiles, by*populateBlockTiles
				cx, cy := float64(x0)+populateBlockTiles/2, float64(y0)+populateBlockTiles/2

				if occupied[[2]int{bx, by}] || math.Hypot(cx-hx, cy-hy) < populateSafeTiles {
					continue
				}

				w, h := minInt(populateBlockTiles, size.Width-x0), minInt(populateBlockTiles, size.Height-y0)
				if v.walkableShare(x0, y0, w, h, reach) < minShare {
					continue
				}

				blocks++

				room := d2monsters.Room{X0: x0 * subtiles, Y0: y0 * subtiles, W: w * subtiles, H: h * subtiles}
				if reach != nil {
					room.Reach = reach.At
					room.WalkTiles = int(v.walkableShare(x0, y0, w, h, reach)*float64(w*h) + 0.5)
				}

				rooms = append(rooms, room)

				if !fill(room) {
					return
				}
			}
		}

		if blocks > 0 {
			break
		}
	}

	// the fractional density of a sparse level can round every block to no group
	// (Kurast Causeway: one bridge block of 0.8 monsters): draw again, a few
	// times, so a level with ground to stand on is never left empty
	for try := 0; groups == 0 && len(rooms) > 0 && try < populateRetries*len(rooms); try++ {
		if !fill(rooms[try%len(rooms)]) {
			return
		}
	}

	// a monster the hero can never reach (an island of floor behind cliffs and
	// water) is no use: it could not be killed, and a quest that wants the
	// level cleared would never finish
	total := len(v.monsters.Monsters()) // the director's list still holds the removed ones until the next frame
	removed := v.removeUnreachableMonsters()

	v.logPopulateTypes(level, natural)

	v.Infof("POPULATE level %d (%s): %d blocks, %d groups, %d monsters (%d unreachable ones removed)", level,
		v.levelName(level), blocks, groups, total-removed, removed)
}

// spawnPlannedPopulation creates the natural monsters the real-map generator
// decided (d2mapgen real_pop.go: the original's room population on the DRLG
// room seeds), links pack followers to their leader and drops the ones the
// hero could not reach.
func (v *Game) spawnPlannedPopulation(level int, plan []d2mapengine.PlannedMonster) {
	made := make([]*d2mapentity.Monster, len(plan))
	natural := map[*d2mapentity.Monster]bool{}
	packs := map[*d2mapentity.Monster]bool{}  // unique / champion packs: their types are logged apart (umon list, not the level's drawn types)
	supers := map[*d2mapentity.Monster]bool{} // super uniques with every follower they came with (superuniques.txt class, not the level's types)
	kinds := classifyPlan(plan)
	units := 0

	for i, pm := range plan {
		stat := v.monsters.FindStat(pm.Key)
		if stat == nil {
			continue
		}

		mon, err := v.monsters.Spawn(stat, pm.X, pm.Y)
		if err != nil {
			v.Warningf("POPULATE level %d: %s at (%d,%d): %v", level, pm.Key, pm.X, pm.Y, err)
			continue
		}

		made[i] = mon

		switch kinds[i] {
		case planSuper:
			supers[mon] = true
		case planPack:
			packs[mon] = true
		default:
			natural[mon] = true
		}

		units++

		applyPlannedRank(mon, pm)

		// the boss shows (and reports in kill events) its own name, "Ismail Vilehand": the Council quest scenarios
		// and the Flail drop read it. The director path (groups.go recordRank) does the same.
		if pm.SuperKey != "" {
			if rec := v.asset.Records.Monster.Unique.Super[pm.SuperKey]; rec != nil && rec.Name != "" {
				mon.SetLabel(rec.Name)
			}
		}

		if pm.Leader >= 0 && pm.Leader < len(made) && made[pm.Leader] != nil {
			v.monsters.Group(made[pm.Leader], mon)
		}
	}

	removed := v.removeUnreachableMonsters()
	v.logPopulateRanks(level)

	v.logPopulateTypes(level, natural)
	v.logPopulateClasses("packs", level, packs)
	v.logPopulateClasses("supers", level, supers)

	v.Infof("POPULATE groups level %d: %d", level, countPlanGroups(plan, kinds))

	v.Infof("POPULATE level %d (%s): %d planned, %d monsters (%d unreachable ones removed)", level,
		v.levelName(level), len(plan), units-removed, removed)
}

type planKind int

const (
	planNatural planKind = iota // plain groups of the level's drawn types (and their party packs)
	planPack                    // rare / champion leaders and everything created around them
	planSuper                   // super uniques and everything created around them
)

// classifyPlan sorts the planned monsters by the kind of pack they belong to. A follower takes the kind of the unit it
// was created around (Origin, which also covers the party packs and extras the original does not link to their
// leader, else Leader), so a super unique's minions are not mistaken for natural monsters of the level's types.
func classifyPlan(plan []d2mapengine.PlannedMonster) []planKind {
	kinds := make([]planKind, len(plan))

	for i, pm := range plan {
		switch {
		case pm.SuperKey != "":
			kinds[i] = planSuper
		case pm.Unique || pm.Champion:
			kinds[i] = planPack
		case pm.Origin > 0 && pm.Origin-1 < i:
			kinds[i] = kinds[pm.Origin-1]
		case pm.Leader >= 0 && pm.Leader < i:
			kinds[i] = kinds[pm.Leader]
		}
	}

	return kinds
}

// countPlanGroups counts the groups the density rolls made: the units that were not created around another one, super
// uniques (placed from presets, not rolled) left out.
func countPlanGroups(plan []d2mapengine.PlannedMonster, kinds []planKind) int {
	n := 0

	for i, pm := range plan {
		if pm.Leader < 0 && pm.Origin == 0 && kinds[i] != planSuper {
			n++
		}
	}

	return n
}

// applyPlannedRank gives a planned monster the rank the original's population
// decided: type bits (the exe's +0x16 mask: 1 modifiers rolled, 2 super unique,
// 4 champion, 8 unique, 0x10 minion), the monumod ids it carries and, for a
// super unique, its superuniques.txt key and hcIdx. A hit point bonus or the
// effect of a modifier is not applied here.
func applyPlannedRank(mon *d2mapentity.Monster, pm d2mapengine.PlannedMonster) {
	switch {
	case pm.SuperKey != "":
		mon.TypeFlags |= d2mapentity.MonTypeSuperUnique | d2mapentity.MonTypeModsRolled
		mon.SuperUnique, mon.SuperUniqueIdx = pm.SuperKey, pm.SuperIdx
	case pm.Champion:
		mon.TypeFlags |= d2mapentity.MonTypeChampion
	case pm.Unique:
		mon.TypeFlags |= d2mapentity.MonTypeUnique | d2mapentity.MonTypeModsRolled
	}

	if pm.Champion && pm.Unique {
		mon.TypeFlags |= d2mapentity.MonTypeModsRolled
	}

	if pm.Minion {
		mon.TypeFlags |= d2mapentity.MonTypeMinion
	}

	if len(pm.Mods) > 0 {
		mon.Modifiers = append([]int(nil), pm.Mods...)
	}
}

// logPopulateRanks logs the ranked monsters that survived the reachability
// filter: one "POPULATE ranks level N:" line with the counts and one
// "POPULATE rank leaders level N:" line with "key:C" (champion), "key:R" (rare)
// and "key:S" (super unique) per pack leader, for the unique / champion
// scenario (scripts/verify.d/9g-uniques-champions.sh checks them against the
// tables of the level).
func (v *Game) logPopulateRanks(level int) {
	champs, rares, minions, supers := 0, 0, 0, 0

	var leaders []string

	for _, mon := range v.monsters.Monsters() {
		if mon.Stat == nil {
			continue
		}

		switch f := mon.TypeFlags; {
		case f&d2mapentity.MonTypeSuperUnique != 0:
			supers++

			leaders = append(leaders, mon.Stat.Key+":S")
		case f&d2mapentity.MonTypeMinion != 0:
			minions++
		case f&d2mapentity.MonTypeChampion != 0:
			champs++

			leaders = append(leaders, mon.Stat.Key+":C")
		case f&d2mapentity.MonTypeUnique != 0:
			rares++

			leaders = append(leaders, mon.Stat.Key+":R")
		}
	}

	sort.Strings(leaders)

	v.Infof("POPULATE ranks level %d: champions=%d rares=%d minions=%d supers=%d", level, champs, rares, minions, supers)
	v.Infof("POPULATE rank leaders level %d: %s", level, strings.Join(leaders, " "))
}

// logPopulateTypes logs the classes of the natural monsters that survived the
// reachability filter, one "POPULATE types" line per level (both population paths),
// for the per-level population scenarios (scripts/verify.d/lib/poplevel.sh), which check every
// class against the level's Levels.txt row (mon1..mon10 plus the minions of
// those classes).
func (v *Game) logPopulateTypes(level int, natural map[*d2mapentity.Monster]bool) {
	v.logPopulateClasses("types", level, natural)
}

// logPopulateClasses logs "POPULATE <what> level N: key:count ..." for a set of monsters.
func (v *Game) logPopulateClasses(what string, level int, natural map[*d2mapentity.Monster]bool) {
	counts := map[string]int{}

	for _, mon := range v.monsters.Monsters() {
		if natural[mon] && mon.Stat != nil {
			counts[mon.Stat.Key]++
		}
	}

	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ":" + strconv.Itoa(counts[k])
	}

	v.Infof("POPULATE %s level %d: %s", what, level, strings.Join(parts, " "))
}

// removeUnreachableMonsters deletes the monsters standing where the hero cannot
// walk to from where he is (an island of floor behind cliffs and water, a
// room the generator did not connect), and returns how many it removed.
func (v *Game) removeUnreachableMonsters() int {
	m := v.gameClient.MapEngine
	hero := v.heroSubtile()
	removed := 0

	for _, mon := range v.monsters.Monsters() {
		x, y := mon.SubtilePos()
		if m.CanWalkTo(hero[0], hero[1], x, y) || mon.SuperUnique != "" { // a quest boss is never dropped
			continue
		}

		v.Infof("POPULATE removing %q at subtile (%d,%d): the hero cannot walk there", mon.Label(), x, y)
		m.RemoveEntity(mon)

		removed++
	}

	return removed
}

// isRealLevel says whether the map is one of the DRLG levels (OD2_REALMAPS=1);
// the old generator's wilderness is not a real level and stays empty.
func (v *Game) isRealLevel() bool {
	return d2mapgen.RealMapsEnabled()
}

func (v *Game) heroSubtile() [2]int {
	x, y := v.heroTilePos()

	return [2]int{int(x * subtiles), int(y * subtiles)}
}

func (v *Game) walkableShare(x0, y0, w, h int, reach *d2mapengine.Reachable) float64 {
	m := v.gameClient.MapEngine
	total, ok := 0, 0

	for sy := y0 * subtiles; sy < (y0+h)*subtiles; sy++ {
		for sx := x0 * subtiles; sx < (x0+w)*subtiles; sx++ {
			total++

			if !m.WalkBlocked(sx, sy) && (reach == nil || reach.At(sx, sy)) {
				ok++
			}
		}
	}

	if total == 0 {
		return 0
	}

	return float64(ok) / float64(total)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
