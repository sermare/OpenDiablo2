// Package d2drop implements Diablo II's loot generation the way Game.exe 1.14b
// does it, as reverse engineered in the d2-re-notes itemgen notes: treasure
// class resolution, the quality roll driven by ItemRatio.txt, and affix
// selection.
//
// The package is table driven and has no dependency on the game's data
// files: callers hand it parsed table data through small interfaces
// (TreasureSource, ItemSource, RatioSource) and a random source (RNG, which
// d2rand.Seed satisfies), so everything is unit testable and, given a seed,
// reproducible.
//
// Parts are marked VERIFIED where the formula was confirmed in the binary and
// UNVERIFIED where it was derived from branch structure only; the unverified
// parts are listed in the doc comment of the function that uses them.
package d2drop
