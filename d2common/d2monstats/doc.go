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
//	UNVERIFIED         the expansion ("L-") monlvl columns are used when the
//	                   game is Lord of Destruction; the HP roll is uniform in
//	                   [min,max] hit points after scaling; MulDiv rounds to
//	                   nearest; boss classes also take the monstats Level.
//	UNVERIFIED         player-count scaling (FUN_0063ff30) is not modelled.
//
// The Ghidra spot check of 0x006551e0 timed out (shared instance busy), so
// the formulas above rest on the notes and on the real data only.
package d2monstats
