package d2gamescreen

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Hero death, respawn and corpse recovery. The rules and their provenance are
// documented in d2core/d2hero/death.go; this file runs them in the game.
//
// Flow: the hero's life reaches 0 -> Die() (DT then DD animation), the penalty
// is applied and the hero is saved at once (the original does both) -> a
// softcore hero respawns in town after the animation, without equipment and
// gold, the corpse lies where it fell and is recovered by walking onto it; a
// hardcore hero stays dead ("You have died") and the character is flagged dead.
const (
	// respawnDelay is how long the hero lies dead before the softcore respawn.
	// UNVERIFIED engine choice (the original timing was not measured).
	respawnDelay = 2.5
	// corpseReach is how close (subtiles) the hero must come to recover a corpse.
	corpseReach = 3
	// goldPileSpread places the dropped gold this many subtiles beside the corpse.
	goldPileSpread = 2
	// deathTestDelay is how long after the hero exists OD2_AUTODEATH starts.
	deathTestDelay = 3.0
	// deathTestStartDistance moves the hero this far (subtiles) from the town
	// start before the fight, so the respawn visibly changes its position.
	deathTestStartDistance = 16
	// deathTestSeconds ends the scenario if the hero somehow survives.
	deathTestSeconds = 60.0
	deathTestSettle  = 1.0

	deathMessage = "You have died" // UNVERIFIED wording (no tbl string was looked up)
)

// deathState is the Game's runtime state of the death system.
type deathState struct {
	townX, townY int // where the hero first appeared: the respawn point
	known        bool
	dead         bool
	deadFor      float64
	hardcore     bool
	corpse       *d2mapentity.Player // entity lying at the death spot
	corpseX      int
	corpseY      int

	test *deathTest
}

// deathTest is the OD2_AUTODEATH scenario.
type deathTest struct {
	elapsed   float64
	started   bool
	respawned bool
	recovered bool
	done      bool
	afterDone float64

	goldBefore, expBefore, levelBefore int
	hpBefore                           string
	equipBefore                        string
	deathPos                           [2]int
	outcome                            d2hero.DeathOutcome
}

// advanceDeath runs once per frame.
func (v *Game) advanceDeath(elapsed float64) {
	p := v.localPlayer
	if p == nil || p.Stats == nil {
		return
	}

	st := &v.death

	if !st.known {
		st.townX, st.townY = int(p.Position.X()), int(p.Position.Y())
		st.known = true
		st.hardcore = p.Hardcore

		// a corpse left by an earlier session lies at the town start (the
		// game's loader places it at the default position, 0x568500)
		if p.Death != nil && p.Death.Corpse != nil {
			v.placeCorpse(st.townX+goldPileSpread, st.townY)
			v.Infof("DEATH pending corpse from the save placed at (%d,%d)", st.corpseX, st.corpseY)
		}
	}

	v.advanceDeathTest(elapsed)

	if !p.IsDead() {
		if p.Stats.Health <= 0 {
			v.heroDies()
		}

		v.advanceCorpseRecovery()

		return
	}

	if !st.dead {
		return
	}

	st.deadFor += elapsed

	if st.hardcore {
		return // final; the player leaves through the escape menu
	}

	if p.DeathAnimationDone() && st.deadFor >= respawnDelay {
		v.respawnHero()
	}
}

// difficulty returns the difficulty the monsters run at (the same switch the
// monster scenarios use).
func (v *Game) difficulty() int {
	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOMONSTER_DIFF")); err == nil && n >= 0 && n <= 2 {
		return n
	}

	return int(d2monster.Normal)
}

func (v *Game) heroDies() {
	p, st := v.localPlayer, &v.death

	st.dead, st.deadFor = true, 0
	st.hardcore = p.Hardcore

	if v.attackTarget != nil {
		v.attackTarget = nil
	}

	level := p.Stats.Level
	next := v.asset.Records.GetExperienceBreakpoint(p.Class, level)
	start := 0

	if level > 1 {
		start = v.asset.Records.GetExperienceBreakpoint(p.Class, level-1)
	}

	if p.Death == nil {
		p.Death = &d2hero.DeathState{}
	}

	x, y := int(p.Position.X()), int(p.Position.Y())

	var equip d2inventory.CharacterEquipment
	if p.Equipment != nil {
		equip = *p.Equipment
	}

	out := p.Death.Die(d2hero.DeathInput{
		Hardcore: p.Hardcore, Difficulty: v.difficulty(), Level: level, Experience: p.Stats.Experience,
		Gold: p.Gold, ExpStart: start, ExpNext: next, X: x, Y: y, Equipment: equip,
	})

	if t := st.test; t != nil {
		t.outcome, t.deathPos = out, [2]int{x, y}
	}

	p.Stats.Experience = out.NewExperience
	st.corpseX, st.corpseY = x, y

	v.takeGold(p.Gold, x, y)
	p.Die()

	v.Infof("DEATH hero=%s hardcore=%v pos=(%d,%d) level=%d exp_lost=%d exp=%d gold_dropped=%d deaths=%d died_flag=%v corpse=%v",
		p.Name(), p.Hardcore, x, y, level, out.ExpLost, p.Stats.Experience, out.GoldDropped, p.Death.Deaths,
		p.Death.Died, describeEquipment(&equip))

	if p.Hardcore {
		v.Infof("DEATH hardcore: the character is dead (d2s status died=0x08), the character select refuses it")

		if v.gameControls != nil {
			v.gameControls.SetZoneChangeText(deathMessage)
			v.gameControls.ShowZoneChangeText()
		}
	}

	// the original saves at once when a hero dies
	v.saveBeforeExit()
}

// takeGold removes the carried gold and drops it as a pile beside the corpse.
func (v *Game) takeGold(amount, x, y int) {
	p := v.localPlayer
	if amount <= 0 {
		return
	}

	if v.gameControls != nil {
		v.gameControls.AddGold(-amount)
	} else {
		p.Gold = 0
	}

	ent, err := v.gameClient.MapEngine.NewGoldPile(amount, v.asset.TranslateString("gld"), x+goldPileSpread, y)
	if err != nil {
		v.Warningf("DEATH could not drop the gold pile: %v", err)
		return
	}

	v.gameClient.MapEngine.AddEntity(ent)
}

// respawnHero stands the softcore hero up in town at the start point. The
// equipment stays on the corpse; life is RespawnLife (unverified).
func (v *Game) respawnHero() {
	p, st := v.localPlayer, &v.death

	if !p.Death.Respawn(p.Hardcore) {
		return
	}

	var corpseEquip d2inventory.CharacterEquipment
	if p.Death.Corpse != nil {
		corpseEquip = p.Death.Corpse.Equipment
	}

	// the dead body stays; the hero walks on without its equipment
	v.placeCorpse(st.corpseX, st.corpseY)

	if p.Equipment != nil {
		*p.Equipment = d2inventory.CharacterEquipment{}
	}

	p.SetPositionSubtile(st.townX, st.townY)
	p.Revive()
	p.ApplyEquipment()
	p.SetIsInTown(true)
	p.Stats.Health = d2hero.RespawnLife(p.Stats.MaxHealth)
	p.Stats.Mana = 0
	st.dead = false

	v.Infof("DEATH respawn pos=(%d,%d) town_start=(%d,%d) hp=%d/%d mana=%d gold=%d equipment=%s corpse_at=(%d,%d) corpse_items=%s",
		int(p.Position.X()), int(p.Position.Y()), st.townX, st.townY, p.Stats.Health, p.Stats.MaxHealth, p.Stats.Mana,
		p.Gold, describeEquipment(p.Equipment), st.corpseX, st.corpseY, describeEquipment(&corpseEquip))

	v.saveBeforeExit()
}

// placeCorpse creates the entity of a dead hero at a subtile position.
func (v *Game) placeCorpse(x, y int) {
	p, st := v.localPlayer, &v.death

	if st.corpse != nil {
		v.gameClient.MapEngine.RemoveEntity(st.corpse)
		st.corpse = nil
	}

	st.corpseX, st.corpseY = x, y

	stats := *p.Stats
	equip := d2inventory.CharacterEquipment{}

	if p.Death != nil && p.Death.Corpse != nil {
		equip = p.Death.Corpse.Equipment
	}

	c := v.gameClient.MapEngine.NewPlayer("corpse:"+p.ID(), p.Name(), x, y, 0, p.Class, &stats, p.Skills,
		&equip, 0, 0, 0)
	c.SetIsInTown(false)
	c.LieDead()
	v.gameClient.MapEngine.AddEntity(c)
	st.corpse = c
}

// advanceCorpseRecovery gives the equipment back when the hero reaches the
// corpse (the original needs a click on it; walking onto it is simpler).
func (v *Game) advanceCorpseRecovery() {
	p, st := v.localPlayer, &v.death
	if st.corpse == nil || p.Death == nil || p.Death.Corpse == nil {
		return
	}

	if d2monster.Distance(int(p.Position.X())-st.corpseX, int(p.Position.Y())-st.corpseY) > corpseReach {
		return
	}

	v.recoverCorpse()
}

func (v *Game) recoverCorpse() {
	p, st := v.localPlayer, &v.death

	eq := p.Death.Recover()
	if eq == nil {
		return
	}

	if p.Equipment == nil {
		p.Equipment = &d2inventory.CharacterEquipment{}
	}

	*p.Equipment = *eq
	p.ApplyEquipment()

	if st.corpse != nil {
		v.gameClient.MapEngine.RemoveEntity(st.corpse)
		st.corpse = nil
	}

	v.Infof("DEATH corpse recovered equipment=%s", describeEquipment(eq))
	v.saveBeforeExit()
}

func describeEquipment(e *d2inventory.CharacterEquipment) string {
	if e == nil {
		return "[]"
	}

	var parts []string

	add := func(slot, code string) {
		if code != "" {
			parts = append(parts, slot+"="+strings.TrimSpace(code))
		}
	}

	if e.Head != nil {
		add("head", e.Head.ItemCode)
	}

	if e.Torso != nil {
		add("torso", e.Torso.ItemCode)
	}

	if e.Legs != nil {
		add("legs", e.Legs.ItemCode)
	}

	if e.RightArm != nil {
		add("gloves", e.RightArm.ItemCode)
	}

	if e.LeftArm != nil {
		add("leftarm", e.LeftArm.ItemCode)
	}

	if e.RightHand != nil {
		add("rhand", e.RightHand.GetItemCode())
	}

	if e.LeftHand != nil {
		add("lhand", e.LeftHand.GetItemCode())
	}

	if e.Shield != nil {
		add("shield", e.Shield.ItemCode)
	}

	return "[" + strings.Join(parts, " ") + "]"
}

// advanceDeathTest implements OD2_AUTODEATH=1. A weak hero (set
// OD2_AUTODEATH_LEVEL=<n> to test the experience penalty at a higher level,
// OD2_AUTOMONSTER_DIFF for the difficulty, OD2_AUTODEATH_MONSTER for the
// killer, default Andariel) is moved away from the town start, strong
// monsters are spawned next to it, and the scenario logs the death, the
// penalties, the respawn, the recovery of the corpse and the exported .d2s.
// OD2_AUTODEATH_HP=<n> starts the fight with n life points.
// OD2_AUTODEATH_HARDCORE=1 plays a hardcore death instead (no respawn).
func (v *Game) advanceDeathTest(elapsed float64) {
	if os.Getenv("OD2_AUTODEATH") == "" {
		return
	}

	st := &v.death

	t := st.test
	if t == nil {
		t = &deathTest{}
		st.test = t
	}

	if t.done {
		return
	}

	p := v.localPlayer
	t.elapsed += elapsed

	if !t.started {
		if t.elapsed < deathTestDelay || v.gameControls == nil || v.monsterDirector() == nil {
			return
		}

		t.started = true
		v.startDeathTest(t)

		return
	}

	switch {
	case p.IsDead() && st.dead && st.hardcore:
		if p.DeathAnimationDone() {
			t.afterDone += elapsed
		}

		if t.afterDone >= deathTestSettle {
			v.finishDeathTest(t)
		}
	case !p.IsDead() && p.Death != nil && p.Death.Deaths > 0 && !t.respawned:
		t.respawned = true
	case t.respawned && !t.recovered:
		// the scenario stands the hero on the corpse (the monsters that killed it
		// are still there, so it cannot walk), then lets the recovery run
		t.recovered = true

		p.SetPositionSubtile(st.corpseX, st.corpseY)
		v.recoverCorpse()
	case t.recovered:
		t.afterDone += elapsed
		if t.afterDone >= deathTestSettle {
			v.finishDeathTest(t)
		}
	case t.elapsed > deathTestDelay+deathTestSeconds:
		v.Infof("DEATHTEST timeout: the hero survived")
		v.finishDeathTest(t)
	}
}

func (v *Game) startDeathTest(t *deathTest) {
	p, st := v.localPlayer, &v.death

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTODEATH_LEVEL")); err == nil && n > 1 && n < 100 {
		v.setHeroLevel(n)
	}

	if os.Getenv("OD2_AUTODEATH_HARDCORE") != "" {
		p.Hardcore, st.hardcore = true, true
	}

	if n, err := strconv.Atoi(os.Getenv("OD2_AUTODEATH_HP")); err == nil && n > 0 && n < p.Stats.MaxHealth {
		p.Stats.Health = n
	}

	if p.Gold < 1000 && v.gameControls != nil {
		v.gameControls.AddGold(1000 - p.Gold)
	}

	// the fight happens outside the safety of town (monsters leave heroes in
	// town alone), away from the respawn point
	p.SetIsInTown(false)
	p.SetPositionSubtile(st.townX+deathTestStartDistance, st.townY)

	t.goldBefore, t.expBefore, t.levelBefore = p.Gold, p.Stats.Experience, p.Stats.Level
	t.hpBefore = fmt.Sprintf("%d/%d", p.Stats.Health, p.Stats.MaxHealth)
	t.equipBefore = describeEquipment(p.Equipment)

	name := os.Getenv("OD2_AUTODEATH_MONSTER")
	if name == "" {
		name = "Andariel"
	}

	stat := v.monsters.FindStat(name)
	if stat == nil {
		v.Errorf("DEATHTEST unknown monster %q", name)
		v.autoTestExit()

		return
	}

	hx, hy := int(p.Position.X()), int(p.Position.Y())

	v.Infof("DEATHTEST start hero=%s class=%v hardcore=%v level=%d exp=%d gold=%d hp=%s pos=(%d,%d) town_start=(%d,%d) "+
		"difficulty=%d killer=%s equipment=%s", p.Name(), p.Class, p.Hardcore, t.levelBefore, t.expBefore, t.goldBefore,
		t.hpBefore, hx, hy, st.townX, st.townY, v.difficulty(), stat.Key, t.equipBefore)

	for i := 0; i < 2; i++ {
		if _, err := v.monsters.SpawnNear(stat, hx+6, hy+i*3, 2); err != nil {
			v.Errorf("DEATHTEST spawn failed: %v", err)
		}
	}
}

// setHeroLevel makes the hero a higher level for the experience test: it
// stands midway through the level.
func (v *Game) setHeroLevel(level int) {
	p := v.localPlayer
	start := v.asset.Records.GetExperienceBreakpoint(p.Class, level-1)
	next := v.asset.Records.GetExperienceBreakpoint(p.Class, level)

	p.Stats.Level = level
	p.Stats.Experience = start + (next-start)/2
	p.Stats.NextLevelExp = next
}

func (v *Game) finishDeathTest(t *deathTest) {
	p := v.localPlayer
	t.done = true

	deaths := 0
	if p.Death != nil {
		deaths = p.Death.Deaths
	}

	v.Infof("DEATHTEST summary deaths=%d exp_lost=%d (%d -> %d) gold_dropped=%d gold_now=%d died_flag=%v hardcore=%v "+
		"equipment_before=%s equipment_now=%s corpse_pending=%v", deaths, t.outcome.ExpLost, t.expBefore,
		p.Stats.Experience, t.outcome.GoldDropped, p.Gold, p.Death != nil && p.Death.Died, p.Hardcore,
		t.equipBefore, describeEquipment(p.Equipment), p.Death != nil && p.Death.Corpse != nil)

	v.autoTestExit()
}
