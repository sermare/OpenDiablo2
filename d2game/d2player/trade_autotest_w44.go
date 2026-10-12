package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The w44 vendor checks of the OD2_AUTOTRADE run (log lines "AUTOTRADE permanent", "AUTOTRADE poor",
// "AUTOTRADE multibuy"): an item the vendor always sells stays in the stock after a purchase, a hero without gold
// gets the vendor's message, and a shift buy of a multibuy potion fills the inventory as far as the gold goes.

func (t *TradeWindow) countStock(code string) int {
	n := 0

	for _, e := range t.stock.Items {
		if e.Code == code {
			n++
		}
	}

	return n
}

// permanentStock is the first stock item the vendor always sells (nil: none).
func (t *TradeWindow) permanentStock(multibuy bool) *diablo2item.Item {
	for _, e := range t.stock.Items {
		item, ok := e.Payload.(*diablo2item.Item)
		if !ok || !t.vendorSells(e.Code) || (multibuy && !item.CommonRecord().Multibuy) {
			continue
		}

		if multibuy && !t.isBeltable(item) {
			continue
		}

		return item
	}

	return nil
}

func (t *TradeWindow) autoW44() {
	if t.gamble {
		return
	}

	parked := t.parkSmallest(6)
	defer func() {
		for _, p := range parked {
			t.unpark(p)
		}
	}()

	pick := t.permanentStock(false)
	if pick == nil {
		t.Infof("AUTOTRADE permanent vendor=%s none", t.vendor.Name)
	} else {
		t.hero.Gold = 100000
		code := pick.GetItemCode()
		before := t.countStock(code)

		var err error

		if t.inv.grid.CanAutoPlace(pick, true) {
			_, err = t.Buy(pick)
		} else {
			err = ErrNoRoom
		}

		t.Infof("AUTOTRADE permanent vendor=%s code=%s stock_before=%d stock_after=%d err=%v",
			t.vendor.Name, code, before, t.countStock(code), err)

		t.removeBought(code)
	}

	// a hero without gold
	for it := range t.entries { // any item of the shown tab
		if item, ok := it.(*diablo2item.Item); ok {
			t.hero.Gold = 0
			_, err := t.Buy(item)
			t.Infof("AUTOTRADE poor vendor=%s err=%v message=%q", t.vendor.Name, err, t.messageText())

			break
		}
	}

	t.autoMultiBuy()
}

// parkSmallest takes the n smallest inventory items out of the grid so that the run's purchases have room.
func (t *TradeWindow) parkSmallest(n int) []*parkedItem {
	items := append([]InventoryItem(nil), t.inv.grid.items...)
	area := func(i InventoryItem) int { w, h := i.InventoryGridSize(); return w * h }

	for a := 1; a < len(items); a++ {
		for b := a; b > 0 && area(items[b]) < area(items[b-1]); b-- {
			items[b], items[b-1] = items[b-1], items[b]
		}
	}

	if len(items) > n {
		items = items[:n]
	}

	var out []*parkedItem

	for _, it := range items {
		p := &parkedItem{item: it}
		p.x, p.y = it.InventoryGridSlot()
		t.inv.grid.Remove(it)
		out = append(out, p)
	}

	return out
}

func (t *TradeWindow) messageText() string {
	_, text := d2trade.ResultMessage(d2trade.ResultNoGold)

	return t.text("TransactionResults10", text)
}

// removeBought takes every inventory item of a code out of the grid (the run's purchases).
func (t *TradeWindow) removeBought(code string) {
	for _, it := range append([]InventoryItem(nil), t.inv.grid.items...) {
		if it.GetItemCode() == code {
			t.inv.grid.Remove(it)
		}
	}
}

func (t *TradeWindow) autoMultiBuy() {
	pick := t.permanentStock(true)
	if pick == nil {
		t.Infof("AUTOTRADE multibuy vendor=%s none", t.vendor.Name)
		return
	}

	belt := t.autoBelt
	t.autoBelt = nil // the run counts the copies in the inventory grid

	defer func() { t.autoBelt = belt }()

	code := pick.GetItemCode()
	unit := t.BuyPrice(pick)
	t.hero.Gold = unit*3 + unit/2 // three potions and a bit

	paid, err := t.BuyMulti(pick, true)

	n := 0

	for _, it := range t.inv.grid.items {
		if it.GetItemCode() == code {
			n++
		}
	}

	t.Infof("AUTOTRADE multibuy vendor=%s code=%s unit=%d paid=%d copies=%d gold_after=%d err=%v",
		t.vendor.Name, code, unit, paid, n, t.hero.Gold, err)

	t.removeBought(code)
}
