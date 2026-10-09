package d2player

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"

// StackableItem is what an item offers when it can stack (diablo2item.Item
// does). The grid only merges items that implement it and report IsStackable.
type StackableItem interface {
	InventoryItem
	IsStackable() bool
	Quantity() int
	SetQuantity(n int)
	// StackLimit is d2inventory.MaxStack(base maxstack, stat 0xfe).
	StackLimit() int
}

// durableItem is implemented by items with a durability (diablo2item.Item).
type durableItem interface {
	Durability() (current, maximum int)
	SetDurability(n int)
}

// mergeIntoStack is the stack-on-drop rule of the server handler for packet
// 0x21 (0x55c600, merge 0x55c3c0, VERIFIED): when the held item and the item
// under it are the same stackable base item, the grid stack takes as much of the
// held quantity as the limit allows. The surplus stays on the cursor (rest is
// the cursor item); when everything fit the cursor item is consumed (rest nil).
// For items with a durability the grid item takes the lower value. merged is
// false when the two do not stack (the caller then swaps as before).
//
// Gameplay change: dropping a stackable item on the same item merges instead
// of swapping. The original's split (packet 0x22) is a stub, so no split exists.
func mergeIntoStack(onGrid, held InventoryItem) (rest InventoryItem, merged bool) {
	a, ok := onGrid.(StackableItem)
	if !ok {
		return nil, false
	}

	b, ok := held.(StackableItem)
	if !ok || !a.IsStackable() || !b.IsStackable() || a.GetItemCode() != b.GetItemCode() {
		return nil, false
	}

	stack, surplus := d2inventory.MergeStacks(a.Quantity(), b.Quantity(), a.StackLimit())

	if da, ok := onGrid.(durableItem); ok {
		if db, ok := held.(durableItem); ok {
			if ca, ma := da.Durability(); ma > 0 {
				if cb, _ := db.Durability(); cb < ca {
					da.SetDurability(d2inventory.MergeDurability(ca, cb))
				}
			}
		}
	}

	a.SetQuantity(stack)

	if surplus == 0 {
		return nil, true
	}

	b.SetQuantity(surplus)

	return held, true
}
