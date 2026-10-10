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

	for l := d2equip.Loc(1); l < d2equip.NumLocs; l++ {
		if it, ok := g.inventory.WornAt(l).(*diablo2item.Item); ok && it != nil && it.Origin() != nil {
			if name := tables.SetItemIndex(it.Origin()); name != "" {
				ctx.WornSetItems[name] = true
			}
		}
	}

	return ctx
}
