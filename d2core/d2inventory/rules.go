package d2inventory

// Container geometry, drop, stack and gold rules of the original game, as pure
// functions (inventory-trade.md, Inventory.cpp / inv.cpp). Rules marked
// UNVERIFIED are not read from the executable yet; the addresses to confirm are
// listed next to them.

// Container names a storage page. The values follow the item page numbers.
type Container int

// The containers.
const (
	ContainerInventory Container = iota
	ContainerTrade
	ContainerCube
	ContainerStash
	ContainerVendor
)

// Dims returns the grid size in cells (Inventory.txt gridX/gridY, 1.14b data:
// player 10x4, Trade pages 10x4, Transmogrify Box 3x4, Bank Page 6x4, Big Bank
// (expansion stash) 6x8, Monster/vendor storage 10x10). A 10x10 stash is a
// later (Resurrected) value and is NOT used here.
func Dims(c Container, expansion bool) (w, h int) {
	switch c {
	case ContainerInventory, ContainerTrade:
		return 10, 4
	case ContainerCube:
		return 3, 4
	case ContainerStash:
		if expansion {
			return 6, 8
		}

		return 6, 4
	case ContainerVendor:
		return 10, 10
	}

	return 0, 0
}

// DropOutcome is what dropping a held item on a footprint does.
type DropOutcome int

// The outcomes (INV_HandleGridClick, 0x48c3c0, using INV_GetOverlappedItems).
const (
	DropPlace  DropOutcome = iota // footprint empty: place
	DropSwap                      // exactly one item underneath: swap with the cursor
	DropRefuse                    // two or more items underneath: nothing happens
)

// ClassifyDrop maps the number of distinct items under the footprint to the
// outcome. A single item that is stackable with the held one merges instead
// (see MergeStacks); the caller decides that first.
func ClassifyDrop(overlapping int) DropOutcome {
	switch {
	case overlapping <= 0:
		return DropPlace
	case overlapping == 1:
		return DropSwap
	default:
		return DropRefuse
	}
}

// MergeStacks adds the quantity held (cursor) to a stack on the grid with a
// stack limit max. It returns the new stack size and what remains on the
// cursor. The surplus staying on the cursor is UNVERIFIED (confirm in the
// handler of packet 0x21, FUN_0055c600; the stackable test 0x62c9a0 is
// verified; the cap is max stack + stat 0xfe via FUN_006297b0).
func MergeStacks(onGrid, held, max int) (stack, rest int) {
	if max < 1 || onGrid >= max {
		return onGrid, held
	}

	total := onGrid + held
	if total <= max {
		return total, 0
	}

	return max, total - max
}

// SplitStack takes n units off a stack for the cursor (packet 0x22 unstack,
// FUN_0055c7f0; UNVERIFIED whether n is chosen in a dialog or fixed). It
// returns the units left behind and the units taken; n is clamped to
// 1..qty-1 (a split cannot take everything or nothing).
func SplitStack(qty, n int) (left, taken int) {
	if qty < 2 {
		return qty, 0
	}

	if n < 1 {
		n = 1
	}

	if n > qty-1 {
		n = qty - 1
	}

	return qty - n, n
}

// Gold limits. UNVERIFIED in the executable (confirm in the setters of the
// gold stats, 0x0e inventory gold and 0x0f stash gold, in D2Common/D2Game;
// addresses not located yet).
const (
	// InventoryGoldPerLevel is the inventory gold cap per character level.
	InventoryGoldPerLevel = 10000
	// StashGoldLimit is the gold the stash holds.
	StashGoldLimit = 2500000
)

// InventoryGoldLimit is the gold a character of a level carries.
func InventoryGoldLimit(level int) int {
	if level < 1 {
		level = 1
	}

	return level * InventoryGoldPerLevel
}

// AddGold adds gold to a purse with a limit and returns the new amount and the
// amount that did not fit (left on the ground when picked up).
func AddGold(have, add, limit int) (total, overflow int) {
	if add <= 0 {
		return have, 0
	}

	if have >= limit {
		return have, add
	}

	if have+add > limit {
		return limit, have + add - limit
	}

	return have + add, 0
}
