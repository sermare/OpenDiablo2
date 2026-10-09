package d2skill

// Weapon masteries: the stats of a true passive with a passiveitype are stored
// by the exe with the item type as the stat param (0x648130, verified), and the
// combat code reads them keyed by the equipped weapon.
//
// Verified in Game.exe (notes: weapon-masteries.md): SKILL_Func_646bc0
// (unit, weapon, skill, mode) reads stat 0x156 / 0x157 / 0x158 for mode
// 0 / 1 / 2 (passive_mastery_melee_th / _dmg / _crit), fetches every
// (param, value) entry of that stat on the unit, and returns the LARGEST value
// among the entries whose param is an item type the weapon IS (ITEM_IsOfType:
// the type or one of its ancestors). The helper 0x646a90 first tries the throw
// family 0x159 / 0x15a / 0x15b (passive_mastery_throw_*) for a throwable
// weapon used with a throwing skill. Callers: to-hit 0x57ba63 (mode 0), hand
// damage 0x5792ab (mode 1), COMBAT_BuildAttackerDamage 0x5795d7 (mode 2, the
// weapon strike chance before passive_critical_strike and item_deadlystrike).
//
// UNVERIFIED: how the mode 0 value enters the to-hit roll (flat attack rating
// added here), how the mode 1 value is combined (added to the damage percent
// here), the exact condition choosing the throw family (here: both families
// are searched; the params never overlap, so this only differs for a throwing
// weapon whose type also matches a melee mastery).

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
// largest value among the mods of the kind whose param the weapon is of
// (isOfType). Zero when nothing matches (also without a weapon).
func MasteryValue(mods []TruePassiveMod, k MasteryKind, isOfType func(itemType string) bool) int {
	names := MasteryStatNames(k)
	best := 0

	for _, m := range mods {
		if m.Param == "" || (m.Stat != names[0] && m.Stat != names[1]) || m.Value <= best {
			continue
		}

		if isOfType != nil && isOfType(m.Param) {
			best = m.Value
		}
	}

	return best
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
// and skills (the hero): Mastery is the 0x646bc0 lookup for the main hand.
type MasteryUnit interface {
	Mastery(k MasteryKind) int
}

func masteryOf(u Unit, k MasteryKind) int {
	if mu, ok := u.(MasteryUnit); ok {
		return mu.Mastery(k)
	}

	return 0
}
