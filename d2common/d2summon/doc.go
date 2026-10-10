// Package d2summon models summoned minions of the player: the stats a minion
// gets from monstats.txt plus the summoning skill's level-scaled modifiers
// (Stats), and the per-owner limits that decide how many of a kind may live
// (Roster). The follow / attack behaviour lives in d2monster (NecroPet,
// Raven, Vines, AssassinSentry AIs).
//
// The package is pure: it consumes the SummonOrder that d2skill produces and
// the tab-separated monstats.txt, and knows nothing about the engine.
//
// Evidence levels follow the repository convention. VERIFIED: the skills.txt
// columns and their meaning (summon, pettype, petmax, aurastat*, passivestat*,
// calc1; skills-combat.md) and the pet AI constants (monster-ai-2.md 6.2).
// UNVERIFIED (marked in the code): how the exe turns the numbers into a unit
// (rounding, the order of the percent steps) and the eviction rule when a
// non-golem summon is cast at the limit.
package d2summon
