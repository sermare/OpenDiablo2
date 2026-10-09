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
// Still unverified (marked U where used): the exact slow percents of
// freeze, group as armor exclusivity, and shrine states.
package d2state

// Rules pinned by tests (states.txt = patch_d2, ids are row numbers).
// Exe addresses are Game.exe (1.14b); notes in d2-re-notes/verify-states.md.
//
//	VERIFIED in the exe:
//	  stun     0x578830: cap 250 frames; a new stun sets the end even when
//	           shorter; ColdEffect is not consulted.
//	  poison   0x578990, burn 0x578b00: one statlist per unit and kind (states
//	           2 and 0x73, stat hpregen 0x4a). A new one replaces strength and
//	           end only when its per-frame value >= the active one's; a weaker
//	           one is ignored. Poison and burn add up (two lists).
//	  chill    0x578ca0: slow = monster ColdEffect of the difficulty, 0 skips,
//	           -50 for non-monsters; length / difficulty divisor (min 1) for a
//	           negative ColdEffect; an active chill only gets a later end.
//	  freeze   0x578f50 (+0x578c50 reads ColdEffect): needs ColdEffect < 0;
//	           length / difficulty divisor; only extends; players, uninterrupt-
//	           able (state 0x36) and special monsters get chill instead or
//	           nothing.
//	  timed    0x56c740: one curse at a time (found through the curse mask
//	           0x63b530, whatever the curse); same state + skill + level only
//	           refreshes the end; lower level of the same skill is rejected;
//	           otherwise old statlist freed and a new one made; curse_resistance
//	           (stat 0x6d) >= 100 rejects, else length - trunc(length*r/100).
//	           The group column is not read here (only by the monster AI check
//	           0x5ea850); the curse column is the exclusion.
//	  bits     0x63aef0 / 0x63af70 set or clear the state bit (bounds checked)
//	           and queue the unit for sending.
//	  death    0x627890 + 0x63b5b0: statlists survive per plrstaydeath (players)
//	           or monstaydeath (all monsters); 0x63b0d0 clears the visible bits
//	           with the boss mask for flagged bosses. Called from 0x57d310
//	           (player) and 0x5a3f20 (monster).
//	  colour   client 0x4d65a0: highest colorpri (strict >, id 0 ignored, ties
//	           to the lowest id) -> colorshift; "blue" is not read; shift 104
//	           is dropped for the local player in 3D mode.
//	data:      group / curse / remhit / *staydeath / shatter / colorpri /
//	           colorshift columns (TestReal*; skipped without D2_TABLES).
//	unverified: armor / aura exclusivity by group (the exe has no such code in
//	           0x56c740), state 0x39 curse immunity and the boss stun cap of
//	           13 frames (not modelled), area-change removal (no bulk
//	           clear-on-area code found; 0x63b0d0 has only the two death
//	           callers), the difficulty divisors are not wired into the skills
//	           engine (the DifficultyLevels record does not parse them).
