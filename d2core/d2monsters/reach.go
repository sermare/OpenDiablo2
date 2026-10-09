package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// missileFor returns the missiles.txt name monstats assigns to an animation
// mode (MissA1, MissA2, MissS1..S4, MissSQ), or "".
func missileFor(st *d2records.MonStatRecord, mode d2monster.Mode) string {
	switch mode {
	case d2monster.ModeAttack1:
		return st.MissileA1
	case d2monster.ModeAttack2:
		return st.MissileA2
	case d2monster.ModeSkill1:
		return st.MissileS1
	case d2monster.ModeSkill2:
		return st.MissileS2
	case d2monster.ModeSkill3:
		return st.MissileS3
	case d2monster.ModeSkill4:
		return st.MissileS4
	case d2monster.ModeCast:
		return st.MissileSQ
	}

	return ""
}

// attackIsRanged says whether an attack made in a mode is a projectile: the
// mode has a monstats missile column, or it is A1 of a class flagged
// rangedtype without a missile column (the engine cannot tell which missile,
// UNVERIFIED). Which skills fly is in skills.txt, which this package does not
// read, so skill modes without a missile column are melee-ranged here.
func attackIsRanged(st *d2records.MonStatRecord, mode d2monster.Mode) bool {
	if missileFor(st, mode) != "" {
		return true
	}

	return mode == d2monster.ModeAttack1 && st.IsRanged
}

// attackReach is the edge distance at which an attack of that kind is used:
// melee reach 7 (VERIFIED value) or the engine's ranged distance (UNVERIFIED).
func attackReach(ranged bool) int {
	if ranged {
		return rangedInRange
	}

	return meleeInRange
}
