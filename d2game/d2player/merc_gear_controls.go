package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// This file is the engine side of giving items to the hero's mercenary and taking
// them back (the panel that shows the merc's body is not built yet). The decision is
// the pure rule of d2equip.Rules.MercGive (Game.exe 0x54b230); here the item moves
// between the cursor, the inventory and the merc's gear (hero.Merc.Items), the
// displaced item goes to the cursor and the gear is saved with the hero.

// MercGearHost connects the controls to the merc in the game: the merc's class and
// table stats (false without a merc) and a function called after the gear changed
// (the director re-applies the gear to the unit).
type MercGearHost struct {
	Ref     func() (d2hero.MercRef, bool)
	Changed func()
}

// SetMercGearHost sets the merc the controls give items to.
func (g *GameControls) SetMercGearHost(h MercGearHost) { g.mercHost = h }

// MercGiveVerdict is what GiveCursorToMerc reports.
type MercGiveVerdict struct {
	OK       bool
	Consumed bool
	Loc      d2equip.Loc
	Reason   d2equip.Reason
	Detail   string
	// Returned is the code of the item that was replaced, now on the cursor.
	Returned string
}

func (g *GameControls) mercRef() (d2hero.MercRef, bool) {
	if g.mercHost.Ref == nil || g.hero.Merc == nil {
		return d2hero.MercRef{}, false
	}

	return g.mercHost.Ref()
}

// GiveCursorToMerc gives the cursor item to the merc. On success the replaced item (if
// any) is on the cursor, a potion is used up, and the gear is saved. A refusal leaves
// the item on the cursor.
func (g *GameControls) GiveCursorToMerc() MercGiveVerdict {
	item := g.inventory.CursorItem()
	if item == nil {
		return MercGiveVerdict{Reason: d2equip.ReasonBodyLoc, Detail: "nothing on the cursor"}
	}

	ref, ok := g.mercRef()
	if !ok {
		return MercGiveVerdict{Reason: d2equip.ReasonBodyLoc, Detail: "no mercenary"}
	}

	s, isItem := g.storedOf(item, d2equip.LocNone)
	if !isItem {
		return MercGiveVerdict{Reason: d2equip.ReasonBodyLoc, Detail: "not an item a mercenary can use"}
	}

	res := g.heroState.GiveMercItem(g.hero.Merc, ref, s)
	v := MercGiveVerdict{OK: res.OK, Consumed: res.Consumed, Loc: res.Loc, Reason: res.Reason, Detail: res.Detail}

	if !res.OK {
		g.Infof("MERC give decision=refused item=%s reason=%q detail=%q", item.GetItemCode(), res.Reason, res.Detail)

		return v
	}

	if res.Consumed {
		g.inventory.SetCursorItem(nil)
		g.Infof("MERC give decision=ok item=%s consumed=true", item.GetItemCode())

		return v
	}

	var back InventoryItem

	if res.Returned != nil {
		realised, err := realiseStored(g.inventory.item, res.Returned)
		if err != nil {
			g.Warningf("MERC give: cannot rebuild the replaced item %q: %v", res.Returned.Code, err)
		} else {
			if res.Returned.D2S != nil {
				g.itemOrigin[realised] = res.Returned.D2S
			}

			back = realised
			v.Returned = res.Returned.Code
		}
	}

	g.inventory.SetCursorItem(back)
	g.afterMercGearChange()
	g.Infof("MERC give decision=ok item=%s slot=%s replaced=%s gear=%s", item.GetItemCode(), res.Loc, orNone(v.Returned),
		g.hero.Merc.MercItemsSummary())

	return v
}

// GiveInventoryToMerc gives the inventory item at a grid cell to the merc (the cursor
// must be empty). A refusal puts the item back where it was; a replaced item lands on
// the cursor.
func (g *GameControls) GiveInventoryToMerc(cellX, cellY int) MercGiveVerdict {
	if g.inventory.CursorItem() != nil {
		return MercGiveVerdict{Reason: d2equip.ReasonBodyLoc, Detail: "the cursor is not empty"}
	}

	item := g.inventory.grid.GetSlot(cellX, cellY)
	if item == nil {
		return MercGiveVerdict{Reason: d2equip.ReasonBodyLoc, Detail: "no item at the cell"}
	}

	x, y := item.InventoryGridSlot()

	g.inventory.grid.Remove(item)
	g.inventory.SetCursorItem(item)

	v := g.GiveCursorToMerc()
	if v.OK {
		return v
	}

	// refused: back to the cell
	g.inventory.SetCursorItem(nil)

	if err := g.inventory.grid.Set(x, y, item); err != nil {
		g.inventory.grid.AutoPlaceForPickup(item)
	}

	return v
}

// TakeMercItemToCursor takes the merc's item at a body location to the cursor (the
// cursor must be empty).
func (g *GameControls) TakeMercItemToCursor(loc d2equip.Loc) bool {
	if g.hero.Merc == nil || g.inventory.CursorItem() != nil || g.hero.Merc.MercItemAt(loc) == nil {
		return false
	}

	s := g.hero.Merc.TakeMercItem(loc)

	item, err := realiseStored(g.inventory.item, s)
	if err != nil {
		g.hero.Merc.Items = append(g.hero.Merc.Items, *s) // keep it on the merc
		g.Warningf("MERC take: cannot rebuild %q: %v", s.Code, err)

		return false
	}

	if s.D2S != nil {
		g.itemOrigin[item] = s.D2S
	}

	g.inventory.SetCursorItem(item)
	g.afterMercGearChange()
	g.Infof("MERC take item=%s slot=%s gear=%s", s.Code, loc, g.hero.Merc.MercItemsSummary())

	return true
}

func (g *GameControls) afterMercGearChange() {
	if g.mercHost.Changed != nil {
		g.mercHost.Changed()
	}

	if !g.equipNoSave {
		g.saveHero()
	}
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}

	return s
}

// MercStatItems is the hero's merc gear as stat list items for the director.
func (g *GameControls) MercStatItems() []d2statlist.Item {
	return g.heroState.MercStatItems(g.hero.Merc)
}
