package d2vendor

import (
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Gamble stock (Gheed, Elzix, Alkor, Drehya/Anya, Jamella). All of this is
// read from Game.exe 1.14b (FUN_005765d0, FUN_005764c0, FUN_00639a60) and
// marked VERIFIED unless stated.
//
// What the real game does, summarised:
//   - the gamble stock is built per player and per vendor the first time the
//     gamble window is opened and kept until the vendor is restocked (the
//     four minute rule, see RestockInterval);
//   - it holds exactly 14 items (the loop stops after the 14th creation or at
//     the first item that does not fit): slot 0 is always a ring, slot 1 is
//     always an amulet, slots 2..13 are random base items of gamble.txt;
//   - every item gets its own item level, player level - 5 + rand(10) clamped
//     to 5..99, and its own pick from the gamble.txt rows whose level
//     requirement is <= that item level (the table is sorted by level);
//   - expansion only: the base item may be replaced by its exceptional or
//     elite version (DifficultyLevels GambleUber / GambleUltra);
//   - the quality is magic unless the 100000-sided roll hits the
//     GambleUnique / GambleSet / GambleRare windows of DifficultyLevels;
//   - the item is created unidentified, so the player sees the base name
//     until it is bought (the client then sends packet 0x37 and the item is
//     revealed; this engine reveals it in Buy).

const (
	// GambleItems is the number of items of a gamble stock (VERIFIED: the
	// creation counter stops when it exceeds 13).
	GambleItems = 14

	// RestockInterval is the vendor restock period in milliseconds
	// (VERIFIED, Game.exe 0x534c20): FUN_00534c20 is not a per-frame loop; it
	// runs from SERVER_HandleTownTransition (0x534d40) when a player leaves a
	// town and, for each vendor record of that act, sets the "needs refresh"
	// flag (+0x27) when GetTickCount is more than 240000 ms past the vendor's
	// last stock (the tick stamped by 0x574780 at record +0x28). The flag is
	// consumed the next time a normal (non-gamble) trade window opens
	// (TRADE_OpenVendorSession 0x577240). Nothing restocks while the player
	// stays in town; see RestockDue.
	RestockInterval = 240000

	gambleMinILvl   = 5
	gambleMaxILvl   = 99
	gambleILvlSpan  = 10 // item level = level - 5 + rand(10)
	gambleILvlShift = 5

	// gambleQualityRange is the modulus of the quality roll (VERIFIED, 100000).
	gambleQualityRange = 100000
	// gambleUpgradeRange is the modulus of the exceptional / elite roll
	// (VERIFIED, 10000).
	gambleUpgradeRange = 10000
)

// GambleParams are the Gamble* columns of DifficultyLevels.txt for one
// difficulty (VERIFIED: struct offsets 0x44, 0x48, 0x4c, 0x50, 0x54).
type GambleParams struct {
	Rare, Set, Unique int // windows of the 100000-sided quality roll
	Uber, Ultra       int // per level chance (of 10000) of an exceptional / elite base
}

// GambleBase is one row of gamble.txt with what the roll needs.
type GambleBase struct {
	Code   string
	Level  int // level requirement (record byte +0xfd)
	Exc    string
	ExcLvl int
	Elite  string
	EliteL int
}

// GambleItem is one gamble stock entry.
type GambleItem struct {
	Code    string
	Quality d2drop.Quality
	ILvl    int
}

// SortGamblePool orders a pool the way the game does: by level requirement
// (VERIFIED, qsort in FUN_00639a60). The order of equal levels is whatever
// the original qsort left (UNVERIFIED); ties are ordered by code here.
func SortGamblePool(pool []GambleBase) {
	sort.SliceStable(pool, func(i, j int) bool {
		if pool[i].Level != pool[j].Level {
			return pool[i].Level < pool[j].Level
		}

		return pool[i].Code < pool[j].Code
	})
}

// GambleItemLevel rolls the item level of one gamble item (VERIFIED).
func GambleItemLevel(rng d2drop.RNG, playerLevel int) int {
	l := int(rng.Roll(gambleILvlSpan)) - gambleILvlShift + playerLevel

	switch {
	case l < gambleMinILvl:
		return gambleMinILvl
	case l > gambleMaxILvl:
		return gambleMaxILvl
	}

	return l
}

// GambleQuality is the quality roll of FUN_005765d0 (VERIFIED): magic by
// default; with r = rand(100000): r < Unique gives unique, r < Unique+Set set,
// r < Unique+Set+Rare rare. The shipped tables have Rare 10000, Set 100,
// Unique 50, so about one gamble item in ten is rare.
func GambleQuality(rng d2drop.RNG, p GambleParams) d2drop.Quality {
	sum := p.Rare + p.Set + p.Unique
	if sum <= 0 {
		return d2drop.QualityMagic
	}

	r := int(rng.Roll(gambleQualityRange))

	switch {
	case r >= sum:
		return d2drop.QualityMagic
	case r < p.Unique:
		return d2drop.QualityUnique
	case r < p.Unique+p.Set:
		return d2drop.QualitySet
	}

	return d2drop.QualityRare
}

// UpgradeGambleBase is FUN_005764c0 (VERIFIED, including the modulus 10000):
// the base item may be replaced by its exceptional version with chance
// ((ilvl - excLevel)*Uber + 1) / 10000 and, failing that, by its elite version
// with ((ilvl - eliteLevel)*Ultra + 1) / 10000 (the elite test only runs when
// the exceptional chance was positive, as in the original).
func UpgradeGambleBase(rng d2drop.RNG, b GambleBase, ilvl int, p GambleParams) string {
	if b.Exc == "" {
		return b.Code
	}

	c := (ilvl-b.ExcLvl)*p.Uber + 1
	if c <= 0 {
		return b.Code
	}

	if int(rng.Roll(gambleUpgradeRange)) < c {
		return b.Exc
	}

	if b.Elite == "" {
		return b.Code
	}

	c = (ilvl-b.EliteL)*p.Ultra + 1
	if c > 0 && int(rng.Roll(gambleUpgradeRange)) < c {
		return b.Elite
	}

	return b.Code
}

// GenerateGamble builds a gamble stock. pool must be sorted (SortGamblePool);
// ring and amulet are the bases of slots 0 and 1; expansion enables the
// exceptional / elite upgrades (the classic game skips them).
//
// Random call order, as in the original per item: item level, pick index
// (also drawn for the ring and amulet slots, whose pick is discarded), upgrade
// rolls, quality roll. UNVERIFIED: that the ring and amulet are also in the
// random pool (gamble.txt lists them), and the tie order of equal levels.
func GenerateGamble(rng d2drop.RNG, pool []GambleBase, ring, amulet GambleBase, p GambleParams,
	playerLevel int, expansion bool) []GambleItem {
	out := make([]GambleItem, 0, GambleItems)

	for len(out) < GambleItems {
		ilvl := GambleItemLevel(rng, playerLevel)

		n := sort.Search(len(pool), func(i int) bool { return pool[i].Level > ilvl })
		if n == 0 {
			return out
		}

		base := pool[rng.Roll(int32(n))]

		switch len(out) {
		case 0:
			base = ring
		case 1:
			base = amulet
		}

		code := base.Code
		if expansion {
			code = UpgradeGambleBase(rng, base, ilvl, p)
		}

		out = append(out, GambleItem{Code: code, Quality: GambleQuality(rng, p), ILvl: ilvl})
	}

	return out
}

// RestockDue reports whether a vendor whose stock was built elapsedMs ago is
// flagged for a restock by the town-leaving check (VERIFIED: strictly more
// than RestockInterval). A caller models the original by calling it when the
// hero leaves town and remembering the flag until the vendor is next opened.
func RestockDue(elapsedMs int64) bool { return elapsedMs > RestockInterval }
