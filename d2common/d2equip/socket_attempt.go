package d2equip

// SocketAttempt is what INV_ServerTrySocketItem (0x560470) reads when a socket filler is dropped on an item.
type SocketAttempt struct {
	TargetIdentified bool // target item flag 0x10
	TargetSocketed   bool // target item flag 0x800 (the item has sockets)
	TargetHasInv     bool // ITEM_BaseHasInventory: the base can hold sockets
	Sockets, Filled  int  // ITEM_GetSocketCount (stat 194) and the items already inside
	FillerIsSocket   bool // ITEM_IsSocketFiller: ItemTypes id 0x35 (Socket Filler: gems, runes, jewels)
	FillerIdentified bool // filler flag 0x10; an unidentified magic or rare jewel is refused
}

// SocketRefusal reports why the exe refuses the socketing (VERIFIED, 0x560470, checked in this order after
// the two units were found): the target must be identified, the filler must be a socket filler and
// identified, the target must carry the socketed flag and a base with an inventory, and have a free
// socket. "" means the filler goes in.
//
// After a successful insert (VERIFIED) the exe applies the filler's effects to the target with
// INV_ServerApplyEquippedItemEffects (gem type: property group 2, rune type: group 5, both from Gems.txt by
// the target's apply type; a jewel keeps its own stat list), which is toggled onto the target only while the
// target is not broken (flag 0x100); then it looks for a runeword (ITEM_FindRunewordForSockets) and, unless the
// row is a "server" row in a game whose +0x74 flag is clear, applies its properties; finally the target gets
// flag 1 and loses flag 0x4000.
func SocketRefusal(a SocketAttempt) string {
	switch {
	case !a.TargetIdentified:
		return "the item is not identified"
	case !a.FillerIsSocket:
		return "that cannot be put into a socket"
	case !a.FillerIdentified:
		return "the socketable item is not identified"
	case !a.TargetSocketed || !a.TargetHasInv || a.Sockets <= 0:
		return "the item has no sockets"
	case a.Filled >= a.Sockets:
		return "the item has no free socket"
	}

	return ""
}
