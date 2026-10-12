package d2trade

// AffixSet is the affix identity of an item as the exe compares it: three magic prefixes, three magic suffixes and the
// automagic prefix (empty strings for the unused slots).
type AffixSet struct {
	Prefix [3]string
	Suffix [3]string
	Auto   string
}

// AffixesOf builds an AffixSet from the affix lists of an item (extra entries beyond three are ignored).
func AffixesOf(prefixes, suffixes []string, auto string) AffixSet {
	var a AffixSet

	copy(a.Prefix[:], prefixes)
	copy(a.Suffix[:], suffixes)
	a.Auto = auto

	return a
}

// TomeMatches is ITEM_HaveSameAffixes 0x55d400 (VERIFIED), the test TRADE_FindStackableTomeInInventory 0x55d460 applies
// between a bought scroll (item type 0x16) and a tome (type 0x12); ITEMACT_ServerScrollToTome 0x55cd70 (the scroll on
// the cursor dropped on a tome) compares the first magic suffix only. Here: the three magic prefixes, the three magic
// suffixes and the automagic prefix must be equal. Only the 16 bit affix ids are compared, not the item codes: the town
// portal and identify books are told apart by the suffix the game gives to scroll and tome alike. The caller adds the
// other conditions: the tome is in the inventory page (page 0, not the stash or cube) and its quantity (stat 0x46) is
// below the maximum stack.
func TomeMatches(scroll, tome AffixSet) bool {
	return scroll == tome
}

// TomeDropMatches is the cursor test of ITEMACT_ServerScrollToTome and INV_HandleGridClick 0x48c3c0: the first magic
// suffix of scroll and tome is equal (VERIFIED).
func TomeDropMatches(scroll, tome AffixSet) bool {
	return scroll.Suffix[0] == tome.Suffix[0]
}

// TomeRoom is the quantity the tome still takes (ITEM_GetMaxStack - stat 0x46, the stat counted as 0 when negative), 0
// when it is full.
func TomeRoom(quantity, maxStack int) int {
	if quantity < 0 {
		quantity = 0
	}

	if maxStack <= quantity {
		return 0
	}

	return maxStack - quantity
}
