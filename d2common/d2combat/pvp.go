package d2combat

// Player versus player. The damage scale is the pre-step 0x579d70 of
// COMBAT_ApplyResistsToDamageStruct (see CombatantScalePercent): when the
// defender is a player and the attacker is player controlled the damage is
// scaled to 17 percent (VERIFIED, the binary spot-check of the skills-combat
// notes); a plain monster attacker keeps 100.

// PvPPercent is the scale between two heroes.
func PvPPercent() int {
	return CombatantScalePercent(false, true, true, true, true)
}

// PvPDamage scales a hero's raw hit for a hero defender.
func PvPDamage(raw int) int {
	if raw <= 0 {
		return 0
	}

	return int(ScaleByPercent(int32(raw), PvPPercent()))
}

// PvPReceive is what a defending hero loses from a scaled hit: the flat damage
// reduction first, then the physical resistance percent (the order of the two
// is UNVERIFIED, the same as for monster hits). It never heals.
func PvPReceive(scaled, physResist, reduce int) int {
	d := ApplyResist(scaled-reduce, physResist)
	if d < 0 {
		return 0
	}

	return d
}

// PvPKillGivesEar reports whether killing a hero leaves the killer an ear.
// UNVERIFIED (public game knowledge, not the binary): ears exist only for
// hardcore kills; a softcore kill has no special drop and the dead hero
// handles its own corpse and items as for any death.
func PvPKillGivesEar(victimHardcore bool) bool { return victimHardcore }
