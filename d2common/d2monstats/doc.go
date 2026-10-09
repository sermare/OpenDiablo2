// Package d2monstats computes the base stats of a freshly created monster
// (level, hit points, armor class, attack ratings, damage, experience) from
// monstats.txt and monlvl.txt, following the original executable.
//
// Evidence (see ~/git/d2-re-notes/monster-ai.md, "Creating a monster unit"
// and MONTBL_GetLevelScaledStats at 0x006551e0):
//
//	VERIFIED (notes)   level: noRatio (flags bit 2) classes use the monstats
//	                   Level column of the difficulty, others the area MonLvl.
//	VERIFIED (notes)   unless noRatio is set the value is
//	                   MulDiv(monlvl.txt[level].column, monstats percent, 100);
//	                   with noRatio the raw per-difficulty monstats columns are used.
//	VERIFIED (data)    real tables only make sense as percentages for bosses
//	                   (Andariel normal: monlvl HP[12]=40 * 2562% = 1024.8).
//	VERIFIED 0x006551e0 the "L-" columns are used when the game is Lord of
//	                   Destruction (the exe's expansion flag selects the
//	                   second column of every monlvl AC/TH/HP/DM/XP pair); the
//	                   row index is clamped to the last row (negative fails).
//	VERIFIED 0x0047f2c0 MulDiv TRUNCATES toward zero (IDIV), not nearest.
//	VERIFIED 0x00571af0 HP: min and max are scaled separately, then the HP is
//	                   min + uniform roll over [0, max-min+1), capped 0x7fffff
//	                   (stored <<8). The exe also adds a percentage bonus
//	                   (from FUN_00571760) which is not modelled.
//	VERIFIED 0x00571af0 level: the monstats Level of the difficulty is the
//	                   default. Only Nightmare/Hell with a known area and a
//	                   non-noRatio, non-boss class take the area MonLvl
//	                   (LEVEL_GetMonsterLevel 0x0061dc00; LoD uses the second
//	                   MonLvl column set). Normal always uses monstats Level.
//	                   The boss exclusion is a flag-bit test (mask 0x6cf268)
//	                   assumed to be the boss bit.
//	VERIFIED 0x0063ff30 FUN_0063ff30 is NOT player-count scaling: it is the
//	                   classic (non-LoD) Nightmare/Hell adjustment (HP x1/2,
//	                   AC x10/12, XP x10/17 N and x10/26 H, level = monstats
//	                   Level + 25*diff). Not modelled; player-count scaling
//	                   lives elsewhere and remains UNVERIFIED/not modelled.
//
// See ~/git/d2-re-notes/verify-monster-hero.md.
package d2monstats
