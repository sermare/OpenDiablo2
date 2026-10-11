package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// A worn stack that ran out (PLRMODE_ServerHandleDepletedStackItem 0x57e020, d2equip.DepletedStackAction):
// the empty stack leaves the body slot, and for item types with the Reload column another stack of the same
// base item from the inventory grid is worn in its place. A magical weapon stack is broken instead (its
// durability goes to 0, which switches its properties off). Notes: gaps-slice-G.md.

// handleDepletedAmmo runs after a shot or throw took the last piece of the stack s.
func (g *GameControls) handleDepletedAmmo(s *diablo2item.Item) {
	if s == nil || s.Quantity() > 0 || g.heroState == nil || g.inventory == nil {
		return
	}

	rules, bases, ok := g.heroState.EquipRules()
	if !ok {
		return
	}

	loc := d2equip.LocRightHand
	if g.inventory.WornAt(loc) != InventoryItem(s) {
		loc = d2equip.LocLeftHand
		if g.inventory.WornAt(loc) != InventoryItem(s) {
			return
		}
	}

	typ := bases[s.GetItemCode()].Type
	cur, max := s.Durability()
	q := s.Quality()

	switch d2equip.DepletedStackAction(d2equip.DepletedItem{
		Stackable: s.IsStackable(), Throwable: rules.IsThrowable(typ), Quantity: s.Quantity(),
		WeaponType:   rules.Types.IsA(typ, "weap"),
		MagicQuality: q >= d2drop.QualityMagic && q <= 9, // 9 is tempered
		Broken:       max > 0 && cur <= 0,
	}) {
	case d2equip.DepletedBreak:
		s.SetDurability(0)
		g.Infof("AMMO depleted magical stack %s broken", s.GetItemCode())
	case d2equip.DepletedUnequip:
		g.inventory.SetWorn(loc, nil)
		g.Infof("AMMO depleted stack %s leaves %s", s.GetItemCode(), loc)

		if !rules.IsReload(typ) {
			return
		}

		var cands []d2equip.StackCandidate

		items := g.inventory.grid.Items()
		for _, it := range items {
			cands = append(cands, d2equip.StackCandidate{Code: it.GetItemCode()})
		}

		if i := d2equip.ReplacementStack(s.GetItemCode(), cands); i >= 0 {
			next := items[i]
			g.inventory.grid.Remove(next)
			g.inventory.SetWorn(loc, next)
			g.Infof("AMMO re-equipped another stack of %s at %s", next.GetItemCode(), loc)
		}
	case d2equip.DepletedNone, d2equip.DepletedKeep:
	}
}
