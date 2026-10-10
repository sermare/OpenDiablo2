package d2player

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"

// Inventory lookups for world objects: the key a locked chest consumes (INV_ConsumeOneKey 0x55cf90) and
// the loose gems the Gem Upgrade shrine reads (0x580b50). Both walk the main inventory grid in its order.
const keyItemCode = "key"

// KeySource returns the first key in the inventory, nil when the hero has none. VERIFIED (0x55cf90): the
// first inventory item of type 0x29 whose page is the inventory page is used.
func (g *GameControls) KeySource() *diablo2item.Item {
	for _, it := range g.inventory.grid.items {
		if item, ok := it.(*diablo2item.Item); ok && item.GetItemCode() == keyItemCode {
			return item
		}
	}

	return nil
}

// ConsumeKey uses up one key: a stack loses one of its quantity, the last key is removed (0x55cf90). It
// returns the quantity left.
func (g *GameControls) ConsumeKey(src *diablo2item.Item) (left int) {
	if src == nil {
		return 0
	}

	if src.Quantity() > 1 {
		src.SetQuantity(src.Quantity() - 1)
		left = src.Quantity()
	} else {
		g.inventory.grid.Remove(src)
	}

	g.saveHero()

	return left
}

// InventoryGems returns the loose gems of the inventory grid (item type "gem"), in grid order. Socketed
// gems are not part of the grid and are not upgraded by the shrine.
func (g *GameControls) InventoryGems() []*diablo2item.Item {
	var out []*diablo2item.Item

	for _, it := range g.inventory.grid.items {
		item, ok := it.(*diablo2item.Item)
		if !ok {
			continue
		}

		if rec := item.CommonRecord(); rec != nil && rec.Type == "gem" {
			out = append(out, item)
		}
	}

	return out
}
