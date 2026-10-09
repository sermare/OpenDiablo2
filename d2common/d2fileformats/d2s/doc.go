// Package d2s reads and writes Diablo II character save files (.d2s).
//
// The layout was verified against Diablo II 1.14b (file version 0x60) with
// real saves: the header, quests, waypoints, NPC flags, attributes, skills
// and items are parsed, and a writer re-encodes a parsed save (an unchanged
// save round-trips byte for byte). New characters can be created too.
package d2s
