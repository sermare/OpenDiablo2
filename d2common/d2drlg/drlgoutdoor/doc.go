// Package drlgoutdoor ports the Act 1 outdoor level generator of Game.exe
// 1.14b (DRLG_GenerateAct1Outdoors, 0x683800, and the cell-to-room pass
// DRLG_CreateOutdoorRooms, 0x677dd0) from the reference implementation in
// d2-re-notes/drlg3-ref (written from drlg3.md and verified bit-exact against
// the real game running in an emulator).
//
// Pipeline (see Generate): boundary polygon with exit notches, exit marks,
// concave cliff marking, border edge presets, the four LvlSub border/cliff
// matcher passes, cave entrances, town transitions, the dirt-road network,
// waypoint and shrine sites, the per-level special features, and finally one
// room per cell (BuildRoomGrids then builds the 9x9 A/B/C tile grids of a
// plain room).
//
// What is NOT covered: the tile-record creation per cell (0x680720 and
// callees) and the DT1 tile library. BuildRoomGrids stops where the game calls
// DRLG_PickRandomTile for the random tile markers of a sub-theme pattern and
// models it as exactly one room-seed step (see RoomBuildOptions.PickTile).
// Act 1 levels 2-7, 0x11 and 0x27 are covered by testdata/outdoor_act1.json.
//
// Acts 4 and 5 (act4.go, act5.go, preset_level.go, world45.go, with
// drlgworld.GenerateAct4/GenerateAct5): the outdoor levels 104, 105, 106, 108,
// 110, 111, 112, 117 and the DrlgType 2 levels 103, 109, 120, 121, 124, 131,
// 132, 136 are compared against testdata/outdoor_act45.json (12 seeds x 3
// difficulties x 16 levels = 576 records: rect, vis, od.flags, neighbour list,
// preset file index, digests of the F/def/B/D grids, the whole room list with
// flags and Dt1Mask, final level seed; see TestOracleAct45).
//
// TODO: level 134 (Forgotten Sands) runs the Act 2 desert generator (branch
// 0x86 of 0x6829f0, drlg-act45-outdoor.md section 6). That generator is not
// ported (LevelType 0x10 is rejected by Generate); the world placement of 134
// is in drlgworld.GenerateAct5. Level 107 and the Act 4/5 mazes are in
// drlgmaze.
package drlgoutdoor
