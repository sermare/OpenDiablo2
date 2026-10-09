// Package d2playertrade is the player-to-player trade: the session both players
// negotiate (request, offers, accept, cancel) and the commit that moves items
// and gold between the two heroes' containers. It is pure: the server owns the
// sessions and runs Commit on the heroes' saved state (d2hero.HeroContainers),
// the clients only show what the server tells them.
//
// What is known of the original (inventory-trade.md): the trade window is the
// "Trade Page 1/2" panel records of inventory.txt (a 10x4 grid per side),
// item packet 0x18 puts an item on page 2 (the trade window) "which needs an
// open player trade", the cursor drop and the NPC buy are refused while a
// player trade is open. PlrTrade.cpp (0x562b30..0x5664e0) was NOT analysed, so
// the session rules below are modelled on the player-visible behaviour and are
// UNVERIFIED: request/accept-request, offers of inventory items and gold, every
// change of an offer revokes both acceptances, a short lock after a change
// (the accept button of the real window is disabled for a moment, the length
// here is a guess), both players accept the unchanged offers, the commit is
// atomic and fails (nothing moves) when an offered item or the gold is gone or
// a side has no room for what it receives.
package d2playertrade
