package d2inventory

// Container geometry, drop, stack and gold rules of the original game, as pure
// functions (inventory-trade.md, Inventory.cpp / inv.cpp). Rules marked
// VERIFIED were read in Game.exe 1.14b (addresses next to them); anything still
// marked UNVERIFIED is a hypothesis.

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

// MaxStack is the stack limit of an item (ITEM_GetMaxStack, 0x6297b0,
// VERIFIED): the base row maxstack plus the item's stat 0xfe (extra stack),
// clamped to 511.
func MaxStack(baseMax, extra int) int {
	if m := baseMax + extra; m < MaxStackLimit {
		return m
	}

	return MaxStackLimit
}

// MaxStackLimit is the hard clamp of MaxStack.
const MaxStackLimit = 511

// MergeStacks adds the quantity held (cursor) to a stack on the grid with a
// stack limit max (ITEMACT_ServerStackItems 0x55c600 and its merge routine
// 0x55c3c0, VERIFIED). When the sum exceeds max the grid stack is set to max
// (even if it was above max) and the surplus stays on the cursor item (stat
// 0x46); otherwise the grid stack takes everything, the cursor item is
// consumed (rest 0) and the cursor is cleared. Nothing is merged when the
// stackable test (0x62c9a0) fails; the caller decides that first.
func MergeStacks(onGrid, held, max int) (stack, rest int) {
	if max < 1 {
		return onGrid, held
	}

	total := onGrid + held
	if total <= max {
		return total, 0
	}

	return max, total - max
}

// MergeDurability is the durability of the stack after a merge: the grid item
// takes the cursor item's durability when that is lower (0x55c3c0, VERIFIED,
// only for items that have durability).
func MergeDurability(grid, cursor int) int {
	if cursor < grid {
		return cursor
	}

	return grid
}

// UnstackSupported is false: the server handler of packet 0x22 (unstack,
// ITEMACT_ServerUnstackItem 0x55c7f0) is an empty stub that returns 0 in 1.14b
// (VERIFIED), so the original never splits a stack with that packet. Splitting
// by shift-click in the original is the UI path of the bank/inventory and is
// not implemented through this packet.
const UnstackSupported = false

// SplitStack takes n units off a stack for the cursor. It is NOT a rule of the
// original (see UnstackSupported); it is only a helper for an OD2 extension and
// clamps n to 1..qty-1 (a split cannot take everything or nothing).
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

// Gold limits, VERIFIED: PLAYER_GetMaxGoldCarry 0x623050 returns stat 0x0c
// (level) * 10000; PLAYER_GetMaxStashGold 0x623640 returns the constant 2500000
// (the same in classic and Lord of Destruction; no stat is involved). The
// shared gold setter TRADE_Helper_53dc10 (0x53dc10) resets stat 0x0e/0x0f to 0
// when a player's new value would exceed its limit and to 0 when negative, so
// callers clamp first (FUN_00558e40 0x558e40 and FUN_0053e640 0x53e640 drop the
// overflow as gold piles on the ground). The save loader (0x531a50) also zeroes
// an over-cap or negative stat 0x0e / 0x0f.
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

// SetGold is the gold stat setter (0x53dc10, VERIFIED): the new amount is
// have+delta; below zero or above the limit the stat becomes 0 (it does not
// clamp). Use AddGold for the clamping callers do before calling it.
func SetGold(have, delta, limit int) int {
	n := have + delta
	if n < 0 || n > limit {
		return 0
	}

	return n
}

// SanitizeLoadedGold is what the save loader does with a stored gold stat
// (0x531a50, VERIFIED): negative or above the limit becomes 0.
func SanitizeLoadedGold(v, limit int) int {
	if v < 0 || v > limit {
		return 0
	}

	return v
}
