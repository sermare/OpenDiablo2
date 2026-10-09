package d2player

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
)

// This file adds the cursor item (the item the hero holds on the mouse, item
// mode 4 in the original) to the inventory panel, together with the click rules
// of INV_HandleGridClick (0x48c3c0) from inventory-trade.md: an empty cursor
// picks the item under the mouse; a held item is placed at the anchor cell when
// its footprint is free, swapped when it overlaps exactly one item and refused
// when it overlaps more.

// AutoPlaceForPickup puts the item at the slot found by the original's free-slot search
// (d2inventory.FindFreeSlot with a player owner) and returns it. If that search
// finds nothing although cells are free (it never accepts a slot without an
// occupied or edge neighbour) the first-fit scan of Add is used as a fallback,
// so a pickup is never refused while there is room.
func (g *ItemGrid) AutoPlaceForPickup(item InventoryItem) (x, y int, ok bool) {
	w, h := item.InventoryGridSize()

	if x, y, ok = g.occupancy().FindFreeSlot(w, h, true); ok && g.canFit(x, y, item) {
		g.set(x, y, item)

		return x, y, true
	}

	if g.add(item) {
		x, y = item.InventoryGridSlot()

		return x, y, true
	}

	return 0, 0, false
}

// Overlapping returns the distinct items under the w x h footprint at (x, y).
func (g *ItemGrid) Overlapping(x, y, w, h int) []InventoryItem {
	var out []InventoryItem

	for _, it := range g.items {
		sx, sy := it.InventoryGridSlot()
		sw, sh := it.InventoryGridSize()

		if x < sx+sw && x+w > sx && y < sy+sh && y+h > sy {
			out = append(out, it)
		}
	}

	return out
}

// Contains reports whether a screen point is over the grid cells.
func (g *ItemGrid) Contains(mx, my int) bool {
	return mx >= g.originX && my >= g.originY &&
		mx < g.originX+g.width*g.slotSize && my < g.originY+g.height*g.slotSize
}

// CursorAnchor is INV_ConvertMouseToGridAnchor: the top-left cell of a w x h
// item held at the pixel (mx, my). The item is held by its middle: for an even
// width the hot spot sits between two cells (rounded), for an odd one inside a
// cell; the result is clamped to the grid.
func CursorAnchor(mx, my, originX, originY, cell, w, h, gridW, gridH int) (ax, ay int) {
	fx := float64(mx-originX) / float64(cell)
	fy := float64(my-originY) / float64(cell)

	if w%2 == 0 {
		ax = int(math.Round(fx)) - w/2
	} else {
		ax = int(math.Floor(fx)) - w/2
	}

	if h%2 == 0 {
		ay = int(math.Round(fy)) - h/2
	} else {
		ay = int(math.Floor(fy)) - h/2
	}

	return clampInt(ax, 0, gridW-w), clampInt(ay, 0, gridH-h)
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}

	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// CursorItem returns the item held on the cursor, or nil.
func (g *Inventory) CursorItem() InventoryItem { return g.cursor }

// SetCursorItem puts an item on the cursor (nil clears it).
func (g *Inventory) SetCursorItem(item InventoryItem) {
	g.cursor = item

	if item != nil {
		g.grid.Load(item)
	}
}

// AutoPlaceCursor moves the cursor item into the inventory using the original's
// auto-placement search. It returns the slot, and false (leaving the item on the
// cursor) when the inventory is full.
func (g *Inventory) AutoPlaceCursor() (x, y int, ok bool) {
	if g.cursor == nil {
		return 0, 0, false
	}

	if x, y, ok = g.grid.AutoPlaceForPickup(g.cursor); ok {
		g.cursor = nil
	}

	return x, y, ok
}

// AddGold adds picked-up gold to the inventory's gold counter.
func (g *Inventory) AddGold(amount int) {
	g.gold += amount
	g.moveGoldPanel.gold += amount
}

// HandleClick handles a left click at (mx, my) while the panel is open and
// reports whether the click was consumed. ctrl auto-places a held item (an
// OD2 convenience; the original has no such shortcut for the cursor).
func (g *Inventory) HandleClick(mx, my int, ctrl bool) bool {
	if !g.isOpen || !g.grid.Contains(mx, my) {
		return false
	}

	held, act, x, y := g.grid.ClickWith(g.cursor, mx, my, ctrl)
	g.lastClick = act

	switch act {
	case ClickPickup:
		g.Infof("picked up %s from the inventory", held.GetItemCode())
	case ClickAuto:
		g.Infof("auto-placed cursor item at (%d,%d)", x, y)
	}

	g.SetCursorItem(held)

	return true
}

// RenderCursorItem draws the held item centred on the mouse.
func (g *Inventory) RenderCursorItem(target d2interface.Surface, mx, my int) {
	if g.cursor == nil {
		return
	}

	sprite := g.grid.sprites[g.cursor.GetItemCode()]
	if sprite == nil {
		return
	}

	w, h := sprite.GetCurrentFrameSize()
	sprite.SetPosition(mx-w/2, my+h/2)
	sprite.Render(target)
}
