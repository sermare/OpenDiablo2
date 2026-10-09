package d2skill

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// tiers5 is SKILL_SumLevelTierBonus (0x645f20, verified): the bonus of the
// five per-level columns. Levels 2..8 add l[0] each, 9..16 l[1], 17..22 l[2],
// 23..28 l[3], above 28 l[4].
func tiers5(l [5]int, lvl int) int {
	switch {
	case lvl < 1:
		return 0
	case lvl <= 8:
		return (lvl - 1) * l[0]
	case lvl <= 16:
		return 7*l[0] + (lvl-8)*l[1]
	case lvl <= 22:
		return 7*l[0] + 8*l[1] + (lvl-16)*l[2]
	case lvl <= 28:
		return 7*l[0] + 8*l[1] + 6*l[2] + (lvl-22)*l[3]
	}

	return 7*l[0] + 8*l[1] + 6*l[2] + 6*l[3] + (lvl-28)*l[4]
}

// tiers3 is the ELevLen version (0x6462f0, verified).
func tiers3(l [3]int, lvl int) int {
	switch {
	case lvl < 1:
		return 0
	case lvl <= 8:
		return (lvl - 1) * l[0]
	case lvl <= 16:
		return 7*l[0] + (lvl-8)*l[1]
	}

	return 7*l[0] + 8*l[1] + (lvl-16)*l[2]
}

// mulDiv is (a*b)/c with a 64 bit intermediate.
func mulDiv(a, b, c int) int { return int(int64(a) * int64(b) / int64(c)) }

// ElemMin is SKILL_GetElementalMinDamage (0x646100, mastery 0), 8.8 fixed
// point after HitShift: (EMin + tiers) << HitShift, plus the synergy percent
// of the shifted base when the base is above 1.0 or EMinLev1 is set.
func (d *DamageSpec) ElemMin(e *Env, lvl int) int32 {
	if lvl < 1 {
		return 0
	}

	base := (d.EMin + tiers5(d.EMinLev, lvl)) << uint(d.HitShift)
	if !d.EDmgSymPer.Empty() && (base > 0x100 || d.EMinLev[0] != 0) {
		base += mulDiv(base, e.eval(d.EDmgSymPer), 100)
	}

	return int32(base)
}

// ElemMax is SKILL_GetElementalMaxDamage (0x646200): the same with EMax; the
// synergy applies when the base is above 1.0.
func (d *DamageSpec) ElemMax(e *Env, lvl int) int32 {
	if lvl < 1 {
		return 0
	}

	base := (d.EMax + tiers5(d.EMaxLev, lvl)) << uint(d.HitShift)
	if !d.EDmgSymPer.Empty() && base > 0x100 {
		base += mulDiv(base, e.eval(d.EDmgSymPer), 100)
	}

	return int32(base)
}

// ElemLen is SKILL_GetElementalLength (0x6462f0): ELen + tiers, plus the
// synergy percent; frames, not shifted.
func (d *DamageSpec) ElemLen(e *Env, lvl int) int {
	if lvl < 1 {
		return 0
	}

	n := d.ELen + tiers3(d.ELevLen, lvl)
	if !d.ELenSymPer.Empty() {
		n += mulDiv(n, e.eval(d.ELenSymPer), 100)
	}

	return n
}

// PhysMin is SKILL_GetPhysicalMinDamage (0x648f90): the weapon part
// (SrcDam*weapon/128, when includeWeapon), plus MinDam and the level tiers,
// plus the DmgSymPerCalc percent, shifted by HitShift. The Kick special case
// is handled by the caller.
func (d *DamageSpec) PhysMin(e *Env, lvl, weaponMin int, includeWeapon bool) int32 {
	return d.phys(e, lvl, weaponMin, includeWeapon, d.MinDam, d.MinLevDam)
}

// PhysMax is the maximum counterpart (0x6490d0).
func (d *DamageSpec) PhysMax(e *Env, lvl, weaponMax int, includeWeapon bool) int32 {
	return d.phys(e, lvl, weaponMax, includeWeapon, d.MaxDam, d.MaxLevDam)
}

func (d *DamageSpec) phys(e *Env, lvl, weapon int, includeWeapon bool, base int, lev [5]int) int32 {
	if lvl < 1 {
		return 0
	}

	v := base + tiers5(lev, lvl)
	if includeWeapon && d.SrcDam > 0 {
		v += d.SrcDam * weapon / 128
	}

	if !d.DmgSymPer.Empty() {
		v += mulDiv(v, e.eval(d.DmgSymPer), 100)
	}

	return int32(v << uint(d.HitShift))
}

// Descriptor builds the missile damage descriptor for a skill at a level
// (MISSILE_BuildDamageDescriptor, 0x64ca60, skill-linked branch): physical
// from the skill with the weapon, the elemental triple from EType. mastery is
// the percent added to the elemental damage (stats 0x149..0x14c); the exact
// operand of the game's MulDiv is UNVERIFIED.
func (s *Skill) Descriptor(e *Env, lvl, weaponMin, weaponMax, mastery int) d2missile.DamageDesc {
	d := d2missile.DamageDesc{
		PhysMin:  s.PhysMin(e, lvl, weaponMin, true),
		PhysMax:  s.PhysMax(e, lvl, weaponMax, true),
		ToHit:    s.ToHitBonus(e, lvl),
		HitClass: s.HitClass,
	}

	emin, emax, elen := s.ElemMin(e, lvl), s.ElemMax(e, lvl), int32(s.ElemLen(e, lvl))
	if mastery != 0 {
		emin += int32(mulDiv(int(emin), mastery, 100))
		emax += int32(mulDiv(int(emax), mastery, 100))
	}

	el := d2missile.Elem{Min: emin, Max: emax, Len: elen}

	switch s.EType {
	case "fire":
		d.Fire = el
	case "ltng":
		d.Lightning = el
	case "mag":
		d.Magic = el
	case "cold":
		d.Cold = el
	case "pois":
		d.Poison = el
	case "burn":
		d.Burn = el
	case "frze":
		d.FreezeLen = elen
	case "stun":
		d.StunLen = elen
	}

	return d
}
