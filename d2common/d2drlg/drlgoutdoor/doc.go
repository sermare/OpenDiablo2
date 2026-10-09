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
// Level.BuildTiles continues where the room list ends: the tile records of every
// room (0x680720 for plain rooms, 0x6696c0 for presets, 0x671680 per cell, the
// DT1 library of the room and DRLG_PickRandomTile 0x6704f0 with the room seed,
// the tree markers of the sub-theme patterns, the border-cell lookup and merge
// with already built neighbour rooms). The game builds rooms in the order the
// player streams them in; BuildTiles fixes the order (plain rooms in creation
// order, then presets in creation order) and the golden uses the same one.
//
// Only levels 2-7, 0x11 and 0x27 of Act 1 are covered by the golden; other
// level types (Acts 2-5) are not implemented. Tile code paths no golden room
// reaches are not ported and make BuildTiles return an error (see tiles.go).
package drlgoutdoor
