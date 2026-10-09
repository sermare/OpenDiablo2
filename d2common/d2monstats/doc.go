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
//	                   min + uniform roll over [0, max-min+1), plus the player
//	                   count bonus (MulDiv(hp, pct, 100), added before the
//	                   cap), capped 0x7fffff (stored <<8). XP gets the same
//	                   kind of bonus.
//	VERIFIED 0x00571760 player-count scaling (see scaling.go): +0,0,50..350%
//	                   for 1..8 players (HP and XP), Align != 0 classes none.
//	VERIFIED 0x00571af0 level: the monstats Level of the difficulty is the
//	                   default. Only Nightmare/Hell with a known area and a
//	                   class without noRatio and without the excluded flag
//	                   take the area MonLvl (LEVEL_GetMonsterLevel 0x0061dc00).
//	                   The excluded flag is bit 6 of the monstats flag dword
//	                   (mask 0x006cf268); by table order that is primeevil,
//	                   NOT boss (UNVERIFIED column, see LevelExclusion).
//	VERIFIED 0x0063ff30 classic-mode adjustment (Options.Classic): HP x1/2,
//	                   AC x10/12, XP x10/17 N / x10/26 H, level = monstats
//	                   Level + 25*diff; skipped for Align 1 classes.
//
// See ~/git/d2-re-notes/verify-monster-hero.md.
package d2monstats
