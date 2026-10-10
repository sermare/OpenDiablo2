package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// The hero's ammunition for the skill engine (d2skills.Options.AmmoLeft / UseAmmo): a bow or crossbow in
// the right hand shoots from the quiver in the left hand, a javelin, throwing knife or throwing axe is its
// own stack. Nothing else counts. The stack sizes are the item quantities the save and the item generator
// hold; UseAmmo takes one off the live item (the game saves the quantity with the other containers).
//
// UNVERIFIED against the exe: a thrown weapon whose stack reaches 0 stays in the hand with quantity 0 (the
// original removes it) and "replenishes quantity" is not modelled.

// ammoStack returns the worn stack the hero's ranged attack draws on, or nil.
func (g *GameControls) ammoStack() *diablo2item.Item {
	if g.heroState == nil || g.inventory == nil {
		return nil
	}

	rules, bases, ok := g.heroState.EquipRules()
	if !ok {
		return nil
	}

	weapon, _ := g.inventory.WornAt(d2equip.LocRightHand).(*diablo2item.Item)
	if weapon == nil {
		return nil
	}

	wt := bases[weapon.GetItemCode()].Type

	if quiver := rules.Shoots(wt); quiver != "" {
		q, _ := g.inventory.WornAt(d2equip.LocLeftHand).(*diablo2item.Item)
		if q == nil || !rules.Types.IsA(bases[q.GetItemCode()].Type, quiver) {
			return nil
		}

		return q
	}

	if rules.IsThrowable(wt) && weapon.IsStackable() {
		return weapon
	}

	return nil
}

// AmmoLeft is the number of arrows, bolts or javelins the hero can still shoot or throw.
func (g *GameControls) AmmoLeft() int {
	if s := g.ammoStack(); s != nil {
		return s.Quantity()
	}

	return 0
}

// UseAmmo takes one arrow, bolt or javelin off the stack; false when there is none.
func (g *GameControls) UseAmmo() bool {
	s := g.ammoStack()
	if s == nil || s.Quantity() < 1 {
		return false
	}

	s.SetQuantity(s.Quantity() - 1)
	resetThrownDurability(s)

	return true
}

// resetThrownDurability is what the exe does with every shot (SKILL_ConsumeThrowQuantity 0x56a090, VERIFIED):
// the stack's durability is set back to its maximum, so a thrown javelin stack never wears down.
func resetThrownDurability(s *diablo2item.Item) {
	if cur, max := s.Durability(); max > 0 && cur != max {
		s.SetDurability(max)
	}
}
