package d2trade

// Mode selects which price TRADE_CalcItemPrice returns.
type Mode int

// Server-side price modes (the client UI notes had 0/1 reversed).
const (
	ModeBuy    Mode = 0 // player buys from a vendor
	ModeSell   Mode = 1 // player sells to a vendor
	ModeGamble Mode = 2 // gamble purchase
	ModeRepair Mode = 3 // repair
)

const (
	// Invalid is the original's 0x7fffffff for unusable inputs.
	Invalid = 0x7fffffff
	// IdentifyCostPerItem is Cain's fixed fee per unidentified item.
	IdentifyCostPerItem = 100

	fixedShift   = 10 // multipliers are 1/1024 fixed point
	overflowEdge = 65536
	maxReduction = 99
)

// QualityLow is the item quality that halves every price.
const QualityLow = 1

// Term is one (multiply, add) pair from an affix, set item, unique or
// ItemStatCost row. Mult is 1/1024 fixed point.
type Term struct {
	Mult int
	Add  int
}

// Item holds everything the price function reads from an item unit and its
// base record. Fields marked unverified follow the notes' hypotheses.
type Item struct {
	BaseCost int // cost column of the base item record
	Quantity int // stat 0x46 (min 1 is applied)
	MaxStack int // base max stack + stat 0xfe, clamped to 511 (min 1 applied)
	Level    int // ear level (ears only)

	Starter    bool // flag 0x20000: always costs 1
	Identified bool // flag 0x10
	Ethereal   bool // flag 0x400000
	IsEar      bool // flag 0x10000

	Quality int // QualityLow halves all prices

	// IsAmmo is arrows and bolts (ItemTypes "Quiver" non-zero). BaseCost is
	// in 1/1024 gold per unit (misc.txt gives 256: a quarter gold).
	IsAmmo bool

	// Tome (type 0x12): TomeBase is the precomputed base (tome cost plus
	// quantity times scroll cost); the tables behind it are unidentified.
	IsTome   bool
	TomeBase int

	// Body part (type 0x28): table driven in the original (unverified).
	IsBodyPart   bool
	BodyPartBase int

	// Armour scaling: when MaxAC != 0 the base is cost*Defense/MaxAC.
	IsArmour bool
	Defense  int
	MaxAC    int

	Stackable bool // base record stackable byte (+0x132)

	// Terms are applied in order to the buy, sell and repair accumulators.
	// Only used when Identified.
	Terms []Term
	// SocketedHalfCost is the sum of base_cost/2 of socketed items.
	SocketedHalfCost int

	// ClassSpecificType is the former open question: the ItemTypes byte at
	// +0x21 tested as "< 7" by FUN_0062c1c0. Checked in Ghidra: the
	// ItemTypes loader spec maps that offset to the "class" column (a
	// character-class code lookup that returns -1, stored as 0xff, for a
	// blank). So it is true only when the item's own ItemTypes row has a
	// non-empty Class (Amazon Bow, Orb, Hand to Hand, ...), NOT for ordinary
	// items; for those, sell value is cost/4 (cost/16 if ethereal), not /8.
	// Parent types are not consulted (FUN_0062b590 yields the item's own row).
	ClassSpecificType bool

	HasDurability bool
	MaxDur        int
	CurDur        int
	Replenishes   bool // stat 0xfc: missing amount measured against MaxDur-1
	Throwable     bool // throwing weapons: quantity instead of durability
	Repairable    bool // TRADE_CheckItemRepairable
	RechargeCost  int  // TRADE_CalcRechargeCost, non-ethereal items

	// StackableRepairable is (base stackable && Repairable): throwing
	// weapons. How ammo behaves in the sell branch is unverified.
	StackableRepairable bool
	ReplenishQty        bool // stat 0xfd
}

// NPC is one npc.txt row in the loader's memory order: the "sell mult"
// column (what the player pays) is stored first, "buy mult" second, the
// reverse of the file's column order. All multipliers are 1/1024.
type NPC struct {
	PlayerPays int // "sell mult" (loader +4): applied to the buy price
	VendorPays int // "buy mult"  (loader +8): applied to the sell price
	Repair     int // "rep mult"  (loader +0xc)
	Quest      [3]Quest
	MaxBuy     [3]int // cap on the sell value per difficulty (0 = none)
}

// Quest is one quest-group override; Active is the quest-bit test result.
type Quest struct {
	Active bool
	Buy    int // multiplies the buy price (questsellmult)
	Sell   int // multiplies the sell price (questbuymult)
	Repair int // multiplies the repair price (questrepmult)
}

// Params are the non-item inputs.
type Params struct {
	Mode          Mode
	Difficulty    int // 0..2, indexes NPC.MaxBuy
	ReducedPrices int // player stat 0x57, clamped to 99
	PlayerLevel   int
	NPC           NPC

	Gamble           Gamble // used for ModeGamble
	GambleFlat       bool   // itemData word +0x30 == 0 (meaning unverified)
	GambleCostColumn int    // base record gamble cost, used when GambleFlat
}

// mulFixed returns x*mult/1024 with the original's overflow guard.
func mulFixed(x, mult int) int {
	if x >= overflowEdge {
		return (x >> fixedShift) * mult
	}

	return x * mult >> fixedShift
}

// applyTerm is x += (mult*x>>10) + add with the overflow guard.
func applyTerm(x int, t Term) int {
	if x >= overflowEdge {
		return x + (x>>fixedShift)*t.Mult + t.Add
	}

	return x + (t.Mult * x >> fixedShift) + t.Add
}

func discount(price, pct int) int {
	if pct > maxReduction {
		pct = maxReduction
	}

	if pct <= 0 {
		return price
	}

	return price - price*pct/100
}

func atLeastOne(v int) int {
	if v < 1 {
		return 1
	}

	return v
}

// ItemPrice implements TRADE_CalcItemPrice for modes 0..3.
func ItemPrice(it *Item, p Params) int {
	if it == nil {
		return Invalid
	}

	if p.Mode == ModeRepair && !it.Repairable {
		return 0
	}

	if it.Starter {
		return 1
	}

	qty := maxInt(it.Quantity, 1)
	maxStack := maxInt(it.MaxStack, 1)

	if p.Mode == ModeGamble {
		price := p.GambleCostColumn
		if !p.GambleFlat {
			price = GamblePrice(p.PlayerLevel, p.Gamble)
		}
		// Whether the discount also applies to the flat cost is unverified.
		return discount(price, p.ReducedPrices)
	}

	a, b, c, d := basePrices(it, qty, maxStack)

	if it.Identified {
		if it.Quality == QualityLow {
			a -= a / 2
			b -= b / 2
			c -= c / 2
		}

		for _, t := range it.Terms {
			a, b, c = applyTerm(a, t), applyTerm(b, t), applyTerm(c, t)
		}

		a, b, c = a/d+it.SocketedHalfCost, b/d+it.SocketedHalfCost, c/d+it.SocketedHalfCost
	}

	if it.Ethereal {
		b >>= 2
	}

	if it.ClassSpecificType {
		b >>= 2
	}

	if p.Mode == ModeSell && it.Ethereal && it.HasDurability && it.CurDur < 1 {
		b = 0
	}

	if p.Mode == ModeRepair {
		c = repairBase(it, c)
	}

	n := p.NPC
	a, b, c = mulFixed(a, n.PlayerPays), mulFixed(b, n.VendorPays), mulFixed(c, n.Repair)

	for _, q := range n.Quest {
		if q.Active {
			a, b, c = mulFixed(a, q.Buy), mulFixed(b, q.Sell), mulFixed(c, q.Repair)
		}
	}

	if !it.IsTome && !it.IsAmmo {
		a *= qty
	}

	switch {
	case !it.StackableRepairable:
		b *= qty
	case qty < maxStack && !it.ReplenishQty:
		c *= maxStack - qty
		b = maxStack*b - c
	default:
		b *= maxStack
		c = 0
	}

	if p.Difficulty >= 0 && p.Difficulty < len(n.MaxBuy) {
		if limit := n.MaxBuy[p.Difficulty]; limit > 0 && b > limit {
			b = limit
		}
	}

	switch p.Mode {
	case ModeSell:
		return atLeastOne(b)
	case ModeRepair:
		return atLeastOne(discount(c, p.ReducedPrices))
	default:
		return atLeastOne(discount(a, p.ReducedPrices))
	}
}

// basePrices returns the three accumulators (buy, sell, repair) and the stack divisor.
func basePrices(it *Item, qty, maxStack int) (a, b, c, d int) {
	d = 1

	switch {
	case it.IsEar:
		a = (it.Level & 0xff) * it.BaseCost
	case it.IsBodyPart:
		a = it.BodyPartBase
	case it.IsTome:
		a = it.TomeBase
	case it.IsAmmo:
		a = (it.BaseCost * qty) >> fixedShift
		return a, a, (it.BaseCost * maxStack) >> fixedShift, d
	default:
		a = it.BaseCost
		if it.Stackable && qty > 1 {
			d = maxStack
		}

		if it.IsArmour && it.MaxAC != 0 {
			a = it.BaseCost * it.Defense / it.MaxAC
		}
	}

	return a, a, a, d
}

// repairBase applies the durability fraction and recharge cost to c.
func repairBase(it *Item, c int) int {
	if it.IsAmmo || it.Throwable || !it.HasDurability {
		return c
	}

	if it.MaxDur == 0 || it.MaxDur <= it.CurDur {
		c = 0
	} else {
		maxDur := it.MaxDur
		if it.Replenishes {
			maxDur--
		}

		c = maxInt(0, c*(maxDur-it.CurDur)/it.MaxDur)
	}

	if !it.Ethereal {
		c += it.RechargeCost
	}

	return c
}

// IdentifyCost is Cain's charge: 100 gold per unidentified item, free once
// the quest flag (state 4) is set.
func IdentifyCost(unidentified int, questDone bool) int {
	if questDone || unidentified <= 0 {
		return 0
	}

	return unidentified * IdentifyCostPerItem
}
