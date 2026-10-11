package d2state

// Curse reduction for resist stats, SKILL_GetCurseDurationReduction (0x5c1300), VERIFIED. Every stat a curse
// or an enemy aura puts on a monster (SKILL_ApplyCurseToTarget 0x5c1380, the aura applier 0x5cd240) passes
// through it. For the six resist stats (ids 0x24, 0x25, 0x27, 0x29, 0x2b, 0x2d: physical, magic, fire,
// lightning, cold, poison) a value of zero or below, aimed at a unit that is neither a player nor a
// hireling and whose base value of that stat is 100 or more (an immune monster), is divided by five
// (truncating towards zero). Everything else is unchanged.

// ImmuneResist is the base resist from which the reduction applies.
const ImmuneResist = 100

// ResistStatIndex returns 0..5 (physical, magic, fire, lightning, cold, poison) for the state stat names of
// the six resist stats, and false for any other name.
func ResistStatIndex(stat string) (int, bool) {
	switch stat {
	case "damageresist":
		return 0, true
	case "magicresist":
		return 1, true
	case "fireresist":
		return 2, true
	case "lightresist":
		return 3, true
	case "coldresist":
		return 4, true
	case "poisonresist":
		return 5, true
	}

	return 0, false
}

// CurseStatReduction is the value a curse stat really gets. baseResist(i) is the target's base resist i
// (order as ResistStatIndex); playerOrHireling targets are never reduced.
func CurseStatReduction(stat string, value int, playerOrHireling bool, baseResist func(i int) int) int {
	i, ok := ResistStatIndex(stat)
	if !ok || value > 0 || playerOrHireling || baseResist == nil {
		return value
	}

	if baseResist(i) >= ImmuneResist {
		return value / 5
	}

	return value
}

// ReduceCurseMods returns mods with the reduction applied (a copy; mods itself is not changed).
func ReduceCurseMods(mods []StatMod, playerOrHireling bool, baseResist func(i int) int) []StatMod {
	out := make([]StatMod, len(mods))

	for i, m := range mods {
		out[i] = StatMod{Stat: m.Stat, Value: CurseStatReduction(m.Stat, m.Value, playerOrHireling, baseResist)}
	}

	return out
}

// CurseNullified reports whether the reduction cut a resist penalty to nothing: SKILL_ApplyCurseToTarget
// (0x5c1380) then does not apply the curse at all (a Lower Resist of -4 on an immune monster is -4/5 = 0).
// orig and reduced are the mods before and after ReduceCurseMods.
func CurseNullified(orig, reduced []StatMod) bool {
	for i := range orig {
		if _, ok := ResistStatIndex(orig[i].Stat); ok && i < len(reduced) && orig[i].Value != 0 && reduced[i].Value == 0 {
			return true
		}
	}

	return false
}

// WalkModeIndex is the animation mode id of walking (monstats2 mWL): SKILL_IsCurseableUnit (0x5c11f0) refuses
// a monster that has no such animation (hydras, sentries, traps, tentacles, nests, maggot queens, ...).
const WalkModeIndex = 2

// CurseableMonster is the monster part of SKILL_IsCurseableUnit: the class needs a walk animation. (The exe
// also refuses a monster with data flag16 bit 0x20 set, whose meaning is not decoded, and needs the
// target hostile to the caster.)
func CurseableMonster(hasWalkMode bool) bool { return hasWalkMode }
