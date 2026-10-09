package d2combat

// ReduceComponent is the per-type step of COMBAT_ApplyResistToDamageType
// (0x579c90, VERIFIED in verify-resist.md and verify-hit-resolution.md) for one
// damage component, in any consistent unit:
//
//	dmg < 1                  -> 0
//	unresistable (ignore)    -> a positive effective resist makes it 0, no flat reduction
//	otherwise                -> dmg -= flat
//	dmg > 0 and res != 0     -> res = min(res, 100); dmg = dmg*(100-res)/100
//
// The result is NOT floored: a flat reduction larger than the damage leaves a
// negative component (the caller floors the total and subtracts life only when
// it is positive). The effective resist comes from EffectiveResist, so a
// defender that is not a non-mercenary monster (players and mercenaries) gets the
// cap and the difficulty penalty. Absorb is not part of this step (see Absorb).
func ReduceComponent(dmg, flat int, in ResistInput, unresistable bool) int {
	if dmg < 1 {
		return 0
	}

	res := EffectiveResist(in)

	if unresistable {
		if res > 0 {
			return 0
		}
	} else {
		dmg -= flat
	}

	if dmg > 0 && res != 0 {
		if res > ImmuneResist {
			res = ImmuneResist
		}

		dmg = dmg * (100 - res) / 100
	}

	return dmg
}
