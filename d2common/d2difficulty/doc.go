// Package d2difficulty holds the rules that depend on the game difficulty
// (Normal, Nightmare, Hell): the DifficultyLevels.txt table, the monster stat
// scaling, resistance and experience penalties, drop rules and the rules that
// unlock the next difficulty. It is pure: no game data is needed.
//
// Evidence. VERIFIED means read from the 1.14b Game.exe or from the extracted
// tables of this install; UNVERIFIED is a reading of column names or of the
// community file format and is said so on the item.
//
//	VERIFIED   MONTBL_GetLevelScaledStats (0x6551e0): the difficulty index is
//	           clamped to 0..2, the monlvl.txt row to count-1; unless the class
//	           has noRatio each stat is MATH_MulDiv(monlvl value, monstats
//	           ratio, 100) which is a plain truncating (a*b)/100 (0x47f2c0);
//	           noRatio classes use the monstats values as they are.
//	VERIFIED   DifficultyLevels.txt of patch_d2.mpq (the 1.14b table, the rows
//	           in Default) and of d2data.mpq (classic, rows in Classic).
//	VERIFIED   COMBAT_GetEffectiveResist (0x579b10): ResistPenalty is added
//	           after pierce, except for damage/magic resist; classic mode uses
//	           -20/-50 (d2data table agrees).
//	VERIFIED   socket caps by difficulty 3/4/6 (ITEMGEN_RollSockets) and the
//	           treasure class upgrade only for monsters in the expansion above
//	           Normal (ITEMGEN_DropFromTreasureClass 0x558d80).
//	UNVERIFIED the progression byte of the .d2s header (status bits 8..12):
//	           5 per difficulty in the expansion (4 in classic); the real
//	           sample has 13 for a hero who finished Normal and Nightmare and
//	           three Hell acts, which agrees.
package d2difficulty
