// Package d2automap is the pure (display-free) part of the Diablo II automap:
// the AutoMap table (txt and the compiled automap.bin), the lookup from a map
// tile to an automap cell, the isometric projection and the reveal model.
//
// What is VERIFIED against Game.exe 1.14b (UI\automap.cpp, D2Common LvlTbls.cpp,
// found with Ghidra, see d2-re-notes) and what is not is marked in each file.
// Short version of the real algorithm:
//
//   - automap.bin: u32 row count, then 44-byte rows: LevelName[16], TileName[8],
//     Style u8, StartSequence u8, EndSequence u8, pad u8, Cel1..Cel4 i32.
//     0xFF in a byte column and -1 in a cel mean "any" and "unused". (VERIFIED)
//   - The level name is "<act> <LvlTypes name>" (index = LvlTypes.txt Id) and the
//     tile name indexes a fixed table (fl wl wr wtlr ... = DT1 orientation).
//     A tile matches the first row of the level whose tile name equals the
//     tile's orientation, whose style is 0xFF or equal to the tile's main index
//     and whose sequence range holds the tile's sub index. One of the (up to
//     four) cels of the row is picked at random from the level seed. (VERIFIED,
//     except that the orientation is the raw DT1 value: strongly suggested)
//   - A cell is drawn at ((tx-ty)*8, (tx+ty)*4) from the 16x32 frame of
//     MaxiMap.dc6 (MaxiMapS.dc6 for the 8x16 mini map). (VERIFIED)
package d2automap
