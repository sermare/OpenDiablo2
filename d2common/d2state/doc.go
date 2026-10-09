// Package d2state is the pure model of timed states, auras and damage over
// time that skills put on units (skills-2.md sections 4.6, 4.15 and 5.3).
//
// In the game a state is a statlist with an expiry frame (SKILL_CreateTimedStateStatList,
// 0x56c740) carrying the stats of the skill's aurastat1..6 / passivestat1..5
// columns; the same skill re-applied refreshes it. This package keeps the
// same data model: a Set per unit holds Instances (name, end frame, stat
// modifiers) and the DoT streams (poison, burn). The engine reads derived
// values through the query methods in query.go (slow, can-act, resists,
// damage percent...).
//
// Frames are 25 Hz game frames. Poison and burn use the game's units: a
// per-frame damage in 8.8 fixed point hit points, so a stream of value v for
// n frames deals v*n/256 hit points in total (verified against the
// Poison Javelin tooltip: 32 per frame for 200 frames = 25 hp).
//
// Unverified (marked U where used): stacking of several poisons (each
// application is an independent stream here), the exact slow percents of
// chill and freeze, and that a state's stats stop applying exactly at Until.
package d2state
