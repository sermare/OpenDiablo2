package d2gamescreen

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
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
	size := m.Size()
	hx, hy := v.heroTilePos()
	groups, units, blocks := 0, 0, 0

	// a block that already holds a monster (DS1 markers) is left alone
	occupied := map[[2]int]bool{}

	for _, mon := range v.monsters.Monsters() {
		x, y := mon.GetPositionF()
		occupied[[2]int{int(x) / populateBlockTiles, int(y) / populateBlockTiles}] = true
	}

	for by := 0; by*populateBlockTiles < size.Height; by++ {
		for bx := 0; bx*populateBlockTiles < size.Width; bx++ {
			x0, y0 := bx*populateBlockTiles, by*populateBlockTiles
			cx, cy := float64(x0)+populateBlockTiles/2, float64(y0)+populateBlockTiles/2

			if occupied[[2]int{bx, by}] || math.Hypot(cx-hx, cy-hy) < populateSafeTiles {
				continue
			}

			w, h := minInt(populateBlockTiles, size.Width-x0), minInt(populateBlockTiles, size.Height-y0)
			if v.walkableShare(x0, y0, w, h) < populateMinWalkableShare {
				continue
			}

			blocks++

			room := d2monsters.Room{X0: x0 * subtiles, Y0: y0 * subtiles, W: w * subtiles, H: h * subtiles}

			res, err := v.monsters.PopulateRoom(room, level)
			if err != nil {
				v.Warningf("POPULATE level %d block (%d,%d): %v", level, bx, by, err)
				return
			}

			groups += len(res)

			for _, r := range res {
				units += len(r.Monsters)
			}
		}
	}

	// a monster the hero can never reach (an island of floor behind cliffs and
	// water) is no use: it could not be killed, and a quest that wants the
	// level cleared would never finish
	removed := v.removeUnreachableMonsters()

	v.Infof("POPULATE level %d (%s): %d blocks, %d groups, %d monsters (%d unreachable ones removed)", level,
		v.levelName(level), blocks, groups, units-removed, removed)
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
		if m.CanWalkTo(hero[0], hero[1], x, y) {
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

func (v *Game) walkableShare(x0, y0, w, h int) float64 {
	m := v.gameClient.MapEngine
	total, ok := 0, 0

	for sy := y0 * subtiles; sy < (y0+h)*subtiles; sy++ {
		for sx := x0 * subtiles; sx < (x0+w)*subtiles; sx++ {
			total++

			if !m.WalkBlocked(sx, sy) {
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
