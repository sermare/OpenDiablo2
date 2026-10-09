package d2combat

// Unit stat ids read by the missile damage builder (MISSILE_BuildDamageStructFromStats, 0x5a63c0).
const (
	statMinDmg       = 0x15
	statMaxDmg       = 0x16
	statDmgPct       = 0x19
	statFireMin      = 0x30
	statFireMax      = 0x31
	statLightMin     = 0x32
	statLightMax     = 0x33
	statMagicMin     = 0x34
	statMagicMax     = 0x35
	statColdMin      = 0x36
	statColdMax      = 0x37
	statColdLen      = 0x38
	statPoisonMin    = 0x39
	statPoisonMax    = 0x3a
	statPoisonLen    = 0x3b
	statLifeLeech    = 0x3c
	statManaLeech    = 0x3e
	statStamLeech    = 0x40
	statStunLen      = 0x42
	statDemonPct     = 0x79
	statUndeadPct    = 0x7a
	statDeadly       = 0x8d
	statIgnoreUndead = 0x67
	statIgnoreDemon  = 0x68
	statIgnoreBeast  = 0x6a
	statBurnLen      = 0x13b
	statBurnMin      = 0x13c
	statBurnMax      = 0x13d
	statPoisonCount  = 0x146
	statFirePct      = 0x149
	statLightPct     = 0x14a
	statColdPct      = 0x14b
	statPoisonPct    = 0x14c
	statMagicPct     = 0x165
)

// MissileTarget describes the optional target of a missile damage build. A
// nil target skips the percent damage bonus.
type MissileTarget struct {
	Demon, Undead bool
}

// MinMaxRoll is helper 0x5a6330: roll between a min and max stat. Verified:
// max <= 0 or min <= 0 give 0 (and no RNG step); min > max swaps; pct (when
// the type has a mastery stat and it is non-zero) scales both ends with the
// truncating MulDiv; the roll is min + Roll(max-min) (one step when max > min).
func MinMaxRoll(r Roller, min, max int32, pct int32, hasPct bool) int32 {
	if max <= 0 || min <= 0 {
		return 0
	}

	if min > max {
		min, max = max, min
	}

	if hasPct && pct != 0 {
		min += MulDiv(min, pct, 100)
		max += MulDiv(max, pct, 100)
	}

	return min + int32(r.Roll(max-min))
}

// BuildMissileDamage reproduces 0x5a63c0: it rolls a combat damage struct from
// the stats of a missile unit (stat reads through get, a missing stat is 0).
// The percent bonus stat 0x19 (+0x79 vs demons, +0x7a vs undead, floored at
// -90) is applied to physical damage only when a target is given, then
// deadly strike (stat 0x8d on the missile) doubles physical and sets result
// 0x2000. Roll order (verified): physical, fire, magic, lightning, cold,
// poison, burn. Unverified: the per-monster-class bonus list (stat 0xb4) that
// the exe adds to the percent when the target is a monster; not modelled.
func BuildMissileDamage(r Roller, get func(stat int) int32, tgt *MissileTarget) Damage {
	var d Damage

	d.Physical = MinMaxRoll(r, get(statMinDmg), get(statMaxDmg), 0, false)
	d.Fire = MinMaxRoll(r, get(statFireMin), get(statFireMax), get(statFirePct), true)
	d.Magic = MinMaxRoll(r, get(statMagicMin), get(statMagicMax), get(statMagicPct), true)
	d.Lightning = MinMaxRoll(r, get(statLightMin), get(statLightMax), get(statLightPct), true)
	d.Cold = MinMaxRoll(r, get(statColdMin), get(statColdMax), get(statColdPct), true)
	d.ColdLen = get(statColdLen)
	d.Poison = MinMaxRoll(r, get(statPoisonMin), get(statPoisonMax), get(statPoisonPct), true)
	d.PoisonLen = get(statPoisonLen)

	if n := get(statPoisonCount); n > 1 {
		d.PoisonLen /= n
	}

	d.Burn = MinMaxRoll(r, get(statBurnMin), get(statBurnMax), get(statFirePct), true)
	d.BurnLen = get(statBurnLen)
	d.ManaLeech = get(statManaLeech)
	d.LifeLeech = get(statLifeLeech)
	d.StaminaLeech = get(statStamLeech)
	d.StunLen = get(statStunLen)

	applyPhysPct(&d, get, tgt)

	if get(statIgnoreUndead) != 0 {
		d.Flags |= DamageFlagIgnoreUndead
	}

	if get(statIgnoreDemon) != 0 {
		d.Flags |= DamageFlagIgnoreDemon
	}

	if get(statIgnoreBeast) != 0 {
		d.Flags |= DamageFlagIgnoreBeast
	}

	return d
}

// BuildMissilePhysical is the physical case (type 0) of the missile hit
// damage function 0x5a6690: the physical roll, the percent bonus (only with a
// target) and the deadly strike doubling, nothing else. Verified.
func BuildMissilePhysical(r Roller, get func(stat int) int32, tgt *MissileTarget) Damage {
	var d Damage

	d.Physical = MinMaxRoll(r, get(statMinDmg), get(statMaxDmg), 0, false)
	applyPhysPct(&d, get, tgt)

	return d
}

// applyPhysPct applies the percent bonus (target required) and deadly strike.
func applyPhysPct(d *Damage, get func(stat int) int32, tgt *MissileTarget) {
	if tgt != nil {
		pct := get(statDmgPct)
		if tgt.Demon {
			pct += get(statDemonPct)
		}

		if tgt.Undead {
			pct += get(statUndeadPct)
		}

		if pct < -90 {
			pct = -90
		}

		d.Physical += int32(int64(d.Physical*pct) / 100) // the product wraps in 32 bits like the exe
	}

	if get(statDeadly) != 0 {
		d.Physical *= 2
		d.Result |= ResultCritical
	}
}
