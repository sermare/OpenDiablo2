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
}

// Empty reports whether the descriptor deals no damage at all.
func (d *DamageDesc) Empty() bool {
	return d.PhysMax <= 0 && d.PhysMin <= 0 && d.Fire.Max <= 0 && d.Lightning.Max <= 0 && d.Magic.Max <= 0 &&
		d.Cold.Max <= 0 && d.Poison.Max <= 0 && d.Burn.Max <= 0 && d.StunLen <= 0 && d.FreezeLen <= 0 &&
		d.Fire.Min <= 0 && d.Lightning.Min <= 0 && d.Magic.Min <= 0 && d.Cold.Min <= 0
}

// roll returns min + Roll(max-min), the game's range roll (SUnitDmg: elemental
// roll is min plus a random value modulo (max-min); "rolls elemMin +
// rand%(max-min)" in the notes, inferred for the exact modulus).
func roll(r d2combat.Roller, min, max int32) int32 {
	if max <= min || r == nil {
		return min
	}

	return min + int32(r.Roll(max-min))
}

// Roll rolls every component into a damage struct and sets the hit bit. Order
// of rolls: physical, fire, lightning, magic, cold, poison, burn.
func (d *DamageDesc) Roll(r d2combat.Roller) d2combat.Damage {
	dmg := d2combat.Damage{Flags: d.Flags, Result: d2combat.ResultHit, HitClass: int32(d.HitClass)}

	if d.PhysMax > 0 || d.PhysMin > 0 {
		p := roll(r, d.PhysMin, d.PhysMax)
		p += int32(int64(p) * int64(d.DamagePct) / 100)

		if p < 0 {
			p = 0
		}

		dmg.Physical = p
	}

	if d.Fire.Max > 0 || d.Fire.Min > 0 {
		dmg.Fire = roll(r, d.Fire.Min, d.Fire.Max)
	}

	if d.Lightning.Max > 0 || d.Lightning.Min > 0 {
		dmg.Lightning = roll(r, d.Lightning.Min, d.Lightning.Max)
	}

	if d.Magic.Max > 0 || d.Magic.Min > 0 {
		dmg.Magic = roll(r, d.Magic.Min, d.Magic.Max)
	}

	if d.Cold.Max > 0 || d.Cold.Min > 0 {
		dmg.Cold = roll(r, d.Cold.Min, d.Cold.Max)
		dmg.ColdLen = d.Cold.Len
	}

	if d.Poison.Max > 0 || d.Poison.Min > 0 {
		dmg.Poison = roll(r, d.Poison.Min, d.Poison.Max)
		dmg.PoisonLen = d.Poison.Len
	}

	if d.Burn.Max > 0 || d.Burn.Min > 0 {
		dmg.Burn = roll(r, d.Burn.Min, d.Burn.Max)
		dmg.BurnLen = d.Burn.Len
	}

	dmg.FreezeLen = d.FreezeLen
	dmg.StunLen = d.StunLen

	return dmg
}
