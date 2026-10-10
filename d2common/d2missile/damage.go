package d2missile

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"

// Elem is one elemental damage triple: min and max in 8.8 fixed point, and a
// length in frames (cold, poison, burn, freeze, stun).
type Elem struct {
	Min, Max, Len int32
}

// DamageDesc is the missile damage descriptor (the 0x7c byte structure that
// MISSILE_BuildDamageDescriptor, 0x64ca60, stores in the missile unit). It is
// rolled into a combat damage struct when the missile hits. Values are 8.8
// fixed point. The build of the descriptor from a skill lives in d2skill.
type DamageDesc struct {
	PhysMin, PhysMax int32
	Fire             Elem
	Lightning        Elem
	Magic            Elem
	Cold             Elem // Len is the chill length
	Poison           Elem // Len is the poison length
	Burn             Elem // min/max per tick, Len duration
	FreezeLen        int32
	StunLen          int32
	// DamagePct is the percent damage bonus applied to physical damage (the
	// notes' descriptor field [0x1e]: stat 0x19 plus extras).
	DamagePct int32
	// ToHit is the to-hit bonus added to the owner's attack rating when the
	// missile rolls to hit.
	ToHit int
	// Flags are d2combat.DamageFlag bits copied into the damage struct.
	Flags    uint32
	HitClass int
	// Crit is descriptor flag 0x2 (verified): the critical / deadly / mastery
	// roll succeeded when the descriptor was built (0x64cce0). 0x64be80 turns
	// it into missile stat 0x8d = 1 and the physical damage struct builder
	// (0x5a673b) doubles the physical damage after the percent bonus.
	Crit bool
}

// Empty reports whether the descriptor deals no damage at all.
func (d *DamageDesc) Empty() bool {
	return d.PhysMax <= 0 && d.PhysMin <= 0 && d.Fire.Max <= 0 && d.Lightning.Max <= 0 && d.Magic.Max <= 0 &&
		d.Cold.Max <= 0 && d.Poison.Max <= 0 && d.Burn.Max <= 0 && d.StunLen <= 0 && d.FreezeLen <= 0 &&
		d.Fire.Min <= 0 && d.Lightning.Min <= 0 && d.Magic.Min <= 0 && d.Cold.Min <= 0
}

// Roll rolls every component into a damage struct and sets the hit bit. The
// rules are the verified ones of the exe's builder (0x5a63c0 / 0x5a6690, see
// d2combat.BuildMissileDamage): a component whose min or max is not positive
// is 0 and consumes no random step, min > max swaps, the roll is
// min + Roll(max-min), and the order of rolls is physical, fire, magic,
// lightning, cold, poison, burn.
func (d *DamageDesc) Roll(r d2combat.Roller) d2combat.Damage {
	dmg := d2combat.Damage{Flags: d.Flags, Result: d2combat.ResultHit, HitClass: int32(d.HitClass)}
	rr := func(e Elem) int32 { return d2combat.MinMaxRoll(r, e.Min, e.Max, 0, false) }

	p := d2combat.MinMaxRoll(r, d.PhysMin, d.PhysMax, 0, false)
	p += int32(int64(p) * int64(d.DamagePct) / 100)

	if p < 0 {
		p = 0
	}

	// Descriptor flag 0x2 (missile stat 0x8d, verified in verify-mastery-formulas /
	// missile-crit.md): the critical/deadly/mastery roll made when the descriptor
	// was built doubles the physical damage after the percent bonus and sets the
	// critical result bit, the same step d2combat.BuildMissileDamage applies for
	// stat 0x8d.
	if d.Crit && p > 0 {
		p *= 2
		dmg.Result |= d2combat.ResultCritical
	}

	dmg.Physical = p
	dmg.Fire = rr(d.Fire)
	dmg.Magic = rr(d.Magic)
	dmg.Lightning = rr(d.Lightning)

	dmg.Cold = rr(d.Cold)
	dmg.ColdLen = d.Cold.Len

	dmg.Poison = rr(d.Poison)
	dmg.PoisonLen = d.Poison.Len

	dmg.Burn = rr(d.Burn)
	dmg.BurnLen = d.Burn.Len

	dmg.FreezeLen = d.FreezeLen
	dmg.StunLen = d.StunLen

	return dmg
}
