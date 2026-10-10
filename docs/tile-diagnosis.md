# Tile diagnosis (numbers only)

Question from the user: do the wrong tiles in the Mac app cluster (a few tile kinds) or are they random? Method: the real Game.exe tile code run in the emulator (`~/git/drlg-oracle`, `gen_tiles2.py`, DT1 loader hooks) on the user's own MPQ data produces the tile records (DT1 file, tile index, orientation, flags, per cell and layer) of every preset level; the Go output is diffed record by record (`drlgoutdoor` tests `TestTileDiffDir`, `TestTileSimDir`, `TestStaleCacheKeys`, `TestOracleTiles`; env `ORACLE_TILES_DIR`, `D2_TABLES`, `D2_DS1_ROOT`). Committed goldens hold numbers only (`d2common/d2drlg/testdata/tiles_towns.json`, `tiles_presets.json`).

## Answer

1. The wrong tiles cluster, and the cluster is one defect, not a set of bad tile ids: **the renderer's tile image cache is keyed by (style, sequence, type, variant) without the DT1 file or the level palette and was not cleared when the level changed**, so a level drew pictures of the levels visited before it for every key both share. Measured on the real records of the five towns walked in travel order (`TestStaleCacheKeys`, seed 0x101d574a): records whose cache key an earlier level had filled with a different DT1 tile: Lut Gholein 1719 of 4569 (38%), Kurast Docks 2599 of 3731 (70%), Pandemonium Fortress 242 of 996 (24%), Harrogath 1409 of 2865 (49%); the Rogue Encampment, loaded first, 0 of 2754. This is why the effect is worst in Act 3 (the most levels cached before it) and "less often" elsewhere. The wrong pictures are whatever the earlier level held under the same key (cave rocks and floors in a town, dark forest/cave floors in a snow level, black or half-size images where the earlier tile was blank or smaller), which is the "white rock formation" and "dark mossy diamond" kind of tile in the screenshots. Fixed on `feat/visual-fidelity` (`resetLevelCaches`, in integration) and measured here: the same Act 3/Act 4 frames taken with and without the reset differ from corrupted to coherent; `TILESTATS` after the reset shows `nofloor=0 fallbacks=0` in every town.
2. Independently of the cache, the old stamp path for the town levels (DS1 cell -> tile by (style, sequence, type) over the level type DT1 union, variant by the engine's own rarity hash) is wrong on measurable record kinds, and these cluster too (section "Old path vs the emulator"): of 442608 non-identical records in the 103 preset and maze levels measured, 96.6% are floors, and the 10 largest (layer, DT1 file, outcome) groups hold 87.6% of all of them; the town floors are mostly the right tile key with the wrong variant, and the walls are second/third wall layers and roof/orientation-15 and orientation-4 pieces that the old path skips because the DS1 wall dword has prop1 = 0. Fixed for the five towns by building them from the exact tile records (all five towns equal the emulator, 5 seeds).
3. The black parts: with the exact records every town cell has a floor (`TILESTATS ... nofloor=0`) and no floor graphic of the emulator's town records is blank (`TestTileSimDir`, "BLANK floor graphic" counter, 0 lines for the towns). So the solid black diamonds/triangles of the Alkor and Act 5 screenshots are not missing tile records: they are stale cache images (item 1). Where the real game fills a cell the DS1 leaves empty it uses `act3/docktown/bridge.dt1` records in Kurast Docks (113 per level) and `act1/outdoors/blank.dt1` in the Fortress and Harrogath (a blank picture on purpose).
4. Other screenshot classes: green and blue/cyan lattices are marker DT1 pixels (fixed on visual-fidelity, which hides marker-only DT1 graphics); the speckled/dithered wall faces in the Act 2 screenshot are the roof/wall fade state, not a tile lookup error (not touched here); half-diamond black triangles are the same stale images, cut off by the neighbouring correct tiles (a stale image from a smaller or blank tile leaves the rest of the diamond unpainted).
5. I could not identify the "white rock formation" tile as one (file, index) from the screenshots alone; the oracle cannot name tiles of a state that never exists in the real game (they are the earlier level's tiles). The statement above is what was measured; the (file, index) pair of a stale picture is by construction a tile of the level that was visited before.

## Exact builder vs the emulator (records, differing/total)

`drlgoutdoor` `GeneratePreset` / `GenerateTown` + `BuildTiles` against the emulator, seed 0x101d574a unless noted. The five towns were also run with seeds 0x1, 0x2, 0x3, 0x12345678: 0 differing records in all 25 (seed, town) runs.

| level | rooms | walls | floors | shadows |
|---|---|---|---|---|
| 1 | 35 | 0/413 | 0/2337 | 0/4 |
| 102 | 24 | 0/322 | 0/1201 | 0/0 |
| 103 | 12 | 0/227 | 0/769 | 0/0 |
| 109 | 25 | 0/826 | 0/1749 | 0/290 |
| 120 | 12 | 0/129 | 0/565 | 0/20 |
| 121 | 12 | 0/443 | 0/594 | 0/0 |
| 124 | 121 | 0/740 | 0/7081 | 0/0 |
| 13 | 9 | 0/5 | 0/580 | 0/0 |
| 131 | 35 | 0/518 | 0/2097 | 0/0 |
| 132 | 49 | 0/234 | 0/3026 | 0/0 |
| 136 | 36 | 0/442 | 0/2189 | 0/0 |
| 14 | 9 | 0/3 | 0/580 | 0/0 |
| 15 | 9 | 0/2 | 0/580 | 0/0 |
| 16 | 9 | 0/5 | 0/580 | 0/0 |
| 20 | 1 | 0/18 | 0/65 | 0/0 |
| 25 | 16 | 0/227 | 0/901 | 0/0 |
| 26 | 24 | 2/264 | 0/1235 | 0/0 |
| 27 | 35 | 220/524 | 257/2248 | 0/0 |
| 32 | 9 | 3/154 | 0/364 | 0/0 |
| 33 | 20 | 3/236 | 0/970 | 0/0 |
| 37 | 12 | 2/162 | 0/682 | 0/0 |
| 38 | 36 | 0/442 | 0/2189 | 0/0 |
| 40 | 49 | 0/769 | 0/3462 | 0/338 |
| 50 | 6 | 0/118 | 0/312 | 0/0 |
| 73 | 24 | 0/230 | 0/1604 | 0/0 |
| 75 | 48 | 0/366 | 0/3189 | 0/176 |
| 90 | 25 | 0/646 | 0/1603 | 0/0 |
| 91 | 25 | 0/804 | 0/1601 | 0/0 |
| 93 | 9 | 0/79 | 0/397 | 0/0 |
| 94 | 9 | 26/206 | 0/577 | 0/0 |
| 95 | 9 | 0/226 | 0/577 | 0/0 |
| 96 | 9 | 0/265 | 0/577 | 0/0 |
| 97 | 9 | 28/242 | 0/577 | 0/0 |
| 98 | 9 | 0/231 | 0/577 | 0/0 |
| 99 | 9 | 0/241 | 0/577 | 0/0 |

Levels the harness cannot build through the Go DRLG port without extra inputs (not compared): 10, 100, 101, 11, 113, 114, 115, 116, 118, 119, 12, 122, 123, 128, 129, 130, 133, 18, 19, 21, 22, 23, 24, 28, 29, 30, 31, 34, 35, 36, 47, 48, 49, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 74, 8, 84, 85, 86, 87, 88, 89, 9, 92 (maze levels outside the preset/outdoor builder: "level N not in the layout", "not an Act 2/3 outdoor level", "LevelType not ported").

Mismatching preset levels at the time of the table above: 26 (2 walls), 27 (220 walls, 257 floors), 32 (3 walls), 33 (3), 37 (2), 94 (26), 97 (28). All seven are now exact (branch `fix/preset-seven`, 10 seeds each, golden `tiles_presets7.json`); causes, each verified in the emulator:
- 26, 32, 33, 37: the tile object table (0x6f0738) and its level table (0x6f0578) were ported for Act 5 only. Wall records of orientation 8/9 of these levels get record flag 0x20 when a table row matches (style, sequence, orientation 9) and the object lands inside the room. The whole tables are in `tiles.go` now (`tileObjLevels`, `tileObjRows`); the one row type that rolls the room seed (ids 0x5b/0x5c, Act 2 mazes) is reported as unported.
- 94, 97: a wall record of orientation 10/11 (cave entrance piece, 0x670f20) is unlinked from its group chain: the game reuses the record's next pointer for the exit entry's list (0x670fae), which makes every older record of that group unreachable for the neighbour lookup (0x671190). A neighbouring room therefore makes its own border records instead of merging with the older cells. Modelled with `RoomTiles.warpHead`.
- 27: not a tile defect. Levels.txt `Depend` (record +0x2c, read by 0x644190) makes the offset relative to another level's rectangle (27 on 26, 33 on 32): 27 sits at (3000, 960). Level 27's data +4 (the exit side, `Layout.BarracksExitSide`) is a forced preset file index (0x66ad50: not -1 replaces the rolled file), like the town file of level 1; `PresetFileOverride` carries both.
`TestOracleTiles` now compares 427 levels, 30129 rooms, 0 mismatches.

## Old path vs the emulator, per level (seed 0x101d574a)

Old path = `PlaceStamp`/`PrepareTile` as in `d2mapengine/map_tile.go` (replicated in `TestTileSimDir`). Per record of the emulator: same = same DT1 file and index; variant = same (style, sequence, type), other tile; other = another key; missing = the old path draws nothing for that record; extra = the old path draws a tile where the emulator has no record. Maze levels use the DRLG port's room list (`drlgmaze`) with the DS1 of each room; rows whose same column is 0 (27, 28, 33, 69, 71) lack level inputs the harness does not have and are not valid.

| level | floors rec | same | variant | other | missing | extra | walls rec | same | variant | other | missing | extra |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | 2337 | 664 | 1572 | 0 | 101 | 0 | 413 | 361 | 12 | 2 | 38 | 0 |
| 8 | 1732 | 699 | 328 | 0 | 705 | 0 | 15 | 15 | 0 | 0 | 0 | 0 |
| 9 | 2888 | 1205 | 519 | 0 | 1164 | 0 | 37 | 37 | 0 | 0 | 0 | 0 |
| 10 | 5196 | 2161 | 958 | 0 | 2077 | 0 | 60 | 60 | 0 | 0 | 0 | 0 |
| 11 | 2888 | 1122 | 538 | 0 | 1228 | 0 | 37 | 37 | 0 | 0 | 0 | 0 |
| 12 | 2888 | 1089 | 469 | 0 | 1330 | 0 | 29 | 29 | 0 | 0 | 0 | 0 |
| 13 | 580 | 350 | 80 | 0 | 150 | 0 | 5 | 5 | 0 | 0 | 0 | 0 |
| 14 | 580 | 318 | 76 | 0 | 186 | 0 | 3 | 3 | 0 | 0 | 0 | 0 |
| 15 | 580 | 309 | 89 | 0 | 182 | 0 | 2 | 2 | 0 | 0 | 0 | 0 |
| 16 | 580 | 221 | 83 | 0 | 276 | 0 | 5 | 5 | 0 | 0 | 0 | 0 |
| 18 | 775 | 109 | 345 | 0 | 321 | 0 | 314 | 167 | 123 | 0 | 24 | 0 |
| 19 | 840 | 107 | 347 | 0 | 386 | 0 | 323 | 166 | 131 | 0 | 26 | 0 |
| 20 | 65 | 9 | 7 | 0 | 49 | 0 | 18 | 15 | 1 | 0 | 2 | 0 |
| 21 | 516 | 65 | 207 | 0 | 244 | 0 | 196 | 110 | 70 | 0 | 16 | 0 |
| 22 | 453 | 69 | 165 | 0 | 219 | 0 | 179 | 81 | 83 | 0 | 15 | 0 |
| 23 | 454 | 65 | 171 | 0 | 218 | 0 | 170 | 79 | 76 | 0 | 15 | 0 |
| 24 | 388 | 46 | 154 | 0 | 188 | 0 | 154 | 83 | 56 | 0 | 15 | 0 |
| 25 | 901 | 111 | 259 | 0 | 531 | 0 | 227 | 99 | 105 | 2 | 21 | 0 |
| 26 | 1235 | 97 | 608 | 0 | 530 | 0 | 264 | 229 | 18 | 1 | 16 | 0 |
| 27 | 2248 | 0 | 0 | 0 | 2248 | 1152 | 484 | 0 | 0 | 0 | 484 | 423 |
| 28 | 1915 | 0 | 0 | 0 | 1915 | 1265 | 732 | 0 | 0 | 0 | 732 | 657 |
| 29 | 1821 | 222 | 1015 | 0 | 584 | 0 | 627 | 295 | 275 | 6 | 51 | 0 |
| 30 | 2282 | 281 | 1320 | 0 | 681 | 0 | 798 | 379 | 334 | 2 | 83 | 0 |
| 31 | 2572 | 314 | 1365 | 0 | 893 | 0 | 937 | 441 | 384 | 11 | 101 | 0 |
| 32 | 364 | 90 | 270 | 0 | 4 | 0 | 154 | 107 | 5 | 0 | 42 | 0 |
| 33 | 970 | 0 | 0 | 0 | 970 | 584 | 236 | 0 | 0 | 0 | 236 | 198 |
| 34 | 1985 | 128 | 1150 | 0 | 707 | 0 | 687 | 217 | 363 | 13 | 94 | 0 |
| 35 | 2277 | 148 | 1379 | 0 | 750 | 0 | 739 | 222 | 447 | 4 | 66 | 0 |
| 36 | 2600 | 184 | 1592 | 0 | 824 | 0 | 851 | 275 | 491 | 6 | 79 | 0 |
| 37 | 682 | 84 | 292 | 0 | 306 | 0 | 162 | 62 | 65 | 0 | 35 | 0 |
| 38 | 2189 | 1025 | 1072 | 0 | 92 | 0 | 442 | 241 | 62 | 0 | 139 | 0 |
| 40 | 3462 | 2278 | 1071 | 0 | 113 | 0 | 769 | 702 | 0 | 6 | 61 | 0 |
| 47 | 2020 | 580 | 283 | 0 | 1157 | 0 | 735 | 683 | 0 | 0 | 52 | 0 |
| 48 | 1732 | 503 | 256 | 0 | 973 | 0 | 617 | 583 | 0 | 0 | 34 | 0 |
| 49 | 2016 | 574 | 279 | 0 | 1163 | 0 | 710 | 665 | 0 | 0 | 45 | 0 |
| 50 | 312 | 71 | 130 | 0 | 111 | 0 | 118 | 92 | 4 | 0 | 22 | 0 |
| 51 | 1032 | 596 | 428 | 0 | 8 | 0 | 463 | 342 | 14 | 1 | 106 | 0 |
| 52 | 1049 | 377 | 643 | 4 | 25 | 0 | 481 | 351 | 12 | 8 | 110 | 0 |
| 53 | 1032 | 387 | 633 | 4 | 8 | 0 | 511 | 374 | 14 | 3 | 120 | 0 |
| 54 | 1057 | 387 | 633 | 4 | 33 | 0 | 478 | 337 | 9 | 7 | 125 | 0 |
| 55 | 1030 | 224 | 138 | 1 | 667 | 0 | 244 | 178 | 48 | 0 | 18 | 0 |
| 56 | 1569 | 281 | 201 | 2 | 1085 | 0 | 327 | 250 | 52 | 1 | 24 | 0 |
| 57 | 1853 | 428 | 281 | 0 | 1144 | 0 | 486 | 363 | 86 | 0 | 37 | 0 |
| 58 | 3493 | 908 | 505 | 1 | 2079 | 0 | 898 | 684 | 152 | 0 | 62 | 0 |
| 59 | 1027 | 213 | 142 | 0 | 672 | 0 | 214 | 158 | 42 | 0 | 14 | 0 |
| 60 | 1627 | 252 | 198 | 1 | 1176 | 0 | 294 | 223 | 52 | 0 | 19 | 0 |
| 61 | 257 | 129 | 3 | 0 | 125 | 0 | 65 | 48 | 14 | 0 | 3 | 0 |
| 62 | 1904 | 282 | 239 | 0 | 1383 | 0 | 732 | 677 | 2 | 0 | 53 | 0 |
| 63 | 1404 | 183 | 134 | 0 | 1087 | 0 | 485 | 455 | 0 | 0 | 30 | 0 |
| 64 | 1300 | 190 | 147 | 0 | 963 | 0 | 504 | 466 | 0 | 0 | 38 | 0 |
| 65 | 1440 | 410 | 175 | 0 | 855 | 0 | 532 | 494 | 0 | 0 | 38 | 0 |
| 66 | 1602 | 350 | 215 | 1 | 1036 | 0 | 348 | 255 | 69 | 0 | 24 | 0 |
| 67 | 1597 | 283 | 221 | 1 | 1092 | 0 | 336 | 224 | 91 | 1 | 20 | 0 |
| 68 | 1589 | 293 | 231 | 1 | 1064 | 0 | 311 | 232 | 59 | 0 | 20 | 0 |
| 69 | 4792 | 0 | 0 | 0 | 4792 | 541 | 1135 | 0 | 0 | 0 | 1135 | 303 |
| 70 | 1871 | 354 | 262 | 1 | 1254 | 0 | 398 | 278 | 96 | 1 | 23 | 0 |
| 71 | 3496 | 60 | 56 | 218 | 3162 | 248 | 819 | 47 | 9 | 42 | 721 | 260 |
| 72 | 1600 | 319 | 199 | 2 | 1080 | 0 | 344 | 259 | 61 | 0 | 24 | 0 |
| 73 | 1604 | 527 | 234 | 0 | 843 | 0 | 230 | 183 | 41 | 0 | 6 | 0 |
| 74 | 8784 | 463 | 1382 | 0 | 6939 | 0 | 1156 | 1150 | 0 | 0 | 6 | 1 |
| 75 | 3189 | 2862 | 214 | 0 | 113 | 0 | 366 | 351 | 0 | 0 | 15 | 0 |
| 84 | 1027 | 318 | 280 | 0 | 429 | 0 | 480 | 424 | 0 | 3 | 53 | 0 |
| 85 | 1027 | 311 | 283 | 0 | 433 | 0 | 430 | 383 | 0 | 3 | 44 | 0 |
| 86 | 1184 | 335 | 205 | 0 | 644 | 0 | 397 | 370 | 0 | 0 | 27 | 0 |
| 87 | 1185 | 315 | 191 | 0 | 679 | 0 | 401 | 374 | 0 | 0 | 27 | 0 |
| 88 | 1184 | 316 | 216 | 0 | 652 | 0 | 426 | 392 | 0 | 0 | 34 | 0 |
| 89 | 1184 | 309 | 186 | 1 | 688 | 0 | 397 | 367 | 0 | 0 | 30 | 0 |
| 90 | 1603 | 478 | 338 | 0 | 787 | 0 | 646 | 601 | 0 | 0 | 45 | 0 |
| 91 | 1601 | 589 | 395 | 0 | 617 | 0 | 804 | 739 | 0 | 0 | 65 | 0 |
| 92 | 4619 | 1300 | 2612 | 0 | 707 | 0 | 1893 | 1135 | 156 | 0 | 602 | 0 |
| 93 | 397 | 30 | 130 | 0 | 237 | 0 | 79 | 42 | 30 | 0 | 7 | 0 |
| 94 | 577 | 53 | 155 | 0 | 369 | 0 | 206 | 126 | 32 | 8 | 40 | 0 |
| 95 | 577 | 71 | 179 | 0 | 327 | 0 | 226 | 161 | 21 | 3 | 41 | 0 |
| 96 | 577 | 89 | 252 | 0 | 236 | 0 | 265 | 192 | 15 | 2 | 56 | 0 |
| 97 | 577 | 78 | 250 | 0 | 249 | 0 | 242 | 196 | 24 | 2 | 20 | 0 |
| 98 | 577 | 57 | 171 | 0 | 349 | 0 | 231 | 183 | 17 | 4 | 27 | 0 |
| 99 | 577 | 69 | 262 | 0 | 246 | 0 | 241 | 199 | 20 | 3 | 19 | 0 |
| 100 | 1602 | 362 | 867 | 1 | 372 | 0 | 537 | 292 | 19 | 29 | 197 | 0 |
| 101 | 1883 | 485 | 926 | 1 | 471 | 0 | 608 | 340 | 17 | 35 | 216 | 0 |
| 102 | 1201 | 311 | 521 | 0 | 369 | 0 | 322 | 228 | 32 | 8 | 54 | 0 |
| 103 | 769 | 194 | 33 | 0 | 542 | 0 | 227 | 209 | 0 | 4 | 14 | 0 |
| 109 | 1749 | 522 | 735 | 0 | 492 | 0 | 826 | 692 | 31 | 4 | 99 | 1 |
| 113 | 3328 | 1058 | 792 | 0 | 1478 | 0 | 1179 | 1062 | 0 | 0 | 117 | 0 |
| 114 | 4096 | 1325 | 1096 | 0 | 1675 | 0 | 703 | 635 | 0 | 0 | 68 | 0 |
| 115 | 3328 | 1131 | 732 | 0 | 1465 | 0 | 1157 | 1043 | 0 | 0 | 114 | 0 |
| 116 | 1024 | 464 | 333 | 0 | 227 | 0 | 201 | 179 | 0 | 0 | 22 | 0 |
| 118 | 4352 | 1389 | 1037 | 0 | 1926 | 0 | 1561 | 1403 | 0 | 0 | 158 | 0 |
| 119 | 1024 | 434 | 363 | 0 | 227 | 0 | 201 | 179 | 0 | 0 | 22 | 0 |
| 120 | 565 | 147 | 189 | 0 | 229 | 0 | 129 | 116 | 7 | 0 | 6 | 0 |
| 121 | 594 | 112 | 179 | 0 | 303 | 0 | 443 | 320 | 52 | 5 | 66 | 0 |
| 122 | 6404 | 364 | 2274 | 0 | 3766 | 0 | 1783 | 1648 | 0 | 10 | 125 | 0 |
| 123 | 6404 | 398 | 2284 | 0 | 3722 | 0 | 1876 | 1732 | 0 | 12 | 132 | 0 |
| 124 | 7081 | 219 | 994 | 0 | 5868 | 0 | 740 | 674 | 0 | 7 | 59 | 0 |
| 125 | 87617 | 398 | 9466 | 11 | 77742 | 0 | 190 | 133 | 57 | 0 | 0 | 0 |
| 126 | 87603 | 415 | 9449 | 13 | 77726 | 0 | 195 | 132 | 63 | 0 | 0 | 0 |
| 127 | 87588 | 402 | 9384 | 4 | 77798 | 0 | 172 | 121 | 51 | 0 | 0 | 0 |
| 128 | 2313 | 322 | 745 | 0 | 1246 | 0 | 556 | 526 | 0 | 0 | 30 | 0 |
| 129 | 2560 | 365 | 861 | 0 | 1334 | 0 | 669 | 629 | 0 | 0 | 40 | 0 |
| 130 | 2569 | 359 | 781 | 0 | 1429 | 0 | 584 | 546 | 0 | 0 | 38 | 0 |
| 131 | 2097 | 135 | 815 | 0 | 1147 | 0 | 518 | 22 | 471 | 0 | 25 | 0 |
| 132 | 3026 | 146 | 252 | 0 | 2628 | 0 | 234 | 102 | 119 | 0 | 13 | 0 |
| 133 | 773 | 111 | 295 | 0 | 367 | 0 | 271 | 130 | 125 | 0 | 16 | 0 |
| 136 | 2189 | 1015 | 1082 | 0 | 92 | 0 | 442 | 236 | 67 | 0 | 139 | 0 |

## Clustering of the non-identical records (all levels, all seeds measured)

Total non-identical records: 442608.

| layer | outcome | reason | records |
|---|---|---|---|
| floors | missing | second record on a cell the old path fills once | 231509 |
| floors | variant |  | 93124 |
| floors | missing | prop1=0 or hidden DS1 cell | 86715 |
| floors | missing | no DS1 cell at all | 15759 |
| walls | variant |  | 5641 |
| walls | missing | no DS1 cell at all | 4032 |
| walls | missing | prop1=0 or hidden DS1 cell | 3027 |
| walls | missing | second record on a cell the old path fills once | 2186 |
| walls | other |  | 318 |
| floors | other |  | 272 |
| shadows | missing | prop1=0 or hidden DS1 cell | 25 |

Top DT1 files (layer, file, outcome), records; the 10 largest files hold 87.6% of all non-identical records, out of 235 distinct (layer, file, outcome) groups:

| layer | DT1 file | outcome | records |
|---|---|---|---|
| floors | act4/lava/floor.dt1 | missing | 233238 |
| floors | act1/outdoors/blank.dt1 | missing | 90241 |
| floors | act4/lava/floor.dt1 | variant | 25892 |
| floors | act1/town/floor.dt1 | variant | 9285 |
| floors | expansion/wildtemple/interior.dt1 | variant | 5552 |
| floors | act3/kurast/floors.dt1 | variant | 5417 |
| floors | act2/town/ground.dt1 | variant | 5388 |
| floors | act1/catacomb/floor.dt1 | variant | 4413 |
| floors | expansion/icecave/interior.dt1 | variant | 4353 |
| floors | act2/tomb/tomb.dt1 | missing | 4131 |
| floors | act1/barracks/floor.dt1 | variant | 3700 |
| floors | expansion/town/ground.dt1 | variant | 3407 |
| floors | act1/caves/cave.dt1 | variant | 3140 |
| floors | act2/tomb/tomb.dt1 | variant | 2838 |
| floors | act4/lava/floornew.dt1 | variant | 2407 |
| floors | expansion/baallair/floor.dt1 | variant | 2387 |
| floors | act1/crypt/floor.dt1 | variant | 1943 |
| floors | act2/palace/cellflr.dt1 | variant | 1909 |
| walls | act2/tomb/tomb.dt1 | missing | 1903 |
| floors | act1/barracks/floor.dt1 | missing | 1828 |
| floors | act3/jungle/dungeon.dt1 | variant | 1531 |
| floors | act2/arcane/sanctuary.dt1 | variant | 1382 |
| walls | act1/catacomb/basewalls.dt1 | variant | 1360 |
| floors | act1/court/floor.dt1 | missing | 1162 |
| floors | expansion/baallair/underflr.dt1 | variant | 1067 |

Wall orientation of the non-identical wall records:

| orientation | missing | variant | other |
|---|---|---|---|
| 1 | 2136 | 2479 | 69 |
| 2 | 1869 | 2445 | 69 |
| 3 | 219 | 153 | 49 |
| 4 | 2972 | 0 | 120 |
| 5 | 185 | 84 | 1 |
| 6 | 230 | 95 | 2 |
| 7 | 318 | 9 | 5 |
| 8 | 91 | 61 | 2 |
| 9 | 80 | 63 | 0 |
| 10 | 45 | 2 | 0 |
| 11 | 130 | 8 | 0 |
| 12 | 345 | 229 | 1 |
| 14 | 250 | 0 | 0 |
| 15 | 353 | 0 | 0 |
| 16 | 6 | 8 | 0 |
| 17 | 11 | 5 | 0 |
| 18 | 5 | 0 | 0 |

The floors "missing" bulk (act4/lava/floor.dt1 and act1/outdoors/blank.dt1) are records where the emulator fills a cell the DS1 leaves empty (the lava level floor and Blank.dt1): a blank or near-black picture, not a visible defect by itself.

## Town numbers before/after

Old stamp path (above) vs the exact records now used by the five town generators (`d2mapgen/town_exact.go`, applied over the stamp, which keeps the NPCs, objects and marker walls): floors differing 0 of 2337 / 3462 / 3189 / 769 / 1749 (towns 1, 40, 75, 103, 109), walls 0 of 413 / 769 / 366 / 227 / 826, shadows 0 of 4 / 338 / 176 / 0 / 290. In game (`OD2_REALMAPS=1`, `scripts/verify.d/9n-town-tiles.sh`) the log line `TILESTATS level=<id> exact=true cells= nofloor=0 ... fallbacks=0` is written per town.

## Outdoor and cave levels (fix/outdoor-cave-tiles)

Outdoor levels (Act 1-5, DrlgType 3, including the Act 3 jungle 76..83) already build their tiles from the exact records (`TestOracleTiles`, 357 levels, 0 mismatches); the jungle frames of the screenshots came from the stale tile image cache (item 1 above) and show correct huts, roofs and ground now with the cache reset (scenarios `9o-jungle-tiles.sh`, `9p-jungle-chain.sh`: levels 75..81 entered by waypoint in travel order, `TILESTATS exact=true nofloor=0 fallbacks=0` for every one).

Cave and maze levels (DrlgType 1: caves, crypts, catacombs, tombs, sewers, palace, Act 3 dungeons, Act 4/5 mazes) were still stamped from the DS1 with the old (style, sequence, type) lookup (the "patchwork of mismatched floor diamonds" and floating walls). They now use `drlgoutdoor.MazeLevel` (`maze_tiles.go`): every chunk of `drlgmaze` is a preset room of its DS1 built by the same preset tile builder (DT1 library from the LvlPrest Dt1Mask, rarity pick with the room seed taken from the level-seed step of the chunk, border merging, warp/outdoors/populate room flags). Measured against the emulator (`TestMazeTilesDir`, `TestMazeTilesGolden`, `testdata/tiles_maze.json`, seed 0x101d574a): 70 maze levels, 2696 rooms, 2481 rooms (92%) record-for-record equal (flag 0x20 left out: it marks tile objects of the Act 1/2 rows of the tile object table, not ported, no picture); 51 of the 70 levels are equal in every room. Update (fix/maze-borders): all 70 maze levels (2696 rooms) are now record-for-record equal to the emulator including flag 0x20, and 35, 92, 69 and 71 build too. The "border merge" suspicion was wrong: the lookup/merge port (0x671250, 0x671190, 0x671420) was already right. The real causes were (1) DRLG_CreateTileObject (0x6706a0) rolls the room seed (bound 3) for the tile object ids 0x5b/0x5c of the Act 2 mazes (levels 55-72) before the spawn, also when 0x671680 calls it without a record for a hidden orientation 8/9 cell, which used to stop the build as "unported"; (2) the harness: levels 100/101 and 31 were off by one level-seed step per room because the tile oracle's emulator has no monster table, so the DS1 object gate of MephNWarpD (3 in the real game) and JailSETheme (1) draw 2 and 0 steps there (the test uses `oracleGateSteps`; the game keeps `DS1GateSteps`); (3) 69/71 "room count differs" because the test did not pass the two special Act 2 tombs (`DrawActExtras`) that the game derives from the seed. `OD2_MAZE_STAMP=1` still forces the old path.
