// Package d2boss holds the trigger logic of the five boss encounters of the
// game: Andariel's quest kill is handled by d2quest alone, the rest is here:
//
//   - Duriel's Tomb (act 2): the Horadric Staff is placed in the orifice of
//     Tal Rasha's tomb, the portal to Duriel's Lair opens, Duriel is met in
//     the lair and Tyrael appears when he is dead;
//   - Mephisto (act 3): asleep in the Durance of Hate level 3 until the hero
//     comes close, a red portal to the Pandemonium Fortress appears after his
//     death;
//   - Diablo (act 4): five seals in the Chaos Sanctuary, three of which
//     spawn the seal bosses (Grand Vizier of Chaos, Lord De Seis, Infector of
//     Souls); Diablo arrives after all seals are open and the three seal
//     bosses are dead;
//   - Baal (act 5): the Throne of Destruction sends five waves, then Baal
//     walks to the Worldstone Chamber portal.
//
// The package is pure: a Manager receives the facts of the game (level
// entered, object operated, monster killed, a frame passed) and returns the
// Actions the engine has to carry out (spawn a monster or object, speak a
// message, open a portal). Every decision is logged through Manager.Log so
// that an autotest can check it.
//
// Evidence (Game.exe 1.14b, read-only Ghidra; d2-re-notes/boss-encounters.md):
// VERIFIED are the ids (levels, objects, super uniques, monster classes), the
// Terror's End bookkeeping (five seal flags, three seal bosses, the one-shot
// Diablo arrival, Diablo's kill), the Baal throne wave cycle (see
// d2monster.thinkBaalThrone) and the wave groups (SuperUniques.txt "Baal
// Subject 1..5"). Marked UNVERIFIED are the delays, the group of the two plain
// seals, the positions and the objects used for the portals; each says so in
// its comment.
package d2boss
