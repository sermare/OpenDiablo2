package d2player

import (
	"errors"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The scroll and the tome of town portal (misc.txt "tsc" and "tbk"). Opening
// the portal itself is the game screen's job; this file finds the item that
// pays for it. A scroll is used up, a tome loses one scroll of its quantity.
// UNVERIFIED: the tome's scroll count is kept in the item quantity like the
// tome of identify's (the original keeps it in stat 0x46).
const (
	townPortalScrollCode = "tsc"
	townPortalTomeCode   = "tbk"
)

// ErrNoTownPortalItem is returned when the inventory has no scroll and no tome
// with a scroll left.
var ErrNoTownPortalItem = errors.New("no Scroll of Town Portal and no Tome of Town Portal with a scroll left")

// IsTownPortalItem reports whether the item is a scroll or tome of town portal.
func IsTownPortalItem(it InventoryItem) bool {
	code := it.GetItemCode()

	return code == townPortalScrollCode || code == townPortalTomeCode
}

// TownPortalSource finds what pays for a town portal: a scroll first (the
// player keeps the tome for later), else a tome with a scroll left.
func (g *GameControls) TownPortalSource() *diablo2item.Item {
	var tome *diablo2item.Item

	for _, it := range g.inventory.grid.items {
		item, ok := it.(*diablo2item.Item)
		if !ok {
			continue
		}

		switch item.GetItemCode() {
		case townPortalScrollCode:
			return item
		case townPortalTomeCode:
			if tome == nil && item.Quantity() > 0 {
				tome = item
			}
		}
	}

	return tome
}

// ConsumeTownPortal uses up one charge of a scroll or tome of town portal and
// saves the hero; it returns the quantity left (0 for a scroll).
func (g *GameControls) ConsumeTownPortal(src *diablo2item.Item) (left int, err error) {
	if src == nil || !IsTownPortalItem(src) {
		return 0, ErrNoTownPortalItem
	}

	if src.GetItemCode() == townPortalTomeCode {
		if src.Quantity() < 1 {
			return 0, ErrNoTownPortalItem
		}

		src.SetQuantity(src.Quantity() - 1)
		left = src.Quantity()
	} else if src.Quantity() > 1 {
		src.SetQuantity(src.Quantity() - 1) // a stack of scrolls
		left = src.Quantity()
	} else {
		g.inventory.grid.Remove(src)
	}

	g.saveHero()

	return left, nil
}
