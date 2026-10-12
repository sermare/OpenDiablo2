package d2player

import "os"

// AutoMercUI is the interface half of the OD2_AUTOMERC_UI probe (scenario 84-mercenary): it drops a healing potion on the
// portrait, gives and takes gear through the portrait and the merc screen, and tries an item the merc cannot wear. It
// uses the code paths a click does and logs "MERC ui ..." lines. The caller has hurt the merc first.
func (g *GameControls) AutoMercUI(log func(format string, a ...interface{})) {
	if _, ok := g.portraitShown(); !ok {
		log("MERC ui portrait not shown")

		return
	}

	px, py := portraitPoint()

	log("MERC ui portrait shown tooltip=%q", g.mercPortraitTooltip())

	// items made in the game cannot be written to the save yet (exportMercItems): the probe leaves the gear as it was
	gear, dirty := append(g.hero.Merc.Items[:0:0], g.hero.Merc.Items...), g.hero.Merc.ItemsDirty

	defer func() {
		g.hero.Merc.Items, g.hero.Merc.ItemsDirty = gear, dirty
		g.afterMercGearChange()
	}()

	// a potion dropped on the portrait is used up and heals the merc (the caller logs the life before and after)
	if p := g.newItem("hp5"); p != nil {
		g.inventory.SetCursorItem(p)
		hit := g.mercPortraitClick(true, px, py)
		log("MERC ui potion drop handled=%v cursor_empty=%v", hit, g.inventory.CursorItem() == nil)
	}

	// right click opens the merc's screen, a second one closes it
	g.mercPortraitClick(false, px, py)
	log("MERC ui right click merc_screen_open=%v", g.mercPanel.IsOpen())

	// OD2_AUTOMERC_UI_HOLD=1 leaves the merc screen open (for a frame by OD2_AUTOSHOT) and skips the rest of the probe
	if os.Getenv("OD2_AUTOMERC_UI_HOLD") != "" && g.mercPanel.IsOpen() {
		return
	}

	// gear dropped on the portrait is worn (a cap), taken off by the screen's slot and put back
	if cap := g.newItem("cap"); cap != nil {
		g.inventory.SetCursorItem(cap)
		g.mercPanel.Close()
		g.mercPortraitClick(true, px, py)
		log("MERC ui gear gear=%s cursor_empty=%v", g.hero.Merc.MercItemsSummary(), g.inventory.CursorItem() == nil)
	}

	// a shield (the Rogue Scout, the Iron Wolf and the Barbarian do not use one; the Desert Mercenary does) is refused or
	// worn by the class rule and stays on the cursor when refused
	for _, code := range []string{"tow", "uow"} {
		heavy := g.newItem(code)
		if heavy == nil {
			continue
		}

		g.inventory.SetCursorItem(heavy)
		g.mercPortraitClick(true, px, py)
		log("MERC ui shield item=%s on_cursor=%v", code, g.inventory.CursorItem() != nil)
		g.inventory.SetCursorItem(nil)
	}
}
