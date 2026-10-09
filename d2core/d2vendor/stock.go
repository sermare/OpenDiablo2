package d2vendor

import (
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

// capByTier are the item level caps of FUN_005746a0, indexed by the per
// vendor record byte at +0x22 (VERIFIED values 12, 20, 28, 36, 45). What that
// byte counts (restock number? game progress?) is UNVERIFIED, so callers pass
// Options.Tier = -1 (no cap) unless they know.
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
	// CanBeMagic stands for FUN_00629e40 ("base item may be magic"). Its
	// exact test is UNVERIFIED; the records adapter uses "not an always
	// normal item type".
	CanBeMagic bool
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
	// Tier is the index into the item level cap table; negative = no cap.
	Tier int
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

// Generate builds a stock. bases should be in a stable order (the original's
// table order is UNVERIFIED; the records adapter sorts by code).
//
// UNVERIFIED simplifications: the Min and MagicMin columns are not read by
// the decompiled code that was followed (only byte[1], the maximum, is);
// permanent items are excluded from the regular and magic passes; permanent
// items other than ammo get no quantity here.
func Generate(rng d2drop.RNG, bases []Base, opt Options) *Stock {
	s := NewStock()
	ilvl := ItemLevel(opt.PlayerLevel, opt.Tier)
	failures := 0

	create := func(b *Base, q d2drop.Quality, perm bool) bool {
		it := &Item{Code: b.Code, Quality: q, ILvl: ilvl, W: b.W, H: b.H, Permanent: perm}

		if perm && b.Ammo {
			it.Quantity = b.MaxStack // VERIFIED: full stack
		}

		if !s.Place(it) {
			failures++

			return failures < maxFailures
		}

		return true
	}

	// The notes list the permanent items last. They go first here so that
	// the always-for-sale goods are never crowded out of the single 10x10
	// grid (the real window pages its stock by tab; that split is
	// UNVERIFIED and not modelled). Permanent creation draws no random
	// numbers, so the other rolls are unaffected.
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
		if b.Permanent {
			continue
		}

		if b.ReqLevel <= ilvl && ilvl < regularItemLevelLimit {
			n := int(rng.Roll(int32(b.Vendor.Max + 1)))
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
			extra := 1
			if ilvl >= regularItemLevelLimit {
				extra = int(rng.Roll(magicLevelExtraRoll)) + 1
			}

			n := int(rng.Roll(int32(b.Vendor.MagicMax + extra)))
			for ; n > 0; n-- {
				if !create(b, d2drop.QualityMagic, false) {
					return s
				}
			}
		}
	}

	return s
}

// GenerateSeeded is Generate with a fresh generator.
func GenerateSeeded(seed uint32, bases []Base, opt Options) *Stock {
	return Generate(d2rand.New(seed), bases, opt)
}
