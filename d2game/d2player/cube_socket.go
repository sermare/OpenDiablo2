package d2player

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"

// trySocketClick handles dropping the held gem, rune or jewel on an item of an
// open panel: it goes into a free socket of the item and a runeword forms when
// the runes fit. It returns false (the click is then an ordinary swap or place)
// when the held item is not a socket filler or the item under the mouse cannot
// take it.
func (g *GameControls) trySocketClick(mx, my int) bool {
	cur, ok := g.inventory.CursorItem().(*diablo2item.Item)
	if !ok || cur == nil {
		return false
	}

	panels := []struct {
		open bool
		grid *ItemGrid
	}{
		{g.inventory.IsOpen(), g.inventory.grid},
		{g.stash.IsOpen(), g.stash.grid},
		{g.cube.IsOpen(), g.cube.grid},
	}

	for _, p := range panels {
		if !p.open || !p.grid.Contains(mx, my) {
			continue
		}

		target, ok := p.grid.ItemAtScreen(mx, my).(*diablo2item.Item)
		if !ok || target == nil || target == cur {
			continue
		}

		return g.socketInto(p.grid, target, cur)
	}

	return false
}

// socketInto puts the filler into the target of a grid and replaces the target
// by the socketed item.
func (g *GameControls) socketInto(grid *ItemGrid, target, filler *diablo2item.Item) bool {
	item, msg, err := g.SocketItem(target, filler)
	if err != nil {
		g.Infof("SOCKET %s into %s refused: %v", filler.CommonCode, target.CommonCode, err)
		return false
	}

	x, y := target.InventoryGridSlot()
	grid.Remove(target)
	delete(g.itemOrigin, target)

	if err := grid.Set(x, y, item); err != nil {
		grid.Remove(item)
		g.Warningf("SOCKET could not put %s back: %v", item.CommonCode, err)
	}

	g.inventory.SetCursorItem(nil)
	g.Infof("SOCKET %s", msg)
	g.saveHero()

	return true
}
