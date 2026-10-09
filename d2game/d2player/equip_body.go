package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
)

// The hero's body is kept in the inventory grid's equipment slots (the places
// the panel draws and the trade windows read). The locations of d2equip map to
// them like this; the weapon set that is not in the hands (d2equip.LocSwapRight
// and LocSwapLeft) lives in Inventory.swapSet and is not drawn.
var locSlots = map[d2equip.Loc]d2enum.EquippedSlot{
	d2equip.LocHead:      d2enum.EquippedSlotHead,
	d2equip.LocNeck:      d2enum.EquippedSlotNeck,
	d2equip.LocTorso:     d2enum.EquippedSlotTorso,
	d2equip.LocRightHand: d2enum.EquippedSlotRightArm, // the weapon: "rArm" of inventory.txt
	d2equip.LocLeftHand:  d2enum.EquippedSlotLeftArm,  // the shield: "lArm"
	d2equip.LocRightRing: d2enum.EquippedSlotRightHand,
	d2equip.LocLeftRing:  d2enum.EquippedSlotLeftHand,
	d2equip.LocBelt:      d2enum.EquippedSlotBelt,
	d2equip.LocFeet:      d2enum.EquippedSlotLegs,
	d2equip.LocGloves:    d2enum.EquippedSlotGloves,
}

// drawnLocs are the locations with a slot on the panel, in a fixed order.
var drawnLocs = []d2equip.Loc{
	d2equip.LocHead, d2equip.LocNeck, d2equip.LocTorso, d2equip.LocRightHand, d2equip.LocLeftHand,
	d2equip.LocRightRing, d2equip.LocLeftRing, d2equip.LocBelt, d2equip.LocFeet, d2equip.LocGloves,
}

// WornAt returns the item at a body location (nil if none).
func (g *Inventory) WornAt(loc d2equip.Loc) InventoryItem {
	if loc.IsSwap() {
		return g.swapSet[loc]
	}

	slot, ok := locSlots[loc]
	if !ok {
		return nil
	}

	return g.grid.equipmentSlots[slot].item
}

// SetWorn puts an item (nil clears) at a body location.
func (g *Inventory) SetWorn(loc d2equip.Loc, item InventoryItem) {
	if loc.IsSwap() {
		if g.swapSet == nil {
			g.swapSet = map[d2equip.Loc]InventoryItem{}
		}

		if item == nil {
			delete(g.swapSet, loc)
		} else {
			g.swapSet[loc] = item
			g.grid.Load(item)
		}

		return
	}

	slot, ok := locSlots[loc]
	if !ok {
		return
	}

	g.grid.ChangeEquippedSlot(slot, item)

	if item != nil {
		g.grid.Load(item)
	}
}

// Body returns every worn item by location (including the other weapon set).
func (g *Inventory) Body() map[d2equip.Loc]InventoryItem {
	out := map[d2equip.Loc]InventoryItem{}

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it := g.WornAt(l); it != nil {
			out[l] = it
		}
	}

	return out
}

// ClearBody removes everything worn (both weapon sets).
func (g *Inventory) ClearBody() {
	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		g.SetWorn(l, nil)
	}
}

// LocAt returns the body location whose slot is under the screen point.
func (g *Inventory) LocAt(mx, my int) (d2equip.Loc, bool) {
	for _, l := range drawnLocs {
		slot := g.grid.equipmentSlots[locSlots[l]]
		if mx > slot.x && mx < slot.x+slot.width && my < slot.y && my > slot.y-slot.height {
			return l, true
		}
	}

	return d2equip.LocNone, false
}

// SlotCenter returns the screen point in the middle of a location's slot (for
// scripted clicks).
func (g *Inventory) SlotCenter(loc d2equip.Loc) (x, y int, ok bool) {
	s, found := locSlots[loc]
	if !found {
		return 0, 0, false
	}

	slot := g.grid.equipmentSlots[s]

	return slot.x + slot.width/2, slot.y - slot.height/2, true
}

// SwapWeaponSets exchanges the weapon set in the hands with the other one and
// flips ActiveArms so the file locations of the items do not change.
func (g *Inventory) SwapWeaponSets() {
	a, b := g.WornAt(d2equip.LocRightHand), g.WornAt(d2equip.LocLeftHand)
	c, d := g.WornAt(d2equip.LocSwapRight), g.WornAt(d2equip.LocSwapLeft)

	g.SetWorn(d2equip.LocRightHand, c)
	g.SetWorn(d2equip.LocLeftHand, d)
	g.SetWorn(d2equip.LocSwapRight, a)
	g.SetWorn(d2equip.LocSwapLeft, b)

	g.activeArms = d2equip.NextSet(g.activeArms)
}

// ActiveArms is the weapon set in the hands: 0 set I, 1 set II.
func (g *Inventory) ActiveArms() int { return g.activeArms }
