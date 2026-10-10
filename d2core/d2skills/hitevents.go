package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Item event callbacks of a landed hit (VERIFIED in the notes: crushing blow
// is stat 136 / item event func 16 at 0x5bdbf0, open wounds is stat 135 / func
// 15 at 0x5bda80). Both run on the attacker's generator AFTER the base damage
// was subtracted from the defender.

const (
	// StateOpenWounds is the name of the timed state 0x3e created by open wounds.
	StateOpenWounds = "openwounds"
	// StateSanctuaryPhysZero names the attacker state 0x2f (States.txt row 47,
	// "sanctuary"; 0x3e is row 62 "openwounds", 0x15 is row 21 "stunned") that
	// zeroes positive physical resist against undead defenders (0x579b10).
	StateSanctuaryPhysZero = "sanctuary"
)

// specialCrushingClasses are the monster classes of helper 0x63fed0 (divisor
// 10 like a player): 0x10f, 0x152, 0x167, 0x230, 0x231.
var specialCrushingClasses = map[int]bool{0x10f: true, 0x152: true, 0x167: true, 0x230: true, 0x231: true}

func (e *Engine) isUndead(m *d2mapentity.Monster) bool {
	return m.Stat != nil && (m.Stat.IsUndeadLow || m.Stat.IsUndeadHigh)
}

// physNullified is the 0x579b10 special case (VERIFIED, verify-undead-helper.md):
// the stat is physical, the attacker has state 0x2f (States.txt row 47
// "sanctuary") and the defender is undead. Helper 0x63f9e0 returns 1 only for
// a monster unit (type 1, class id in range) whose monstats flag byte +0xd has
// bit 0x08 (lUndead, flag bit 11) or 0x10 (hUndead, flag bit 12). It is NOT the
// boss flag (bit 6, byte +0xc). The same helper adds stat 0x7c to attack rating.
func (e *Engine) physNullified(m *d2mapentity.Monster, src *d2mapentity.Player) bool {
	return src != nil && e.isUndead(m) && e.setOf(src.ID()).Active(e.frame, StateSanctuaryPhysZero)
}

// rawPhysResist is the defender's RAW stat 36 (monstats value of the
// difficulty, no curses, pierce or cap), which crushing blow uses.
func rawPhysResist(m *d2mapentity.Monster) int {
	s := m.Stat
	diff := int(m.Vitals.Difficulty)

	return [3]int{s.ResistancePhysicalNormal, s.ResistancePhysicalNightmare, s.ResistancePhysicalHell}[diff]
}

// crushingClass classifies a monster for CrushingBlowDivisor. The monster data
// flag +0x16 & 2 (also boss-like) is not mapped to a field and is UNVERIFIED
// here.
func crushingClass(m *d2mapentity.Monster) d2combat.CrushingBlowDefender {
	switch {
	case specialCrushingClasses[m.MonstatID()]:
		return d2combat.CBSpecial
	case m.Stat.IsSpecialBoss:
		return d2combat.CBBossMonster
	}

	return d2combat.CBNormalMonster
}

// hitEventPlan is the result of the pure part of the item events.
type hitEventPlan struct {
	CrushRemoved int // 8.8 life removed by crushing blow
	CrushKilled  bool
	Wounds       d2combat.OpenWoundsEffect
}

// planHitEvents runs crushing blow and then open wounds, in the order of the
// item event table, on the attacker's generator r. life is the defender's 8.8
// life after the base damage.
func planHitEvents(r d2combat.Roller, crushChance, woundsChance int, class d2combat.CrushingBlowDefender, players int,
	missile bool, life, rawPhys, attackerLevel int, halved bool, now int) hitEventPlan {
	var plan hitEventPlan

	cb := d2combat.RollCrushingBlow(r, d2combat.CrushingBlowInput{
		Chance:             crushChance,
		DefenderLife:       life,
		Divisor:            d2combat.CrushingBlowDivisor(class, players, missile),
		DefenderPhysResist: rawPhys,
	})
	plan.CrushRemoved, plan.CrushKilled = cb.Removed, cb.Killed

	// a defender crushing blow just killed gets no wound (no roll consumed);
	// UNVERIFIED whether the exe still rolls for a dead unit
	if !cb.Killed {
		v := d2combat.OpenWoundsValue(attackerLevel, false, true, halved, missile)
		plan.Wounds = d2combat.RollOpenWounds(r, woundsChance, v, now)
	}

	return plan
}

// itemEvents applies crushing blow and open wounds of the hero's items after
// a hit that already hurt the monster. melee is false for missiles (event 6:
// the crushing blow divisor doubles); spells without physical damage are
// excluded by the caller (UNVERIFIED which hits dispatch the events).
func (e *Engine) itemEvents(m *d2mapentity.Monster, p *d2mapentity.Player, melee bool) {
	if p == nil || p.Stats == nil || p.Stats.Totals == nil || !m.Alive() {
		return
	}

	t := p.Stats.Totals
	if t.SlowTarget > 0 {
		e.applySlowTarget(m, t.SlowTarget)
	}

	if t.CrushingBlow <= 0 && t.OpenWounds <= 0 {
		return
	}

	plan := planHitEvents(e.hero(p).seed, t.CrushingBlow, t.OpenWounds, crushingClass(m), maxInt(len(e.heroes), 1), !melee,
		m.Vitals.HP<<d2combat.FixedShift, rawPhysResist(m), p.Stats.Level, false, e.frame)

	if whole := plan.CrushRemoved >> d2combat.FixedShift; whole > 0 {
		e.Counters.Damage += whole
		e.emit("damage", "CRUSHING BLOW target=%s removed=%d hp=%d->%d/%d", m.Label(), whole, m.Vitals.HP,
			maxInt(m.Vitals.HP-whole, 0), m.Vitals.MaxHP)
		e.monsters.Damage(m, whole, p)

		if !m.Alive() {
			e.Counters.Kills++
			e.emit("damage", "KILL skill=%q target=%s", "crushing blow", m.Label())

			return
		}
	}

	if plan.Wounds.Triggered {
		e.applyOpenWounds(m, p, plan.Wounds)
	}
}

// applyOpenWounds attaches state 0x3e (200 frames, skill 0, level 1, hp regen
// = -value). The same state just refreshes the expiry (0x56c740). The life
// drained per tick is NOT applied: the unit of stat 0x4a per tick is
// UNVERIFIED in the notes.
func (e *Engine) applyOpenWounds(m *d2mapentity.Monster, p *d2mapentity.Player, w d2combat.OpenWoundsEffect) {
	set := e.setOf(m.ID())

	if cur := set.Get(e.frame, StateOpenWounds); cur != nil {
		next := *cur
		next.Until = w.Expires
		set.Apply(e.frame, next)
	} else {
		set.Apply(e.frame, d2state.Instance{
			Name: StateOpenWounds, Until: w.Expires, Level: 1, Source: p.ID(),
			Mods: []d2state.StatMod{{Stat: "hpregen", Value: w.RegenStat}},
		})
	}

	e.emit("state", "STATE open_wounds target=%s hpregen=%d until=%d", m.Label(), w.RegenStat, w.Expires)
}
