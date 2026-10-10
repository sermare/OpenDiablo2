package d2combat

// Port of COMBAT_RollPhysicalDamage (0x579120), verified against the real
// function in the emulator (golden wep_golden.json). Observations only.

// Stat ids read by the physical roll.
const (
	statStrength     = 0x00
	statDexterity    = 0x02
	statMaxDmgPct    = 0x11 // item_maxdamage_percent: percent applied to the max end
	statMinDmgPct    = 0x12 // item_mindamage_percent: percent applied to the min end
	statSecMinDmg    = 0x17 // used when the hand classification is 2 (second damage profile)
	statSecMaxDmg    = 0x18
	statDmgFlatBonus = 0x6f // added to min and max (whole points)
)

// WeaponRollIn is the input of RollPhysical.
type WeaponRollIn struct {
	Get func(stat int) int32 // attacker stats
	// UseWeapon is the second register argument of the exe (edx): read the damage
	// from the unit's stats instead of using Min/Max.
	UseWeapon bool
	// Weapon is the first stack argument; ActiveWeapon is what the exe finds when
	// that is absent (0x5335c0). True means "a weapon item is present".
	Weapon, ActiveWeapon bool
	// HandMode is 0x63e3e0 (2 selects stats 0x17/0x18).
	HandMode int
	// StrBonus and DexBonus are the weapon's strength/dexterity percent factors
	// (0x629a50, 0x629a80, signed 16 bit). Mastery is 0x646bc0 for the weapon.
	StrBonus, DexBonus int16
	Mastery            int32
	Min, Max           int32 // a2, a3 (8.8) used when UseWeapon is false
	PctAdd             int32 // a4: percent added to the enhanced damage
	Flat               int32 // a5: 8.8 value added after the roll
	Scale              uint8 // a6: 0x80 means unscaled
}

// RollPhysical returns the rolled physical damage in 8.8 units.
func RollPhysical(r Roller, in WeaponRollIn) int32 {
	get := in.Get
	minV, maxV := in.Min, in.Max
	weapon := in.Weapon || (in.UseWeapon && in.ActiveWeapon)

	if in.UseWeapon {
		switch {
		case !weapon:
			minV = get(statMinDmg)
			if minV < 1 {
				minV = 1
			}

			maxV = get(statMaxDmg)
			if maxV < 2 {
				maxV = 2
			}

			minV <<= 8
			maxV <<= 8
		case in.HandMode == 2:
			minV, maxV = get(statSecMinDmg)<<8, get(statSecMaxDmg)<<8
		default:
			minV, maxV = get(statMinDmg)<<8, get(statMaxDmg)<<8
		}
	}

	flat := get(statDmgFlatBonus) << 8
	minV += flat
	maxV += flat

	if minV < 1 {
		minV = 0x100
	}

	if maxV <= minV {
		maxV = minV + 0x100
	}

	pct := in.PctAdd + get(statDmgPct)

	switch {
	case weapon:
		if in.StrBonus != 0 {
			pct += int32(int64(get(statStrength)*int32(in.StrBonus)) / 100)
		}

		if in.DexBonus != 0 {
			pct += int32(int64(get(statDexterity)*int32(in.DexBonus)) / 100)
		}

		pct += in.Mastery
	case in.UseWeapon:
		pct += get(statStrength) // unarmed: strength counts as percent
	}

	if pct < -90 {
		pct = -90
	}

	lo := minV + MulDiv(minV, get(statMinDmgPct)+pct, 100)
	hi := maxV + MulDiv(maxV, get(statMaxDmgPct)+pct, 100)
	res := lo + in.Flat

	if hi > lo {
		res += int32(r.Roll(hi - lo))
	}

	if res < 0 {
		res = 0
	}

	if in.Scale != 0x80 {
		res = MulDiv(res, int32(in.Scale), 0x80)
	}

	return res
}
