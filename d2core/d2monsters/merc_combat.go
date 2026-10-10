package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// MercDamageType is a damage type a mercenary can resist.
type MercDamageType int

// The damage types of the resist pipeline that apply to a mercenary.
const (
	MercPhysical MercDamageType = iota
	MercFire
	MercCold
	MercLightning
	MercPoison
	MercMagic
)

// gearIdx maps an elemental type to the index in d2hireling.Gear.Resist (fire, cold, lightning, poison).
func gearIdx(t MercDamageType) int {
	switch t {
	case MercFire:
		return 0
	case MercCold:
		return 1
	case MercLightning:
		return 2
	case MercPoison:
		return 3
	}

	return -1
}

// effectiveGear is the gear effect in force: the computed one, or the plain table
// stats when no gear was applied (so a merc without gear resists exactly its table resist).
func (mu *mercUnit) effectiveGear() d2hireling.Gear {
	if mu.gear.MaxHP > 0 {
		return mu.gear
	}

	return d2hireling.ApplyGear(mu.base, nil)
}

// mercResistInput builds the ResistInput of a mercenary defender. VERIFIED
// (verify-resist.md 0x579ef0): the "no cap, no difficulty penalty" context flag is set only
// for non-mercenary monsters, so a mercenary is capped (75 + the items' max resist, 95
// at most; physical 50) and takes the difficulty penalty, like a player. Physical and
// magic resist are exempt from the penalty. Mercs carry no pierce stats of their own.
func (d *Director) mercResistInput(mu *mercUnit, t MercDamageType) d2combat.ResistInput {
	g := mu.effectiveGear()
	in := d2combat.ResistInput{}

	switch t {
	case MercPhysical:
		in.Resist, in.IsPhysical, in.NoDifficultyPenalty = g.PhysResist, true, true
	case MercMagic:
		in.HasMaxResist, in.NoDifficultyPenalty = true, true // gear gives no magic resist in the engine yet
	default:
		i := gearIdx(t)
		in.Resist = g.RawResist[i]
		in.HasMaxResist, in.MaxResistBonus = true, g.MaxResistBonus[i]
		in.DifficultyPenalty = d2difficulty.ResistPenalty(!d.opt.Expansion, d2difficulty.Level(d.opt.Difficulty))
	}

	return in
}

// mercReduceFor is what a mercenary actually loses from one damage component of the
// given type: flat damage reduction (stat 34, physical only: the gear has no stat 35 yet),
// then the effective resist (d2combat.ReduceComponent). The result is not floored.
func (d *Director) mercReduceFor(tu *unit, t MercDamageType, dmg int) int {
	flat := 0

	if t == MercPhysical {
		flat = tu.merc.effectiveGear().DamageReduction
	}

	// mercs have no absorb stats in the engine yet
	out, _ := d2combat.ReduceComponent(dmg, flat, d2combat.EffectiveResist(d.mercResistInput(tu.merc, t)), false, false, 0, 0)

	return out
}

// MercReduce reduces one damage component aimed at the owner's mercenary and returns what
// the merc would lose (not floored). It returns dmg unchanged when the owner has no merc.
func (d *Director) MercReduce(owner *d2mapentity.Player, t MercDamageType, dmg int) int {
	if u := d.mercs[owner]; u != nil {
		return d.mercReduceFor(u, t, dmg)
	}

	return dmg
}

// takenByMerc is the physical hit of a monster after the merc's gear: a hit never
// drops below 0 here (the total is floored, and life is subtracted only when positive).
func (d *Director) takenByMerc(tu *unit, dmg int) int {
	if v := d.mercReduceFor(tu, MercPhysical, dmg); v > 0 {
		return v
	}

	return 0
}

// mercRegen is the over-time part of the potions given to a merc.
type mercRegen struct {
	r   d2inventory.Regen
	acc float64
}

// DrinkMerc gives a potion effect to the owner's merc (d2inventory.PotionEffectOf of the
// potion, the same effect the hero's belt uses). The instant part is a percent of the
// merc's maximum life (d2inventory.ApplyInstant), the over-time part (healing potions) is
// restored by the merc's regen every frame. Mana parts are ignored (a merc has no mana).
// A potion without life in it (antidote, thawing, stamina) is used up with no effect: the
// engine models no unit states on mercenaries, so the state it cures is never there
// (UNVERIFIED for the real state handling). A dead merc takes nothing. Reports whether
// life was or will be restored.
func (d *Director) DrinkMerc(owner *d2mapentity.Player, e d2inventory.PotionEffect) bool {
	u := d.mercs[owner]
	if u == nil || !u.m.Alive() {
		return false
	}

	v := &u.m.Vitals
	before := v.HP

	if e.InstantHPPercent > 0 {
		v.HP = d2inventory.ApplyInstant(v.HP, v.MaxHP, e.InstantHPPercent)
	}

	if e.HP > 0 {
		hpOnly := e
		hpOnly.Mana = 0
		u.merc.regen.r.Add(hpOnly)
	}

	d.emit("merc", "MERC potion name=%s instant=%d%% over_time=%.0f seconds=%.1f hp=%d->%d/%d", u.m.Label(),
		e.InstantHPPercent, e.HP, e.Seconds, before, v.HP, v.MaxHP)

	return e.InstantHPPercent > 0 || e.HP > 0
}

// stepRegen restores the over-time life of the potions given to the merc, one 25 Hz frame.
func (mu *mercUnit) stepRegen(v *d2mapentity.MonsterVitals) {
	if !mu.regen.r.Active() {
		return
	}

	hp, _ := mu.regen.r.Tick(frameSeconds)

	mu.regen.acc += hp
	if whole := int(mu.regen.acc); whole > 0 {
		mu.regen.acc -= float64(whole)

		if v.HP += whole; v.HP > v.MaxHP {
			v.HP = v.MaxHP
		}
	}
}
