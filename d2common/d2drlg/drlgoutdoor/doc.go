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
// Only levels 2-7, 0x11 and 0x27 of Act 1 are covered by the golden; other
// level types (Acts 2-5) are not implemented.
package drlgoutdoor
