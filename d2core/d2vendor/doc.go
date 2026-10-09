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

// Audit status (oracle_test.go pins the VERIFIED rules on real tables via
// D2_TABLES) and exe addresses still to confirm in Game.exe 1.14b:
//
//   - 0x5746a0 item level cap table (12/20/28/36/45) indexed by the vendor
//     record byte +0x22: what the index counts is unknown, so no cap is
//     applied (Tier -1); a cap changes late-game vendor item levels.
//   - 0x574780 whether the Min/MagicMin columns are read (only Max is);
//     whether the count is random(Max+1) or between Min and Max.
//   - 0x574cf0 hp4/hp5/mp4/mp5 "sold on non-normal difficulty" rule: potion
//     tiers by difficulty are NOT modelled (hp5/mp5 are sold by nobody here).
//   - 0x534c20 / 0x576970 restock trigger: the 240000 ms flag is documented
//     in gamble.go, but session-core.md lists 0x534c20 as called from the
//     town transition handler (0x534d40); confirm whether entering town also
//     restocks. RNG source of a stock (npc unit seed? player record?) is
//     unknown; StockSeed is a deterministic stand-in.
//   - 0x629e40 "may be magic" test, and the quest group columns of npc.txt
//     (which of questbuymult / questsellmult hits the player-pays side).
//   - Gamble: DifficultyLevels Gamble* values come from difficultylevels.bin
//     (the .txt has no such columns), see ShippedGambleParams.
