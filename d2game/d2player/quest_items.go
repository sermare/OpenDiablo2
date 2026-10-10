package d2player

import "strings"

// ItemCountsByCode counts the items of the inventory grid and the cursor by
// their base code (used by the quest system to know which quest items the
// hero carries).
func (g *GameControls) ItemCountsByCode() map[string]int {
	out := map[string]int{}

	for _, it := range g.inventory.grid.Items() {
		out[strings.TrimSpace(it.GetItemCode())]++
	}

	if it := g.inventory.CursorItem(); it != nil {
		out[strings.TrimSpace(it.GetItemCode())]++
	}

	return out
}

// RemoveItemByCode deletes one item with the given base code from the cursor
// or the inventory grid; it reports whether one was found.
func (g *GameControls) RemoveItemByCode(code string) bool {
	if it := g.inventory.CursorItem(); it != nil && strings.TrimSpace(it.GetItemCode()) == code {
		g.inventory.SetCursorItem(nil)
		return true
	}

	for _, it := range g.inventory.grid.Items() {
		if strings.TrimSpace(it.GetItemCode()) == code {
			g.inventory.grid.Remove(it)
			return true
		}
	}

	return false
}

// SetQuestItemUse installs the hook that gives the usable quest items (Book of Skill, Potion of Life, Scroll
// of Resistance) their effect: it gets the item code and reports whether the item took effect and is consumed.
func (g *GameControls) SetQuestItemUse(use func(code string) bool) { g.questItemUse = use }

// useQuestItem runs the hook for an inventory item; it reports whether the item was consumed.
func (g *GameControls) useQuestItem(from *ItemGrid, item InventoryItem) bool {
	if g.questItemUse == nil || !g.questItemUse(item.GetItemCode()) {
		return false
	}

	from.Remove(item)

	return true
}

// UseInventoryItem uses the first inventory item with this base code the way a right click does
// (quest items with an effect are consumed); it reports whether the item took effect.
func (g *GameControls) UseInventoryItem(code string) bool {
	for _, it := range g.inventory.grid.Items() {
		if strings.TrimSpace(it.GetItemCode()) == code {
			if g.useQuestItem(g.inventory.grid, it) {
				g.saveHero()

				return true
			}

			return false
		}
	}

	return false
}

// ClearInventoryGrid removes every item of the inventory grid (not the equipped ones, the belt or the
// stash) and returns how many were removed (debug: makes room for a quest walkthrough).
func (g *GameControls) ClearInventoryGrid() int {
	n := 0

	// (a copy: Items() is the grid's own slice and Remove shifts it, which skipped every second item)
	for _, it := range append([]InventoryItem{}, g.inventory.grid.Items()...) {
		g.inventory.grid.Remove(it)

		n++
	}

	return n
}

// MoveInventoryItemToCube takes the first inventory item with this base code and puts it into the Horadric Cube
// (what dragging it onto the open cube does); it reports whether it was moved.
func (g *GameControls) MoveInventoryItemToCube(code string) bool {
	for _, it := range g.inventory.grid.Items() {
		if strings.TrimSpace(it.GetItemCode()) != code {
			continue
		}

		g.inventory.grid.Remove(it)

		if g.CubePut(it) {
			return true
		}

		g.inventory.grid.AutoPlace(it, true)

		return false
	}

	return false
}
