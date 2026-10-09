package d2player

import (
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

// autoPick returns the cheapest affordable stock item that fits the bag.
func (t *TradeWindow) autoPick() *diablo2item.Item {
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

func (t *TradeWindow) autoBuySell() {
	pick := t.autoPick()

	// the bag of a real (imported) hero can be full: lift items out until a
	// purchase fits, and put them back where they were afterwards
	var lifted []InventoryItem

	for pick == nil && len(t.inv.grid.items) > 0 {
		it := t.inv.grid.items[len(t.inv.grid.items)-1]
		t.inv.grid.Remove(it)
		lifted = append(lifted, it)
		pick = t.autoPick()
	}

	if len(lifted) > 0 {
		defer func() {
			for i := len(lifted) - 1; i >= 0; i-- {
				x, y := lifted[i].InventoryGridSlot()
				if err := t.inv.grid.Set(x, y, lifted[i]); err != nil {
					t.Infof("AUTOTRADE could not put %s back: %v", lifted[i].GetItemCode(), err)
				}
			}
		}()
	}

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

	before = t.hero.Gold
	price, err = t.Sell(pick)
	t.Infof("AUTOTRADE sell vendor=%s item=%q price=%d gold_before=%d gold_after=%d err=%v",
		t.vendor.Name, name, price, before, t.hero.Gold, err)
}

func (t *TradeWindow) autoRepair() {
	// the bag first, then what the hero wears
	candidates := append([]InventoryItem{}, t.inv.grid.items...)

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

	t.Infof("AUTOTRADE repair vendor=%s skipped: no item with durability in the inventory", t.vendor.Name)
}
