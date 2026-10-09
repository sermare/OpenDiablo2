// Package d2monsters runs hostile monsters inside the engine: it builds
// d2monster brains from monstats, feeds them the map and the players through
// the d2monster.World interface, resolves attacks with d2combat, applies
// hero damage, and rolls loot with d2drop on death.
//
// The AI is the pure d2common/d2monster package; this package is the glue and
// therefore the place where engine-level simplifications are listed:
//
//   - Units occupy one collision cell each (monster flag 0x100, player 0x80,
//     corpse 0x8000) and block each other through d2path.MaskUnits. Which mask
//     the original uses for unit blocking is not recorded (UNVERIFIED).
//   - Monster attacks resolve at the animation's halfway frame. Melee connects
//     within the reach of the mode (7 subtiles); ranged attacks (a missile
//     column for the mode) launch a Shot through the Launcher hook: the
//     built-in launcher flies a straight bolt that is stopped by walls and by
//     the first hero cell it enters. missiles.txt behaviour beyond velocity is
//     not simulated; feat/skill-pipeline can replace the launcher.
//   - Not ported (notes: not read / unverified): forced AI states (flee, fear,
//     confuse, charm: the state table at 0x73a548), the Summoner, Vulture and
//     the other bosses, monster-vs-monster targeting, aidel throttling after
//     hit recovery, champion/unique modifiers (monumod).
//   - Hero defense is dexterity/4 only (equipment defense is not read).
//   - Monster level is the monstats Level column (not the area's MonLvl).
//   - Monster stats use monlvl.txt L-* columns scaled by the monstats ratios.
package d2monsters
