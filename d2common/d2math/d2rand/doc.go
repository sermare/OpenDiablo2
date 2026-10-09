// Package d2rand implements the random number generator Diablo II uses for
// level generation, item drops and unit seeds, and the seed hierarchy the
// level generator (DRLG) derives from the game seed.
//
// Reverse engineered from Game.exe 1.14b: the state is a 64-bit pair of 32-bit
// words advanced by a multiply-with-carry step with the multiplier 0x6AC690C5.
// Reproducing it exactly is what makes real maps reproducible from a seed.
package d2rand
