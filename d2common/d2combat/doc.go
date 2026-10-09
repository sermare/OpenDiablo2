// Package d2combat holds Diablo II's combat math as pure functions.
//
// Everything here was reverse engineered from Game.exe 1.14b (see the
// skills-combat notes). Inputs are plain numbers: callers read them from
// skills.txt, charstats.txt, monstats.txt or unit stats and pass them in. The
// package has no engine dependencies and touches no game files.
//
// Randomness goes through the Roller interface, which *d2rand.Seed satisfies.
// The game rolls percentages as Roll(100) < chance, so a seeded Roller
// reproduces results exactly and the number of consumed steps matches the
// game: a function documents when it does NOT consume a step.
//
// Anything not read directly from the decompiled binary is marked "inferred"
// in its doc comment. Discrepancies between the binary and the notes found
// while spot-checking are marked "NOTE (binary)".
package d2combat

// Roller is the random source used for all rolls. Roll(n) returns a value in
// [0, n); *d2rand.Seed implements it.
type Roller interface {
	Roll(n int32) uint32
}

// FixedShift is the number of fractional bits of the game's 8.8 fixed point
// representation used for hit points, mana, stamina and damage.
const FixedShift = 8

// ToFixed converts a whole number to 8.8 fixed point.
func ToFixed(v int) int { return v << FixedShift }

// FromFixed truncates an 8.8 fixed point value to a whole number (toward
// negative infinity, like the game's shift).
func FromFixed(v int) int { return v >> FixedShift }

// roll100 reports whether a percent roll succeeds: Roll(100) < chance. It
// always consumes one step.
func roll100(r Roller, chance int) (ok bool, roll int) {
	roll = int(r.Roll(100))

	return roll < chance, roll
}
