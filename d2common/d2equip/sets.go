package d2equip

// Weapon sets. A character has two weapon sets: set I lives in LocRightHand /
// LocLeftHand (4, 5), set II in LocSwapRight / LocSwapLeft (11, 12), and the
// header field at 0x10 of the .d2s ("active_arms") says which one is in the
// hands: 0 set I, 1 set II (UNVERIFIED which value is which: the sample has 0
// with the weapons in 4 and 5).
//
// The slots of the file are physical: when set II is active the weapon the hero
// holds is the item in slot 11, not the one in slot 4. EffectiveLoc maps a file
// slot to where the item currently is.

// SetCount is the number of weapon sets.
const SetCount = 2

// EffectiveLoc maps a physical location to the one the game treats the item as
// being at: with set II active, 11/12 act as 4/5 and 4/5 as 11/12.
func EffectiveLoc(loc Loc, activeArms int) Loc {
	if activeArms%SetCount == 0 {
		return loc
	}

	switch loc {
	case LocRightHand:
		return LocSwapRight
	case LocLeftHand:
		return LocSwapLeft
	case LocSwapRight:
		return LocRightHand
	case LocSwapLeft:
		return LocLeftHand
	}

	return loc
}

// NextSet is the active set after a swap.
func NextSet(activeArms int) int { return (activeArms + 1) % SetCount }

// HandsOf returns the right and left hand items of the active set from a
// physical body.
func HandsOf(body map[Loc]*Item, activeArms int) (right, left *Item) {
	if activeArms%SetCount == 0 {
		return body[LocRightHand], body[LocLeftHand]
	}

	return body[LocSwapRight], body[LocSwapLeft]
}
