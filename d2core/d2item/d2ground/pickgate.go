package d2ground

// Pickup gate for unique and quest items: ITEMMODE_CanPickupQuestOrUniqueItem 0x55aa20, whose carried-item
// conflict test is ITEMMODE_PlayerHasConflictingUniqueOrQuestItem 0x55a7d0 (both VERIFIED by decompilation in w49).
// The quest-flag half of 0x55aa20 (per quest code, records of the difficulty) is not ported here: quest state lives
// in the client, see the notes in d2gamescreen.

// Held is one item the hero carries, as the conflict scan sees it.
type Held struct {
	// UniqueCode identifies a unique item row ("" when the item is not a unique).
	UniqueCode string
	// Code is the base item code of a quest item.
	Code string
	// Quest is the base record's quest byte (0 = not a quest item).
	Quest int
	// Worn is true for an item with inventory page 1 in the exe (UNVERIFIED which container that is; the scan skips it).
	Worn bool
}

// Candidate is the ground item to pick up.
type Candidate struct {
	UniqueCode string
	// SingleCopy is the unique row's "carry1" bit (bit 4 of the row flags at +0x2c).
	SingleCopy bool
	Code       string
	Quest      int
}

// questCodePairs are the codes the exe treats as the same quest object in either direction.
var questCodePairs = [][2]string{
	{"j34", "g34"}, {"bks", "bkd"}, {"d33", "g33"}, {"hst", "msf"}, {"hst", "vip"}, {"qf1", "qf2"},
	{"qf2", "qhr"}, {"qf2", "qey"}, {"qf2", "qbr"},
}

func sameQuestCode(a, b string) bool {
	if a == b {
		return true
	}

	for _, p := range questCodePairs {
		if (a == p[0] && b == p[1]) || (a == p[1] && b == p[0]) {
			return true
		}
	}

	return false
}

// Conflicts reports the 0x55a7d0 answer: a carried item blocks the pickup when it is the same single-copy unique,
// or a quest item of the same quest byte whose code is the candidate's (or its paired code).
func Conflicts(held []Held, c Candidate) bool {
	for _, h := range held {
		if h.Worn {
			continue
		}

		if c.SingleCopy && c.UniqueCode != "" && h.UniqueCode == c.UniqueCode {
			return true
		}

		if h.Quest != 0 && c.Quest != 0 && h.Quest == c.Quest && sameQuestCode(c.Code, h.Code) {
			return true
		}
	}

	return false
}

// CanPickUp is the part of 0x55aa20 that depends only on what the hero carries: a single-copy unique, or a quest
// item, is refused when Conflicts says so. Other items are always allowed.
func CanPickUp(held []Held, c Candidate) bool {
	if !c.SingleCopy && c.Quest == 0 {
		return true
	}

	return !Conflicts(held, c)
}
