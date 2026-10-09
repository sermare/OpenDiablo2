package d2herostats

// Kill experience scaling, from the exe (see ~/git/d2-re-notes/verify-xp-gain.md):
// 0x0057c300 (level difference), 0x0057c490 (full per-recipient pipeline),
// 0x0057c3a0 (ExpRatio), 0x0047f2c0 (mul/div helper).

// killXPCap is the clamp the original applies to one kill's experience
// before scaling (0x7fffff). VERIFIED.
const killXPCap = 0x7fffff

// KillXPCap is killXPCap for callers that clamp before their own checks.
const KillXPCap = killXPCap

// Level difference multipliers, in 1/256, indexed by the level difference
// capped at 10. Numbers read from the exe's data tables. VERIFIED.
//
// charAbove is 0x006e2960, used when the monster level is <= the character
// level (index = character level - monster level).
// monsterAbove is 0x006e298c, used when the monster is above the character
// and the character level is below 25 (index = monster level - char level).
var (
	charAbove    = [11]int{256, 256, 256, 256, 256, 256, 207, 159, 110, 61, 13}
	monsterAbove = [11]int{256, 256, 256, 256, 256, 256, 225, 174, 92, 38, 5}
)

// CharAboveTable returns a copy of the table at 0x006e2960.
func CharAboveTable() [11]int { return charAbove }

// MonsterAboveTable returns a copy of the table at 0x006e298c.
func MonsterAboveTable() [11]int { return monsterAbove }

// mulDiv is the exe's a*b/c helper (0x0047f2c0): 0 for c == 0, otherwise a*b/c
// truncating toward zero. The original orders the operations to avoid 32-bit
// overflow; a 64-bit product gives the same value for the sizes reachable here.
func mulDiv(a, b, c int) int {
	if c == 0 {
		return 0
	}

	return int(int64(a) * int64(b) / int64(c))
}

// LevelScaleXP scales a kill's experience by the monster/character level
// difference (0x0057c300). VERIFIED rules:
//   - monster level <= character level: table charAbove[min(diff,10)];
//   - monster above and character level >= 25 (and monster level > 0):
//     xp * charLevel / monsterLevel;
//   - monster above and character level < 25: table monsterAbove[min(diff,10)].
//
// A multiplier of 256 returns xp untouched, else xp*mult/256.
func LevelScaleXP(xp, monsterLevel, charLevel int) int {
	var mult int

	switch {
	case monsterLevel <= charLevel:
		d := charLevel - monsterLevel
		if d > 10 {
			d = 10
		}

		mult = charAbove[d]
	case charLevel >= 25 && monsterLevel > 0:
		return mulDiv(xp, charLevel, monsterLevel)
	default:
		d := monsterLevel - charLevel
		if d > 10 {
			d = 10
		}

		mult = monsterAbove[d]
	}

	if mult == 256 {
		return xp
	}

	return mulDiv(xp, mult, 256)
}

// ExpRatioScale is 0x0057c3a0: xp * ratio >> shift, where ratio is the
// Experience.txt ExpRatio of the recipient's level and shift is the ratio of
// level 0 (the MaxLvl row). A non-positive xp is returned as is; a shift outside
// 1..31 returns xp. VERIFIED code, but the ratio DATA is not in the extracted
// table (UNVERIFIED values), so the engine does not apply it.
func ExpRatioScale(xp, ratio, shift int) int {
	if xp <= 0 || shift < 1 || shift > 31 {
		return xp
	}

	return int((int64(xp) * int64(ratio)) >> uint(shift))
}

// KillXP is the per-recipient pipeline of 0x0057c490 up to (not including)
// the final add: clamp to 0x7fffff, a non-positive xp counts as 1, a
// recipient at or above maxLevel gets 0, then the level difference scaling and
// the item "+% experience" bonus (stat 0x55): xp += xp*itemPct/100. The ExpRatio
// step is omitted (see ExpRatioScale). VERIFIED.
func KillXP(xp, monsterLevel, charLevel, maxLevel, itemPct int) int {
	if xp > killXPCap {
		xp = killXPCap
	} else if xp <= 0 {
		return 1
	}

	if charLevel >= maxLevel {
		return 0
	}

	xp = LevelScaleXP(xp, monsterLevel, charLevel)
	if itemPct != 0 {
		xp += mulDiv(xp, itemPct, 100)
	}

	return xp
}

// MercKillShare is what a merc is credited from a kill that it did not make
// itself (0x0057c990): xp*86/256 (VERIFIED); then its experience stat grows by
// twice that (0x0057c860, see the merc code). A kill by the merc itself is full.
func MercKillShare(xp int, killedByMerc bool) int {
	if killedByMerc {
		return xp
	}

	return xp * 0x56 >> 8
}
