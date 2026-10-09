// Package d2level is a pure model of how a player moves between levels in
// Diablo II 1.14b: level ids and acts, the level graph (which exit leads
// where), waypoints, portals, cooldowns, arrival start types and the LoadAct
// packet. It has no engine dependencies, so the rules can be tested without
// game data. Facts come from the reverse-engineering notes (session-core.md,
// drlg.md, drlg2.md); anything not verified there is marked UNVERIFIED.
//
// Every level change in the original goes through one function,
// SERVER_ChangePlayerLevel (0x5389f0). PlanTransition mirrors its decision:
// same act moves the unit inside the already built act, a different act builds
// the destination act on demand from the same game seed and sends LoadAct.
package d2level
