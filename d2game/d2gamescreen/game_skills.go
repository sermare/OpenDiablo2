package d2gamescreen

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2skills"
)

const (
	castTestInterval     = 0.6 // seconds between scripted casts
	castTestDelayed      = 1.6 // seconds between casts of a skill with a delay (cooldown)
	castTestDefaultCount = 5
	castTestDefaultLevel = 10
	castTestMoveSeconds  = 0.4
	castTestMaxSeconds   = 900.0 // longest a skill scenario may run
	castTestWalkGiveUp   = 6.0   // seconds a melee skill walks before the hero is put next to the target
	castTestEngage       = 6     // subtiles: how close the nearest monster should be before a ranged skill is cast
	castTestEngageWait   = 3.0   // seconds to wait for that
	castTestSettle       = 3.0   // seconds after the last cast before the scenario ends
	castTestItemSettle   = 2.5   // seconds after a skill's last cast before the next skill starts
	castTestCorpseWait   = 2.5   // seconds a corpse skill waits for a fresh corpse
)

// castItem is one skill of the OD2_AUTOCAST list.
type castItem struct {
	skill string
	id    int
	count int
	casts int

	started bool
	// doneFor is the time since the last cast was made.
	doneFor float64
	// counters at the start, to print the skill's own totals
	c0 d2skills.Counters
	// refusedSeen is the engine's refused count already accounted for; retries
	// is how many refused casts were given back to be made again.
	refusedSeen, retries int
}

// castItemMaxRetries bounds how often a refused cast is repeated.
const castItemMaxRetries = 5

// castTest is the state of the OD2_AUTOCAST scenario.
type castTest struct {
	items    []*castItem
	idx      int
	level    int
	since    float64
	moveAcc  float64
	corpseW  float64
	walking  float64
	engageW  float64
	refused  int
	reported bool
	finished bool
	settle   float64
	skill    string // the whole list, for the summary
}

// skillEngine returns the skill engine, creating it once the hero and the
// monster director exist.
func (v *Game) skillEngine() *d2skills.Engine {
	if v.localPlayer == nil {
		return v.skills
	}

	md := v.monsterDirector()
	if md == nil {
		return v.skills
	}

	if v.skills != nil {
		// the area changed: the Director was rebuilt, the hero keeps his state
		if v.skills.Monsters() != md {
			v.skills.AreaChanged(md)
		}

		return v.skills
	}

	scenario := os.Getenv("OD2_AUTOCAST") != ""
	v.skills = d2skills.New(v.asset, v.gameClient.MapEngine, md, v.logLevel, d2skills.Options{
		Seed:         uint32(v.gameClient.MapEngine.Seed()),
		IgnoreTown:   scenario || os.Getenv("OD2_AUTOMONSTER") != "",
		InfiniteAmmo: scenario,
		TeleportFlag: func() int {
			if det := v.asset.Records.GetLevelDetails(v.currentLevel()); det != nil {
				return int(det.TeleportFlag)
			}

			return 1
		},
		Act:          v.currentAct,
		Difficulty:   int(v.gameClient.Difficulty),
	})
	v.skills.Rivals = v.skillRivals
	v.skills.OnPvPHit = v.sendSkillPvP

	if st := v.localPlayer.Stats; st != nil {
		id, eng := v.localPlayer.ID(), v.skills
		st.SkillStats = func() map[string]int {
			out := eng.StateStats(id)

			for k, v := range eng.PassiveTotals(id) {
				if out == nil {
					out = map[string]int{}
				}

				out[k] += v
			}

			return out
		}
	}

	v.skills.OnSound = v.onSkillSound

	return v.skills
}

func (v *Game) advanceSkills(elapsed float64) {
	eng := v.skillEngine()
	if eng == nil {
		return
	}

	eng.Advance(elapsed)

	// the hero's buffs and auras are part of its stats: recompute the totals
	// when the set of active skill stats changes
	if st := v.localPlayer.Stats; st != nil && st.Recalc != nil {
		sig := fmt.Sprint(eng.StateStats(v.localPlayer.ID()))
		if sig != v.skillStatSig {
			v.skillStatSig = sig
			st.Recalc()
		}
	}
}

// castWithPipeline runs a skill through the skill pipeline when it supports
// it; false means the caller should use the old client-only cast.
func (v *Game) castWithPipeline(skillID int, tileX, tileY float64) bool {
	eng := v.skillEngine()
	if eng == nil || !eng.Supported(skillID) {
		return false
	}

	eng.Cast(v.localPlayer, skillID, tileX, tileY)

	return true
}

// parseCastTest reads OD2_AUTOCAST=<skill>[,count][;<skill>[,count]...]: a
// list of skills (name or id) cast one after the other, each count times
// (default 5).
func (v *Game) parseCastTest(eng *d2skills.Engine) *castTest {
	ref := os.Getenv("OD2_AUTOCAST")
	t := &castTest{level: castTestDefaultLevel, skill: ref}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOCAST_LEVEL")); err == nil && n > 0 {
		t.level = n
	}

	for _, part := range strings.Split(ref, ";") {
		it := &castItem{skill: strings.TrimSpace(part), count: castTestDefaultCount}

		if i := strings.LastIndex(part, ","); i >= 0 {
			if n, err := strconv.Atoi(strings.TrimSpace(part[i+1:])); err == nil && n > 0 {
				it.skill, it.count = strings.TrimSpace(part[:i]), n
			}
		}

		it.id = eng.SkillID(it.skill)
		if it.id < 0 {
			v.Errorf("AUTOCAST: unknown skill %q", it.skill)
			v.autoTestExit()

			continue
		}

		t.items = append(t.items, it)
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOCAST_MANA")); err == nil && n > 0 {
		v.localPlayer.Stats.MaxMana, v.localPlayer.Stats.Mana = n, n
	}

	return t
}

// startCastItem grants the skill if the class cannot have it, refills life and
// mana and logs the start.
func (v *Game) startCastItem(eng *d2skills.Engine, t *castTest, it *castItem) {
	it.started = true
	it.c0 = eng.Counters
	it.refusedSeen = eng.Counters.Refused

	p := v.localPlayer
	if s := p.Skills[it.id]; s == nil || s.SkillPoints < 1 {
		if p.Skills == nil {
			p.Skills = map[int]*d2hero.HeroSkill{}
		}

		p.Skills[it.id] = &d2hero.HeroSkill{SkillRecord: v.asset.Records.Skill.Details[it.id], SkillPoints: t.level}
		v.Infof("AUTOCAST granted skill=%q level=%d (the hero did not have it)", it.skill, t.level)
	}

	p.Stats.Health = p.Stats.MaxHealth

	if os.Getenv("OD2_AUTOCAST_MANA") != "" || len(t.items) > 1 {
		p.Stats.Mana = p.Stats.MaxMana
	}

	v.Infof("AUTOCAST start skill=%q id=%d level=%d count=%d hero_mana=%s class=%v", it.skill, it.id, p.Skills[it.id].SkillPoints,
		it.count, eng.Hero(p).ManaString(), p.Class)
}

// finishCastItem prints what the skill alone did.
func (v *Game) finishCastItem(eng *d2skills.Engine, it *castItem) {
	c, c0 := eng.Counters, it.c0
	v.Infof("AUTOCAST skill_done skill=%q casts=%d missiles=%d hits=%d melee=%d area=%d kills=%d damage=%d dot=%d refused=%d",
		it.skill, c.Casts-c0.Casts, c.Missiles-c0.Missiles, c.Hits-c0.Hits, c.Melee-c0.Melee, c.AreaHits-c0.AreaHits,
		c.Kills-c0.Kills, c.Damage-c0.Damage, c.DotDamage-c0.DotDamage, c.Refused-c0.Refused)
}

// autoCast implements OD2_AUTOCAST=<skill>[,count][;...]: the hero casts the
// skills at the nearest living monster (OD2_AUTOMONSTER spawns them; they are
// spawned again when all are dead) every castTestInterval seconds; melee range
// skills first walk up to the monster, corpse skills aim at a corpse (one is
// made when none is lying around).
func (v *Game) autoCast(elapsed float64) {
	eng := v.skillEngine()
	if eng == nil {
		return
	}

	t := v.castTestState
	if t == nil {
		t = v.parseCastTest(eng)
		v.castTestState = t
	}

	if t.finished {
		return
	}

	// a cast the engine refused (the target died in the cast animation, say) does not
	// count: it is made again, a few times at most
	if t.idx < len(t.items) {
		if it := t.items[t.idx]; it.started && eng.Counters.Refused > it.refusedSeen {
			if d := eng.Counters.Refused - it.refusedSeen; it.retries < castItemMaxRetries && it.casts > 0 {
				if it.casts -= d; it.casts < 0 {
					it.casts = 0
				}
				it.retries++
				it.doneFor = 0
				v.Infof("AUTOCAST skill=%q cast refused, trying again (%d)", it.skill, it.retries)
			}

			it.refusedSeen = eng.Counters.Refused
		}
	}

	// a skill is done castTestItemSettle seconds after its last cast, so what
	// the missiles and delayed strikes did is counted for it
	for t.idx < len(t.items) && t.items[t.idx].casts >= t.items[t.idx].count {
		it := t.items[t.idx]
		if it.doneFor += elapsed; it.doneFor < castTestItemSettle {
			return
		}

		v.finishCastItem(eng, it)
		t.idx++
	}

	if t.idx >= len(t.items) {
		t.settle += elapsed
		if t.settle >= castTestSettle {
			t.finished = true
		}

		return
	}

	if v.localPlayer.IsCasting() {
		return
	}

	it := t.items[t.idx]
	if !it.started {
		v.startCastItem(eng, t, it)
	}

	t.since += elapsed
	if t.since < castTestInterval {
		return
	}

	rec := v.asset.Records.Skill.Details[it.id]
	if rec != nil && rec.Delay != nil && !rec.Delay.Empty() && t.since < castTestDelayed {
		return
	}

	m := v.nearestMonster()
	if m == nil {
		v.respawnCastMonsters()

		return
	}

	aimX, aimY := m.SubtilePos()

	// let the monsters come close: skills that act around the hero need them there
	if hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y()); d2monster.Distance(hx-aimX, hy-aimY) > castTestEngage &&
		t.engageW < castTestEngageWait && (rec == nil || rec.Range != "h2h") {
		t.engageW += elapsed

		return
	}

	t.engageW = 0

	if rec != nil && rec.TargetCorpse {
		cx, cy, ok := v.corpseForCast(m, elapsed, t)
		if !ok {
			return
		}

		aimX, aimY = cx, cy
	}

	mx, my := aimX, aimY
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	if rec != nil && rec.Range == "h2h" && d2monster.EdgeDistance(hx-mx, hy-my, 1) > heroMeleeReach {
		t.moveAcc += elapsed
		t.walking += elapsed

		if t.walking > castTestWalkGiveUp { // cannot get there: the scenario puts the hero next to it
			t.walking = 0
			v.putHeroNear(mx, my)

			return
		}

		if t.moveAcc >= castTestMoveSeconds {
			t.moveAcc = 0
			x, y := m.GetPositionF()
			v.OnPlayerMove(x, y)
		}

		return
	}

	t.walking = 0
	t.since = 0
	t.corpseW = 0
	it.casts++

	if !eng.CastAt(v.localPlayer, it.id, aimX, aimY) {
		t.refused++
	}
}

// corpseForCast finds the corpse nearest to the hero; with none lying around it
// kills the given monster so one appears, and asks the caller to wait.
func (v *Game) corpseForCast(victim *d2mapentity.Monster, elapsed float64, t *castTest) (int, int, bool) {
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	var best *d2mapentity.Monster

	bd := 0

	for _, c := range v.monsters.Corpses() {
		if v.skills != nil && v.skills.Looted(c.ID()) { // Find Potion / Find Item / Grim Ward used it up
			continue
		}

		cx, cy := c.SubtilePos()
		if d := d2monster.Distance(hx-cx, hy-cy); best == nil || d < bd {
			best, bd = c, d
		}
	}

	if best != nil {
		x, y := best.SubtilePos()

		return x, y, true
	}

	if t.corpseW == 0 {
		v.Infof("AUTOCAST making a corpse: killing %s", victim.Label())
		v.monsters.Damage(victim, 1<<20, v.localPlayer)
	}

	t.corpseW += elapsed
	if t.corpseW > castTestCorpseWait {
		t.corpseW = 0
	}

	return 0, 0, false
}

// respawnCastMonsters spawns the OD2_AUTOMONSTER group again when every monster
// of the scenario is dead and casts are left.
func (v *Game) respawnCastMonsters() {
	mt := v.monsterTest
	if mt == nil || !mt.spawned || mt.pack {
		return
	}

	stat := v.monsters.FindStat(mt.ref)
	if stat == nil {
		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	for i := 0; i < mt.count; i++ {
		angle := 2 * math.Pi * float64(i) / float64(mt.count)
		x := hx + int(math.Round(math.Cos(angle)*monsterTestRing))
		y := hy + int(math.Round(math.Sin(angle)*monsterTestRing))

		_, _ = v.monsters.SpawnNear(stat, x, y, 2)
	}

	v.Infof("AUTOCAST respawned %d x %s", mt.count, mt.ref)
}

// putHeroNear places the hero on a free subtile next to a position (scenario
// only: the monster could not be reached on foot).
func (v *Game) putHeroNear(x, y int) {
	p, ok := d2path.NearestFree(v.monsters.Grid(), d2path.MaskMonster, d2path.Point{X: x - 3, Y: y}, 6)
	if !ok {
		return
	}

	v.localPlayer.StopMoving()
	v.localPlayer.Position = d2vector.NewPosition(float64(p.X)+0.5, float64(p.Y)+0.5)
	v.localPlayer.StopMoving()
	v.Infof("AUTOCAST moved the hero next to the target at (%d,%d)", p.X, p.Y)
}

// castTestDone reports whether the OD2_AUTOCAST list is finished.
func (v *Game) castTestDone() bool {
	return v.castTestState != nil && v.castTestState.finished
}

func (v *Game) nearestMonster() *d2mapentity.Monster {
	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	var best *d2mapentity.Monster

	bestDist := 0

	for _, m := range v.monsters.Monsters() {
		if !m.Alive() {
			continue
		}

		mx, my := m.SubtilePos()
		if d := d2monster.Distance(hx-mx, hy-my); best == nil || d < bestDist {
			best, bestDist = m, d
		}
	}

	return best
}

// logCastSummary prints the OD2_AUTOCAST totals once.
func (v *Game) logCastSummary() {
	t := v.castTestState
	if t == nil || t.reported || v.skills == nil {
		return
	}

	t.reported = true
	c := v.skills.Counters
	m := v.monsters.Counters
	v.Infof("AUTOCAST summary skill=%q casts=%d refused=%d missiles=%d hits=%d misses=%d walls=%d expired=%d melee=%d "+
		"kills=%d damage=%d mana_spent=%.2f hero_mana=%s area=%d dot=%d minions=%d minion_hits=%d", t.skill, c.Casts, c.Refused,
		c.Missiles, c.Hits, c.Misses, c.Walls, c.Expired, c.Melee, c.Kills, c.Damage, float64(c.ManaSpent)/256,
		v.skills.Hero(v.localPlayer).ManaString(), c.AreaHits, c.DotDamage, m.Minions, m.MinionHits)
}
