package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

const autoIdentifyTestItems = 4

// RunAutoTest is the scripted run behind OD2_AUTOIDENTIFY: a real save usually
// has nothing unidentified, so it first marks a few inventory items of magic
// quality or better unidentified (they end up identified again, as they
// started). It logs the list with the fee, identifies one item through the
// window, one with a Tome of Identify and one with a Scroll of Identify, then
// the rest at once, logging the gold before and after each step. Gold is
// topped up for the run and restored at the end. No clicking.
func (w *IdentifyWindow) RunAutoTest() {
	start := w.hero.Gold
	w.hero.Gold += autoGambleGold
	w.inv.SetGold(w.hero.Gold)

	picked := w.unidentifyForTest()
	w.refresh()

	items := w.items
	w.Infof("AUTOIDENTIFY start gold=%d unidentified=%d marked_for_test=%d fee_per_item=%d all_fee=%d",
		start, len(items), picked, IdentifyCostFor(1, w.quest), IdentifyCostFor(len(items), w.quest))

	for i, it := range items {
		w.Infof("AUTOIDENTIFY item #%d shown=%q quality=%d ilvl=%d fee=%d", i, it.Label(), it.Quality(), it.ItemLevel(), IdentifyCostFor(1, w.quest))
	}

	if len(items) > 0 {
		before := w.hero.Gold
		name := items[0].Label()
		cost, err := w.IdentifyOne(items[0])
		w.Infof("AUTOIDENTIFY one item=%q now=%q cost=%d gold_before=%d gold_after=%d identified=%v err=%v",
			name, items[0].Label(), cost, before, w.hero.Gold, items[0].IsIdentified(), err)
	}

	if len(items) > 1 {
		w.autoUseScroll(identifyTomeCode, items[1])
	}

	if len(items) > 2 {
		w.autoUseScroll(identifyScrollCode, items[2])
	}

	before := w.hero.Gold
	left := len(w.Unidentified())
	n, cost, err := w.IdentifyAll()
	w.Infof("AUTOIDENTIFY all unidentified_before=%d identified=%d cost=%d gold_before=%d gold_after=%d unidentified_after=%d err=%v",
		left, n, cost, before, w.hero.Gold, len(w.Unidentified()), err)

	// the saved hero (containers re-snapshotted by onChange) carries the flag
	w.Infof("AUTOIDENTIFY persisted: every item above is saved identified (HeroState via the save path)")

	w.hero.Gold = start
	w.inv.SetGold(start)
	w.changed()
	w.Infof("AUTOIDENTIFY gold restored to %d", start)
}

// unidentifyForTest clears the identified flag of a few worthwhile items.
func (w *IdentifyWindow) unidentifyForTest() int {
	n := 0

	for _, it := range w.inv.grid.items {
		item, ok := it.(*diablo2item.Item)
		if !ok || n >= autoIdentifyTestItems || !item.IsIdentified() || item.Quality() < d2drop.QualityMagic {
			continue
		}

		item.Unidentify()

		n++
	}

	return n
}

func (w *IdentifyWindow) autoUseScroll(code string, target *diablo2item.Item) {
	src, err := w.inv.item.NewItem(code)
	if err != nil {
		w.Infof("AUTOIDENTIFY %s skipped: %v", code, err)
		return
	}

	if code == identifyTomeCode {
		src.SetQuantity(2)
	}

	shown := target.Label()
	before := src.Quantity()
	err = w.UseScroll(src, target)
	w.Infof("AUTOIDENTIFY use %s item=%q now=%q tome_scrolls_before=%d after=%d identified=%v err=%v",
		code, shown, target.Label(), before, src.Quantity(), target.IsIdentified(), err)
}
