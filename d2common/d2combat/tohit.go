package d2combat

// Defense returns a unit's total defense. Verified in COMBAT_GetDefense
// (0x6225a0, read from the disassembly):
//
//	base = armorClass + dex/4          (truncating toward zero)
//	def  = base + base*bonusPct/100    (truncating toward zero) when base > 0
//	def  = base - base*bonusPct/100    when base <= 0 (a bonus shrinks a
//	                                   negative defense toward zero)
//	def += def*overridePct/100         (stat 0xb6 armor_override_percent,
//	                                   MulDiv on the running total; 0 skips)
//
// bonusPct is the sum of the percent stats 0xab (skill_armor_percent) and 0x10
// (item_armor_percent). The game also adds the Holy Shield calc (skills.txt
// calc1 of the skill whose aurastate is state 0x65) into bonusPct when the
// unit has that state and a hand item that passes 0x63d9d0; callers pass it in
// bonusPct. Which item that test wants is not decoded (U).
func Defense(armorClass, dex, bonusPct int) int {
	return DefenseOverride(armorClass, dex, bonusPct, 0)
}

// DefenseOverride is Defense with the stat 0xb6 term (see Defense).
func DefenseOverride(armorClass, dex, bonusPct, overridePct int) int {
	base := armorClass + dex/4

	def := base + base*bonusPct/100
	if base <= 0 {
		def = base - base*bonusPct/100
	}

	if overridePct != 0 {
		def += def * overridePct / 100
	}

	return def
}

// PlayerAttackRating is the base attack rating of a player:
// toHit + (dex-7)*5 + classBase (charstats +0x3C). From the notes
// (GetPlayerAttackRating, 0x622710); not re-checked in the binary.
func PlayerAttackRating(toHit, dex, classBase int) int {
	return toHit + (dex-7)*5 + classBase
}

// MonsterAttackRating is the attack rating of a monster or mercenary. Verified
// in COMBAT_RollToHit (0x57b9c0): stat 0x13 + bonus + dex*5, where bonus is
// the skill or missile ToHit passed by the caller.
func MonsterAttackRating(toHit, bonus, dex int) int {
	return toHit + bonus + dex*5
}

// ToHitInput holds the inputs of the to-hit roll.
type ToHitInput struct {
	AttackRating  int // attacker AR
	Defense       int // defender defense, already including the vs-missile/vs-melee armor stat (0x20/0x21)
	AttackerLevel int
	DefenderLevel int
	// AttackRatingPct is the percent AR bonus (stat 0x77). The game adds
	// MulDiv(AR, pct, 100); the operands of that call are hidden in the
	// decompile, so this is inferred.
	AttackRatingPct int
}

// ToHitChance returns the percent chance to hit, in [5, 95]. Verified in
// COMBAT_RollToHit (0x57b9c0):
//
//	AR += AR*pct/100
//	if DEF < 0 { AR -= DEF; DEF = 0 }
//	if AR  < 0 { DEF -= AR; AR = 0 }
//	chance = 100 if AR+DEF == 0 else AR*100/(AR+DEF)
//	chance = chance*Alvl*2/(Alvl+Dlvl)
//	chance < 6 -> 5, chance > 94 -> 95
//
// If both levels are 0 the game divides by zero; here the level factor is
// skipped instead (inferred, never reachable in the game).
func ToHitChance(in ToHitInput) int {
	ar, def := in.AttackRating, in.Defense
	ar += ar * in.AttackRatingPct / 100

	if def < 0 {
		ar -= def
		def = 0
	}

	if ar < 0 {
		def -= ar
		ar = 0
	}

	chance := 100
	if ar+def != 0 {
		chance = ar * 100 / (ar + def)
	}

	if sum := in.AttackerLevel + in.DefenderLevel; sum != 0 {
		chance = chance * in.AttackerLevel * 2 / sum
	}

	switch {
	case chance < 6:
		return 5
	case chance > 94:
		return 95
	}

	return chance
}

// RollToHit performs the to-hit roll. It always consumes exactly one step of
// r, even when the chance is clamped. NOTE (binary): the notes marked the
// owner of the seed unverified; the decompile shows param_1 (the attacker, the
// unit whose level is Alvl) supplying the seed, so r should be the attacker's
// generator. Hit iff roll < chance.
func RollToHit(r Roller, in ToHitInput) (hit bool, chance, roll int) {
	chance = ToHitChance(in)
	hit, roll = roll100(r, chance)

	return hit, chance, roll
}

// DefenseWithFinalPct is the same as DefenseOverride (kept for the skill
// oracle audit tests): finalPct is unit stat 0xb6.
func DefenseWithFinalPct(armorClass, dex, bonusPct, finalPct int) int {
	return DefenseOverride(armorClass, dex, bonusPct, finalPct)
}
