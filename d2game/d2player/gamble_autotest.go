package d2player

import (
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

const autoGambleGold = 1000000

// RunGambleAutoTest is the scripted run behind OD2_AUTOGAMBLE: it logs the
// gamble stock (unidentified names, quality, item level, price), buys the
// cheapest item that fits and logs what it turned out to be, sells it back,
// then moves the clock past the four minute restock and reopens the window to
// log that the stock is regenerated. Gold is topped up for the run and
// restored at the end. No clicking.
func (t *TradeWindow) RunGambleAutoTest() {
	if !t.IsGamble() {
		t.Infof("AUTOGAMBLE window not open")
		return
	}

	start := t.hero.Gold
	t.hero.Gold += autoGambleGold
	t.syncGold()

	t.Infof("AUTOGAMBLE vendor=%s level=%d gold=%d stock=%d", t.vendor.Name, t.playerLevel(), start, len(t.stock.Items))

	first := t.logGambleStock()

	pick := t.cheapestFitting()
	if pick == nil {
		t.Infof("AUTOGAMBLE buy vendor=%s skipped: no room in the inventory", t.vendor.Name)
	} else {
		before := t.hero.Gold
		shown := pick.Label()
		q, il := pick.Quality(), pick.ItemLevel()
		wasIdentified := pick.IsIdentified()
		price, err := t.Buy(pick)
		t.Infof("AUTOGAMBLE buy vendor=%s shown=%q identified_before=%v price=%d gold_before=%d gold_after=%d err=%v",
			t.vendor.Name, shown, wasIdentified, price, before, t.hero.Gold, err)

		if err == nil {
			t.Infof("AUTOGAMBLE result vendor=%s name=%q quality=%d ilvl=%d identified=%v",
				t.vendor.Name, pick.Label(), q, il, pick.IsIdentified())

			before = t.hero.Gold
			price, err = t.Sell(pick)
			t.Infof("AUTOGAMBLE sell vendor=%s item=%q price=%d gold_before=%d gold_after=%d err=%v",
				t.vendor.Name, pick.Label(), price, before, t.hero.Gold, err)
		}
	}

	t.autoRestock(first)

	t.hero.Gold = start
	t.syncGold()
	t.changed()
	t.Infof("AUTOGAMBLE vendor=%s gold restored to %d", t.vendor.Name, start)
}

// logGambleStock logs every stock line and returns the codes joined.
func (t *TradeWindow) logGambleStock() string {
	codes := ""
	counts := map[d2drop.Quality]int{}

	for i, l := range t.Stock() {
		t.Infof("AUTOGAMBLE stock vendor=%s #%d code=%s name=%q quality=%d ilvl=%d price=%d",
			t.vendor.Name, i, l.Code, l.Name, l.Quality, l.ILvl, l.Price)

		codes += l.Code + ","
		counts[l.Quality]++
	}

	t.Infof("AUTOGAMBLE stock vendor=%s quality_counts magic=%d rare=%d set=%d unique=%d",
		t.vendor.Name, counts[d2drop.QualityMagic], counts[d2drop.QualityRare], counts[d2drop.QualitySet], counts[d2drop.QualityUnique])

	return codes
}

// autoRestock reopens the window five minutes later and checks the restock.
func (t *TradeWindow) autoRestock(first string) {
	v := t.vendor
	real := t.now
	n := len(t.stock.Items)

	t.Close()

	t.now = func() time.Time { return real().Add(5 * time.Minute) }
	_ = t.OpenGamble(v, 1, nil)
	t.now = real

	t.Infof("AUTOGAMBLE restock vendor=%s items_before=%d items_after=%d changed=%v",
		v.Name, n, len(t.stock.Items), t.logGambleStock() != first)
}
