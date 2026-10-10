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
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
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

	// the skills of the fight (fightskill.go)
	plan        *fightPlan
	dropped     map[int]bool    // attack skills the engine refused again and again in this fight
	castAfter   float64         // kill clock before which no attack is tried again
	castFails   int             // refusals in a row that count against the skill
	lastCast    float64         // kill clock of the last accepted cast of the left skill
	supportAt   map[int]float64 // support skill -> kill clock of its last cast
	lastMana    float64         // game clock of the last mana potion
	manaPotions int             // mana potions drunk
	casts       map[string]int  // accepted casts by skill name, for the summary
	refusals    map[string]int  // refusals by reason, for the summary
}

// newKillState returns a fight with its maps made (the defensive fights of a walk make one too).
func newKillState(radius, seconds float64, defend bool) *killState {
	return &killState{
		radius: radius, deadline: seconds, skip: map[*d2mapentity.Monster]float64{}, defend: defend,
		dropped: map[int]bool{}, lastCast: -standSeconds * 10, casts: map[string]int{}, refusals: map[string]int{},
	}
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

	k := newKillState(radius, seconds, false)
	k.start = len(v.killCandidates(k))
	v.levels.kill = k

	v.Infof("KILL start radius=%.0f seconds=%.0f candidates=%d level=%d", radius, seconds, k.start, v.currentLevel())

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
	radius := warpChaseRadius
	if v.gameClient.MapEngine.World().Level == 0 && d2level.ActOfLevel(v.currentLevel()) < 4 {
		// a fight in a dungeon may pass the stairs, but not stand on them: with the tougher natural
		// monsters (packs, skill hits) the Halls of the Dead fight chased a monster on the arrival stairs
		// and the order to walk there took the hero back up (Act 2 playthrough, pass5)
		radius = warpClickRadius + 0.5
	}

	for i := range v.levels.warps {
		w := &v.levels.warps[i]
		if math.Hypot(float64(w.TileX)+0.5-x, float64(w.TileY)+0.5-y) <= radius {
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
		if !m.Alive() || m.Stat == nil || !d2monsters.IsHostile(m.Stat) {
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
		if !m.Alive() || m.Stat == nil || !d2monsters.IsHostile(m.Stat) {
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

		v.logFightSkills(k)

		v.levels.kill = nil
		v.attackTarget = nil

		return
	}

	if len(cands) == 0 {
		return // every monster left is waiting out its retry time
	}

	v.drinkIfHurt(k)

	if k.plan == nil {
		k.plan = v.planFight()
		v.logFightPlan(k)
	}

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
	// a caster that stands and casts at the target is not stuck
	casting := k.elapsed-k.lastCast < killStuckSeconds/2

	if dist < k.bestDist-0.5 || inReach || casting {
		k.bestDist, k.since = math.Min(dist, k.bestDist), 0
	} else if k.since += elapsed; k.since > killStuckSeconds {
		v.Warningf("KILL giving up for now on %q at (%.1f,%.1f): hero at (%.1f,%.1f) cannot get closer than %.1f tiles",
			k.target.Label(), mx, my, hx, hy, k.bestDist)
		k.skip[k.target] = k.elapsed + killRetrySeconds
		k.target = nil
		v.attackTarget = nil

		return
	}

	if v.castSupport(k) {
		return
	}

	v.castAtTarget(k, dist)

	if v.holdGround(k, dist) {
		return
	}

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
//
// A hero whose left button holds an attack skill (the class heroes of herogen) casts that skill instead:
// a melee skill when the target is within reach, a ranged one from where he stands (fightskill.go).
func (v *Game) castAtTarget(k *killState, dist float64) {
	eng := v.skillEngine()
	if eng == nil || v.localPlayer.IsCasting() {
		return
	}

	if k.plan != nil && k.plan.hasLeft {
		// a class hero casts his left skill; one that was dropped in this fight swings (not the legacy spell)
		if !k.dropped[k.plan.left.ID] {
			v.castLeftSkill(k, eng, k.plan.left, dist)
		}

		return
	}

	if dist > killCastRange || dist < 2.5 {
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

// targetInReach says whether a monster is within the hero's melee reach.
func (v *Game) targetInReach(m *d2mapentity.Monster) bool {
	hsx, hsy := v.localPlayer.Position.X(), v.localPlayer.Position.Y()
	msx, msy := m.SubtilePos()

	return d2monster.EdgeDistance(int(hsx)-msx, int(hsy)-msy, 1) <= heroMeleeReach
}

// castLeftSkill casts the hero's left skill at the fight's target. A refused cast is tried again after a
// short wait; a skill refused again and again (other than for mana or its delay) is dropped for the
// rest of the fight, and the hero swings instead.
func (v *Game) castLeftSkill(k *killState, eng *d2skills.Engine, pick fightPick, dist float64) {
	if k.elapsed < k.castAfter {
		return
	}

	if pick.Melee {
		if !v.targetInReach(k.target) {
			return // the chase brings the hero within reach
		}
	} else if dist > killCastRange {
		return
	}

	if v.dryFor(pick.ID) {
		v.drinkMana(k)

		if !eng.CanAfford(v.localPlayer, pick.ID) {
			return // waiting for the mana to come back; a melee hero swings meanwhile (advanceHeroAttack)
		}
	}

	mx, my := k.target.SubtilePos()
	if eng.CastAt(v.localPlayer, pick.ID, mx, my) {
		k.castAcc++
		k.castFails = 0
		k.lastCast = k.elapsed
		k.casts[v.skillName(pick.ID)]++

		return
	}

	reason := eng.LastRefusal()
	k.refusals[reason]++
	k.castAfter = k.elapsed + castRetrySeconds

	if !refusalCounts(reason) {
		return
	}

	if k.castFails++; k.castFails >= castFailLimit {
		k.dropped[pick.ID] = true
		// an Infof, not a warning: the hero swings instead and the fight goes on (scripts/verify_classes.sh lists it)
		v.Infof("KILL left skill %q refused %d times (%s): swinging instead for the rest of this fight",
			v.skillName(pick.ID), k.castFails, reason)
	}
}

// skillName is the skills.txt name of one of the hero's skills.
func (v *Game) skillName(id int) string {
	if s := v.localPlayer.Skills[id]; s != nil && s.SkillRecord != nil {
		return s.SkillRecord.Skill
	}

	return fmt.Sprintf("#%d", id)
}

// holdGround keeps a hero with a ranged left skill standing while he casts at a target within range, as a
// player does: no walking into the monsters. It reports whether the hero stands.
func (v *Game) holdGround(k *killState, dist float64) bool {
	if k.plan == nil || !k.plan.hasLeft || k.plan.left.Melee || k.dropped[k.plan.left.ID] ||
		dist > standRange || k.elapsed-k.lastCast > standSeconds {
		return false
	}

	v.attackTarget = nil
	v.localPlayer.StopMoving()

	return true
}

// castSupport casts the support skill that is due (the right button, the summons and buffs of the
// hotkeys). It reports whether a cast started, which takes the tick: the attack comes next tick.
func (v *Game) castSupport(k *killState) bool {
	if k.plan == nil || len(k.plan.supports) == 0 || v.localPlayer.IsCasting() {
		return false
	}

	if v.levels.supportAt == nil || v.levels.supportLevel != v.currentLevel() {
		v.levels.supportAt, v.levels.supportLevel = map[int]float64{}, v.currentLevel()
	}

	now, at := v.levels.clock, v.levels.supportAt

	s, due := dueSupport(k.plan.supports, at, now)
	if !due {
		return false
	}

	eng := v.skillEngine()
	st := v.localPlayer.Stats

	// the attack has the mana first; a support skill that cannot be paid now is tried again soon
	if eng == nil || !eng.CanAfford(v.localPlayer, s.ID) || float64(st.Mana) < supportManaFraction*float64(st.MaxMana) {
		at[s.ID] = now - s.Every + 3

		return false
	}

	ax, ay := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	if rec := v.localPlayer.Skills[s.ID]; rec != nil && rec.SkillRecord != nil && rec.SkillRecord.TargetCorpse {
		if cx, cy, ok := v.nearestCorpse(ax, ay); ok {
			ax, ay = cx, cy
		}
	}

	if eng.CastAt(v.localPlayer, s.ID, ax, ay) {
		at[s.ID] = now
		k.casts[s.Name]++
		v.Infof("KILL support %q cast (next in %.0fs)", s.Name, s.Every)

		return true
	}

	k.refusals[eng.LastRefusal()]++
	at[s.ID] = now - s.Every + 5 // refused (no corpse, ...): try again in a few seconds

	return false
}

// nearestCorpse is the subtile position of the corpse nearest to a subtile position, within 12 tiles.
func (v *Game) nearestCorpse(x, y int) (cx, cy int, ok bool) {
	best := float64(killCastRange * 5)

	for _, m := range v.monsters.Corpses() {
		mx, my := m.SubtilePos()
		if d := math.Hypot(float64(mx-x), float64(my-y)); d < best {
			best, cx, cy, ok = d, mx, my, true
		}
	}

	return cx, cy, ok
}

// logFightPlan tells what the fight casts, for the log (and the class matrix).
func (v *Game) logFightPlan(k *killState) {
	if !k.plan.hasLeft {
		v.Infof("KILL skills: plain attack on the left button (legacy: right or best spell %d)", v.attackSpell())

		return
	}

	var names []string
	for _, s := range k.plan.supports {
		names = append(names, s.Name)
	}

	kind := "ranged"
	if k.plan.left.Melee {
		kind = "melee"
	}

	v.Infof("KILL skills: left=%q (%s) supports=%v", v.skillName(k.plan.left.ID), kind, names)
}

// logFightSkills summarises the casts of a fight.
func (v *Game) logFightSkills(k *killState) {
	if k.plan == nil || !k.plan.hasLeft {
		return
	}

	var dropped []string
	for id := range k.dropped {
		dropped = append(dropped, v.skillName(id))
	}

	v.Infof("KILL skill summary: casts=%v refusals=%v dropped=%v mana_potions=%d", k.casts, k.refusals, dropped, k.manaPotions)
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
}

// Loot picks up the ground items around the hero, nearest first.
func (h autoScriptHost) Loot(radius, seconds float64) error {
	h.v.levels.loot = &lootState{radius: radius, deadline: seconds, tried: map[*d2mapentity.Item]bool{}}
	h.v.Infof("LOOT start radius=%.0f seconds=%.0f items=%d", radius, seconds, len(h.v.lootCandidates(h.v.levels.loot)))
	h.v.lootDiagnose(radius)

	return nil
}

func (v *Game) lootCandidates(l *lootState) []*d2mapentity.Item {
	var out []*d2mapentity.Item

	hx, hy := v.heroTilePos()

	for _, e := range v.gameClient.MapEngine.Entities() {
		it, ok := e.(*d2mapentity.Item)
		if !ok || l.tried[it] {
			continue
		}

		if x, y := it.GetPositionF(); math.Hypot(x-hx, y-hy) <= l.radius && (!v.nearWarpTile(x, y) || v.lootClickSafe(x, y, hx, hy)) {
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
			v.Infof("LOOT no room in the inventory for %q", it.GetItemCode())
			v.levels.loot = nil

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
		if d := math.Hypot(x-hx, y-hy); d < bd {
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

// lootDiagnose logs the ground items near the hero and why a loot might ignore them (a loot that found
// nothing is the first thing to look at when a scripted pickup fails).
func (v *Game) lootDiagnose(radius float64) {
	hx, hy := v.heroTilePos()
	total, near, warp := 0, 0, 0

	for _, e := range v.gameClient.MapEngine.Entities() {
		it, ok := e.(*d2mapentity.Item)
		if !ok {
			continue
		}

		total++

		x, y := it.GetPositionF()
		if math.Hypot(x-hx, y-hy) <= radius {
			near++

			if v.nearWarpTile(x, y) {
				warp++
			}
		}
	}

	v.Infof("LOOT ground items=%d within radius=%d, of those near a warp tile=%d; hero at (%.1f,%.1f) level=%d", total, near, warp, hx, hy, v.currentLevel())
}

// lootClickSafe says whether an item next to a warp tile can still be picked up by a scripted loot: it lies within
// the hero's reach and farther from every warp tile than a click targets one (targetWarpAt), so the walk to it
// is a step and takes no gate. The hero of the item export scenario stands at the town gate of the Rogue
// Encampment and drops items at his feet.
func (v *Game) lootClickSafe(x, y, hx, hy float64) bool {
	if math.Hypot(x-hx, y-hy) > 1.5 {
		return false
	}

	for i := range v.levels.warps {
		w := &v.levels.warps[i]
		if math.Hypot(float64(w.TileX)+0.5-x, float64(w.TileY)+0.5-y) <= warpClickRadius+0.25 {
			return false
		}
	}

	return true
}
