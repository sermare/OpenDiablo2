// Package d2cube implements the Horadric Cube: CubeMain.txt recipe parsing and
// matching, the generation of the recipe results, and Runes.txt runeword
// validation for socketing.
//
// The package is pure and table driven. It reads the raw game tables (CubeMain,
// Runes, ItemTypes, weapons, armor, misc, MagicPrefix/Suffix) from byte slices
// handed in by the caller, works on its own plain Item value, and rolls with a
// d2drop.RNG, so everything is unit testable and reproducible. It never
// contains game data: the compiled table (CubeMain.txt of the player's install)
// is authoritative and every recipe comes from it.
//
// Behaviour taken from the table semantics (d2mods.info CubeMain article) and
// not from the binary is marked UNVERIFIED where it matters (the Ghidra
// bridge was not available when this was written):
//
//   - the op/param/value condition: only op 28 (used by the quest recipes) is
//     present in the 1.14b table and is treated as "always true"; other ops
//     make a recipe unusable (they are not in the table);
//   - lvl/plvl/ilvl of an output are additive: lvl, plus plvl percent of the
//     player level, plus ilvl percent of the main input's item level (crafted
//     items are known to be 50% character level and 50% item level);
//   - pre=N / suf=N are 0-based row numbers of MagicPrefix.txt/MagicSuffix.txt
//     (checked against the table: Prismatic is pre=331);
//   - the cube must contain exactly the recipe's inputs and nothing else.
package d2cube
