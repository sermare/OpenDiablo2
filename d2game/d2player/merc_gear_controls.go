package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
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
	// Drink gives a potion's effect to the merc (healing and rejuvenation heal it; others
	// are used up with no effect). Optional.
	Drink func(d2inventory.PotionEffect)
	// DrinkState gives an antidote or thawing potion's timed state to the merc (the resist stats; the exe runs
	// the same item effect with the merc as the user, 0x54b040 -> 0x5bce10). Optional.
	DrinkState func(d2inventory.StatePotion)
	// View gives the numbers the mercenary panel shows (false without a merc). Optional.
	View func() (MercView, bool)
}

// SetMercGearHost sets the merc the controls give items to.
func (g *GameControls) SetMercGearHost(h MercGearHost) {
	g.mercHost = h
	g.mercPanel.SetHost(MercPanelHost{View: h.View, ItemCode: g.mercSlotCode})
}

// mercSlotCode is the code of the item the hero's merc wears at a panel slot.
func (g *GameControls) mercSlotCode(slot string) string {
	loc, ok := MercSlotLoc(slot)
	if !ok || g.hero.Merc == nil {
		return ""
	}

	if it := g.hero.Merc.MercItemAt(loc); it != nil {
		return it.Code
	}

	return ""
}

// ToggleMercPanel opens or closes the mercenary screen (it opens only when the hero has a merc).
func (g *GameControls) ToggleMercPanel() {
	if g.hero.Merc == nil && !g.mercPanel.IsOpen() {
		return
	}

	g.openLeftPanel(g.mercPanel)
}

// mercPanelClick handles a left click on a body slot of the open mercenary screen: the item on the cursor
// is given to the merc (the rules of GiveCursorToMerc), or, with an empty cursor, the merc's item is taken.
func (g *GameControls) mercPanelClick(mx, my int) bool {
	if !g.mercPanel.IsOpen() {
		return false
	}

	slot := Mode800.MercSlotAt(mx, my)
	loc, ok := MercSlotLoc(slot)

	if !ok {
		return false
	}

	if g.inventory.CursorItem() != nil {
		v := g.GiveCursorToMerc()
		if !v.OK {
			g.Infof("MERC panel: refused slot=%s reason=%v %s", slot, v.Reason, v.Detail)
		}
	} else {
		g.TakeMercItemToCursor(loc)
	}

	g.mercPanel.refresh()

	return true
}

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
		if it, isPotion := item.(*diablo2item.Item); isPotion && g.mercHost.Drink != nil {
			// a non-player drinker: life x2, mana x1 (PLAYER_ScaleValueByClassGroupA/B 0x62a730 / 0x62a780)
			g.mercHost.Drink(d2inventory.ScaledPotion(d2inventory.PotionEffectOf(it.CommonRecord()), 0, false, 0, 0, nil))

			if sp, isState := d2inventory.StatePotionOf(it.CommonRecord()); isState && g.mercHost.DrinkState != nil {
				g.mercHost.DrinkState(sp)
			}
		}

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

// renderMercPortrait draws the hireling portrait of the HUD (UI_DrawHirelingPortraitStrip). It shares the top left
// corner with the left panels, so it is hidden while one is open. The exe hides the strip (and resets its tooltip
// flags) when the automap/view display mode (AUTOMAP_GetDisplayMode 0x4565d0) is 2 or 3, which is the view shift of an
// open left panel (UI_SetPanelState 0x4515f0 sets it for the character, quest log, party, mercenary... panels), and
// when panel 9 (game menu) or 0xb (key configuration) is open (UI_DrawHirelingPortraitStrip 0x4904f0, w44; the
// register argument of the mode setter is not shown by the decompiler, so the mode number per panel is inferred).
func (g *GameControls) renderMercPortrait(target d2interface.Surface) {
	v, ok := g.portraitShown()
	if !ok {
		return
	}

	// the exe draws it in screen coordinates (the strip starts at the left edge of the screen), not in the interface column
	ox, oy := d2display.ColumnOrigin()

	target.PushTranslation(-ox, -oy)
	g.mercPortrait.Render(target, v)

	if sx, sy := g.lastMouseX+ox, g.lastMouseY+oy; MercPortraitHit(sx, sy) {
		g.mercPortrait.RenderTooltip(target, g.mercPortraitTooltip(), sx, sy)
	}

	target.Pop()
}

// portraitPoint is a point of the portrait picture in interface-column coordinates (what the mouse handlers receive).
func portraitPoint() (x, y int) {
	ox, oy := d2display.ColumnOrigin()

	return mercPortraitX + 5 - ox, mercPortraitBarTop + 10 - oy
}

func (g *GameControls) mercPortraitTooltip() []string {
	return MercPortraitTooltip(g.asset.TranslateString("hireiconinfo2"), g.asset.TranslateString("hireiconinfo1"), "O")
}

// portraitShown reports whether the portrait is on screen (a merc exists and no left panel covers it).
func (g *GameControls) portraitShown() (MercView, bool) {
	if g.mercPortrait == nil || g.mercHost.View == nil || g.isLeftPanelOpen() {
		return MercView{}, false
	}

	v, ok := g.mercHost.View()

	return v, ok && MercPortraitIcon(v.Class) != ""
}

// mercPortraitClick handles a click on the portrait: the left button drops the cursor item on the merc (a potion
// heals it, gear is equipped: GiveCursorToMerc), the right button opens the merc's inventory ("Right-click to Open
// Inventory", "Drop Potion on Portrait to Heal": strings hireiconinfo2 / hireiconinfo1).
func (g *GameControls) mercPortraitClick(left bool, mx, my int) bool {
	ox, oy := d2display.ColumnOrigin()

	if !MercPortraitHit(mx+ox, my+oy) {
		return false
	}

	if _, ok := g.portraitShown(); !ok {
		return false
	}

	if !left {
		g.ToggleMercPanel()

		return true
	}

	if g.inventory.CursorItem() == nil {
		return false
	}

	if v := g.GiveCursorToMerc(); !v.OK {
		g.Infof("MERC portrait: refused reason=%v %s", v.Reason, v.Detail)
	} else {
		g.Infof("MERC portrait: item given consumed=%v", v.Consumed)
	}

	return true
}

// OpenMercPanel opens the mercenary screen if it is closed (the first hire does it once: PopupHireling).
func (g *GameControls) OpenMercPanel() {
	if g.hero.Merc != nil && !g.mercPanel.IsOpen() {
		g.openLeftPanel(g.mercPanel)
	}
}

// IsMercPanelOpen says whether the mercenary screen is showing.
func (g *GameControls) IsMercPanelOpen() bool { return g.mercPanel.IsOpen() }
