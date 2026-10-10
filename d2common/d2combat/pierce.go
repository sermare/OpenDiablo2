package d2combat

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// Missile pierce charges, verified against Game.exe in the emulator (golden
// pierce_golden.json). Observations only.

// StatPierceIdx is the missile stat holding the remaining pierce charges.
const StatPierceIdx = 0x148

// PierceMaxCharges is the number of attempts of 0x59d4e0.
const PierceMaxCharges = 4

// RollPierceCharges is 0x59d4e0: for a missile unit (type 3) with the Pierce
// flag whose owner is a player or a monster, chance = skill_pierce (0xa6) +
// item_pierce (0x9c); when non-zero up to four attempts are made, each adding a
// charge while (value % 100) < chance. The attempts do NOT use the owner's
// generator: a private seed is built from the OWNER's value of stat 0x148 (low
// word; nothing ever sets it, so 0) and the usual high word 0x29a. The result
// is a pure function of the chance: the attempts compare the constant sequence
// 66, 70, ... against it, so a chance of 66 or less never gives a charge. The
// returned count is stored in the MISSILE's stat 0x148; when nothing is rolled
// the missile's stat is left alone (cur).
func RollPierceCharges(chance int32, ownerKind, missileKind int, flag bool, ownerStat, cur int32) int32 {
	if !flag || missileKind != 3 || (ownerKind != 0 && ownerKind != 1) || chance == 0 {
		return cur
	}

	s := &d2rand.Seed{Lo: uint32(ownerStat), Hi: 0x29a}
	n := int32(0)

	for n < PierceMaxCharges {
		if int32(s.Step()%100) >= chance {
			break
		}

		n++
	}

	return n
}

// SpendPierce is 0x5ab550: B is 3 normally (may destroy, runs the hit
// function) and 2 when a charge was spent (the stat is decremented).
func SpendPierce(flag bool, stat int32) (b int, newStat int32) {
	if flag && stat != 0 {
		return 2, stat - 1
	}

	return 3, stat
}
