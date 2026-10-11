package d2equip

// Depleted stacks and broken weapons (PLRMODE_ServerHandleDepletedStackItem 0x57e020 and
// SRVST_DestroyBrokenEquippedItems 0x57e370; notes in gaps-slice-G.md).

// DepletedAction says what the server does with a worn item whose quantity (stat 0x46) ran out.
type DepletedAction int

// The outcomes of a depleted item.
const (
	// DepletedNone: not a stack, or the quantity is still above zero.
	DepletedNone DepletedAction = iota
	// DepletedKeep: the item has stat 0x7d (a refilling stack, meaning unverified); only its quantity is set to
	// 0 and it stays in the hand.
	DepletedKeep
	// DepletedBreak: a magical weapon stack is broken (ITEM_BreakEquippedItem) unless it is already broken.
	DepletedBreak
	// DepletedUnequip: the stack leaves the body slot; Reload types then look for a replacement in the
	// inventory, ReEquip types get a pending re-equip.
	DepletedUnequip
)

// DepletedItem is what the decision reads.
type DepletedItem struct {
	Stackable    bool // ITEM_IsStackable
	Throwable    bool // ITEM_IsThrowableType
	HasStat7d    bool // stat 0x7d non-zero (full stat)
	Quantity     int  // stat 0x46
	WeaponType   bool // ITEM_IsOfType(0x2d), the Weapon type
	MagicQuality bool // ITEM_IsMagicalQuality: an item of quality 4..9
	Broken       bool // item flag 0x100
}

// DepletedStackAction is the first half of 0x57e020.
func DepletedStackAction(it DepletedItem) DepletedAction {
	if !(it.Stackable || it.Throwable || it.HasStat7d) || it.Quantity > 0 {
		return DepletedNone
	}

	if it.HasStat7d {
		return DepletedKeep
	}

	if it.WeaponType && it.MagicQuality {
		if it.Broken {
			return DepletedNone
		}

		return DepletedBreak
	}

	return DepletedUnequip
}

// ReplacementStack picks the inventory item that takes the place of a spent stack: the first one (grid walk
// order) with the same base item code that is not broken. It returns -1 for none. The exe only does this for
// types with the Reload column, and re-checks that the body slot is empty before equipping.
func ReplacementStack(code string, candidates []StackCandidate) int {
	for i, c := range candidates {
		if c.Code == code && !c.Broken {
			return i
		}
	}

	return -1
}

// StackCandidate is an inventory item considered by ReplacementStack.
type StackCandidate struct {
	Code   string
	Broken bool // flag 0x100
}

// IsReload reports whether the item type has the Reload column set (the type's own row).
func (r Rules) IsReload(itemType string) bool {
	t := r.Types.Get(itemType)

	return t != nil && t.Reload
}

// IsReEquip reports whether the item type has the ReEquip column set (the type's own row).
func (r Rules) IsReEquip(itemType string) bool {
	t := r.Types.Get(itemType)

	return t != nil && t.ReEquip
}

// WornItemBreaks is the test of the scan SRVST_DestroyBrokenEquippedItems makes over the worn items: a
// Weapon type item with durability applicable, durability of zero or less and no broken flag yet is broken
// (flag 0x100 set, durability stat 0, properties removed). The function name says "destroy" but
// ITEM_BreakEquippedItem (0x55d660) keeps the item on the body.
func WornItemBreaks(weaponType, durabilityApplicable bool, durability int, alreadyBroken bool) bool {
	return weaponType && durabilityApplicable && durability <= 0 && !alreadyBroken
}
