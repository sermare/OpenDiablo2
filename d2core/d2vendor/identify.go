package d2vendor

// Cain's identify service (TRADE_ServerIdentifyAllItems, Game.exe 1.14b
// 0x576290; VERIFIED). The server charges 100 gold per unidentified item
// (the count comes from FUN_0062a6a0) with TRADE_DeductGold: no npc.txt
// multiplier and no reduced-prices discount. It is free once quest record
// bit 4 (flag 4, either state bit 0 or 1) is set in the quest record of the
// game's current difficulty. The packet is only accepted from the Deckard
// Cain classes 244, 245, 246, 265 and 520.

// IdentifyPricePerItem is the gold per unidentified item.
const IdentifyPricePerItem = 100

// identifyQuestFlag is the quest record flag that makes identifying free.
const identifyQuestFlag = 4

// CainClassIDs are the monster classes that accept the identify packet.
//
//nolint:gochecknoglobals // static lookup data
var CainClassIDs = [...]int{244, 245, 246, 265, 520}

// IsCain reports whether a monster class id is one of Cain's.
func IsCain(classID int) bool {
	for _, c := range CainClassIDs {
		if c == classID {
			return true
		}
	}

	return false
}

// IdentifyQuestFlag is the quest flag number whose bits make identifying free.
func IdentifyQuestFlag() int { return identifyQuestFlag }

// IdentifyPrice is the total price of identifying unidentified items;
// questDone is quest flag 4 (state bit 0 or bit 1) of the current difficulty.
func IdentifyPrice(unidentified int, questDone bool) int {
	if questDone || unidentified <= 0 {
		return 0
	}

	return unidentified * IdentifyPricePerItem
}
