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
