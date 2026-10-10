// Package herogen builds a ready-to-play level 94 sample hero as a .d2s file at
// run time, from the game tables, for the long playthrough scenarios
// (scripts/verify.d/9*-*playthrough.sh, switched by OD2_HERO=barb).
//
// The generated Barbarian replaces the sample Sorceress in scenarios that only
// need "a strong hero": the Sorceress runs out of mana (the engine has no mana
// regeneration yet) and melees with a broken flail, while the Barbarian wields
// a unique two-hand sword with the strength and dexterity it needs and a full
// set of unique gear.
//
// Everything is derived from the tables and nothing is stored in the repository:
//
//   - the writer is the d2s package (Promote, SetStat, NewItem, Write);
//   - items are created by the d2drop item generator with the unique row forced
//     (Request.ForcedID), so their rolls are the ones the game rolls; the
//     generator seed is fixed, which makes the file deterministic;
//   - life, mana and stamina follow charstats (d2statlist.Class.BaseMax) and
//     the current values equal the totals with the worn items, like a save the
//     game writes at full health.
//
// Hero saves must never be committed: callers write the file to a scratch
// folder (see scripts/d2s-make-hero.go).
package herogen
