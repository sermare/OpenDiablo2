// Package d2calc implements the calculation language of skills.txt and
// missiles.txt (calc1..4, ToHitCalc, delay, aurastatcalc1..6, DmgSymPerCalc,
// SrvCalc1, ...), reverse engineered from Game.exe 1.14b (see skills-2.md,
// section 3 of the notes).
//
// The language has 32 bit signed integers only. A string is compiled once
// with Compile into reverse polish code (the game uses a shunting-yard
// compiler to byte code) and evaluated against an Env, which supplies the
// skill level context: bare identifiers are "field codes" (lvl, par1..par8,
// ln12, dm34, edmn, blvl, clc1, ...), looked up by their first four
// characters; and skill('Name'.field), miss('Name'.field), stat('name'.mode),
// min, max, rand, sklvl are function calls.
//
// Quirks that are reproduced because the shipped tables rely on them (or
// are at least compatible with them):
//
//   - lnXY is ParamX + (lvl-1)*ParamY (the txt description says lvl, not lvl-1),
//   - '^' is left associative, '/' truncates toward zero and divides by zero
//     to 0,
//   - the '?' operator binds tighter than every binary operator: write
//     (cond) ? (a) : (b) fully parenthesised; ':' itself is a no-op,
//   - comparison operators are the loosest and all have the same rank,
//   - unknown identifiers are the literal 0, a lone '=' or '!' is ignored.
//
// Verified from the decompile unless a comment says otherwise: the token set,
// the function list and arities, the opcode semantics and the precedence
// order. Unverified: rand's exact modulus, sklvl's argument order, how quoted
// names behave outside skill/miss/stat, and the dm formula for levels < 1.
package d2calc
