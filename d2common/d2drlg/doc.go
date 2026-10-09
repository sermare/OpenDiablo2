// Package d2drlg is a pure-Go port of Diablo II's dungeon level generator
// (DRLG) as documented in the reverse-engineering notes (drlg.md, drlg2.md).
//
// It has no MPQ dependency: every table is supplied through the small
// interfaces in this package (Levels, LvlMaze, LvlPrest, LvlTypes, LvlSub),
// loaded from the extracted txt files and, for LvlPrest, from the compiled
// lvlprest.bin, which is authoritative where the txt is stripped.
//
// Status of exactness: the random number generator and seed hierarchy
// (d2rand) and the Act 1 maze algorithm and world layout were read from the
// real binary. There is no oracle (the original game cannot run here), so
// bit-exact equality with the real game's maps is NOT proven. Parts the
// notes mark as unread are left as TODO interfaces and are never invented.
package d2drlg
