// Package drlgmaze ports the maze level generator (Levels.txt DrlgType 1):
// Act 1 caves, crypts, jail, catacombs and barracks, Act 2 sewers, harem,
// palace cellar, tombs, lairs and the Arcane Sanctuary, Act 3 spider caves,
// dungeons, sewers and Durance of Hate (levels 84-101), from drlg2.md part A
// and the drlg-act23 notes. Act 4 and Act 5 maze types (maze_act45.go) are covered too.
//
// Provenance of each step is noted in comments. Everything here is compared
// against the emulated real game in oracle_test.go.
//
// The level-seed draw order is reproduced exactly: room allocation steps,
// FillMazeRooms rolls, finisher counters, theme pass (31 steps) and the commit
// pass (file rolls, per-chunk room allocations and the DS1 object RNG gates of
// the few preset files that have any, see gates.go; Params.GateSteps overrides
// the built-in table).
package drlgmaze
