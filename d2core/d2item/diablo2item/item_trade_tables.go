package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// staffModTypes are the ItemTypes rows with a StaffMods class (the exe tests
// the byte at +0x1f of the item's own type row against 7 = none). The
// record's StaffMods field cannot tell "none" from the first hero, so the
// rows of the shipped ItemTypes.txt are listed (like classItemTypes).
var staffModTypes = map[string]bool{
	"scep": true, "wand": true, "staf": true, "orb": true, "head": true,
	"phlm": true, "pelt": true, "cloa": true, "h2h2": true,
}

// tomeType is the item type 0x12 of TRADE_CalcItemPrice (ItemTypes "book").
const tomeType = "book"

// priceTables builds the stat and skill price rows once.
func (f *ItemFactory) priceTables() *d2trade.Tables {
	f.priceOnce.Do(func() {
		t := &d2trade.Tables{Stats: map[int]d2trade.StatCost{}, Skills: map[int]d2trade.SkillCost{}}

		for _, s := range f.asset.Records.Item.Stats {
			if s != nil {
				t.Stats[s.Index] = d2trade.StatCost{
					Encode: int(s.Encode), ValShift: s.ValShift, Mult: s.CostMultiply, Add: s.CostAdd,
				}
			}
		}

		for id, s := range f.asset.Records.Skill.Details {
			if s != nil {
				t.Skills[id] = d2trade.SkillCost{Mult: s.CostMult, Add: s.CostAdd, ReqLevel: s.Reqlevel}
			}
		}

		f.priceTbl = t
	})

	return f.priceTbl
}

// priceStats returns the item's stat list entries as the price code reads them
// (the bonus the item's property lists add, in stored form: shifted left by
// the stat's valshift), merged by stat id and parameter. Items read from a
// save use the exact save contents (own, runeword and socketed stats);
// created items use what the creator wrote to their own lists (set bonus
// tiers are not part of the item's own list). Items with neither return nil.
func (i *Item) priceStats() []d2trade.StatEntry {
	t := i.factory.priceTables()
	merged := map[[2]int]int{}

	var order [][2]int

	add := func(id, param, v int) {
		k := [2]int{id, param}
		if _, ok := merged[k]; !ok {
			order = append(order, k)
		}

		merged[k] += v
	}

	switch {
	case i.origin != nil && i.factory.DescriptionTables() != nil:
		for _, s := range i.factory.DescriptionTables().ItemStats(i.origin) {
			add(s.ID, s.Param, s.Value<<uint(t.Stats[s.ID].ValShift))
		}
	case i.rolled != nil:
		for _, wr := range i.rolled.Writes {
			if wr.Kind == 'S' || (wr.List >= setListFirst && wr.List <= setListLast) {
				continue
			}

			add(wr.Stat, wr.Param, wr.Value)
		}
	default:
		return nil
	}

	out := make([]d2trade.StatEntry, 0, len(order))
	for _, k := range order {
		v := merged[k]
		if i.chargesFull && k[0] == d2trade.StatChargedSkill {
			mx := v >> 8 & 0xff
			v = v&^0xffff | mx | mx<<8 // current charges = maximum
		}

		out = append(out, d2trade.StatEntry{ID: k[0], Param: k[1], Value: v})
	}

	return out
}

// entriesOf returns the entries of one stat id, with the parameter as is.
func entriesOf(all []d2trade.StatEntry, id int) []d2trade.StatEntry {
	var out []d2trade.StatEntry

	for _, e := range all {
		if e.ID == id {
			out = append(out, e)
		}
	}

	return out
}

// socketedCosts are the base costs of the items in the item's sockets (the
// created ones in socketed, gems and runes of a cube or vendor item by code).
func (i *Item) socketedCosts() []int {
	var out []int

	all := i.factory.asset.Records.Item.All

	for _, code := range i.SocketCodes {
		if r := all[code]; r != nil {
			out = append(out, r.Cost)
		}
	}

	for _, s := range i.socketed {
		if s != nil {
			if r := s.CommonRecord(); r != nil {
				out = append(out, r.Cost)
			}
		}
	}

	if i.origin != nil && len(out) == 0 {
		for _, ch := range i.origin.Children {
			if r := all[ch.Code]; r != nil {
				out = append(out, r.Cost)
			}
		}
	}

	return out
}

// tomeScrollCost is books.txt CostPerCharge of the tome (record +0x14).
func (i *Item) tomeScrollCost(code string) int {
	for _, b := range i.factory.asset.Records.Item.Books {
		if b != nil && b.BookSpellCode == code {
			return b.CostPerCharge
		}
	}

	return 0
}

// fillPriceExtras sets the price inputs that need the tables: stat, skill and
// socket prices, recharge cost, tome price, unique and set rows.
func (i *Item) fillPriceExtras(it *d2trade.Item, typeCode string) {
	rec := i.CommonRecord()

	it.Tables = i.factory.priceTables()
	it.StaffMods = staffModTypes[typeCode]
	it.SocketedCosts = i.socketedCosts()

	if typeCode == tomeType {
		it.IsTome = true
		it.TomeScrollCost = i.tomeScrollCost(rec.Code)
	}

	if !it.Identified {
		return
	}

	switch i.quality {
	case d2drop.QualityUnique:
		if u := i.UniqueRecord(); u != nil {
			it.Affixes.Unique = &d2trade.Term{Mult: u.CostMultiplier, Add: u.CostAdd}
		}
	case d2drop.QualitySet:
		if s := i.SetItemRecord(); s != nil {
			it.Affixes.Set = &d2trade.Term{Mult: s.CostMult, Add: s.CostAdd}
		}
	}

	stats := i.priceStats()
	it.Stats = stats
	it.SingleSkills = entriesOf(stats, d2trade.StatSingleSkill)

	it.RechargeCost = d2trade.RechargeCost(it.Tables, entriesOf(stats, d2trade.StatChargedSkill))
}
