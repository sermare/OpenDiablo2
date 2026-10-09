package d2gamescreen

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client"
)

// Level changes and world objects (doors, waypoints, portals, stairs). The
// rules live in d2common/d2level; this file is the glue to the engine: the
// walk to an object, the fade, rebuilding the map through the level providers
// and the waypoint panel. Every change goes through startLevelChange, the
// analogue of SERVER_ChangePlayerLevel (0x5389f0).

const (
	fadeSeconds       = 0.4  // each way
	objectUseTimeout  = 12.0 // seconds without progress before a walk to an object is abandoned
	walkProgressStep  = 0.5  // tiles the hero must get closer to count as progress
	doorRange         = 2.5  // tiles; the hero stops next to a closed door
	portalRange       = 2.0  // tiles
	warpClickRadius   = 1.5  // a click this close to a warp tile targets it
	portalObjectID    = 59   // objects.txt "Portal" (town portal)
	portalStateSecond = float64(d2level.PortalStateFrames) / d2level.FramesPerSecond
)

type fadePhase int

const (
	fadeOut fadePhase = iota
	fadeIn
)

type levelTransition struct {
	phase  fadePhase
	t      float64
	target int
	start  d2level.StartType
	via    string
	// actFinished: the server marks the act left as finished (forward NPC trips)
	actFinished bool
	// edge crossings: the world position at which the hero left the old level
	edgeWX, edgeWY float64
}

type pendingUse struct {
	ob      *d2mapentity.Object
	elapsed float64 // seconds since the hero last got closer to the object
	best    float64 // the smallest distance so far (0: not measured yet)
}

// levelState is the level-change state of the game screen.
type levelState struct {
	clock      float64 // game seconds, drives the cooldowns
	cooldown   d2level.Cooldown
	trans      *levelTransition
	use        *pendingUse
	wpObj      *d2mapentity.Object // the waypoint whose panel is open
	wpLevel    int
	warps      []d2mapengine.WarpTile
	warpTarget *d2mapengine.WarpTile
	warpWait   float64 // seconds since the hero last got closer to warpTarget
	warpBest   float64 // smallest distance to warpTarget so far
	warpBestOf *d2mapengine.WarpTile
	warpSeen   map[[2]int]bool
	changes    int
	// edgeArmed is true once the hero stood away from every level border since
	// the last level change; crossing a border needs it, so a hero who arrives
	// near the border does not bounce between two levels
	edgeArmed bool
	exitWalk  *exitWalk
	kill      *killState // the scripted fight of a kill: step
	loot      *lootState // the scripted pickup of a loot: step
	// portalStateUntil is when the 75-frame state 0x66 after a portal jump ends.
	portalStateUntil float64
}

// Busy reports that the hero is walking to an object or a level change runs.
func (v *Game) levelBusy() bool {
	return v.levels.trans != nil || v.levels.use != nil || v.levels.warpTarget != nil || v.levels.exitWalk != nil
}

// currentLevel returns the level the hero is in.
func (v *Game) currentLevel() int {
	if v.gameClient.Level == 0 {
		return d2level.RogueEncampment
	}

	return v.gameClient.Level
}

func (v *Game) levelName(id int) string {
	if rec := v.asset.Records.Level.Details[id]; rec != nil && rec.LevelDisplayName != "" {
		return rec.LevelDisplayName
	}

	if n := d2level.DefaultLevelNames[id]; n != "" {
		return n
	}

	return fmt.Sprintf("Level %d", id)
}

func (v *Game) heroTilePos() (x, y float64) { return v.localPlayer.GetPositionF() }

func (v *Game) distanceToObject(ob *d2mapentity.Object) float64 {
	px, py := v.heroTilePos()
	ox, oy := ob.GetPositionF()

	return math.Hypot(px-ox, py-oy)
}

// advanceLevels runs the pending walks, the fade and the warp tiles.
func (v *Game) advanceLevels(elapsed float64) {
	v.levels.clock += elapsed

	if v.localPlayer == nil || v.gameControls == nil {
		return
	}

	if v.levels.warpSeen == nil {
		v.scanWarps() // first frame: the map of the starting level
	}

	v.closeWaypointPanelWhenFar()
	v.advanceObjectUse(elapsed)
	v.advanceWarpUse(elapsed)
	v.advanceEdges()
	v.advanceExitWalk(elapsed)
	v.advanceFade(elapsed)
}

// closeWaypointPanelWhenFar closes the waypoint panel when the hero walks away
// from the waypoint (like the NPC menu).
func (v *Game) closeWaypointPanelWhenFar() {
	ob := v.levels.wpObj
	if ob == nil || !v.gameControls.Waypoints.IsOpen() || v.levels.trans != nil {
		return
	}

	if v.distanceToObject(ob) > objectRange(d2level.ObjectWaypoint)+waypointLeaveSlack {
		v.Infof("WAYPOINT panel closed: walked away from the waypoint")
		v.gameControls.Waypoints.Close()
	}
}

// waypointLeaveSlack is the extra distance (tiles) before the panel closes.
const waypointLeaveSlack = 2.0

// renderFade darkens the screen during a level change.
func (v *Game) renderFade(screen d2interface.Surface) {
	t := v.levels.trans
	if t == nil {
		return
	}

	a := t.t / fadeSeconds
	if t.phase == fadeIn {
		a = 1 - a
	}

	if a <= 0 {
		return
	}

	if a > 1 {
		a = 1
	}

	screen.DrawRect(screenWidth, screenHeight, d2util.Color(uint32(a*255)))
}

func (v *Game) advanceFade(elapsed float64) {
	t := v.levels.trans
	if t == nil {
		return
	}

	t.t += elapsed
	if t.t < fadeSeconds {
		return
	}

	t.t = 0

	if t.phase == fadeOut {
		v.performLevelChange(t)
		t.phase = fadeIn

		return
	}

	v.levels.trans = nil
}

// startLevelChange begins a level change with a fade. It refuses when a change
// is already running or no level provider can build the level.
func (v *Game) startLevelChange(level int, start d2level.StartType, via string) bool {
	if v.levels.trans != nil {
		v.Infof("LEVEL change to %d refused: another change is running", level)
		return false
	}

	if !v.gameClient.CanLoadLevel(level) {
		v.Infof("LEVEL change to %d (%s) refused: the engine cannot load that level yet", level, v.levelName(level))
		return false
	}

	v.levels.trans = &levelTransition{target: level, start: start, via: via}

	return true
}

// performLevelChange rebuilds the map for the target level and puts the hero in
// it (at the black point of the fade).
func (v *Game) performLevelChange(t *levelTransition) {
	from := v.currentLevel()

	plan, err := d2level.PlanTransition(from, t.target, t.start, uint32(v.gameClient.MapEngine.Seed()), 0)
	if err != nil {
		v.Errorf("LEVEL change to %d: %v", t.target, err)
		return
	}

	var prefer d2client.ArrivalFunc

	switch t.via {
	case "waypoint":
		prefer = nextToWaypoint // waypoint travel lands you at the destination's waypoint
	case "edge":
		prefer = edgeArrival(from, t.target, t.edgeWX, t.edgeWY)
	case "warp":
		prefer = nextToWarpBackTo(from, t.target) // the stairs or cave entrance you came through
	}

	v.saveLevel(from)

	arrival, err := v.gameClient.ChangeLevelAct(t.target, prefer, t.actFinished)
	if err != nil {
		v.Errorf("LEVEL change to %d failed: %v; going back to level %d", t.target, err, from)

		if _, err2 := v.gameClient.ChangeLevel(from, nil); err2 != nil {
			v.Errorf("LEVEL could not rebuild level %d either: %v", from, err2)
		}

		v.resetLevelState()
		v.restoreLevel(from)

		return
	}

	v.afterLevelBuilt(from, t.target, t.via)

	px, py := v.heroTilePos()
	v.Infof("LEVEL CHANGE from=%d to=%d (%s) act=%d via=%s start=%#x townTransition=%v actChange=%v arrival=(%.1f,%.1f) hero=(%.1f,%.1f)",
		from, t.target, v.levelName(t.target), plan.ToAct, t.via, int(plan.StartType), plan.TownTransition,
		plan.ActChange, arrival.X, arrival.Y, px, py)

	if plan.ActChange {
		v.Infof("ACT CHANGE %d -> %d LoadAct packet % x", plan.FromAct, plan.ToAct, plan.LoadAct.Encode())
		v.logActArrival(t.target)
	}

	if t.via == "portal" {
		v.levels.portalStateUntil = v.levels.clock + portalStateSecond
		v.Infof("LEVEL portal state %#x for %d frames (%.1f s)", d2level.PortalStateID, d2level.PortalStateFrames, portalStateSecond)
	}
}

// snapCamera puts the camera on the hero at once; the normal follow eases the
// camera towards him, which after a level change would show the new level
// from where the old hero stood.
func (v *Game) snapCamera() {
	if v.localPlayer == nil {
		return
	}

	w := v.localPlayer.Position.World()
	rx, ry := v.mapRenderer.WorldToOrtho(w.X(), w.Y())
	pos := d2vector.NewPosition(rx, ry)

	v.mapRenderer.MoveCameraTo(&pos)
	v.mapRenderer.SetCameraTarget(&pos)
}

// afterLevelBuilt is the bookkeeping after the map of a new level was built
// and the hero put into it: the old level's pending things go, the warp tiles
// of the new map are listed, the quest system learns the new area and the
// corpse of a hero who died here comes back.
func (v *Game) afterLevelBuilt(from, to int, via string) {
	v.resetLevelState()
	v.gameControls.Speech.Clear() // the NPC who was speaking stayed behind
	v.snapCamera()

	v.levels.cooldown.Mark(v.levels.clock)
	v.levels.changes++
	v.levels.edgeArmed = false
	v.scanWarps()
	v.questArea(to) // the quest system follows the hero between areas
	v.restoreCorpse()
	v.restoreLevel(to)

	v.Infof("LEVEL built: level %d (%s) via=%s from=%d", to, v.levelName(to), via, from)
}

// nextToWaypoint picks the waypoint object of a freshly built level as the
// arrival point (the original's special arrival, start type 0xD, UNVERIFIED).
func nextToWaypoint(m *d2mapengine.MapEngine) (x, y float64, ok bool) {
	for _, e := range m.Entities() {
		if ob, isObj := e.(*d2mapentity.Object); isObj && ob.Kind() == d2level.ObjectWaypoint {
			x, y = ob.GetPositionF()
			return x, y, true
		}
	}

	return 0, 0, false
}

// resetLevelState drops everything tied to the old map.
func (v *Game) resetLevelState() {
	v.monsters, v.attackTarget, v.npcTarget = nil, nil, nil
	v.ground.item, v.ground.chest = nil, nil
	v.levels.use, v.levels.warpTarget, v.levels.wpObj, v.levels.exitWalk = nil, nil, nil, nil
	v.lastRegionType = d2enum.RegionNone

	v.gameControls.NPCMenu.Close()
	v.gameControls.Waypoints.Close()
}

// scanWarps lists the warp tiles of the freshly loaded map.
func (v *Game) scanWarps() {
	v.levels.warps = v.gameClient.MapEngine.WarpTiles()
	v.levels.warpSeen = map[[2]int]bool{}

	v.Infof("LEVEL %d: %d warp tile(s)", v.currentLevel(), len(v.levels.warps))
	for _, w := range v.levels.warps {
		v.Infof("LEVEL warp tile at (%d,%d) style=%d", w.TileX, w.TileY, w.Style)
	}
}

// targetWarpAt selects the warp tile near a clicked point, if any.
func (v *Game) targetWarpAt(x, y float64) {
	v.levels.warpTarget, v.levels.warpWait = nil, 0

	for i := range v.levels.warps {
		w := &v.levels.warps[i]
		if math.Hypot(float64(w.TileX)+0.5-x, float64(w.TileY)+0.5-y) <= warpClickRadius {
			v.levels.warpTarget = w
			return
		}
	}
}

// advanceWarpUse enters a clicked warp tile once the hero is closer than
// d2level.WarpRange tiles (SERVER_InteractWithUnitByType case 5).
func (v *Game) advanceWarpUse(elapsed float64) {
	w := v.levels.warpTarget
	if w == nil || v.levels.trans != nil {
		return
	}

	px, py := v.heroTilePos()
	dist := math.Hypot(float64(w.TileX)+0.5-px, float64(w.TileY)+0.5-py)

	// the walk is only abandoned when the hero stops getting closer: a far
	// stair takes longer than objectUseTimeout to reach
	if v.levels.warpBestOf != w || dist < v.levels.warpBest-walkProgressStep {
		v.levels.warpBestOf, v.levels.warpBest, v.levels.warpWait = w, dist, 0
	}

	if v.levels.warpWait += elapsed; v.levels.warpWait > objectUseTimeout {
		v.Warningf("LEVEL gave up walking to the warp tile at (%d,%d): the hero is %.1f tiles away", w.TileX, w.TileY, dist)
		v.levels.warpTarget = nil

		return
	}

	if dist >= d2level.WarpRange {
		return
	}

	v.levels.warpTarget = nil
	cur := v.currentLevel()

	dest, ok := warpDest(cur, w)
	if !ok {
		key := [2]int{w.TileX, w.TileY}
		if !v.levels.warpSeen[key] {
			v.levels.warpSeen[key] = true
			v.Infof("LEVEL warp tile style=%d at (%d,%d) in level %d has no known destination", w.Style, w.TileX, w.TileY, cur)
		}

		return
	}

	v.Infof("LEVEL warp tile style=%d at (%d,%d): level %d -> %d", w.Style, w.TileX, w.TileY, cur, dest)
	v.startLevelChange(dest, d2level.StartDefault, "warp")
}

// useObject walks the hero to an object and operates it on arrival.
func (v *Game) useObject(ob *d2mapentity.Object) {
	v.ground.item, v.ground.chest, v.npcTarget = nil, nil, nil
	v.levels.warpTarget = nil
	v.levels.use = &pendingUse{ob: ob}

	x, y := ob.GetPositionF()
	hx, hy := v.heroTilePos()
	v.Infof("OBJECT walking to use %s %q (id %d) at (%.1f,%.1f) from (%.1f,%.1f)", ob.Kind(), ob.Label(), ob.Record().Index, x, y, hx, hy)
	v.movePlayerTo(x, y)
}

// objectReachSlack is added to the operate range: objects are measured from
// their origin, but the hero stops at the edge of the object's footprint and of
// the walls around it (UNVERIFIED: the original measures to the object's
// bounding box).
const objectReachSlack = 1.0

func objectRange(k d2level.ObjectKind) float64 {
	switch k {
	case d2level.ObjectDoor:
		return doorRange + objectReachSlack
	case d2level.ObjectPortal:
		return portalRange + objectReachSlack
	}

	return d2level.WaypointRange + objectReachSlack
}

func (v *Game) advanceObjectUse(elapsed float64) {
	u := v.levels.use
	if u == nil {
		return
	}

	if d := v.distanceToObject(u.ob); u.best == 0 || d < u.best-walkProgressStep {
		u.best, u.elapsed = d, 0 // still getting closer
	}

	u.elapsed += elapsed

	switch {
	case v.gameClient.MapEngine.Entities()[u.ob.ID()] == nil:
		v.levels.use = nil // the object is gone (level changed)
	case v.distanceToObject(u.ob) <= objectRange(u.ob.Kind()):
		v.levels.use = nil
		v.localPlayer.StopMoving()
		v.operateObject(u.ob)
	case u.elapsed > objectUseTimeout:
		hx, hy := v.heroTilePos()
		v.Warningf("OBJECT gave up walking to %q: hero at (%.1f,%.1f), %.1f tiles away", u.ob.Label(), hx, hy, v.distanceToObject(u.ob))
		v.levels.use = nil
	}
}

func (v *Game) operateObject(ob *d2mapentity.Object) {
	switch ob.Kind() {
	case d2level.ObjectDoor:
		v.operateDoor(ob)
	case d2level.ObjectWaypoint:
		v.operateWaypoint(ob)
	case d2level.ObjectPortal:
		v.operatePortal(ob)
	default:
		v.operateWorldObject(ob)
	}
}

// operateDoor opens a closed door or closes an open one and updates the
// collision overlay of the map engine.
func (v *Game) operateDoor(ob *d2mapentity.Object) {
	open := !ob.IsOpened()

	changed, err := v.gameClient.MapEngine.SetDoorOpen(ob, open)
	if err != nil {
		v.Warningf("door %q: %v", ob.Label(), err)
	}

	x, y := ob.GetPositionF()
	fx, fy, fw, fh := ob.Footprint()
	blocked := v.gameClient.MapEngine.DoorBlocks(fx, fy)

	v.Infof("OBJECT door %q id=%d open=%v changed=%v blocks=%v footprint=(%d,%d %dx%d) pos=(%.1f,%.1f)",
		ob.Label(), ob.Record().Index, ob.IsOpened(), changed, blocked, fx, fy, fw, fh, x, y)
}

// waypointLevelOf returns the level a waypoint object belongs to: objects.txt
// row 119 is the town waypoint of act 1; other waypoints belong to the level
// the hero is in.
func (v *Game) waypointLevelOf(ob *d2mapentity.Object) int {
	if ob.Record().Index == 119 {
		return d2level.RogueEncampment
	}

	return v.currentLevel()
}

// operateWaypoint activates the waypoint (setting and saving its bit) and opens
// the waypoint panel for the act.
func (v *Game) operateWaypoint(ob *d2mapentity.Object) {
	level := v.waypointLevelOf(ob)

	if _, ok := d2level.WaypointBit(level); !ok {
		v.Infof("WAYPOINT object %q is in level %d, which has no waypoint bit", ob.Label(), level)
		return
	}

	changed, err := v.gameClient.SetWaypoint(level, true)
	if err != nil {
		v.Errorf("WAYPOINT activate level %d: %v", level, err)
	}

	if changed {
		bit, _ := d2level.WaypointBit(level)
		v.Infof("WAYPOINT activated level=%d (%s) bit=%d difficulty=%d", level, v.levelName(level), bit, v.gameClient.Difficulty)
	}

	v.levels.wpObj, v.levels.wpLevel = ob, level
	v.openWaypointPanel(level)
}

func (v *Game) openWaypointPanel(level int) {
	act := d2level.ActOfLevel(level)
	entries := d2level.WaypointList(v.gameClient.Progress.Waypoints, d2level.Difficulty(v.gameClient.Difficulty), act, v.gameClient.CanLoadLevel)

	rows := make([]d2player.WaypointRow, 0, len(entries))
	desc := make([]string, 0, len(entries))
	active, enabled := 0, 0

	for _, e := range entries {
		name := v.levelName(e.Level)
		rows = append(rows, d2player.WaypointRow{WaypointEntry: e, Name: name})

		state := "off"

		switch {
		case e.Enabled():
			state = "on"
		case e.Active:
			state = "grey"
		}

		if e.Active {
			active++
		}

		if e.Enabled() {
			enabled++
		}

		desc = append(desc, fmt.Sprintf("%d:%s:%s", e.Level, name, state))
	}

	v.gameControls.NPCMenu.Close()
	v.gameControls.Waypoints.Open(act, level, rows, v.onWaypointChosen)

	v.Infof("WAYPOINT PANEL act=%d level=%d rows=%d active=%d enabled=%d entries=[%s]",
		act, level, len(rows), active, enabled, strings.Join(desc, ", "))
}

// onWaypointChosen is C2S 0x49 (take waypoint): validate like the server does,
// then change level.
func (v *Game) onWaypointChosen(level int) {
	cur := v.currentLevel()
	if level == cur {
		v.Infof("WAYPOINT already in level %d", level)
		return
	}

	dist := math.MaxFloat64
	if v.levels.wpObj != nil {
		dist = v.distanceToObject(v.levels.wpObj)
	}

	err := d2level.ValidateWaypoint(d2level.WaypointRequest{
		PlayerAct: d2level.ActOfLevel(cur),
		ObjectAct: d2level.ActOfLevel(v.levels.wpLevel),
		Distance:  dist,
		Range:     objectRange(d2level.ObjectWaypoint),
		Target:    level,
		Diff:      d2level.Difficulty(v.gameClient.Difficulty),
		Waypoints: v.gameClient.Progress.Waypoints,
	})
	if err != nil {
		v.Infof("WAYPOINT refused level=%d: %v", level, err)
		return
	}

	if !v.levels.cooldown.Ready(v.levels.clock, d2level.WaypointCooldownSeconds) {
		v.Infof("WAYPOINT refused level=%d: %v", level, d2level.ErrWaypointCooldown)
		return
	}

	v.Infof("WAYPOINT travel level=%d (%s)", level, v.levelName(level))
	v.startLevelChange(level, d2level.WaypointStartType(level), "waypoint")
}

// operatePortal uses a portal object: cooldown, owner and quest rules, then the
// level change with the portal arrival rule.
func (v *Game) operatePortal(ob *d2mapentity.Object) {
	if !v.levels.cooldown.Ready(v.levels.clock, d2level.PortalCooldownSeconds) {
		v.Infof("PORTAL refused: %v", d2level.ErrPortalCooldown)
		return
	}

	req := d2level.PortalRequest{
		Dest:   ob.PortalDest,
		Owner:  ob.PortalOwner,
		Player: v.gameClient.PlayerID,
		// the quest flag of the destination is not wired in yet (UNVERIFIED rule input)
	}
	if err := d2level.ValidatePortal(req); err != nil {
		v.Infof("PORTAL refused: %v", err)
		return
	}

	if a := d2level.ActOfLevel(ob.PortalDest); a != v.currentAct() && ob.PortalDest == d2level.ActStartLevel(a) {
		_ = v.travelToAct(a, "portal") // an act change: Mephisto's portal to the Pandemonium Fortress
		return
	}

	v.Infof("PORTAL used dest=%d (%s)", ob.PortalDest, v.levelName(ob.PortalDest))
	v.startLevelChange(ob.PortalDest, d2level.StartPortal, "portal")
}

// commandSpawnPortal is the console command "spawnportal <level>": a town
// portal object leading to that level, next to the hero.
func (v *Game) commandSpawnPortal(args []string) error {
	level, err := strconv.Atoi(args[0])
	if err != nil || level < 1 || level > 136 { // 133 to 136: the Pandemonium areas the cube opens
		return fmt.Errorf("invalid level %q", args[0])
	}

	rec := v.asset.Records.Object.Details[portalObjectID]
	if rec == nil {
		return fmt.Errorf("no objects.txt row %d", portalObjectID)
	}

	px, py := v.heroTilePos()
	sx, sy := int(math.Floor(px*subtilesInTile))+portalSpawnOffset, int(math.Floor(py*subtilesInTile))

	ob, err := v.gameClient.MapEngine.NewObject(sx, sy, rec, d2resource.PaletteUnits)
	if err != nil {
		return err
	}

	ob.PortalDest, ob.PortalOwner = level, v.gameClient.PlayerID
	v.gameClient.MapEngine.AddEntity(ob)
	v.Infof("OBJECT spawned portal to level %d at (%d,%d)", level, sx, sy)

	return nil
}

const portalSpawnOffset = 10 // sub-tiles east of the hero

// commandSetWaypoint is the console command "setwaypoint <level> <0|1>".
func (v *Game) commandSetWaypoint(args []string) error {
	level, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid level %q", args[0])
	}

	changed, err := v.gameClient.SetWaypoint(level, args[1] != "0")
	if err == nil {
		v.Infof("WAYPOINT set level=%d active=%v changed=%v", level, args[1] != "0", changed)
	}

	return err
}

// findObject returns the object nearest to the hero whose name or objects.txt
// id matches target.
func (v *Game) findObject(target string) *d2mapentity.Object {
	id, numeric := strconv.Atoi(target)

	var best *d2mapentity.Object

	bestDist := math.MaxFloat64

	for _, e := range v.gameClient.MapEngine.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok {
			continue
		}

		rec := ob.Record()

		match := strings.EqualFold(ob.Label(), target) || strings.EqualFold(rec.Name, target)
		if numeric == nil {
			match = rec.Index == id
		}

		if !match {
			continue
		}

		// objects that can be operated win over look-alikes of the same name
		d := v.distanceToObject(ob)
		if !usableKind(ob.Kind()) {
			d += unusablePenalty
		}

		if d < bestDist {
			best, bestDist = ob, d
		}
	}

	return best
}

// unusablePenalty makes findObject prefer operable objects over same-named ones.
const unusablePenalty = 1e6

func usableKind(k d2level.ObjectKind) bool {
	return k == d2level.ObjectDoor || k == d2level.ObjectWaypoint || k == d2level.ObjectPortal
}

// Use, Waypoint, Level and Busy implement d2autoscript.LevelHost.

func (h autoScriptHost) Use(target string) error {
	ob := h.v.findObject(target)
	if ob == nil {
		return fmt.Errorf("no object %q on this map", target)
	}

	if usableKind(ob.Kind()) {
		h.v.useObject(ob)
		return nil
	}

	r := ob.Record()

	return fmt.Errorf("object %q (%s, id %d, operate fn %d, subclass %d) cannot be used yet",
		ob.Label(), ob.Kind(), r.Index, r.OperateFn, r.SubClass)
}

func (h autoScriptHost) Waypoint(level int) error {
	if !h.v.gameControls.Waypoints.IsOpen() {
		return fmt.Errorf("the waypoint panel is not open (use:Waypoint first)")
	}

	return h.v.gameControls.Waypoints.Choose(level)
}

func (h autoScriptHost) Level() (level int, x, y float64) {
	x, y = h.v.heroTilePos()

	return h.v.currentLevel(), x, y
}

func (h autoScriptHost) Busy() bool { return h.v.levelBusy() || h.v.playBusy() }
