package d2gamescreen

// goldPurse is the hero's purse: PickUpGold adds up to the carry cap and returns
// what did not fit (GameControls.PickUpGold).
type goldPurse interface {
	PickUpGold(amount int) (overflow int)
}

// groundGold is the ground-item side of a gold pickup: the pile being picked
// up, and the place where the remainder is left.
type groundGold interface {
	// Remove takes the picked-up pile off the ground.
	Remove()
	// Leave puts a new pile of amount where the picked-up one lay.
	Leave(amount int)
}

// pickUpGoldPile applies a gold pickup (0x558e40, VERIFIED): the purse takes
// what fits; a pile that does not fit at all stays untouched; otherwise the
// pile is removed and the overflow (if any) becomes a new pile at the same
// spot. It reports whether any gold was taken.
func pickUpGoldPile(purse goldPurse, pile groundGold, amount int) (taken bool) {
	over := purse.PickUpGold(amount)
	if over >= amount {
		return false
	}

	pile.Remove()

	if over > 0 {
		pile.Leave(over)
	}

	return true
}

// goldPileFuncs adapts two closures of the running game to groundGold.
type goldPileFuncs struct {
	remove func()
	leave  func(amount int)
}

func (g goldPileFuncs) Remove()          { g.remove() }
func (g goldPileFuncs) Leave(amount int) { g.leave(amount) }
