package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2itemdesc"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// describeContext tells the item descriptions who is looking at the item:
// the hero's level and attributes (unmet requirements are shown red), class,
// and the set pieces that are worn (set bonuses).
func (g *GameControls) describeContext() *d2itemdesc.Context {
	if g.hero == nil || g.hero.Stats == nil {
		return nil
	}

	ctx := &d2itemdesc.Context{
		HasHero: true, CharLevel: g.hero.Stats.Level,
		Strength: g.hero.Stats.Strength, Dexterity: g.hero.Stats.Dexterity, Class: -1,
		WornSetItems: make(map[string]bool),
	}

	if class, ok := d2hero.D2SClassOf(g.hero.Class); ok {
		ctx.Class = int(class)
	}

	tables := g.inventory.item.DescriptionTables()
	if tables == nil {
		return ctx
	}

	ctx.CarriedSetItems = make(map[string]bool)

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it, ok := g.inventory.WornAt(l).(*diablo2item.Item); ok && it != nil && it.Origin() != nil {
			if name := tables.SetItemIndex(it.Origin()); name != "" {
				ctx.CarriedSetItems[name] = true

				// INV_CountEquippedSetMates 0x62a4e0 (VERIFIED) leaves out a broken piece and one with flag 0x4000
				// (a weapon at 0 durability carries no broken flag, so the right hand never counts as broken)
				if !it.HasBrokenFlag() && !(it.IsBroken() && l != d2equip.LocRightHand) {
					ctx.WornSetItems[name] = true
				}
			}
		}
	}

	// the piece list of the set tooltip paints a piece green for any identified copy the hero carries: backpack,
	// cube, stash (INV_FindSetItemMatchInInventory 0x4828e0, VERIFIED)
	for _, grid := range []*ItemGrid{g.inventory.grid, g.stash.grid, g.cube.grid} {
		if grid == nil {
			continue
		}

		for _, gi := range grid.Items() {
			it, ok := gi.(*diablo2item.Item)
			if !ok || it == nil || it.Origin() == nil || !(it.Origin().Identified || it.IsIdentified()) {
				continue
			}

			if name := tables.SetItemIndex(it.Origin()); name != "" {
				ctx.CarriedSetItems[name] = true
			}
		}
	}

	return ctx
}
