package d2gamescreen

import (
	"fmt"
	"math"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client"
)

// Walking out of a level through its border (the seamless exits between the
// Act 1 outdoor levels and the town) and through the warp tiles (cave
// entrances, stairs). The geometry is d2level.EdgeExit and TileDestination;
// this file is the glue to the hero, the map engine and the fade.

const (
	edgeMargin  = 1.5 // tiles from the border at which the hero crosses
	edgeInset   = 3.0 // tiles inside the new level where the hero arrives
	exitTimeout = 60.0
	// exitStuckSeconds is how long the hero may stand still before the walk
	// to an exit tries the next candidate cell
	exitStuckSeconds = 1.5
	subtiles         = 5
)

// heroWorld returns the hero's position in world tiles and the world of the map.
func (v *Game) heroWorld() (w d2mapengine.World, wx, wy float64, ok bool) {
	w = v.gameClient.MapEngine.World()
	if w.Level == 0 {
		return w, 0, 0, false
	}

	hx, hy := v.heroTilePos()

	return w, float64(w.OriginX) + hx, float64(w.OriginY) + hy, true
}

// advanceEdges starts the level change when the hero reaches a level border.
func (v *Game) advanceEdges() {
	if v.levels.trans != nil || v.localPlayer == nil || v.localPlayer.IsDead() {
		return
	}

	w, wx, wy, ok := v.heroWorld()
	if !ok {
		return
	}

	to, near := d2level.EdgeExit(w.Rects, w.Level, wx, wy, edgeMargin)
	if !near {
		v.levels.edgeArmed = true
		return
	}

	if !v.levels.edgeArmed {
		return
	}

	if !v.gameClient.CanLoadLevel(to) {
		v.levels.edgeArmed = false

		v.Infof("LEVEL edge of level %d towards level %d (%s) refused: the engine cannot load that level yet",
			w.Level, to, v.levelName(to))

		return
	}

	v.Infof("LEVEL edge: level %d -> %d (%s) at world (%.1f,%.1f)", w.Level, to, v.levelName(to), wx, wy)

	v.levels.edgeArmed = false

	if v.startLevelChange(to, d2level.StartDefault, "edge") {
		v.levels.trans.edgeWX, v.levels.trans.edgeWY = wx, wy
	}
}

// edgeArrival places the hero at the world position he left the old level at.
func edgeArrival(from, to int, wx, wy float64) d2client.ArrivalFunc {
	return func(m *d2mapengine.MapEngine) (x, y float64, ok bool) {
		w := m.World()

		ax, ay, ok := d2level.EdgeArrival(w.Rects, from, to, wx, wy, edgeInset)
		if !ok {
			return 0, 0, false
		}

		return ax - float64(w.OriginX), ay - float64(w.OriginY), true
	}
}

// nextToWarpBackTo picks the warp tile of the new level that leads back to the
// level the hero came from (the cave entrance of the wilderness, the stairs)
// as the arrival point.
func nextToWarpBackTo(from, to int) d2client.ArrivalFunc {
	return func(m *d2mapengine.MapEngine) (x, y float64, ok bool) {
		for _, t := range m.WarpTiles() {
			if dest, found := d2level.TileDestination(to, t.Style); found && dest == from {
				return float64(t.TileX) + 0.5, float64(t.TileY) + 0.5, true
			}
		}

		return 0, 0, false
	}
}

// exitWalk is a scripted walk to an exit of the level (walkto:exit=<level>).
type exitWalk struct {
	level      int
	elapsed    float64
	candidates [][2]float64 // local tile positions to try, nearest first
	next       int
	standStill float64
	paused     bool // a defensive fight interrupted the walk
	lastX      float64
	lastY      float64
	warp       *d2mapengine.WarpTile
}

// exitCandidates lists local positions from which the hero leaves towards
// level: the cells of the shared border (inside the crossing margin) or the
// warp tiles that lead there.
func (v *Game) exitCandidates(level int) (out [][2]float64, warp *d2mapengine.WarpTile, err error) {
	cur := v.currentLevel()
	m := v.gameClient.MapEngine
	hx, hy := v.heroTilePos()

	for i := range v.levels.warps {
		t := &v.levels.warps[i]
		if dest, ok := d2level.TileDestination(cur, t.Style); ok && dest == level {
			if warp == nil || math.Hypot(float64(t.TileX)-hx, float64(t.TileY)-hy) <
				math.Hypot(float64(warp.TileX)-hx, float64(warp.TileY)-hy) {
				warp = t
			}
		}
	}

	if warp != nil {
		return [][2]float64{{float64(warp.TileX) + 0.5, float64(warp.TileY) + 0.5}}, warp, nil
	}

	w := m.World()
	if w.Level == 0 {
		return nil, nil, fmt.Errorf("level %d has no exit towards level %d", cur, level)
	}

	a, okA := w.Rects[cur]
	b, okB := w.Rects[level]

	bd, touch := d2level.SharedBorder(a, b)
	if !okA || !okB || !touch {
		return nil, nil, fmt.Errorf("level %d does not border level %d", cur, level)
	}

	// every walkable cell inside the crossing margin along the shared border
	for along := float64(bd.From) + 0.5; along < float64(bd.To); along++ {
		for depth := 0.5; depth <= edgeMargin-0.5; depth++ {
			wx, wy := float64(bd.Pos), along

			switch bd.Side {
			case d2level.West:
				wx += depth
			case d2level.East:
				wx -= depth
			case d2level.North:
				wx, wy = along, float64(bd.Pos)+depth
			default:
				wx, wy = along, float64(bd.Pos)-depth
			}

			lx, ly := wx-float64(w.OriginX), wy-float64(w.OriginY)
			if lx < 0 || ly < 0 || lx >= float64(m.Size().Width) || ly >= float64(m.Size().Height) {
				continue
			}

			if !m.WalkBlocked(int(lx*subtiles), int(ly*subtiles)) {
				out = append(out, [2]float64{lx, ly})
			}
		}
	}

	if len(out) == 0 {
		return nil, nil, fmt.Errorf("no walkable cell on the border towards level %d", level)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return math.Hypot(out[i][0]-hx, out[i][1]-hy) < math.Hypot(out[j][0]-hx, out[j][1]-hy)
	})

	return out, nil, nil
}

// walkToExit starts the walk to the exit towards level (script step walkto:exit).
func (v *Game) walkToExit(level int) error {
	if v.currentLevel() == level {
		return nil
	}

	cands, warp, err := v.exitCandidates(level)
	if err != nil {
		return err
	}

	v.levels.exitWalk = &exitWalk{level: level, candidates: cands, warp: warp}
	hx, hy := v.heroTilePos()
	v.levels.exitWalk.lastX, v.levels.exitWalk.lastY = hx, hy
	v.levels.use, v.levels.warpTarget = nil, nil

	v.Infof("EXIT walking from level %d towards level %d: %d candidate cell(s), first (%.1f,%.1f)%s",
		v.currentLevel(), level, len(cands), cands[0][0], cands[0][1], map[bool]string{true: " (warp tile)", false: ""}[warp != nil])

	v.stepExitWalk(v.levels.exitWalk)

	return nil
}

// stepExitWalk sends the hero to the current candidate.
func (v *Game) stepExitWalk(e *exitWalk) {
	c := e.candidates[e.next%len(e.candidates)]
	if e.warp != nil {
		v.levels.warpTarget, v.levels.warpWait = e.warp, 0
	}

	v.movePlayerTo(c[0], c[1])
}

// threatRadius is the distance (tiles) at which a hostile monster makes the
// walking hero turn round and fight, like a player who is attacked on the way.
const threatRadius = 5.0

// defendOnTheWay pauses the walk and fights when a monster is close, and
// resumes the walk when the fight is over. It reports that the walk is paused.
func (v *Game) defendOnTheWay(e *exitWalk) bool {
	if v.monsters == nil {
		return false
	}

	if v.levels.kill != nil {
		e.paused = true
		return true
	}

	if e.paused {
		e.paused = false
		e.standStill = 0
		hx, hy := v.heroTilePos()
		e.lastX, e.lastY = hx, hy
		v.stepExitWalk(e)

		return false
	}

	k := &killState{radius: threatRadius, deadline: defendSeconds, skip: map[*d2mapentity.Monster]bool{}, defend: true}
	if k.start = len(v.killCandidates(k)); k.start == 0 {
		return false
	}

	v.levels.kill, e.paused = k, true
	v.Infof("KILL defending against %d monster(s) near the hero on the way to level %d", k.start, e.level)

	return true
}

// advanceExitWalk keeps the scripted walk going until the level changed, the
// hero is stuck for good or the time is up.
func (v *Game) advanceExitWalk(elapsed float64) {
	e := v.levels.exitWalk
	if e == nil {
		return
	}

	if v.currentLevel() == e.level {
		v.levels.exitWalk = nil
		return
	}

	if v.levels.trans != nil {
		return // crossing right now
	}

	if v.defendOnTheWay(e) {
		return
	}

	e.elapsed += elapsed
	if e.elapsed > exitTimeout {
		hx, hy := v.heroTilePos()
		v.Warningf("EXIT gave up walking towards level %d: hero at (%.1f,%.1f) after %.0f s", e.level, hx, hy, e.elapsed)
		v.levels.exitWalk = nil

		return
	}

	hx, hy := v.heroTilePos()
	if math.Hypot(hx-e.lastX, hy-e.lastY) > 0.05 {
		e.lastX, e.lastY, e.standStill = hx, hy, 0
		return
	}

	if e.standStill += elapsed; e.standStill < exitStuckSeconds {
		return
	}

	e.standStill = 0

	if e.next++; e.next >= 4*len(e.candidates) {
		v.Warningf("EXIT no way found towards level %d: hero stands at (%.1f,%.1f)", e.level, hx, hy)
		v.levels.exitWalk = nil

		return
	}

	v.Infof("EXIT hero stopped at (%.1f,%.1f); trying candidate %d", hx, hy, e.next%len(e.candidates))
	v.stepExitWalk(e)
}
