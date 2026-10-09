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

// Rules pinned by tests (states.txt = patch_d2, ids are row numbers):
//
//	verified:   cold slow = monstats ColdEffect of the difficulty (stats
//	            velocitypercent, attackrate, other_animrate); stun length cap
//	            250 frames; states are 25 Hz frames, active on [apply, Until);
//	            curse_resistance >= 100 rejects a timed skill state.
//	data:       group / curse / remhit / *staydeath / shatter / colorpri /
//	            colorshift columns (TestReal*; skipped without D2_TABLES).
//	unverified: exe use of group and curse as mutual exclusion (Defs.exclusive),
//	            staydeath = "survives death", blue overriding colorpri, rounding
//	            of length reductions, a shorter stun replacing a longer one,
//	            ColdEffect 0 skipping stun, poison stacking, area-change removal
//	            (no column in states.txt; just_portaled and sync_warped only).
//
// Exe addresses for Ghidra to confirm: 0x578830 (stun), 0x578990 (poison),
// 0x578b00 (burn), 0x578ca0 (chill), 0x578f50 (freeze), 0x56c740
// (SKILL_CreateTimedStateStatList: replacement and curse rules), 0x63af70 and
// 0x63aef0 (set / clear a state bit), the state removal on death and the
// states.txt row consumers of colorpri / colorshift (client draw code).
