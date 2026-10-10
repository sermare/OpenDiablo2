// Package herogen builds a ready-to-play level 94 hero of any of the seven
// classes as a .d2s file at run time, from the game tables, for the long
// playthrough scenarios (scripts/verify.d/9*-*playthrough.sh, switched by
// OD2_HERO=amazon|sorc|necro|paladin|barb|druid|assassin, see
// scripts/verify.d/lib/hero.sh and scripts/verify_classes.sh).
//
// The generated heroes replace the sample Sorceress in scenarios that only need
// "a strong hero": the sample runs out of mana and melees with a broken flail,
// while a preset wears class-legal unique gear, spends all its points on a
// coherent build around one main attack (the left skill, which a scripted fight
// casts: d2game/d2gamescreen/fightskill.go) and carries a full belt.
//
// Everything is derived from the tables and nothing is stored in the repository:
//
//   - the writer is the d2s package (Promote, SetStat, NewItem, Write);
//   - items are created by the d2drop item generator with the unique row forced
//     (Request.ForcedID), so their rolls are the ones the game rolls; the
//     generator seed is fixed, which makes the file deterministic;
//   - life, mana and stamina follow charstats (d2statlist.Class.BaseMax) and
//     the current values equal the totals with the worn items, like a save the
//     game writes at full health;
//   - Generate refuses a hero the game would not let be: the worn set is checked
//     with the game's equip rules (d2equip: body location, class, strength,
//     dexterity, level, hands, belt size) and the skills with skills.txt
//     (class, maximum, level, prerequisites).
//
// Hero saves must never be committed: callers write the file to a scratch
// folder (see scripts/d2s-make-hero.go).
package herogen
