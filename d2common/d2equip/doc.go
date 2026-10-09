// Package d2equip holds the pure rules of equipping and wearing items: which
// item may sit in which body location, the strength/dexterity/level/class
// requirements, the two weapon sets, which armor piece is damaged when the
// hero is hit and what a broken or unqualified item does.
//
// Sources (Game.exe 1.14b, notes in d2-re-notes/inventory-trade.md):
//
//	VERIFIED   requirement formula (INV_CheckItemRequirements 0x62ebf0), the
//	           disabling of equipped items that fail it (0x55b9c0: flag 0x4000,
//	           stat list removed, re-enabled when they pass again), the armor
//	           piece weights and chances of the durability loss (0x57b380 and
//	           0x557d90, table at 0x730148), the effect of a broken armor
//	           piece (0x55d660: flag 0x100, stat list removed).
//	UNVERIFIED is marked in the comments where it applies.
package d2equip
