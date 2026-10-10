package d2gamescreen

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// The play steps of OD2_AUTOSCRIPT (d2autoscript.PlayHost): walk to exits and
// objects, fight, choose NPC menu rows. They drive the same code paths the
// player's clicks reach.

const (
	killRepathSeconds = 0.4
	// killStuckSeconds is how long the hero may fail to get closer to a target
	// before the target is given up as unreachable.
	killStuckSeconds = 6.0
	// killCastRange is the distance (tiles) up to which a spell is preferred over the staff.
	killCastRange = 12.0
	// killPotionLife is the life fraction below which the hero drinks a belt potion.
	killPotionLife = 0.4
	fireBoltName   = "Fire Bolt"
	defendSeconds  = 40.0
	// retargetGain is how much nearer (tiles) another monster must be for the
	// hero to turn from the one he fights.
	retargetGain = 3.0
	// killRetrySeconds is how long a monster the hero could not reach is left alone.
	killRetrySeconds = 10.0
)

// killState is a running kill: step of the script.
type killState struct {
	radius     float64 // tiles, <= 0: the whole level
	deadline   float64
	elapsed    float64
	kills      int
	start      int // monsters alive at the start
	target     *d2mapentity.Monster
	since      float64 // seconds spent on the current target
	bestDist   float64
	skip       map[*d2mapentity.Monster]float64 // target -> the kill clock at which it may be tried again
	castAcc    float64
	potions    int
	lastPotion float64 // game clock of the last potion
	defend     bool    // a fight started by a walk that was attacked on the way
	name       string  // only monsters whose name contains this (kill:name=), lower case
}

// matches says whether a monster is one the fight is about (every monster, or the named ones).
func (k *killState) matches(m *d2mapentity.Monster) bool {
	return k.name == "" || strings.Contains(strings.ToLower(m.Label()), k.name)
}

// WalkToExit implements d2autoscript.PlayHost.
func (h autoScriptHost) WalkToExit(level int) error { return h.v.walkToExit(level) }

// WalkToObject walks to the nearest object of that name and operates it like a click does.
func (h autoScriptHost) WalkToObject(name string) error {
	ob := h.v.findObject(name)
	if ob == nil {
		return fmt.Errorf("no object %q on this map", name)
	}

	h.v.walkToObject(ob)

	return nil
}

// Menu chooses a row of the open NPC menu by its label or action name.
func (h autoScriptHost) Menu(row string) error {
	menu := h.v.gameControls.NPCMenu
	if !menu.IsOpen() {
		return fmt.Errorf("no NPC menu is open")
	}

	var have []string

	for i, r := range menu.Rows() {
		label := menu.RowLabel(r)
		have = append(have, label)

		if strings.EqualFold(label, row) || strings.EqualFold(r.Action.String(), row) || strings.Contains(strings.ToLower(label), strings.ToLower(row)) {
			h.v.Infof("AUTOSCRIPT menu row %d %q", i, label)
			menu.Choose(i)

			return nil
		}
	}

	return fmt.Errorf("the NPC menu has no row %q (rows: %v)", row, have)
}

// Kill starts a fight.
func (h autoScriptHost) Kill(radius, seconds float64) error {
	v := h.v
	if v.monsterDirector() == nil {
		return fmt.Errorf("no monster director")
	}

	k := &killState{radius: radius, deadline: seconds, skip: map[*d2mapentity.Monster]float64{}}
	k.start = len(v.killCandidates(k))
	v.levels.kill = k

	v.Infof("KILL start radius=%.0f seconds=%.0f candidates=%d level=%d", radius, seconds, k.start, v.currentLevel())

	return nil
}

// KillNamed implements d2autoscript.NamedKillHost: a fight against the monsters whose name contains the text,
// wherever they stand in the level.
func (h autoScriptHost) KillNamed(name string, seconds float64) error {
	v := h.v
	if v.monsterDirector() == nil {
		return fmt.Errorf("no monster director")
	}

	k := &killState{radius: 0, deadline: seconds, skip: map[*d2mapentity.Monster]float64{}, name: strings.ToLower(name)}
	if k.start = len(v.killCandidates(k)); k.start == 0 {
		return fmt.Errorf("no monster named %q on this level", name)
	}

	v.levels.kill = k

	v.Infof("KILL start name=%q seconds=%.0f candidates=%d level=%d", name, seconds, k.start, v.currentLevel())

	return nil
}

// chaseBorderSlack is the extra distance (tiles) from a level border inside
// which a scripted fight leaves monsters alone.
const chaseBorderSlack = 2.0

// nearLevelBorder says whether a position (local tiles) is close to a border
// the hero would cross into another level.
func (v *Game) nearLevelBorder(x, y, margin float64) bool {
	w := v.gameClient.MapEngine.World()
	if w.Level == 0 {
		return false
	}

	_, near := d2level.EdgeExit(w.Rects, w.Level, float64(w.OriginX)+x, float64(w.OriginY)+y, margin)

	return near || v.nearWarpTile(x, y)
}

// warpChaseRadius is the distance (tiles) around a warp tile of an outdoor level (cave, tomb or
// temple entrance) inside which a scripted fight or pickup leaves monsters and items alone: the
// order to walk there lands on the tile, which counts as a click on the entrance (OnPlayerMove,
// targetWarpAt) and takes the hero down the stairs in the middle of a fight.
const warpChaseRadius = warpClickRadius + 1.5

// nearWarpTile says whether a position (local tiles) lies near a warp tile of an outdoor level.
func (v *Game) nearWarpTile(x, y float64) bool {
	// dungeons: the warp tiles are stairs the fight may pass; the exits of the Act 4 and 5 mazes (the bridge
	// of the River of Flame, the Worldstone Keep stairs) are areas of floor tiles and are left alone too
	if v.gameClient.MapEngine.World().Level == 0 && d2level.ActOfLevel(v.currentLevel()) < 4 {
		return false
	}

	for i := range v.levels.warps {
		w := &v.levels.warps[i]
		if math.Hypot(float64(w.TileX)+0.5-x, float64(w.TileY)+0.5-y) <= warpChaseRadius {
			return true
		}
	}

	return false
}

// killAlive counts the living hostile monsters in range, those waiting for a retry included.
func (v *Game) killAlive(k *killState) int {
	n := 0

	hx, hy := v.heroTilePos()

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() || m.Stat == nil || !d2monsters.IsHostile(m.Stat) || !k.matches(m) {
			continue
		}

		x, y := m.GetPositionF()
		if (k.radius <= 0 || math.Hypot(x-hx, y-hy) <= k.radius) &&
			(k.defend || !v.nearLevelBorder(x, y, edgeMargin+chaseBorderSlack)) {
			n++
		}
	}

	return n
}

// killCandidates lists the living hostile monsters the fight may go after.
func (v *Game) killCandidates(k *killState) []*d2mapentity.Monster {
	var out []*d2mapentity.Monster

	hx, hy := v.heroTilePos()

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() || m.Stat == nil || !d2monsters.IsHostile(m.Stat) || !k.matches(m) {
			continue
		}

		if until, ok := k.skip[m]; ok && k.elapsed < until {
			continue // given up for now; it moves, try again later
		}

		mx, my := m.GetPositionF()

		if k.radius > 0 && math.Hypot(mx-hx, my-hy) > k.radius {
			continue
		}

		// a scripted fight does not chase into the border of the level: the hero would
		// walk into the next level (a player decides that himself)
		if !k.defend && v.nearLevelBorder(mx, my, edgeMargin+chaseBorderSlack) {
			continue
		}

		// not even a defensive fight goes for a monster standing on an exit tile of the River of Flame:
		// the walk-through exits there made the hero leave for the Chaos Sanctuary on the way to the stairs
		if k.defend && v.nearWarpTile(mx, my) {
			continue
		}

		out = append(out, m)
	}

	return out
}

// advanceKill runs the scripted fight: the hero goes for the nearest monster,
// casting Fire Bolt when he has it, and drinks a belt potion when hurt.
func (v *Game) advanceKill(elapsed float64) {
	k := v.levels.kill
	if k == nil || v.monsters == nil || v.localPlayer == nil {
		return
	}

	k.elapsed += elapsed

	if v.localPlayer.IsDead() {
		v.Infof("KILL stopped: the hero died (kills=%d)", k.kills)
		v.levels.kill = nil

		return
	}

	cands := v.killCandidates(k)
	alive := v.killAlive(k)

	if alive == 0 || k.elapsed >= k.deadline {
		v.Infof("KILL done: kills=%d of %d remaining=%d skipped=%d elapsed=%.1fs potions=%d",
			k.kills, k.start, alive, len(k.skip), k.elapsed, k.potions)

		if alive > 0 {
			v.Infof("KILL time limit reached with %d monster(s) left", alive)
		}

		v.levels.kill = nil
		v.attackTarget = nil

		return
	}

	if len(cands) == 0 {
		return // every monster left is waiting out its retry time
	}

	v.drinkIfHurt(k)

	if k.target != nil && !k.target.Alive() {
		k.kills++
		k.target = nil
		v.attackTarget = nil
	}

	hx, hy := v.heroTilePos()

	// the nearest monster is the one to fight; a monster that leaves the hero
	// alone for a far target is hit first (a player swings at what bites him)
	best, bd := (*d2mapentity.Monster)(nil), math.MaxFloat64

	for _, m := range cands {
		mx, my := m.GetPositionF()
		if d := math.Hypot(mx-hx, my-hy); d < bd {
			best, bd = m, d
		}
	}

	if k.target != nil {
		tx, ty := k.target.GetPositionF()
		if best != k.target && bd+retargetGain < math.Hypot(tx-hx, ty-hy) {
			k.target = nil
		}
	}

	if k.target == nil {
		k.target, k.since, k.bestDist = best, 0, bd
		v.OnPlayerAttack(best)

		return
	}

	mx, my := k.target.GetPositionF()
	dist := math.Hypot(mx-hx, my-hy)

	// give up on a target only when nothing happens: the hero neither gets
	// closer nor stands in reach and swings
	hsx, hsy := v.localPlayer.Position.X(), v.localPlayer.Position.Y()
	msx, msy := k.target.SubtilePos()
	inReach := d2monster.EdgeDistance(int(hsx)-msx, int(hsy)-msy, 1) <= heroMeleeReach

	if dist < k.bestDist-0.5 || inReach {
		k.bestDist, k.since = math.Min(dist, k.bestDist), 0
	} else if k.since += elapsed; k.since > killStuckSeconds {
		v.Warningf("KILL giving up for now on %q at (%.1f,%.1f): hero at (%.1f,%.1f) cannot get closer than %.1f tiles",
			k.target.Label(), mx, my, hx, hy, k.bestDist)
		k.skip[k.target] = k.elapsed + killRetrySeconds
		k.target = nil
		v.attackTarget = nil

		return
	}

	v.castAtTarget(k, dist)

	if v.attackTarget != k.target {
		v.OnPlayerAttack(k.target) // melee takes over again
	}
}

// attackSpells are the skills a scripted fight casts at a target: the
// projectile and area attacks of the sorceress (name as in skills.txt).
var attackSpells = map[string]bool{
	"Fire Bolt": true, "Ice Bolt": true, "Charged Bolt": true, "Frozen Orb": true, "Blizzard": true,
	"Meteor": true, "Lightning": true, "Chain Lightning": true, "Fire Ball": true, "Glacial Spike": true,
	"Nova": true, "Hydra": true, "Fire Wall": true, "Inferno": true,
}

// attackSpell picks the skill to throw: the hero's right-hand or left-hand
// skill when it is an attack spell he has points in, else his attack spell with
// the most points. -1: none.
func (v *Game) attackSpell() int {
	p := v.localPlayer
	usable := func(s *d2hero.HeroSkill) bool {
		return s != nil && s.SkillRecord != nil && s.SkillPoints > 0 && attackSpells[s.SkillRecord.Skill]
	}

	for _, sel := range []*d2hero.HeroSkill{p.RightSkill, p.LeftSkill} {
		if usable(sel) && p.Skills[sel.ID] != nil {
			return sel.ID
		}
	}

	best, bestPoints := -1, 0

	for id, s := range p.Skills {
		if usable(s) && (s.SkillPoints > bestPoints || (s.SkillPoints == bestPoints && id < best)) {
			best, bestPoints = id, s.SkillPoints
		}
	}

	return best
}

// castAtTarget throws an attack spell at a target that is in range but not yet
// in reach, like a sorceress does.
func (v *Game) castAtTarget(k *killState, dist float64) {
	eng := v.skillEngine()
	if eng == nil || dist > killCastRange || dist < 2.5 || v.localPlayer.IsCasting() {
		return
	}

	id := v.attackSpell()
	if id < 0 || !eng.Supported(id) || v.localPlayer.Stats.Mana < fireBoltManaFloor {
		return
	}

	mx, my := k.target.SubtilePos()
	if eng.CastAt(v.localPlayer, id, mx, my) {
		k.castAcc++
	}
}

// fireBoltManaFloor is a cheap guard; the skill pipeline checks the exact cost.
const fireBoltManaFloor = 3

// potionSeconds is how long the hero waits after a potion before the next one:
// a minor healing potion works over several seconds.
const potionSeconds = 8.0

// drinkIfHurt drinks a healing (or rejuvenation) potion of the belt when life
// is low, like a player pressing the hotkey, and not more often than a potion
// takes to work. Mana potions stay in the belt.
func (v *Game) drinkIfHurt(k *killState) {
	st := v.localPlayer.Stats
	if st.MaxHealth <= 0 || float64(st.Health) >= killPotionLife*float64(st.MaxHealth) ||
		v.levels.clock-k.lastPotion < potionSeconds {
		return
	}

	for col := 0; col < 4; col++ {
		code := v.gameControls.BeltFrontCode(col)
		if !strings.HasPrefix(code, "hp") && !strings.HasPrefix(code, "rv") {
			continue
		}

		if v.gameControls.UseBeltColumnNoSave(col) {
			k.potions++
			k.lastPotion = v.levels.clock
			v.Infof("KILL drank the belt potion %s of column %d at %d/%d life", code, col+1, st.Health, st.MaxHealth)

			return
		}
	}
}

// lootState is a running loot: step.
type lootState struct {
	radius   float64
	deadline float64
	elapsed  float64
	tried    map[*d2mapentity.Item]bool
	picked   int
	nofit    map[string]bool // item codes that did not fit in the inventory (left on the ground)
	questing bool            // only quest items (the lootquest command)
}

// Loot picks up the ground items around the hero, nearest first.
func (h autoScriptHost) Loot(radius, seconds float64) error {
	h.v.levels.loot = &lootState{radius: radius, deadline: seconds, tried: map[*d2mapentity.Item]bool{}, nofit: map[string]bool{}}
	h.v.Infof("LOOT start radius=%.0f seconds=%.0f items=%d", radius, seconds, len(h.v.lootCandidates(h.v.levels.loot)))

	return nil
}

func (v *Game) lootCandidates(l *lootState) []*d2mapentity.Item {
	var out []*d2mapentity.Item

	hx, hy := v.heroTilePos()

	for _, e := range v.gameClient.MapEngine.Entities() {
		it, ok := e.(*d2mapentity.Item)
		if !ok || l.tried[it] || (it.Item != nil && l.nofit[strings.TrimSpace(it.Item.GetItemCode())]) || (l.questing && !isQuestItemCode(it)) {
			continue
		}

		if x, y := it.GetPositionF(); math.Hypot(x-hx, y-hy) <= l.radius && !v.nearWarpTile(x, y) {
			out = append(out, it)
		}
	}

	return out
}

// advanceLoot walks to the next item once the previous pickup is over.
func (v *Game) advanceLoot(elapsed float64) {
	l := v.levels.loot
	if l == nil || v.localPlayer == nil {
		return
	}

	if l.elapsed += elapsed; l.elapsed > l.deadline {
		v.Infof("LOOT time limit reached (%d item(s) picked)", l.picked)
		v.levels.loot, v.ground.item = nil, nil

		return
	}

	if v.ground.item != nil {
		return // still walking to / picking up the current one
	}

	// a picked-up item sits on the cursor: put it into the inventory like the player would
	if it := v.gameControls.CursorItem(); it != nil {
		if x, y, ok := v.gameControls.AutoPlaceCursor(); ok {
			v.Infof("LOOT stored %q in the inventory at (%d,%d)", it.GetItemCode(), x, y)
		} else {
			// like a player: put it back on the ground and go on with the other items
			v.Infof("LOOT no room in the inventory for %q: dropped again", it.GetItemCode())
			l.nofit[strings.TrimSpace(it.GetItemCode())] = true
			v.gameControls.SetCursorItem(nil)
			v.OnPlayerDropItem(it)

			return
		}
	}

	cands := v.lootCandidates(l)
	if len(cands) == 0 {
		v.Infof("LOOT done: %d item(s) walked to, %.1fs", l.picked, l.elapsed)
		v.levels.loot = nil

		return
	}

	hx, hy := v.heroTilePos()
	best, bd := cands[0], math.MaxFloat64

	for _, it := range cands {
		x, y := it.GetPositionF()
		d := math.Hypot(x-hx, y-hy)

		if isQuestItemCode(it) {
			d -= 1000 // quest items first
		}

		if d < bd {
			best, bd = it, d
		}
	}

	l.tried[best] = true
	l.picked++
	v.walkToItem(best)
}

// playBusy reports that a scripted play step still runs.
func (v *Game) playBusy() bool {
	g := v.ground

	return v.levels.kill != nil || v.levels.loot != nil || g.item != nil || g.chest != nil || g.stash != nil || g.questObj != nil
}

// autoPlayPopulate says whether the levels get their natural monsters. They do
// in a normal game and in scripts that play (walkto: / kill: steps); the
// scenarios that test something else (waypoints, automap, maps, monsters, mercs,
// death, quests) keep the levels empty or place their own monsters.
// OD2_POPULATE=1 or 0 forces it.
func autoPlayPopulate() bool {
	switch os.Getenv("OD2_POPULATE") {
	case "1":
		return true
	case "0":
		return false
	}

	for _, name := range []string{"OD2_AUTOMONSTER", "OD2_AUTOMERC", "OD2_AUTODEATH", "OD2_AUTOQUEST", "OD2_NOPOPULATE"} {
		if os.Getenv(name) != "" {
			return false
		}
	}

	if spec := os.Getenv("OD2_AUTOSCRIPT"); spec != "" {
		return strings.Contains(spec, "kill:") || strings.Contains(spec, "walkto:")
	}

	return true
}

// isQuestItemCode says whether a ground item is one of the quest items the quest system tracks.
func isQuestItemCode(it *d2mapentity.Item) bool {
	if it.Item == nil {
		return false
	}

	code := strings.TrimSpace(it.Item.GetItemCode())

	for _, q := range questItemCodes {
		if q == code {
			return true
		}
	}

	return false
}
