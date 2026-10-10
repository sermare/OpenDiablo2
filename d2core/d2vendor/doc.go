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

// Audit status. Everything below was checked in Game.exe 1.14b (see
// d2-re-notes/verify-vendor.md; verify_test.go and oracle_test.go pin it, the
// real-table tests need D2_TABLES).
//
// VERIFIED:
//   - 0x5746a0 item level cap (12/20/28/36/45): indexed by the 0-based act of
//     the vendor's town (ActIndex), applied as min(level+5, cap) on Normal
//     difficulty only.
//   - 0x574780 reads Min and Max (regular count Min+random(Max+1-Min)) and
//     MagicMin and MagicMax (MagicMin+random(MagicMax+extra-MagicMin), extra 1
//     below ilvl 25 else 2..3). ReqLevel <= ilvl and the item version gate
//     wrap both passes; the magic pass also needs the may-be-magic bit and
//     MagicLvl <= ilvl.
//   - 0x574cf0 VendorSells: hp4/hp5/mp4/mp5 count as sold on non-normal
//     difficulty; the potions arrive through the difficulty upgrade of
//     0x574110 (NightmareUpgrade / HellUpgrade, player level >= 26).
//   - 0x534c20 restocks flag a vendor when the hero LEAVES town and the last
//     stock is more than 240000 ms old; consumed at the next window open.
//   - RNG: the game's single shared generator (0x534510 returns game+0x1d24+8),
//     not a per-NPC seed. The quality, count and upgrade rolls all use it.
//   - 0x629e40 is bit 0 of the load-time flag word at base record +0xdc.
//   - npc.txt quest columns: the player pays with "questsellmult", the vendor
//     pays with "questbuymult"; a group needs a non-zero questflag.
//   - Cain (0x576290): 100 gold per unidentified item, free with quest flag 4.
//
// UNVERIFIED:
//   - which item types set bit 0 of +0xdc (the adapter's rule is consistent
//     with every vendor magic column on the real tables);
//   - whether PermStoreItem rows are really absent from the Min/Max pass, and
//     the original table order (the adapter sorts by code, permanents first);
//   - the meaning of game field +0x70 (gate of the hell elite upgrade,
//     Options.EliteUpgrade) and the difficulty source in the trade window;
//   - StockSeed is a stand-in: the original consumes the shared game stream, so
//     a stock depends on everything rolled before it.
//   - Gamble: DifficultyLevels Gamble* values come from difficultylevels.bin
//     (the .txt has no such columns), see ShippedGambleParams.
