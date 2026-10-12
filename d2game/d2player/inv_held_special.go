package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The held-item click on the inventory grid, the branches of INV_HandleGridClick (0x48c3c0) that are not a plain
// place, swap or stack: a Horadric Cube under the footprint takes the item (packet 0x2a), a scroll goes into its
// tome (packet 0x29) and a cube without room makes the hero say "impossible". d2inventory.HeldDrop holds the
// decision; this file applies it.

// tomeOf maps a scroll code to the code of its tome (the Books rows of the game: town portal and identify).
var tomeOf = map[string]string{townPortalScrollCode: townPortalTomeCode, identifyScrollCode: identifyTomeCode}

// VERIFIED (ITEMACT_ServerScrollToTome 0x55cd70, INV_FindMatchingStackWithRoom 0x63d4b0): the exe pairs a scroll and a
// tome by comparing their magic suffix (ITEM_GetMagicSuffix 0x628260, slot 0) with the type ids (0x16 scroll, 0x12
// book) and needs the tome's quantity (stat 0x46) below its max stack; the table above is the equivalent for the two
// scroll kinds of 1.14b.

// itemsUnder lists the distinct items under a w x h footprint at (x, y) in the scan order of
// INV_GetSingleOverlappedItem (0x63cc50): rows from the top, cells from the left.
func (g *ItemGrid) itemsUnder(x, y, w, h int) []InventoryItem {
	var out []InventoryItem

	for cy := y; cy < y+h; cy++ {
		for cx := x; cx < x+w; cx++ {
			it := g.GetSlot(cx, cy)
			if it == nil {
				continue
			}

			seen := false

			for _, o := range out {
				seen = seen || o == it
			}

			if !seen {
				out = append(out, it)
			}
		}
	}

	return out
}

// stackProbe describes an item for d2inventory.CanStack. INV_CheckItemsStackable 0x62c9a0 (read) compares txt number, item
// file index, stackable, ethereal, that both are non-magical quality, damage stats 0x15..0x18 and 0x9f/0xa0, and that
// neither has sockets. The damage stats are not tracked by the engine; for non-magical stackable bases they are all
// zero, so leaving them out cannot change the answer.
func stackProbe(it InventoryItem) d2inventory.StackProbe {
	// an item that is not a rolled game item (a test double) counts as a plain normal quality stackable
	p := d2inventory.StackProbe{Code: it.GetItemCode(), Quality: 2, Stackable: true}

	if s, ok := it.(*diablo2item.Item); ok {
		p.Quality = int(s.Quality())
		p.Stackable = s.IsStackable()
		p.Ethereal = s.IsEthereal()
		p.Sockets = s.NumSockets()
	}

	return p
}

// heldInput builds the decision input for the held item over the inventory grid at (mx, my).
func (g *GameControls) heldInput(cur InventoryItem, mx, my int) (in d2inventory.HeldInput, under []InventoryItem, ok bool) {
	grid := g.inventory.grid
	w, h := cur.InventoryGridSize()

	ax, ay, fits := CursorAnchor(mx, my, grid.originX, grid.originY, grid.slotSize, w, h, grid.width, grid.height)
	if !fits {
		return in, nil, false
	}

	under = grid.itemsUnder(ax, ay, w, h)
	in.HeldIsCube = cur.GetItemCode() == cubeItemCode
	in.CubeFits = !in.HeldIsCube && g.cube.Accepts(cur) && g.cube.grid.CanAutoPlace(cur, true)

	for _, it := range under {
		u := d2inventory.HeldUnder{IsCube: it.GetItemCode() == cubeItemCode}

		u.Stack = d2inventory.CanStack(stackProbe(it), stackProbe(cur))

		if tome, isScroll := tomeOf[cur.GetItemCode()]; isScroll && it.GetItemCode() == tome {
			if t, isItem := it.(*diablo2item.Item); isItem {
				u.Tome = true
				u.TomeFull = t.Quantity() >= t.StackLimit()
			}
		}

		in.Under = append(in.Under, u)
	}

	return in, under, true
}

// tryInventoryHeldSpecial applies the cube and tome branches of a held-item click on the inventory grid. It reports
// whether the click was consumed; everything else (place, swap, stack, refusal) stays with ItemGrid.ClickWith.
func (g *GameControls) tryInventoryHeldSpecial(mx, my int) bool {
	cur := g.inventory.CursorItem()
	if cur == nil || !g.inventory.IsOpen() || !g.inventory.grid.Contains(mx, my) {
		return false
	}

	in, under, ok := g.heldInput(cur, mx, my)
	if !ok {
		return false
	}

	switch d2inventory.HeldDrop(in) {
	case d2inventory.HeldIntoCube:
		if !g.cube.grid.AutoPlace(cur, true) {
			return false
		}

		g.Infof("INVENTORY held %s dropped into the cube", cur.GetItemCode())
		itemSound(cur, "item-drop")
		g.inventory.SetCursorItem(nil)
		g.cube.changed()
		g.saveHero()

		return true
	case d2inventory.HeldImpossible:
		if in.HeldIsCube {
			return true // OD2: the cube into the cube is refused by the server without a line (the exe's rule was not traced)
		}

		g.Infof("INVENTORY held %s: the cube has no room", cur.GetItemCode())
		g.sayHeroVoice(0x13)

		return true
	case d2inventory.HeldScrollToTome:
		tome, isItem := firstTome(under, cur).(*diablo2item.Item)
		if !isItem {
			return false
		}

		tome.SetQuantity(tome.Quantity() + 1)

		if s, isScroll := cur.(*diablo2item.Item); isScroll && s.Quantity() > 1 {
			s.SetQuantity(s.Quantity() - 1) // a stack of scrolls gives one
		} else {
			g.inventory.SetCursorItem(nil)
		}

		g.Infof("INVENTORY scroll %s went into the tome: quantity=%d", cur.GetItemCode(), tome.Quantity())
		itemSound(cur, "item-drop")
		g.saveHero()

		return true
	case d2inventory.HeldNothing:
		if len(in.Under) == 1 && in.Under[0].Tome && in.Under[0].TomeFull {
			g.Infof("INVENTORY scroll %s refused: the tome is full", cur.GetItemCode())

			return true
		}
	}

	return false
}

func firstTome(under []InventoryItem, scroll InventoryItem) InventoryItem {
	want := tomeOf[scroll.GetItemCode()]

	for _, it := range under {
		if it.GetItemCode() == want {
			return it
		}
	}

	return nil
}
