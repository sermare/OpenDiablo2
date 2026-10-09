package d2player

import (
	"os"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// RunAutoTest is the scripted transaction run behind OD2_AUTOTRADE: it logs
// the stock with its computed buy prices, buys the cheapest affordable item
// that fits and sells it back, and for a repairing vendor damages one
// inventory item and repairs it. No clicking is involved. The hero's gold is
// restored at the end so the run does not change the player's saved gold for
// good (the transactions themselves go through the normal save path).
func (t *TradeWindow) RunAutoTest() {
	if !t.isOpen {
		t.Infof("AUTOTRADE window not open")
		return
	}

	start := t.hero.Gold

	t.Infof("AUTOTRADE vendor=%s gold=%d stock=%d", t.vendor.Name, start, len(t.stock.Items))

	for i, l := range t.Stock() {
		t.Infof("AUTOTRADE stock vendor=%s #%d code=%s name=%q quality=%d ilvl=%d qty=%d buy=%d",
			t.vendor.Name, i, l.Code, l.Name, l.Quality, l.ILvl, l.Quantity, l.Price)
	}

	t.autoBuySell()

	if t.vendor.Repairs {
		t.autoRepair()
	}

	t.hero.Gold = start
	t.syncGold()
	t.changed()
	t.Infof("AUTOTRADE vendor=%s gold restored to %d", t.vendor.Name, start)
}

// cheapestFitting returns the cheapest vendor item the hero can pay and has room for.
func (t *TradeWindow) cheapestFitting() *diablo2item.Item {
	var pick *diablo2item.Item

	best := 0

	for _, e := range t.stock.Items {
		item, ok := e.Payload.(*diablo2item.Item)
		if !ok {
			continue
		}

		price := t.BuyPrice(item)
		if price > t.hero.Gold || !t.inv.grid.CanAutoPlace(item, true) {
			continue
		}

		if pick == nil || price < best {
			pick, best = item, price
		}
	}

	return pick
}

// parkedItem is an inventory item the autotest took out of the grid to make room.
type parkedItem struct {
	item InventoryItem
	x, y int
}

// parkForRoom frees the smallest inventory item whose removal lets the
// cheapest affordable vendor item fit (an imported save's inventory can be
// completely full). The item goes back to its cell with unpark.
func (t *TradeWindow) parkForRoom() (*parkedItem, *diablo2item.Item) {
	items := append([]InventoryItem(nil), t.inv.grid.items...)

	area := func(i InventoryItem) int { w, h := i.InventoryGridSize(); return w * h }

	sort.SliceStable(items, func(a, b int) bool { return area(items[a]) < area(items[b]) })

	for _, it := range items {
		p := &parkedItem{item: it}
		p.x, p.y = it.InventoryGridSlot()

		t.inv.grid.Remove(it)

		if pick := t.cheapestFitting(); pick != nil {
			return p, pick
		}

		_ = t.inv.grid.Set(p.x, p.y, it)
	}

	return nil, nil
}

func (t *TradeWindow) unpark(p *parkedItem) {
	if p == nil {
		return
	}

	if err := t.inv.grid.Set(p.x, p.y, p.item); err != nil && !t.inv.grid.AutoPlace(p.item, true) {
		t.Infof("AUTOTRADE could not give the parked %s back", p.item.GetItemCode())
	}
}

func (t *TradeWindow) autoBuySell() {
	pick := t.cheapestFitting()

	var parked *parkedItem

	if pick == nil {
		parked, pick = t.parkForRoom()
		if parked != nil {
			t.Infof("AUTOTRADE inventory full: %s taken out of the grid for the test", parked.item.GetItemCode())
		}
	}

	defer t.unpark(parked)

	if pick == nil {
		t.Infof("AUTOTRADE buy vendor=%s skipped: nothing affordable (gold=%d) or no room", t.vendor.Name, t.hero.Gold)
		return
	}

	before := t.hero.Gold
	name := pick.Label()
	price, err := t.Buy(pick)
	t.Infof("AUTOTRADE buy vendor=%s item=%q price=%d gold_before=%d gold_after=%d err=%v",
		t.vendor.Name, name, price, before, t.hero.Gold, err)

	if err != nil {
		return
	}

	if os.Getenv("OD2_AUTOTRADE_KEEP") == "1" {
		// the item stays in the inventory: the .d2s export scenario checks that it is written
		t.Infof("AUTOTRADE keep vendor=%s code=%s quality=%d ilvl=%d", t.vendor.Name, pick.GetItemCode(), pick.Quality(), pick.ItemLevel())
		return
	}

	before = t.hero.Gold
	price, err = t.Sell(pick)
	t.Infof("AUTOTRADE sell vendor=%s item=%q price=%d gold_before=%d gold_after=%d err=%v",
		t.vendor.Name, name, price, before, t.hero.Gold, err)
}

func (t *TradeWindow) autoRepair() {
	// the inventory first, then the worn items (an imported inventory holds
	// charms and potions, which have no durability)
	candidates := append([]InventoryItem(nil), t.inv.grid.items...)

	for _, slot := range t.inv.grid.equipmentSlots {
		if slot.item != nil {
			candidates = append(candidates, slot.item)
		}
	}

	for _, it := range candidates {
		item, ok := it.(*diablo2item.Item)
		if !ok {
			continue
		}

		if _, maxDur := item.Durability(); maxDur > 1 {
			item.SetDurability(maxDur / 2) //nolint:gomnd // half worn

			name := item.Label()
			cost := t.RepairPrice(item)
			before := t.hero.Gold
			total, err := t.RepairAll()
			t.Infof("AUTOTRADE repair vendor=%s item=%q cost=%d total=%d gold_before=%d gold_after=%d err=%v",
				t.vendor.Name, name, cost, total, before, t.hero.Gold, err)

			return
		}
	}

	t.Infof("AUTOTRADE repair vendor=%s skipped: no item with durability in the inventory or worn", t.vendor.Name)
}
