package d2combat

// ManaCost computes a skill's mana cost in 8.8 fixed point.
//
// Verified in SKILL_PayManaCost (0x569d70):
//
//	cost = (mana + lvlmana*max(level-1, 0)) << (manashift & 0x1f)
//	cost = max(cost, minmana*256)
//
// A skill with mana == 0 and lvlmana == 0 is free (cost 0, minmana is not
// applied). The arithmetic is 32 bit, as in the game. The binary matches the
// notes.
func ManaCost(mana, lvlmana, minmana, manashift int16, level int) int {
	if mana == 0 && lvlmana == 0 {
		return 0
	}

	lv := int32(level) - 1
	if lv < 0 {
		lv = 0
	}

	cost := (int32(lvlmana)*lv + int32(mana)) << (uint(manashift) & 0x1f)

	if floor := int32(minmana) * 256; cost < floor {
		cost = floor
	}

	return int(cost)
}

// PayMana deducts cost (8.8 fixed point) from current mana (8.8) and reports
// whether the skill could be paid for. On failure mana is unchanged. Verified:
// the game pays when cost <= current.
func PayMana(current, cost int) (remaining int, ok bool) {
	if cost <= current {
		return current - cost, true
	}

	return current, false
}
