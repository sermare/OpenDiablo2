package d2player

// ClickAction names what a click on a grid did.
type ClickAction string

// The outcomes of ItemGrid.ClickWith.
const (
	ClickNone     ClickAction = ""
	ClickPickup   ClickAction = "pickup"   // the item under the mouse went to the cursor
	ClickPlace    ClickAction = "place"    // the cursor item went into free cells
	ClickSwap     ClickAction = "swap"     // the cursor item went in, the item it covered went to the cursor
	ClickAuto     ClickAction = "auto"     // the cursor item was auto-placed (ctrl)
	ClickRefused  ClickAction = "refused"  // the footprint covers two or more items, or the grid is full
	ClickNoTarget ClickAction = "notarget" // empty cursor on empty cells
	ClickMerge    ClickAction = "merge"    // the cursor item was stacked onto the item under it; any surplus stays on the cursor
)

// ClickWith applies the left-click rules of INV_HandleGridClick (0x48c3c0,
// inventory-trade.md) for the grid under (mx, my): with an empty cursor the
// item under the mouse is picked up; with a held item its footprint is
// anchored on the mouse (CursorAnchor); no item under it places it, exactly one
// swaps, two or more refuse. ctrl auto-places the held item (an OpenDiablo2
// convenience). It returns the item now on the cursor, what happened and the
// cell involved. The caller must have checked that the point is on the grid.
func (g *ItemGrid) ClickWith(cursor InventoryItem, mx, my int, ctrl bool) (held InventoryItem, act ClickAction, x, y int) {
	if cursor == nil {
		cx, cy := g.ScreenToSlot(mx, my)

		it := g.GetSlot(cx, cy)
		if it == nil {
			return nil, ClickNoTarget, cx, cy
		}

		x, y = it.InventoryGridSlot()
		g.Remove(it)

		return it, ClickPickup, x, y
	}

	if ctrl {
		if x, y, ok := g.AutoPlaceForPickup(cursor); ok {
			return nil, ClickAuto, x, y
		}

		return cursor, ClickRefused, 0, 0
	}

	w, h := cursor.InventoryGridSize()
	ax, ay := CursorAnchor(mx, my, g.originX, g.originY, g.slotSize, w, h, g.width, g.height)

	if ax+w > g.width || ay+h > g.height {
		return cursor, ClickRefused, ax, ay // the item is bigger than the grid
	}

	switch over := g.Overlapping(ax, ay, w, h); len(over) {
	case 0:
		g.set(ax, ay, cursor)

		return nil, ClickPlace, ax, ay
	case 1:
		if rest, merged := mergeIntoStack(over[0], cursor); merged {
			return rest, ClickMerge, ax, ay
		}

		g.Remove(over[0])
		g.set(ax, ay, cursor)

		return over[0], ClickSwap, ax, ay
	default:
		return cursor, ClickRefused, ax, ay
	}
}
