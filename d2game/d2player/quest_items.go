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
