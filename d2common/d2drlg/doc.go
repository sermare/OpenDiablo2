// Package d2drlg is a pure-Go port of Diablo II's dungeon level generator
// (DRLG) as documented in the reverse-engineering notes (drlg.md, drlg2.md).
//
// It has no MPQ dependency: every table is supplied through the small
// interfaces in this package (Levels, LvlMaze, LvlPrest, LvlTypes, LvlSub),
// loaded from the extracted txt files and, for LvlPrest, from the compiled
// lvlprest.bin, which is authoritative where the txt is stripped.
//
// Status of exactness: the real Game.exe DRLG was run in an emulator (see
// ~/git/d2-re-notes/drlg-oracle.md) and the Act 1 world layout, all 17 Act 1
// maze levels (rooms and final level seed) and the Act 2/3 creation draws were
// proven equal to it (oracle_test.go files, golden numbers in testdata/). The
// Act 1 outdoor generator is still a partial skeleton, and parts the notes
// mark as unread are left as TODO interfaces and are never invented.
package d2drlg
