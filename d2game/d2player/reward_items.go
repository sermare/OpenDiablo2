package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The item side of the NPC rewards (Larzuk, Anya, Charsi): the hero takes an
// item from the inventory onto the cursor and clicks the NPC with it. The
// screen decides whether the NPC wants the item (OnItemDropOnNPC).

// ItemDropListener is implemented by the game screen: it is told that the hero
// clicked an NPC with an item on the cursor, and returns true when the NPC
// takes care of it (the item then does not fall to the ground).
type ItemDropListener interface {
	OnItemDropOnNPC(npc d2interface.MapEntity, item InventoryItem) bool
}

// dropOnNPC offers the cursor item to the NPC under the mouse.
func (g *GameControls) dropOnNPC(npc d2interface.MapEntity) bool {
	l, ok := g.inputListener.(ItemDropListener)
	if !ok || npc == nil || g.inventory.CursorItem() == nil {
		return false
	}

	return l.OnItemDropOnNPC(npc, g.inventory.CursorItem())
}

// ItemFactory returns the factory that makes the hero's items.
func (g *GameControls) ItemFactory() *diablo2item.ItemFactory { return g.inventory.item }

// OpenInventoryPanel shows the inventory (the reward rows ask for an item).
func (g *GameControls) OpenInventoryPanel() {
	if !g.inventory.IsOpen() {
		g.inventory.Open()
	}
}

// PickItem moves the first inventory item the predicate accepts onto the
// cursor, as a click on it would (the autotest has no mouse). It returns
// the item, or nil.
func (g *GameControls) PickItem(pred func(*diablo2item.Item) bool) *diablo2item.Item {
	if g.inventory.CursorItem() != nil {
		return nil
	}

	for _, it := range g.inventory.grid.Items() {
		if d, ok := it.(*diablo2item.Item); ok && pred(d) {
			g.inventory.grid.Remove(it)
			g.inventory.SetCursorItem(it)

			return d
		}
	}

	return nil
}

// ReplaceCursorItem swaps the item on the cursor for another one (Charsi hands
// back a new, rare item).
func (g *GameControls) ReplaceCursorItem(item InventoryItem) error {
	if g.inventory.CursorItem() == nil {
		return fmt.Errorf("no item on the cursor")
	}

	g.inventory.cursor = nil
	g.inventory.SetCursorItem(item)

	return nil
}

// ReplaceInventoryItem swaps an item of the inventory grid for another one of
// the same size at the same place (the console reward command, which has no
// cursor). It reports whether the old item was in the grid.
func (g *GameControls) ReplaceInventoryItem(old, fresh InventoryItem) bool {
	found := false

	for _, it := range g.inventory.grid.Items() {
		if it == old {
			found = true

			break
		}
	}

	if !found {
		return false
	}

	x, y := old.InventoryGridSlot()

	g.inventory.grid.Remove(old)
	g.inventory.grid.Load(fresh)

	if !g.inventory.grid.canFit(x, y, fresh) {
		return g.inventory.grid.AutoPlace(fresh, true)
	}

	g.inventory.grid.set(x, y, fresh)

	return true
}

// PutCursorItemAway puts the cursor item into the inventory; it reports
// whether there was room.
func (g *GameControls) PutCursorItemAway() bool {
	_, _, ok := g.inventory.AutoPlaceCursor()

	return ok
}

// RefreshSkills redraws the skill tree and the stat buttons after the skill and
// stat points changed (Akara's reset).
func (g *GameControls) RefreshSkills() {
	g.skilltree.refresh()
	g.setAddButtons()
}
