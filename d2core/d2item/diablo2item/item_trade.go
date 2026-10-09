package diablo2item

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

var errUnknownItemCode = errors.New("unknown item code")

// ItemFromCode creates an item of a base code with a given quality and item
// level, the way a vendor does (quality chosen by the caller). Equal
// arguments give equal items.
func (f *ItemFactory) ItemFromCode(code string, q d2drop.Quality, ilvl int, seed uint32) (*Item, error) {
	if f.asset.Records.Item.All[code] == nil {
		return nil, fmt.Errorf("%w: %q", errUnknownItemCode, code)
	}

	item, err := f.Create(CreateParams{Code: code, ILvl: ilvl, Quality: q, Seed: seed, Difficulty: f.Difficulty, Classic: f.Classic})
	if err != nil {
		if !errors.Is(err, errNoCreator) {
			return nil, err
		}

		rng := d2rand.New(seed)

		if item = f.legacyItemFromDrop(rng, &d2drop.Drop{Code: code, Quality: q, ILvl: ilvl}); item == nil {
			return nil, fmt.Errorf("%w: %q", errUnknownItemCode, code)
		}

		item.quality = q
	}

	return item, nil
}

// Quality returns the quality the item was created with (QualityNone if it
// was not created through ItemFromCode / DropItems).
func (i *Item) Quality() d2drop.Quality {
	return i.quality
}

// IsIdentified reports whether the item is identified.
func (i *Item) IsIdentified() bool {
	return i.attributes != nil && i.attributes.identitified
}

// Quantity returns the stack size (0 when unset).
func (i *Item) Quantity() int {
	return i.attributes.currentStackSize
}

// SetQuantity sets the stack size.
func (i *Item) SetQuantity(n int) {
	i.attributes.currentStackSize = n
}

// Durability returns the current and maximum durability; max is 0 for items
// without durability.
func (i *Item) Durability() (current, maximum int) {
	if !i.attributes.durable {
		return 0, 0
	}

	return i.attributes.currentDurability, i.attributes.durability.max
}

// IsIndestructible reports the "indestructible" property (stat 152): the item
// never loses durability.
func (i *Item) IsIndestructible() bool {
	return i.attributes != nil && i.attributes.indestructable
}

// SetDurability sets the current durability.
func (i *Item) SetDurability(n int) {
	i.attributes.currentDurability = n
}

// InventoryFileName returns the DC6 name of the item's inventory picture
// ("invrbk" for a tome of identify), without directory or extension; empty
// when the record has none.
func (i *Item) InventoryFileName() string {
	name := i.CommonRecord().InventoryFile
	if len(name) < len("inv") {
		return ""
	}

	return name
}

// TradeItem converts the item to the input of d2trade.ItemPrice.
//
// Not covered (price terms are left out rather than invented): the
// ItemStatCost per-stat price columns, the single skill price, the socketed
// items' cost, ethereal (the factory does not roll it), the recharge cost of
// charged items, tome and body part tables, automagic rows and gamble flags.
// Magic affixes use PriceScale/PriceAdd (1/1024 fixed point, checked against
// MagicPrefix.txt); unique and set rows are not priced.
func (i *Item) TradeItem() *d2trade.Item {
	rec := i.CommonRecord()
	typ := i.factory.asset.Records.Item.Types[rec.Type] // TypeCode is only set for dropped items

	maxStack := rec.MaxStack
	cur, maxDur := i.Durability()

	it := &d2trade.Item{
		BaseCost:   rec.Cost,
		Quantity:   i.Quantity(),
		MaxStack:   maxStack,
		Identified: i.IsIdentified(),
		Ethereal:   i.attributes.ethereal,
		Quality:    int(i.quality),
		Stackable:  rec.Stackable,
		IsArmour:   rec.MaxAC > 0 && rec.Source == d2enum.InventoryItemTypeArmor,
		Defense:    i.attributes.defense,
		MinAC:      rec.MinAC,
		MaxAC:      rec.MaxAC,

		HasDurability: maxDur > 0,
		MaxDur:        maxDur,
		CurDur:        cur,
		Throwable:     rec.Throwable,

		// ClassSpecificType: see the field's documentation in d2trade; the
		// item's own ItemTypes row has a class.
		ClassSpecificType: classItemTypes[rec.Type],
	}

	if typ != nil {
		it.IsAmmo = typ.Quiver != ""
		it.Repairable = typ.Repair && maxDur > 0 && it.Identified && !it.Ethereal
	}

	if !it.Identified {
		return it
	}

	// The price function picks the affix rows by quality (one prefix and one
	// suffix for magic items, three of each for rare and crafted ones).
	for n, p := range i.PrefixRecords() {
		if n < len(it.Affixes.Prefix) {
			it.Affixes.Prefix[n] = &d2trade.Term{Mult: p.PriceScale, Add: p.PriceAdd}
		}
	}

	for n, s := range i.SuffixRecords() {
		if n < len(it.Affixes.Suffix) {
			it.Affixes.Suffix[n] = &d2trade.Term{Mult: s.PriceScale, Add: s.PriceAdd}
		}
	}

	// Unique and set rows ("cost mult" is a small integer such as 5 in the
	// shipped tables, not 1/1024 fixed point) are UNVERIFIED and not priced.
	return it
}

// GambleBase converts the item's base record to the input of
// d2trade.GamblePrice. TRADE_CalcGamblePrice works on the normal-tier row of
// the item's family (its NormalCode), whose exceptional (UberCode) and elite
// (UltraCode) rows it also reads; the levels it uses are the base item levels
// (byte +0xfd, the "level" column), not the required levels. VERIFIED.
func (i *Item) GambleBase() d2trade.Gamble {
	all := i.factory.asset.Records.Item.All
	rec := i.CommonRecord()

	if n := all[rec.NormalCode]; n != nil && rec.NormalCode != "" {
		rec = n
	}

	g := d2trade.Gamble{
		ReqLevel: rec.Level, Cost: rec.Cost, MinStack: rec.MinStack, MaxStack: rec.MaxStack,
		GambleCost:     rec.GambleCost,
		IsRingOrAmulet: rec.Code == "rin" || rec.Code == "amu",
	}

	if x := all[rec.UberCode]; x != nil && rec.UberCode != "" {
		g.HasExc, g.ExcReq, g.ExcCost = true, x.Level, x.Cost
	}

	if x := all[rec.UltraCode]; x != nil && rec.UltraCode != "" {
		g.HasElite, g.EliteReq, g.EliteCost = true, x.Level, x.Cost
	}

	return g
}
