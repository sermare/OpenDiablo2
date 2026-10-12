package diablo2item

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// questType is ItemTypes row 0x27, the type TRADE_CheckItemSellable refuses.
const (
	questType  = "ques"
	playerPart = "play" // item type 7: a player ear
	// lowQualityCracked is the LowQualityItems row of "Cracked" in the save id.
	lowQualityCracked = 1
)

// IsBroken is the broken state (item flag 0x100) as far as the engine models
// it: a durable item with no durability left. The exe sets flag 0x100 only in
// ITEM_BreakEquippedItem (0x55d660, when a worn item runs out) and clears it
// in ITEM_RestoreBrokenItem (0x55d710), so a loose item at 0 without the saved
// flag is UNVERIFIED as broken (the engine treats it as broken).
func (i *Item) IsBroken() bool {
	cur, maxDur := i.Durability()

	return maxDur > 0 && cur <= 0
}

// hasContents reports whether anything is socketed into the item.
func (i *Item) hasContents() bool {
	if len(i.SocketCodes) > 0 || len(i.socketed) > 0 {
		return true
	}

	return i.origin != nil && len(i.origin.Children) > 0
}

// SellInfo returns the inputs of d2trade.Sellable. The no-sell flag 0x1000 is
// not modelled (nothing sets it in the engine).
func (i *Item) SellInfo() d2trade.SellInfo {
	rec := i.CommonRecord()
	if rec == nil {
		return d2trade.SellInfo{}
	}

	return d2trade.SellInfo{Quest: rec.Quest != 0, QuestType: rec.Type == questType}
}

// ResaleInfo returns the inputs of d2trade.Resellable. The caller sets
// GambleMode and VendorSells, which depend on the vendor.
func (i *Item) ResaleInfo() d2trade.ResaleInfo {
	rec := i.CommonRecord()
	r := d2trade.ResaleInfo{
		Broken:       i.IsBroken(),
		Ethereal:     i.IsEthereal(),
		HasContents:  i.hasContents(),
		Personalized: i.attributes != nil && i.attributes.personalization != "",
	}

	if rec != nil {
		r.PlayerPart = rec.Type == playerPart
	}

	if i.quality == d2drop.QualityLow && i.origin != nil {
		r.Cracked = i.origin.LowQualityID == lowQualityCracked
	}

	if i.quality == d2drop.QualityUnique {
		if u := i.UniqueRecord(); u != nil {
			r.SingleCopy = u.SingleCopy
		}
	}

	return r
}

// Refill restores what TRADE_RefillItemForResale (0x573fb0) restores on a
// repairable item: the quantity of stacked throwing weapons and the
// durability (a broken item is repaired too). Skill charges are restored too
// (ITEM_UpdateSkillChargesToMax 0x55dc40: every stat 0xcc entry with cur < max is
// raised to max; VERIFIED in Game.exe): chargesFull makes priceStats report them full.
func (i *Item) Refill() {
	ti := i.TradeItem()
	if !ti.Repairable {
		return
	}

	i.chargesFull = true

	if ti.Throwable && ti.Stackable {
		if limit := i.StackLimit(); limit > 0 {
			i.SetQuantity(limit)
		}
	}

	if _, maxDur := i.Durability(); maxDur > 0 {
		i.SetDurability(maxDur)
	}
}
