package d2trade

// Result codes of the server's trade reply (SCMD_SendPacketLen15_B4D0 0x53b4d0, sent by TRADE_ServerBuyItem
// 0x575640, TRADE_ServerSellItem 0x577320, TRADE_ServerRepairItem 0x575e80 and TRADE_ServerIdentifyAllItems).
// VERIFIED: the client side (UI_ApplyNpcTransactionResultA/B 0x4b1c50, 0x4b1ea0) plays a sound for 0 and 1 and
// opens a modal message with an OK button for every code above 6; the strings are string.tbl keys
// TransactionResults1..11 (ids 0xd19..0xd23).
const (
	ResultBought   = 0    // item bought (sound: the item's drop sound)
	ResultSold     = 1    // item sold (sound 0xdd)
	ResultDone     = 2    // repaired / identified
	ResultStacked  = 5    // bought into an existing stack or tome
	ResultBusy     = 7    // cursor not empty, unknown item or not in the right inventory
	ResultInvalid  = 9    // "I cannot complete that request"
	ResultNoRoom   = 0xa  // no free space for the bought item
	ResultNoItem   = 0xb  // the sold item is not in the player's inventory
	ResultNoGold   = 0xc  // not enough gold
	ResultMercBusy = 0xf  // the mercenary cannot come now
	ResultMercMax  = 0x10 // already the maximum number of mercenaries
)

// ResultMessage is the string table key and the English text the client shows for a trade reply code, "" for the
// codes that only play a sound (0..6).
func ResultMessage(code int) (key, text string) {
	if code <= 6 {
		return "", ""
	}

	switch code {
	case 7, 8:
		return "TransactionResults6", "That item has just been traded."
	case ResultNoRoom:
		return "TransactionResults4", "You do not have enough free space to do that."
	case ResultNoItem:
		return "TransactionResults8", "Something tells me that you do not have that item."
	case ResultNoGold:
		return "TransactionResults10", "You do not have enough gold for that."
	case ResultMercBusy:
		return "TransactionResults11", "She cannot come right now.  Try again later."
	case ResultMercMax:
		return "TransactionResults5", "You already have the maximum number of Mercenaries."
	}

	return "TransactionResults9", "I cannot complete that request."
}

// StackBuy is the quantity bought by a multi buy (shift) of an auto stack item: as many pieces as the gold pays
// (gold / unit price, unit = the price of one piece) up to room (the missing quantity of the stack that is topped
// up, or the maximum stack when a new stack is made). A new stack is never smaller than 1 (the purchase then fails
// on the gold). ok is false when not even one piece is affordable and a stack is topped up (TRADE_ServerBuyItem).
func StackBuy(unit, gold, room int, topUp bool) (qty, cost int, ok bool) {
	if unit <= 0 {
		unit = 1
	}

	afford := gold / unit

	qty = room
	if afford < qty {
		qty = afford
	}

	if topUp {
		if qty < 1 {
			return 0, 0, false
		}
	} else if qty < 2 {
		qty = 1
	}

	return qty, qty * unit, true
}

// MultiBuy reports whether a shift click buys more than one piece: the shift bit travels in the packet only for
// base items whose items.txt "multibuy" column is set (UI_SendNpcItemTransactionPacket 0x4aedf0), and the server
// keeps it only for items the vendor always sells (TRADE_CheckVendorSellsItem) that also go on the belt or auto
// stack (0x575640).
func MultiBuy(shift, multibuyColumn, vendorSells bool) bool {
	return shift && multibuyColumn && vendorSells
}
