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

	overflowEdge = 0xffff // above this a multiplication divides first
	maxReduction = 99
)

// Item qualities the price code distinguishes.
const (
	QualityLow      = 1
	QualityNormal   = 2
	QualitySuperior = 3
	QualityMagic    = 4
	QualitySet      = 5
	QualityRare     = 6
	QualityUnique   = 7
	QualityCrafted  = 8
	QualityTempered = 9
)

// Term is one (multiply, add) pair from an affix, set item or unique row.
// Mult is 1/1024 fixed point.
type Term struct {
	Mult int
	Add  int
}

// Affixes are the price rows of an item's affixes. A nil pointer means that
// the row does not exist (id 0 or unknown). VERIFIED against the real
// function: which of them count depends on the quality (see ItemPrice).
type Affixes struct {
	Auto           *Term // automagic row (item data +0x36): counts for every quality
	Prefix, Suffix [3]*Term
	Unique, Set    *Term
}

// Item holds everything the price function reads from an item unit and its
// base record. Every field was checked against the real function (see the
// oracle test), except where a comment says otherwise.
type Item struct {
	BaseCost int // cost column of the base item record
	Quantity int // stat 0x46 (anything below 1 counts as 1)
	MaxStack int // base max stack + stat 0xfe, at most 511 (TRADE 6297b0)
	EarLevel int // level of a player ear (only for IsEar)

	Starter    bool // flag 0x20000: always costs 1
	Identified bool // flag 0x10
	Ethereal   bool // flag 0x400000
	IsEar      bool // flag 0x10000

	Quality int

	// IsAmmo is arrows and bolts (ItemTypes "Quiver" word non-zero). BaseCost is
	// in 1/1024 gold per unit (misc.txt gives 256: a quarter gold).
	IsAmmo bool

	// Tome (item type 0x12): base is cost + quantity * TomeScrollCost.
	IsTome         bool
	TomeScrollCost int

	// Body part (item type 0x28): base is cost + 8 * BodyPartWord, the word
	// being the monster record's (difficulty-indexed) price word. nil when the
	// item has no monster record.
	IsBodyPart   bool
	BodyPartWord *int

	// Armour (item type 0x32): when MaxAC-MinAC+1 != 0 and MaxAC != 0 the base
	// is cost*Defense/MaxAC.
	IsArmour  bool
	Defense   int
	MinAC     int
	MaxAC     int
	Stackable bool // base record stackable byte (+0x132)

	Affixes Affixes

	// The three adders the price function calls out to, each a delta for the
	// buy, sell and repair accumulators: ItemStatCost columns of every stat on
	// the item (TRADE_AddStatPropertyPrices), the single skill price
	// (TRADE_AddSingleSkillPrice) and the cost of socketed items
	// (TRADE_AddSocketedItemPrices). The package leaves their computation to
	// the caller.
	StatPrice, SkillPrice, SocketPrice [3]int

	// ClassSpecificType is the ItemTypes byte at +0x21 tested as "< 7" by
	// FUN_0062c1c0: true when the item's own ItemTypes row has a class
	// (Amazon Bow, Orb, Hand to Hand, ...); the sell value is then divided by 4.
	ClassSpecificType bool

	HasDurability bool // TRADE 629b00: base durability, not indestructible, max > 0
	MaxDur        int
	CurDur        int
	Replenishes   bool // stat 0xfc
	Throwable     bool // ItemTypes Throwable of the item's own type
	Repairable    bool // TRADE_CheckItemRepairable (62e770)
	RechargeCost  int  // TRADE_CalcRechargeCost, added to repairs of non-ethereal items
	ReplenishQty  bool // stat 0xfd
}

// NPC is one npc.txt row in the loader's memory order: the "sell mult"
// column (what the player pays) is stored first, "buy mult" second, the
// reverse of the file's column order. All multipliers are 1/1024.
type NPC struct {
	PlayerPays int // "sell mult" (loader +4): applied to the buy price
	VendorPays int // "buy mult"  (loader +8): applied to the sell price
	Repair     int // "rep mult"  (loader +0xc)
	Quest      [3]Quest
	// MaxBuy caps the sell value per difficulty. The original always applies
	// it (a row without a cap would sell everything for 1); here 0 means
	// "no cap" so that incomplete rows stay usable.
	MaxBuy [3]int
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
	ReducedPrices int // player stat 0x57, at most 99
	PlayerLevel   int
	NPC           NPC

	Gamble Gamble // used for ModeGamble
	// GambleFlat is true when the item data word +0x30 is below 1: the gamble
	// price is then the base record's gamble cost column, without any
	// discount (VERIFIED).
	GambleFlat       bool
	GambleCostColumn int // gamble cost of the item's normal-tier row, used when GambleFlat
}

// The original computes in 32-bit registers. w wraps a value to int32.
func w(x int) int { return int(int32(x)) }

// div1024 is (x + (x<0 ? 1023 : 0)) >> 10: division by 1024 truncating toward 0.
func div1024(x int) int {
	x32 := int32(x)

	return int((x32 + (x32>>31)&0x3ff) >> 10)
}

// mulFixed is x*mult/1024. VERIFIED guard: above 0xffff (and for a non-zero
// multiplier) the division comes first.
func mulFixed(x, mult int) int { return mulFixedGuard(x, mult, mult != 0) }

// mulFixedGuard is mulFixed with an explicit "multiplier is non-zero" test:
// the original tests the buy multiplier (a stale register) when it guards the
// sell multiplication.
func mulFixedGuard(x, mult int, nonZero bool) int {
	if x > overflowEdge && nonZero {
		return w(div1024(x) * mult)
	}

	return div1024(w(mult * x))
}

// termValue is the contribution of an affix row to an accumulator of base x;
// big says whether the BUY base exceeds 0xffff (that one decides for all
// three accumulators).
func termValue(x int, big bool, t Term) int {
	if big && t.Mult != 0 {
		return w(w(div1024(x)*t.Mult) + t.Add)
	}

	return w(div1024(w(t.Mult*x)) + t.Add)
}

// mulDiv is the reduced-price helper FUN_0047f2c0 (a*b/c with overflow
// handling), as used for the discounts.
func mulDiv(a, b, c int) int {
	a, b, c = w(a), w(b), w(c)

	switch {
	case c == 0:
		return 0
	case a > 0x100000:
		if c <= a>>4 {
			return w(w(a/c) * b)
		}

		return int(int32(int64(a) * int64(b) / int64(c)))
	case b > 0x10000:
		if c <= b>>4 {
			return w(w(b/c) * a)
		}

		return int(int32(int64(a) * int64(b) / int64(c)))
	}

	return w(a*b) / c
}

func discount(price, pct int) int {
	if pct > maxReduction {
		pct = maxReduction
	}

	if pct == 0 {
		return price
	}

	return w(price - mulDiv(price, pct, 100))
}

func atLeastOne(v int) int {
	if v < 1 {
		return 1
	}

	return v
}

// magicOrBetter is TRADE 62a290: quality 4 to 9.
func magicOrBetter(q int) bool { return q >= QualityMagic && q <= QualityTempered }

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

	qty := it.Quantity
	if qty < 1 {
		qty = 1
	}

	red := p.ReducedPrices
	if red > maxReduction {
		red = maxReduction
	}

	if p.Mode == ModeGamble {
		if p.GambleFlat {
			return p.GambleCostColumn
		}

		price := GamblePrice(p.PlayerLevel, p.Gamble)
		if red != 0 {
			price = w(price - mulDiv(price, red, 100))
		}

		return price
	}

	a, b, c, d := basePrices(it, qty)

	if !magicOrBetter(it.Quality) {
		a, b, c = w(a+it.SkillPrice[0]), w(b+it.SkillPrice[1]), w(c+it.SkillPrice[2])
	}

	if it.Identified {
		a, b, c = identifiedPrices(it, a, b, c, d)
	}

	a, b, c = w(a+it.SocketPrice[0]), w(b+it.SocketPrice[1]), w(c+it.SocketPrice[2])

	if it.Ethereal {
		b = w(b) / 4
	}

	if it.ClassSpecificType {
		b /= 4
	}

	switch p.Mode {
	case ModeRepair:
		if !it.IsAmmo && !it.Throwable && it.HasDurability {
			c = repairBase(it, c)
		}
	case ModeSell:
		if it.Ethereal && it.HasDurability && it.MaxDur != 0 && it.CurDur <= 0 {
			b = 0
		}
	}

	n := p.NPC
	// VERIFIED quirk: the guard of the sell multiplication tests the buy multiplier.
	a, b, c = mulFixed(a, n.PlayerPays), mulFixedGuard(b, n.VendorPays, n.PlayerPays != 0), mulFixed(c, n.Repair)

	for _, q := range n.Quest {
		if q.Active {
			a, b, c = mulFixed(a, q.Buy), mulFixed(b, q.Sell), mulFixed(c, q.Repair)
		}
	}

	if !it.IsTome && !it.IsAmmo {
		a = w(a * qty)

		if it.Stackable && it.Repairable {
			m := it.MaxStack

			if qty < m && !it.ReplenishQty {
				c = w(c * (m - qty))
				b = w(w(m*b) - c)
			} else {
				b, c = w(m*b), 0
			}
		} else {
			b = w(b * qty)
		}
	}

	if p.Mode == ModeRepair && !it.Ethereal {
		c = w(c + it.RechargeCost)
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
		return atLeastOne(discount(c, red))
	default:
		return atLeastOne(discount(a, red))
	}
}

// basePrices returns the buy, sell and repair accumulators and the divisor
// of the affix contributions (the max stack of stackable items).
func basePrices(it *Item, qty int) (a, b, c, d int) {
	d = 1

	switch {
	case it.IsEar:
		a = w((it.EarLevel & 0xff) * it.BaseCost)
		b = a
	case it.IsBodyPart:
		a = it.BaseCost

		if it.BodyPartWord != nil {
			a = w(a + 8**it.BodyPartWord)
		}

		b = a
	case it.IsTome:
		a = w(it.BaseCost + w(it.TomeScrollCost*qty))
		b = a
	case it.IsAmmo:
		a = div1024(w(it.BaseCost * qty))

		return a, a, div1024(w(it.BaseCost * it.MaxStack)), d
	default:
		a = it.BaseCost
		b, c = a, a

		if it.Stackable && it.MaxStack > 1 {
			d = it.MaxStack
		}
	}

	if it.IsArmour && it.MaxAC-it.MinAC+1 != 0 && it.MaxAC != 0 {
		a = w(it.BaseCost*it.Defense) / it.MaxAC
		b, c = a, a
	}

	return a, b, c, d
}

// identifiedPrices adds the contributions of the affixes, the stat prices
// and the single skill price to an identified item (everything is added to
// the base values, never compounded).
func identifiedPrices(it *Item, a, b, c, d int) (int, int, int) {
	var ta, tb, tc int

	big := a > overflowEdge
	add := func(t *Term) {
		if t == nil {
			return
		}

		ta, tb, tc = w(ta+termValue(a, big, *t)), w(tb+termValue(b, big, *t)), w(tc+termValue(c, big, *t))
	}

	add(it.Affixes.Auto)

	stat := true

	switch it.Quality {
	case QualityLow:
		ta, tb, tc = -(w(a) / 2), -(w(b) / 2), -(w(c) / 2)
		stat = false
	case QualityNormal:
		stat = false
	case QualityMagic:
		add(it.Affixes.Prefix[0])
		add(it.Affixes.Suffix[0])
	case QualitySet:
		// VERIFIED: the set and unique paths skip the stat prices.
		stat = false

		add(it.Affixes.Set)
	case QualityRare, QualityCrafted:
		for i := 0; i < 3; i++ {
			add(it.Affixes.Prefix[i])
			add(it.Affixes.Suffix[i])
		}
	case QualityUnique:
		if it.Affixes.Unique != nil {
			stat = false

			add(it.Affixes.Unique)
		} else {
			add(it.Affixes.Prefix[0])
			add(it.Affixes.Suffix[0])
		}
	case QualitySuperior, QualityTempered:
	default:
		stat = false
	}

	if stat {
		a, b, c = w(a+it.StatPrice[0]), w(b+it.StatPrice[1]), w(c+it.StatPrice[2])
	}

	a, b, c = w(a+w(ta)/d), w(b+w(tb)/d), w(c+w(tc)/d)

	if magicOrBetter(it.Quality) {
		a, b, c = w(a+it.SkillPrice[0]), w(b+it.SkillPrice[1]), w(c+it.SkillPrice[2])
	}

	return a, b, c
}

// repairBase scales the repair accumulator by the missing durability.
func repairBase(it *Item, c int) int {
	cur, max := it.CurDur, it.MaxDur

	if max == 0 || cur >= max {
		return 0
	}

	if it.Replenishes {
		// VERIFIED oddity: with the replenish stat the factor is (max-1)/max
		// whatever the current durability, and 0 once cur >= max-1.
		if cur >= max-1 {
			return 0
		}

		return w(c*(max-1)) / max
	}

	return w(c*(max-cur)) / max
}

// IdentifyCost is Cain's charge: 100 gold per unidentified item, free once
// the quest flag (state 4) is set.
func IdentifyCost(unidentified int, questDone bool) int {
	if questDone || unidentified <= 0 {
		return 0
	}

	return unidentified * IdentifyCostPerItem
}
