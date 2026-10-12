package d2statlist

// The hand skill boxes of the character screen (wave 46). Port of SKILLDESC_ComputeWeaponDamageRange (0x4e5860),
// SKILLDESC_ComputeAttackRatingBonus (0x4e62b0) and SKILLDESC_ComputeThrownWeaponDamage (0x4e7470), all read in
// Game.exe 1.14b (VERIFIED wave 46 unless marked UNVERIFIED). Observations only.

// BoxColour is the font colour the box code stores: 0 white, 1 red, 3 blue (the exe's colour index).
type BoxColour int

// Box colours.
const (
	BoxWhite BoxColour = 0
	BoxRed   BoxColour = 1
	BoxBlue  BoxColour = 3
)

// SkillBox is the per skill input of the damage box.
type SkillBox struct {
	// SrcDam is skills.txt srcdam (0..128): the share of the weapon range the skill uses (0: none, 128: all).
	SrcDam int
	// SkillMin and SkillMax are the skill's own physical damage in whole points (the accumulators the caller hands
	// in): they go through the percent pool like the weapon part.
	SkillMin, SkillMax int
	// Extra is the last argument of the exe (param_5): whole points added after the percent to both ends.
	Extra int
	// Bit8 is the skills.txt flag word bit 8 (dwBits4 & 8); with StateBonus != 0 the percent pool grows and the
	// box turns blue. UNVERIFIED which column sets it (the finishing moves are the skills that carry it).
	Bit8       bool
	StateBonus int
	// DmgBlue and DmgRed are the dmgblue (0xe8) / dmgred (0xec) state masks of the hero (d2state.PanelFlags);
	// red wins.
	DmgBlue, DmgRed bool
	// Mastery is SKILL_GetWeaponMasteryBonus for the weapon and skill (UNVERIFIED here: 0 when not supplied).
	Mastery int
}

// DamageBox is the number pair and colour of a damage box.
type DamageBox struct {
	Min, Max int
	Colour   BoxColour
}

// SkillDamageBox is the damage of a hand skill (0x4e5860):
//
//	weapon range (stats 0x15/0x16; bare hands 0x15 + 1 and 0x16 + 2; the two handed range 0x17/0x18 when used so)
//	scaled by SrcDam/128 (SrcDam 0 drops the weapon part), floored at 1 / 2 and added to the skill's own damage,
//	pool = stat 0x19 + mastery + (state bonus of a bit 8 skill) + the weapon's strength and dexterity bonus
//	(bare hands: strength), floored at -90,
//	min = (100 + pool + item_mindamage_percent) * acc / 100 + item_normaldamage(weapon) + Extra
//	max = the same with item_maxdamage_percent
//
// Note the flat is added AFTER the percent here, before it in the hit roll (0x579120). The exe pairs the min end with
// stat 0x12 and the max end with stat 0x11, which are the ids of item_mindamage_percent / item_maxdamage_percent.
func (t *Totals) SkillDamageBox(sk SkillBox) DamageBox {
	wp := t.Weapon

	// the unit stats 0x15/0x16: WeaponParts holds them with the weapon's own enhanced damage folded in, so the
	// separate 0x11/0x12 terms of the exe are already inside (they stay 0 here).
	// Bare hands: the exe reads stat 0x15 + 1 and 0x16 + 2, which is the 1 / 2 base WeaponParts starts from.
	wmin, wmax := wp.Min, wp.Max

	srcDam := sk.SrcDam
	if srcDam != 128 {
		wmin = wmin * srcDam / 128
		wmax = wmax * srcDam / 128
	}

	accMin, accMax := sk.SkillMin, sk.SkillMax

	if srcDam != 0 {
		if wmin < 1 {
			wmin = 1
		}

		if wmax < 2 {
			wmax = 2
		}

		accMin += wmin
		accMax += wmax
	}

	pool := t.DamagePct + sk.Mastery
	out := DamageBox{}

	if sk.Bit8 && sk.StateBonus != 0 {
		pool += sk.StateBonus
		out.Colour = BoxBlue
	}

	if wp.Armed {
		pool += wp.BonusPct
	} else {
		pool += t.Str
	}

	if sk.DmgBlue {
		out.Colour = BoxBlue
	}

	if sk.DmgRed {
		out.Colour = BoxRed
	}

	if pool < -90 {
		pool = -90
	}

	flat := wp.Flat
	if !wp.Armed {
		flat = 0
	}

	out.Min = (100+pool)*accMin/100 + flat + sk.Extra
	out.Max = (100+pool)*accMax/100 + flat + sk.Extra

	return out
}

// ThrownBox is the damage range the character screen shows for a thrown weapon skill (SKILLDESC_ComputeThrownWeaponDamage
// 0x4e7470). Only a throwing weapon (ITEM_IsThrowableType) qualifies; a weapon of type 0x26 takes another branch
// (the missile record's damage, not ported). The path differs from the melee box: the base is the thrown range
// (stats 0x9f/0xa0), item_normaldamage is NOT added, there is no SrcDam scaling, and the pool is
// stat 0x19 + the weapon's strength and dexterity bonus, floored at -90, min scaled by (stat 0x12 + pool) and max by
// (stat 0x11 + pool): v + v * (ed + pool) / 100. The weapon mastery bonus is added only while state 0x4e is on, and
// the max end is raised to the min end. ok is false when the unit holds no throwing weapon.
func (t *Totals) ThrownBox(mastery int) (min, max int, ok bool) {
	wp := t.Weapon
	if !wp.Armed || !wp.Throwing {
		return 0, 0, false
	}

	pool := wp.BonusPct + t.DamagePct
	if pool < -89 {
		pool = -90
	}

	// ThrowMin/ThrowMax carry the weapon's enhanced damage already (see WeaponParts), so the 0x11/0x12 terms are 0.
	min = pool*wp.ThrowMin/100 + wp.ThrowMin
	max = pool*wp.ThrowMax/100 + wp.ThrowMax
	min += wp.ThrowMin * mastery / 100
	max += wp.ThrowMax * mastery / 100

	if max < min {
		max = min
	}

	return min, max, true
}

// SkillAttackRating is the attack rating of a hand skill (0x4e62b0):
//
//	ar = flat attack rating (COMBAT_GetPlayerAttackRating, 0x13 plus dexterity term and class factor)
//	ar = ar * (mastery(state 0x4e only) + stat 0x77 + SKILL_GetToHitBonus(skill, level)) / 100 + ar
//
// A weapon of item type 0x26 shows 0 and no colour. The colour is blue for the attblue (0xf0) state mask, red for
// attred (0xf4), red wins.
func SkillAttackRating(flatAR, toHitPct, mastery, skillToHit int, typ26, attBlue, attRed bool) (int, BoxColour) {
	if typ26 {
		return 0, BoxWhite
	}

	ar := flatAR*(mastery+toHitPct+skillToHit)/100 + flatAR

	switch {
	case attRed:
		return ar, BoxRed
	case attBlue:
		return ar, BoxBlue
	}

	return ar, BoxWhite
}
