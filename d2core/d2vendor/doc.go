// Package d2vendor models the stock of a town vendor: how it is generated
// when the player opens the trade window, and the 10x10 vendor inventory that
// holds it. The generator follows TRADE_GenerateVendorStock (Game.exe 1.14b
// 0x574780) as documented in d2-re-notes/inventory-trade.md. The stock model is
// pure: base items come in through Base values, randomness through
// d2drop.RNG (d2rand.Seed implements it), and the result is a list of Items
// placed on an OccupancyGrid, so everything is reproducible from a seed.
//
// Verified facts are marked VERIFIED, hypotheses UNVERIFIED in the comments.
package d2vendor
