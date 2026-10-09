// Package d2hireling is a pure model of Diablo II mercenaries (Game.exe 1.14b,
// notes hirelings.md): the hireling.txt record, the per-game hire offer table,
// offer stats and price, per-level stats, weighted skill choice, experience
// and level, and the revive cost.
//
// Faithfulness: V = verified in the notes' decompile, U = unverified.
// Nothing here reads game files; Parse takes the text of hireling.txt.
package d2hireling
