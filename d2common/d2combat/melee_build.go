package d2combat

// Port of COMBAT_BuildAttackerDamage (0x5794e0) for the parts that do not
// depend on the wielded weapon, verified against the real function in the
// emulator (golden dm_mel). The physical roll of the weapon (0x579120) and the
// weapon strike chance (0x646bc0) are NOT covered: callers pass the rolled
// physical value in and the strike chance is limited to stats 0x151 and 0x8d.
// Unverified: the per monster class bonus list (helper 0x5793c0, stat 0xb4) is
// not modelled.

// Stat ids read by the attacker damage builder (in addition to those used by
// BuildMissileDamage).
const (
	statDefenderAC    = 0x1f  // defender stat that stat 0x78 adjusts (player vs monster)
	statAttackerDefAd = 0x78  // added to the defender's stat 0x1f by a player attacker
	statLeechMonMin   = 0x3d  // monster life leech max (min is 0x3c)
	statManaMonMax    = 0x3f  // monster mana leech max
	statStamMonMax    = 0x41  // monster stamina leech max
	statPoisonOver    = 0x65  // skill_poison_override_length
	statStrikeCrit    = 0x151 // passive_critical_strike
)

// ConvertClass* are the values of the damage struct byte +0x65 (conversion of
// part of the physical damage into an element). 10 picks one of 1..5 at random.
const ConvertRandom = 10

// MeleeIn is the input of BuildAttackerDamage.
type MeleeIn struct {
	Get          func(stat int) int32 // attacker stats
	AttackerKind int                  // unit type of the attacker: 0 player, 1 monster
	AttackerMerc bool                 // helper 0x625c10(attacker) == 2
	Special      bool                 // helper 0x63fed0(attacker)
	DefenderKind int
	DefenderDemon,
	DefenderUndead bool
	DefenderAC int32 // defender stat 0x1f before the call
	Scale      uint8 // SrcDam scale byte (0 means 128)
	// SkillOverride is the third stack argument (disables the weapon strike roll).
	SkillOverride bool
	Phys          int32 // physical damage rolled by 0x579120 (already scaled)
	// Weapon, when non-nil, makes BuildAttackerDamage roll the physical damage
	// itself with RollPhysical (the real call 0x579120(unit, true, 0, 0, 0,
	// struct pct, struct physical, scale)); Phys is then ignored. Get, PctAdd,
	// Flat and Scale of the value are filled in here.
	Weapon *WeaponRollIn
	// ActiveWeapon is helper 0x5335c0 (a weapon is wielded) and WeaponStrike
	// the weapon deadly strike chance 0x646bc0(mode 2); both only matter with
	// SkillOverride false.
	ActiveWeapon bool
	// UndeadBlunt: the wielded weapon is of item type 0x39 (0x629d70), worth
	// +50 percent damage against undead (0x579380).
	UndeadBlunt  bool
	WeaponStrike int32
	// Damage is the struct as the caller prepared it (Flags).
	Damage Damage
	// ConvClass is the byte at struct +0x65 and ConvPct the dword at +0x68.
	ConvClass byte
	ConvPct   int32
}

// MeleeOut is the result: the filled struct and the new defender stat 0x1f
// (valid when DefenderACSet).
type MeleeOut struct {
	Damage        Damage
	Conversion    byte
	DefenderAC    int32
	DefenderACSet bool
}

func scaleDiv(x int32, scale uint8) int32 { return (x * int32(scale)) / 128 }

// accumRoll is helper 0x5785d0: cur + min' + Roll(max'-min'), floored at 0,
// where min' and max' include the percent bonus. Nothing is added when max <= 0.
func accumRoll(r Roller, cur, minV, maxV, pct int32) int32 {
	res := cur

	if maxV > 0 {
		minP := minV + MulDiv(minV, pct, 100)
		maxP := maxV + MulDiv(maxV, pct, 100)
		res += minP

		if maxP > minP {
			res += int32(r.Roll(maxP - minP))
		}
	}

	if res < 0 {
		res = 0
	}

	return res
}

// elemRoll is helper 0x578630: stats in whole units are shifted to 8.8; nothing
// is rolled when the max stat is below 1. pctStat < 0 means none.
func elemRoll(r Roller, get func(int) int32, minStat, maxStat, pctStat int, cur int32) int32 {
	max8 := get(maxStat) << 8
	if max8 < 8 {
		if cur < 0 {
			return 0
		}

		return cur
	}

	var pct int32
	if pctStat >= 0 {
		pct = get(pctStat)
	}

	return accumRoll(r, cur, get(minStat)<<8, max8, pct)
}

// BuildAttackerDamage fills the elemental, poison, burn, leech, stun and
// conversion parts of a melee damage struct. Roll order (verified): strike,
// fire, lightning, cold, magic, (monster leeches), poison, burn, conversion.
func BuildAttackerDamage(r Roller, in MeleeIn) MeleeOut {
	d := in.Damage
	get := in.Get
	scale := in.Scale

	if scale == 0 {
		scale = 0x80
	}

	var out MeleeOut

	d.Flags |= DamageFlagComputed

	if (in.AttackerKind == 0 || in.AttackerMerc) && in.DefenderKind == 1 {
		if v := get(statAttackerDefAd); v != 0 {
			n := in.DefenderAC + v
			if n < 0 {
				n = 0
			}

			out.DefenderAC, out.DefenderACSet = n, true
		}

		if in.DefenderDemon {
			if v := get(statDemonPct); v > 0 {
				d.DamagePct += v
			}
		}

		if in.DefenderUndead {
			// helper 0x579380: +50 for a wielded weapon of item type 0x39 (blunt)
			v := get(statUndeadPct)
			if in.UndeadBlunt {
				v += 50
			}

			if v > 0 {
				d.DamagePct += v
			}
		}
	}

	if d.Flags&DamageFlagNoPhysical == 0 {
		d.Physical = in.Phys

		if in.Weapon != nil {
			// the exe passes the struct's existing physical value (a5) into the roll
			d.Physical = in.Damage.Physical
			w := *in.Weapon
			w.Get, w.UseWeapon, w.Weapon = get, true, false
			w.PctAdd, w.Flat, w.Scale = d.DamagePct, d.Physical, scale
			w.Min, w.Max = 0, 0
			w.ActiveWeapon = in.ActiveWeapon
			d.Physical = RollPhysical(r, w)
		}

		d.ApplyStrike(r, StrikeInput{
			SkipWeapon: in.SkillOverride || !in.ActiveWeapon, WeaponChance: int(in.WeaponStrike),
			CriticalChance: int(get(statStrikeCrit)), DeadlyChance: int(get(statDeadly)),
		})
	}

	sc := func(v int32) int32 {
		if scale == 0x80 {
			return v
		}

		return scaleDiv(v, scale)
	}

	d.Fire = sc(elemRoll(r, get, statFireMin, statFireMax, statFirePct, d.Fire))
	d.Lightning = sc(elemRoll(r, get, statLightMin, statLightMax, statLightPct, d.Lightning))
	d.Cold = sc(elemRoll(r, get, statColdMin, statColdMax, statColdPct, d.Cold))
	d.Magic = sc(elemRoll(r, get, statMagicMin, statMagicMax, statMagicPct, d.Magic))

	switch {
	case in.AttackerKind == 1 && !in.Special:
		// monsters: leech ranges are rolled and scaled
		if d.Flags&DamageFlagNoLifeLeech == 0 {
			d.LifeLeech += scaleDiv(elemRoll(r, get, statLifeLeech, statLeechMonMin, -1, d.LifeLeech), scale)
		}

		if d.Flags&DamageFlagNoManaLeech == 0 {
			d.ManaLeech += scaleDiv(elemRoll(r, get, statManaLeech, statManaMonMax, -1, d.ManaLeech), scale)
		}

		if d.Flags&DamageFlagNoStamLeech == 0 {
			d.StaminaLeech += scaleDiv(elemRoll(r, get, statStamLeech, statStamMonMax, -1, d.StaminaLeech), scale)
		}
	case in.AttackerKind == 0 || in.Special:
		if d.Flags&DamageFlagNoLifeLeech == 0 {
			d.LifeLeech += get(statLifeLeech)
		}

		if d.Flags&DamageFlagNoManaLeech == 0 {
			d.ManaLeech += get(statManaLeech)
		}

		if get(statIgnoreUndead) != 0 {
			d.Flags |= DamageFlagIgnoreUndead
		}

		if get(statIgnoreDemon) != 0 {
			d.Flags |= DamageFlagIgnoreDemon
		}

		if get(statIgnoreBeast) != 0 {
			d.Flags |= DamageFlagIgnoreBeast
		}
	}

	// poison: raw (not shifted) stats
	pmax := get(statPoisonMax)

	var ppct int32
	if pmax > 0 {
		ppct = get(statPoisonPct)
	}

	d.Poison = scaleDiv(accumRoll(r, d.Poison, get(statPoisonMin), pmax, ppct), scale)

	if d.Poison != 0 {
		if ov := get(statPoisonOver); ov > 0 {
			d.PoisonLen += ov
		} else {
			d.PoisonLen += get(statPoisonLen)
			if n := get(statPoisonCount); n > 1 {
				d.PoisonLen /= n
			}
		}
	}

	if d.Cold > 0 {
		d.ColdLen += scaleDiv(get(statColdLen), scale)
	}

	if d.StunLen == 0 {
		d.StunLen += scaleDiv(get(statStunLen), scale)
	}

	// burn: the exe always adds scale*burn/128 with a constant 0x13c base and consumes one step
	b := d.Burn*int32(scale) + 0x13c + int32(r.Roll(1))
	if b < 0 {
		b = 0
	}

	d.Burn += b / 128

	if d.BurnLen != 0 {
		d.BurnLen += scaleDiv(get(statBurnLen), scale)
	}

	// conversion of physical into an element
	if class := in.ConvClass; class > 0 {
		moved := MulDiv(d.Physical, in.ConvPct, 100)
		d.Physical -= moved

		if d.Physical < 0 {
			d.Physical = 0
		}

		if class == ConvertRandom {
			class = byte(r.Roll(5)) + 1
		}

		switch class {
		case 1:
			d.Fire += moved
		case 2:
			d.Lightning += moved
		case 3:
			d.Magic += moved
		case 4:
			d.Cold += moved
			if d.ColdLen < 50 {
				d.ColdLen = 50
			}
		case 5:
			d.Poison += moved / 8
			if d.PoisonLen < 50 {
				d.PoisonLen = 50
			}
		case 11:
			d.Fire += moved
			if d.BurnLen < 50 {
				d.BurnLen = 50
			}
		case 12:
			d.Cold += moved
			if d.FreezeLen < 50 {
				d.FreezeLen = 50
			}
		}
	}

	out.Damage = d
	out.Conversion = in.ConvClass

	return out
}
