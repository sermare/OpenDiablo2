package d2player

import (
	"errors"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"
)

// The gamble window. The stock rules (14 items, ring and amulet first, item
// level and quality rolls) are VERIFIED from Game.exe and live in
// d2vendor/gamble.go; this file adds the window part.

// ErrNoGamble is returned when the vendor has no gamble window or the tables
// hold no gamble rows.
var ErrNoGamble = errors.New("this vendor does not gamble")

// stockKey separates a vendor's regular stock from its gamble stock.
func (t *TradeWindow) stockKey(v d2vendor.Vendor) string {
	if t.gamble {
		return v.Name + "#gamble"
	}

	return v.Name
}

// OpenGamble shows the vendor's gamble stock (the "Gamble" row of the NPC
// menu, packet 0x38 with action 2 in the original).
func (t *TradeWindow) OpenGamble(v d2vendor.Vendor, seed uint32, quests *d2s.QuestRecord) error {
	if !v.Gambles {
		return ErrNoGamble
	}

	if _, _, _, ok := d2vendor.GamblePoolFor(t.asset.Records); !ok {
		return ErrNoGamble
	}

	t.open(v, seed, quests, true)

	return nil
}

// IsGamble reports whether the open window is a gamble window.
func (t *TradeWindow) IsGamble() bool { return t.isOpen && t.gamble }

// generateGamble rolls the gamble stock and builds its (unidentified) items.
func (t *TradeWindow) generateGamble(v d2vendor.Vendor, seed uint32) *d2vendor.Stock {
	stock := d2vendor.NewStock()

	pool, ring, amulet, ok := d2vendor.GamblePoolFor(t.asset.Records)
	if !ok {
		return stock
	}

	params := d2vendor.GambleParamsFor(t.asset.Records, t.difficulty)
	// The expansion check of the original (record version < 100 skips
	// classic-only rows, upgrades need the expansion) is not modelled: this
	// engine always runs with the Lord of Destruction tables.
	picks := d2vendor.GenerateGamble(d2rand.New(seed), pool, ring, amulet, params, t.playerLevel(), true)

	for idx, g := range picks {
		item, err := t.factory.ItemFromCode(g.Code, g.Quality, g.ILvl, seed+uint32(idx)+1)
		if err != nil {
			t.Errorf("gamble item %q: %v", g.Code, err)
			continue
		}

		e := &d2vendor.Item{Code: g.Code, Quality: g.Quality, ILvl: g.ILvl, Payload: item}
		e.W, e.H = item.InventoryGridSize()

		if !stock.Place(e) {
			break // the original stops at the first item that does not fit
		}
	}

	t.Infof("gamble stock rolled: vendor=%s level=%d items=%d", v.Name, t.playerLevel(), len(stock.Items))

	return stock
}

// gamblePrice is TRADE_CalcItemPrice mode 2 (VERIFIED formula, d2trade).
// UNVERIFIED: the "flat gamble cost" switch (itemData word +0x30 == 0) is
// never taken here, so only rings and amulets use the gamble cost column.
func (t *TradeWindow) gamblePrice(item *diablo2item.Item) int {
	p := t.params(d2trade.ModeGamble)
	p.Gamble = item.GambleBase()

	return d2trade.ItemPrice(item.TradeItem(), p)
}

// reveal identifies a bought gamble item (packet 0x37 / TRADE_ServerIdentify
// GambleItem: the item keeps the quality rolled when the stock was built).
func (t *TradeWindow) reveal(item *diablo2item.Item) {
	item.Identify()
	t.Infof("gamble reveal: %q quality=%d ilvl=%d", item.Label(), item.Quality(), item.ItemLevel())
}
