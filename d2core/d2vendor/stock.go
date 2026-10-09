package d2vendor

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

const (
	// GridCols and GridRows are the size of the vendor's own storage
	// ("Monster" record of Inventory.txt, VERIFIED: 10 x 10 cells).
	GridCols = 10
	GridRows = 10

	// maxFailures is the number of failed placements after which generation
	// stops (VERIFIED constant 0x20 in the decompilation; that it is a global
	// counter rather than per base item is UNVERIFIED).
	maxFailures = 0x20

	// regularItemLevelLimit: from item level 25 on, the regular pass creates
	// nothing and only the magic pass runs (VERIFIED).
	regularItemLevelLimit = 25

	// ilvlBonus is added to the player level to get the vendor item level
	// (VERIFIED: playerLevel + 5, then capped by ItemLevel).
	ilvlBonus = 5

	// Quality thresholds of FUN_00574710 (VERIFIED from the decompilation).
	qualityRollRange = 100
	lowItemLevel     = 5
	midItemLevel     = 10
	lowRollAbove     = 0x5a // ilvl < 5: roll > 90 gives quality 1 (low)
	midRollAbove     = 0x55 // ilvl < 10: roll > 85 gives quality 3 (superior)
	highRollAbove    = 0x4a // ilvl >= 10: roll > 74 gives quality 3 (superior)

	magicLevelExtraRoll = 3
)

// capByTier are the item level caps of FUN_005746a0 (Game.exe 1.14b
// 0x5746a0), VERIFIED: 12, 20, 28, 36, 45. The index is the per-vendor record
// byte at +0x22, which is the 0-based act of the vendor's town (the town
// transition handler 0x534d40 -> 0x534c20 matches that byte against the act
// it derives from the town level ids 1/40/75/103/109). The cap is applied
// only on Normal difficulty (game byte +0x6d == 0) and only for an index
// below 5; the result is min(playerLevel+5, cap). Callers pass
// Options.Tier = ActIndex(vendor), or -1 for no cap.
//
//nolint:gochecknoglobals // constant table
var capByTier = [...]int{0xc, 0x14, 0x1c, 0x24, 0x2d}

// Params are the "<Npc>Min/Max/MagicMin/MagicMax/MagicLvl" columns of one
// base item for one vendor.
type Params struct {
	Min, Max           int
	MagicMin, MagicMax int
	MagicLevel         int
}

// Base is one base item a vendor may stock.
type Base struct {
	Code     string
	ReqLevel int // required level (record +0xfd)
	W, H     int // inventory size in cells
	Vendor   Params

	// Permanent is PermStoreItem: the item is always for sale, never rolled.
	Permanent bool
	// Gear marks armour and weapons; only those get quality rolls other
	// than normal (UNVERIFIED for misc items, treated as always normal).
	Gear bool
	// CanBeMagic stands for ITEM_TestBitfield1Flag1 (0x629e40): bit 0 of the
	// base record's load-time flag word at +0xdc (VERIFIED; a derived flag,
	// not a txt column). Which item types set it is UNVERIFIED; the records
	// adapter uses "not an always normal item type and armour or weapon" and
	// the oracle test checks that every vendor magic column sits on such an
	// item.
	CanBeMagic bool
	// Version is the items.txt version column (+0xf6): 0 classic, 100
	// expansion. VERIFIED: an item with Version >= 100 is skipped (both the
	// regular and the magic pass) unless the game is an expansion game
	// (game word +0x78 >= 100), see Options.Classic.
	Version int
	// Uber, Ultra, NightmareUpgrade and HellUpgrade are the item's
	// exceptional / elite versions and its per-difficulty replacements
	// (+0x88, +0x8c, +0x19c, +0x1a0, VERIFIED use in FUN_00574110).
	Uber, Ultra, NightmareUpgrade, HellUpgrade string

	// Ammo is arrows and bolts ("aqv ", "cqv "): sold as a full stack.
	Ammo     bool
	MaxStack int
}

// Item is one item of the vendor's stock.
type Item struct {
	Code      string
	Quality   d2drop.Quality
	ILvl      int
	Quantity  int // 0 when not set (the item factory decides)
	Permanent bool
	// SoldByPlayer marks items the player sold; they stay until the session
	// ends (VERIFIED, flag 0x10 in unit+0xc8).
	SoldByPlayer bool

	W, H int
	X, Y int // top-left cell in the vendor grid

	// Payload lets the UI attach the realised item object.
	Payload interface{}
}

// Options are the inputs of Generate.
type Options struct {
	PlayerLevel int
	// Tier is the index into the item level cap table: the 0-based act of
	// the vendor's town (see ActIndex); negative = no cap. Ignored (no cap)
	// when Difficulty != 0.
	Tier int
	// Difficulty is 0 normal, 1 nightmare, 2 hell. VERIFIED effects: the cap
	// table applies on normal only, and from player level 26 on nightmare
	// and hell vendors upgrade items (see upgradeCode).
	Difficulty int
	// Classic marks a classic (non-expansion) game: items with Version >= 100
	// are not stocked (VERIFIED). The zero value is an expansion game.
	Classic bool
	// EliteUpgrade is game field +0x70, which enables the hell-only
	// elite (ultracode) upgrade (VERIFIED as a gate, its meaning is not).
	EliteUpgrade bool
	// Resolve returns the base data of an upgraded code (size, codes). When
	// nil or when it does not know the code, an upgrade is skipped.
	Resolve func(code string) (Base, bool)
}

// Stock is a vendor inventory.
type Stock struct {
	Items []*Item
	grid  *d2inventory.OccupancyGrid
}

// NewStock creates an empty stock grid.
func NewStock() *Stock {
	return &Stock{grid: d2inventory.NewOccupancyGrid(GridCols, GridRows)}
}

// Place puts an item on the first free spot (the vendor owner uses first fit
// column-major, VERIFIED) and reports whether there was room.
func (s *Stock) Place(it *Item) bool {
	x, y, ok := s.grid.FindFreeSlot(it.W, it.H, false)
	if !ok {
		return false
	}

	it.X, it.Y = x, y
	s.grid.Fill(x, y, it.W, it.H, true)
	s.Items = append(s.Items, it)

	return true
}

// Remove takes an item out of the stock (a purchase: the vendor does not
// recreate it, VERIFIED).
func (s *Stock) Remove(it *Item) {
	for i, o := range s.Items {
		if o == it {
			s.grid.Fill(it.X, it.Y, it.W, it.H, false)
			s.Items = append(s.Items[:i], s.Items[i+1:]...)

			return
		}
	}
}

// ItemAt returns the item covering a grid cell, or nil.
func (s *Stock) ItemAt(x, y int) *Item {
	for _, it := range s.Items {
		if x >= it.X && x < it.X+it.W && y >= it.Y && y < it.Y+it.H {
			return it
		}
	}

	return nil
}

// ItemLevel is the item level of everything a vendor creates for a player
// of the given level (VERIFIED: FUN_005746a0(level+5)); a negative tier
// means "no cap".
func ItemLevel(playerLevel, tier int) int {
	ilvl := playerLevel + ilvlBonus

	if tier >= 0 && tier < len(capByTier) && ilvl > capByTier[tier] {
		return capByTier[tier]
	}

	return ilvl
}

// RollQuality is FUN_00574710 (VERIFIED): the quality of a regular vendor
// item. Below item level 5 there is a 9% chance of low quality, otherwise
// 14% (ilvl 5..9) or 25% (ilvl >= 10) of superior, else normal.
func RollQuality(rng d2drop.RNG, ilvl int) d2drop.Quality {
	r := int(rng.Roll(qualityRollRange))

	switch {
	case ilvl < lowItemLevel:
		if r > lowRollAbove {
			return d2drop.QualityLow
		}
	case ilvl < midItemLevel:
		if r > midRollAbove {
			return d2drop.QualitySuperior
		}
	default:
		if r > highRollAbove {
			return d2drop.QualitySuperior
		}
	}

	return d2drop.QualityNormal
}

// upgradeRange is the modulus of the difficulty upgrade roll (VERIFIED:
// 0x186a0 in FUN_00574110), upgradeMinPlayerLevel the first player level
// with upgrades (VERIFIED: player level > 25).
const (
	upgradeRange          = 100000
	upgradeMinPlayerLevel = 26
)

func validCode(c string) bool {
	c = strings.TrimSpace(c)

	return c != "" && c != "xxx" && c != "0"
}

// upgradeCode is the difficulty rule of FUN_00574110 (VERIFIED), applied to
// every created vendor item when Difficulty > 0 and the player level is 26 or
// more, with one Roll(100000) per created item:
//
//   - nightmare: roll < ilvl*64+4000 and an exceptional version exists:
//     that; otherwise the NightmareUpgrade code if there is one;
//   - hell: with EliteUpgrade set, roll < ilvl*16+1000 and an elite version
//     exists: that; else roll < ilvl*128+5000 and an exceptional version
//     exists: that; afterwards a HellUpgrade code, if any, replaces the
//     result. This is how healing and mana potions of tier 4 / 5 appear:
//     hp1..hp3 -> hp4 (NightmareUpgrade) -> hp5 (HellUpgrade).
//
// It returns the code to create (b.Code when nothing changes).
func upgradeCode(rng d2drop.RNG, b *Base, opt Options, ilvl int) string {
	if opt.Difficulty <= 0 || opt.PlayerLevel < upgradeMinPlayerLevel {
		return b.Code
	}

	r := int(rng.Roll(upgradeRange))

	if opt.Difficulty == 1 {
		if r < ilvl*64+4000 && validCode(b.Uber) {
			return b.Uber
		}

		if validCode(b.NightmareUpgrade) {
			return b.NightmareUpgrade
		}

		return b.Code
	}

	code := b.Code

	switch {
	case opt.EliteUpgrade && r < ilvl*16+1000 && validCode(b.Ultra):
		code = b.Ultra
	case r < ilvl*128+5000 && validCode(b.Uber):
		code = b.Uber
	}

	if validCode(b.HellUpgrade) {
		code = b.HellUpgrade
	}

	return code
}

// rollRange is CTRL_Helper_4b8e00 (0x4b8e00): a number in [lo, hi); lo
// without drawing when hi <= lo. VERIFIED.
func rollRange(rng d2drop.RNG, lo, hi int) int {
	if hi <= lo {
		return lo
	}

	return lo + int(rng.Roll(int32(hi-lo)))
}

// extraMagic is the amount added to MagicMax (VERIFIED): 1 below item level
// 25, else 2 or 3 (random(1..2)+1, one roll).
func extraMagic(rng d2drop.RNG, ilvl int) int {
	if ilvl < regularItemLevelLimit {
		return 1
	}

	return rollRange(rng, 1, magicLevelExtraRoll) + 1
}

// VendorSells is TRADE_CheckVendorSellsItem (0x574cf0, VERIFIED): whether the
// vendor counts as selling code, which the server's buy and sell paths use
// (the vendor does not take such an item back). That is the vendor's
// permanent list, and on any non-normal difficulty also the tier 4 and 5
// potions hp4, hp5, mp4 and mp5 (which the difficulty upgrade can put on the
// shelves, see upgradeCode).
func VendorSells(difficulty int, code string, permanent []string) bool {
	if difficulty != 0 {
		switch code {
		case "hp4", "hp5", "mp4", "mp5":
			return true
		}
	}

	for _, p := range permanent {
		if p == code {
			return true
		}
	}

	return false
}

// ActIndex is the 0-based act of a vendor's town, the index into the item
// level cap table; -1 for a vendor of no modelled act.
func ActIndex(v Vendor) int {
	for i, l := range [][]Vendor{Act1, Act2, Act3, Act4, Act5} {
		for _, o := range l {
			if o.ClassID == v.ClassID {
				return i
			}
		}
	}

	return -1
}

// Generate builds a stock. bases should be in a stable order (the original's
// table order is UNVERIFIED; the records adapter sorts by code).
//
// VERIFIED against 0x574780 (see doc.go): both Min and Max columns are read,
// the regular count is Min + random(Max+1-Min) (Min when Max < Min), the magic
// count is MagicMin + random(MagicMax+extra-MagicMin) with extra 1 below item
// level 25 and 2..3 from there on; the regular and the magic pass both need
// ReqLevel <= ilvl and an item version the game accepts; the magic pass
// additionally needs the may-be-magic flag and MagicLvl <= ilvl. All rolls use
// the game's one shared generator.
//
// UNVERIFIED simplifications: permanent items are excluded from the regular
// and magic passes (assumed to live in a separate list); permanent items other
// than ammo get no quantity here; permanent items are created first (the
// original creates them last) because the single 10x10 grid would otherwise
// crowd them out.
func Generate(rng d2drop.RNG, bases []Base, opt Options) *Stock {
	s := NewStock()

	tier := opt.Tier
	if opt.Difficulty != 0 {
		tier = -1 // VERIFIED: the cap table only applies on normal difficulty
	}

	ilvl := ItemLevel(opt.PlayerLevel, tier)
	failures := 0

	create := func(b *Base, q d2drop.Quality, perm bool) bool {
		it := &Item{Code: b.Code, Quality: q, ILvl: ilvl, W: b.W, H: b.H, Permanent: perm}

		if code := upgradeCode(rng, b, opt, ilvl); code != b.Code && opt.Resolve != nil {
			if nb, ok := opt.Resolve(code); ok {
				it.Code, it.W, it.H = code, nb.W, nb.H
				b = &nb
			}
		}

		if perm && b.Ammo {
			it.Quantity = b.MaxStack // VERIFIED: full stack
		}

		if !s.Place(it) {
			failures++

			return failures < maxFailures
		}

		return true
	}

	for i := range bases {
		b := &bases[i]
		if b.Permanent && b.Vendor.Max > 0 {
			if !create(b, d2drop.QualityNormal, true) {
				return s
			}
		}
	}

	for i := range bases {
		b := &bases[i]
		if b.Permanent || b.ReqLevel > ilvl || (opt.Classic && b.Version >= 100) {
			continue
		}

		if ilvl < regularItemLevelLimit {
			n := rollRange(rng, b.Vendor.Min, b.Vendor.Max+1)
			for ; n > 0; n-- {
				q := RollQuality(rng, ilvl)
				if !b.Gear {
					q = d2drop.QualityNormal
				}

				if !create(b, q, false) {
					return s
				}
			}
		}

		if b.CanBeMagic && b.Vendor.MagicLevel <= ilvl {
			n := rollRange(rng, b.Vendor.MagicMin, b.Vendor.MagicMax+extraMagic(rng, ilvl))
			for ; n > 0; n-- {
				if !create(b, d2drop.QualityMagic, false) {
					return s
				}
			}
		}
	}

	return s
}

// StockSeed derives the generator seed of one vendor's stock from the game's
// vendor seed, so that every vendor of a game has its own stream (previously
// the same seed served all of them) and a restock with the same restock count
// reproduces. restock is the number of restocks so far (0 for the first
// stock), gamble selects the gamble stock of the same vendor. The original
// seeds from the NPC unit / player record; how exactly is UNVERIFIED (see the
// open addresses in the package notes), this is a deterministic stand-in.
func StockSeed(gameSeed uint32, classID int, restock uint32, gamble bool) uint32 {
	const (
		classMix  = 0x9E3779B1
		gambleMix = 0x5BD1E995
	)

	s := gameSeed ^ uint32(classID)*classMix
	if gamble {
		s ^= gambleMix
	}

	g := d2rand.New(s)
	g.Step()

	return g.Step() + restock
}

// GenerateSeeded is Generate with a fresh generator.
func GenerateSeeded(seed uint32, bases []Base, opt Options) *Stock {
	return Generate(d2rand.New(seed), bases, opt)
}
