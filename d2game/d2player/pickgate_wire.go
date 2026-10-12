package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2ground"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// heldOf describes a carried item for the pickup gate (ITEMMODE_PlayerHasConflictingUniqueOrQuestItem 0x55a7d0).
func heldOf(it InventoryItem, worn bool) (d2ground.Held, bool) {
	item, ok := it.(*diablo2item.Item)
	if !ok || item == nil {
		return d2ground.Held{}, false
	}

	h := d2ground.Held{UniqueCode: item.UniqueCode, Code: item.CommonCode, Worn: worn}
	if rec := item.CommonRecord(); rec != nil {
		h.Quest = rec.Quest
	}

	return h, true
}

// CanPickUpGround is the carried-item half of the quest/unique pickup gate (0x55aa20) for a ground item: a
// single-copy unique is refused while the same one is carried in the backpack, stash or cube, and a quest item while
// its twin is. Worn items and the belt are skipped (the exe skips inventory page 1; which container that is is
// UNVERIFIED). The quest-flag half is not ported.
func (g *GameControls) CanPickUpGround(it *diablo2item.Item) bool {
	if it == nil {
		return true
	}

	c := d2ground.Candidate{UniqueCode: it.UniqueCode, Code: it.CommonCode}

	if u := it.UniqueRecord(); u != nil && it.UniqueCode != "" {
		c.SingleCopy = u.SingleCopy
	}

	if rec := it.CommonRecord(); rec != nil {
		c.Quest = rec.Quest
	}

	var held []d2ground.Held

	for _, grid := range []*ItemGrid{g.inventory.grid, g.stash.grid, g.cube.grid} {
		for _, x := range grid.items {
			if h, ok := heldOf(x, false); ok {
				held = append(held, h)
			}
		}
	}

	return d2ground.CanPickUp(held, c)
}
