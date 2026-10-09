// Package d2trade implements the Diablo II 1.14b vendor price formulas
// (buy, sell, gamble, repair, identify) as pure integer functions.
//
// The formulas follow the reverse-engineering notes for TRADE_CalcItemPrice
// (Game.exe 0x62f100) and TRADE_CalcGamblePrice (0x629570). All prices are
// computed with 32-bit integer arithmetic and 1/1024 fixed point multipliers,
// as the original does. Items that need data from several tables (affix
// multiply/add columns, ItemStatCost, ...) are described by the caller
// through Item; this package does not read any game files.
//
// Mode numbering is the server's: 0 buy, 1 sell, 2 gamble, 3 repair.
package d2trade
