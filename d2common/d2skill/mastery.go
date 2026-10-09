package d2skill

// Weapon masteries: the stats of a true passive with a passiveitype are stored
// by the exe with the item type as the stat param (0x648130, verified), and the
// combat code reads them keyed by the equipped weapon.
//
// VERIFIED in Game.exe (notes: weapon-masteries.md, verify-mastery-formulas.md):
//
// Lookup SKILL_Func_646bc0 (unit, weapon, skill, mode): mode 0 / 1 / 2 reads
// stat 0x156 / 0x157 / 0x158 (passive_mastery_melee_th / _dmg / _crit), every
// (param, value) entry on the unit, and returns the LARGEST value among the
// entries whose param is an item type the weapon IS (ITEM_IsOfType: the type or
// an ancestor). The helper 0x646a90 runs first. When the weapon is throwable
// and the skill (the argument, else the unit's current skill) has an item type
// A1 that inherits from item type 0x30 (thro) and a record byte +0x14 equal to
// 2 (the range column "rng"; the byte meaning is inferred from the layout:
// every thro/jave skill is range rng, no h2h skill is), it reads ONLY the
// throw family 0x159..0x15b and returns that, even when 0. Otherwise only the
// melee family.
//
// Use of the value:
//   - to-hit (COMBAT_RollToHit 0x57ba63, mode 0): a PERCENT of the attacker's
//     attack rating, summed with stat 0x77 (item_tohit_percent) and the skill's
//     to-hit bonus percent; AR += AR * sum / 100. Not applied on the missile
//     hit path (0x5abb74 passes the skip flag).
//   - damage (COMBAT_RollPhysicalDamage 0x5792ab and MISSILE_BuildDamageDescriptor
//     0x64cd32, mode 1): added into the same additive percent pool as
//     damagepercent (ED) and the strength / dexterity bonus; min and max are
//     scaled once by (100 + pool) / 100 (pool clamped at -90). Missiles do it
//     only when the missile's SrcDam byte is non-zero.
//   - crit (COMBAT_BuildAttackerDamage 0x5795d7, mode 2; missiles: 0x64ba70
//     mode 2): a SEPARATE roll, rand(100) < value, besides the
//     passive_critical_strike (0x151) and item_deadlystrike (0x8d) rolls; any
//     success doubles physical damage. Melee order: mastery, critical, deadly;
//     missile order: critical, deadly, mastery. Skipped when the caller's gate
//     argument is non-zero (0 at COMBAT_FinalizeDamageStruct).
//
// NOT wired: the missile-side crit flag (descriptor flag 0x2; its consumer is
// not traced), so only the melee crit is modelled.

// MasteryKind selects which mastery value is read.
type MasteryKind int

// Mastery kinds, equal to the mode argument of 0x646bc0.
const (
	MasteryToHit MasteryKind = iota
	MasteryDamage
	MasteryCrit
)

// MasteryStatNames are the ItemStatCost names of a kind: the melee family
// (0x156..0x158) and the throw family (0x159..0x15b).
func MasteryStatNames(k MasteryKind) [2]string {
	switch k {
	case MasteryToHit:
		return [2]string{"passive_mastery_melee_th", "passive_mastery_throw_th"}
	case MasteryDamage:
		return [2]string{"passive_mastery_melee_dmg", "passive_mastery_throw_dmg"}
	case MasteryCrit:
		return [2]string{"passive_mastery_melee_crit", "passive_mastery_throw_crit"}
	}

	return [2]string{}
}

// MasteryValue is the exe's lookup over a unit's weapon-keyed stats: the
// largest value among the mods of the kind (the throw family when thrown, else
// the melee family) whose param the weapon is of (isOfType). Zero when nothing
// matches (also without a weapon).
func MasteryValue(mods []TruePassiveMod, k MasteryKind, thrown bool, isOfType func(itemType string) bool) int {
	names := MasteryStatNames(k)
	name := names[0]

	if thrown {
		name = names[1]
	}

	best := 0

	for _, m := range mods {
		if m.Param == "" || m.Stat != name || m.Value <= best {
			continue
		}

		if isOfType != nil && isOfType(m.Param) {
			best = m.Value
		}
	}

	return best
}

// SkillThrows is the skill half of the throw family test of 0x646a90: the
// skill's itypea1 inherits from thro and its range is rng. typeIs reports
// whether item type have inherits from want. The weapon half (a throwable
// weapon) is the caller's.
func SkillThrows(sk *Skill, typeIs func(have, want string) bool) bool {
	return sk != nil && sk.Range == "rng" && sk.IType1 != "" && typeIs != nil && typeIs(sk.IType1, "thro")
}

// TypeIs reports whether item type have is want or inherits from it, walking
// the Equiv1/Equiv2 parents (ITEM_IsOfType). equiv returns the two parents of
// a type code ("" for none).
func TypeIs(equiv func(code string) (string, string), have, want string) bool {
	seen := map[string]bool{}

	var walk func(c string) bool

	walk = func(c string) bool {
		if c == "" || seen[c] {
			return false
		}

		if c == want {
			return true
		}

		seen[c] = true
		a, b := equiv(c)

		return walk(a) || walk(b)
	}

	return walk(have)
}

// MasteryUnit is implemented by units that know their equipped weapon's type
// and skills (the hero): Mastery is the 0x646bc0 lookup for the main hand
// while performing skill sk.
type MasteryUnit interface {
	Mastery(k MasteryKind, sk *Skill) int
}

func masteryOf(u Unit, k MasteryKind, sk *Skill) int {
	if mu, ok := u.(MasteryUnit); ok {
		return mu.Mastery(k, sk)
	}

	return 0
}

// missileMastery is the damage mastery percent that MISSILE_BuildDamageDescriptor
// (0x64cd32) adds to the descriptor's percent pool: only for missiles whose
// skill has a non-zero SrcDam (weapon-based damage).
func missileMastery(u Unit, sk *Skill) int {
	if sk == nil || sk.SrcDam <= 0 {
		return 0
	}

	return masteryOf(u, MasteryDamage, sk)
}
