package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// handleContainerClick gives a left click to the belt, the stash, the cube and
// the inventory, in that order, and reports whether one of them took it.
func (g *GameControls) handleContainerClick(mx, my int, ctrl bool) bool {
	if g.belt.HandleClick(mx, my) {
		return true
	}

	if g.stash.HandleClick(mx, my, ctrl) || g.cube.HandleClick(mx, my, ctrl) {
		return true
	}

	if g.inventory.HandleClick(mx, my, ctrl) {
		switch g.inventory.lastClick {
		case ClickPlace, ClickSwap, ClickAuto:
			g.saveHero()
		}

		return true
	}

	return false
}

// handleContainerRightClick opens the Horadric Cube from a right click on it
// and drinks a potion that is right-clicked in the inventory.
func (g *GameControls) handleContainerRightClick(mx, my int) bool {
	var item InventoryItem

	var from *ItemGrid

	switch {
	case g.inventory.IsOpen() && g.inventory.grid.Contains(mx, my):
		from = g.inventory.grid
	case g.stash.IsOpen() && g.stash.grid.Contains(mx, my):
		from = g.stash.grid
	default:
		return false
	}

	item = from.ItemAtScreen(mx, my)
	if item == nil {
		return true
	}

	if item.GetItemCode() == cubeItemCode {
		g.OpenCube()
		return true
	}

	if it, ok := item.(*diablo2item.Item); ok && !d2inventory.PotionEffectOf(it.CommonRecord()).IsEmpty() {
		from.Remove(item)
		g.drink(it)
		g.saveHero()
	}

	return true
}

// OpenStash opens the stash next to the inventory (the stash object of the
// town does this; the original also shows the inventory with it).
func (g *GameControls) OpenStash() {
	g.NPCMenu.Close()
	g.clearScreen()
	g.inventory.Open()
	g.stash.Open()
	g.updateLayout()
	g.Infof("stash opened: items=%d", len(g.stash.grid.items))
}

// OpenCube opens the Horadric Cube panel next to the inventory.
func (g *GameControls) OpenCube() {
	g.clearLeftScreenSide()
	g.inventory.Open()
	g.cube.Open()
	g.updateLayout()
	g.Infof("cube opened: items=%d", len(g.cube.grid.items))
}

// IsStashOpen reports whether the stash panel is open.
func (g *GameControls) IsStashOpen() bool { return g.stash.IsOpen() }

// UseBeltColumn drinks the front potion of a belt column (hotkeys 1 to 4).
func (g *GameControls) UseBeltColumn(col int) {
	if g.UseBeltColumnNoSave(col) {
		g.saveHero()
	}
}

// UseBeltColumnNoSave drinks the potion like UseBeltColumn but leaves saving
// to the caller; it reports whether a potion was drunk.
func (g *GameControls) UseBeltColumnNoSave(col int) bool {
	item, ok := g.belt.Use(col)
	if !ok {
		g.Infof("BELT slot %d is empty", col+1)
		return false
	}

	if it, isItem := item.(*diablo2item.Item); isItem {
		g.Infof("BELT use slot=%d code=%s name=%q", col+1, it.GetItemCode(), itemName(it))
		g.drink(it)
	}

	return true
}

// AutoBeltUse is the scripted potion test (OD2_AUTOBELT): it puts the hero at
// 40% life and mana, drinks the potion of a belt column, lets the over-time part
// run out, logs the numbers and restores the belt and the stats, saving
// nothing. It returns the log lines.
func (g *GameControls) AutoBeltUse(col int) []string {
	st := g.hero.Stats
	savedBelt, savedHP, savedMana, savedRegen := g.belt.items, st.Health, st.Mana, g.regen

	defer func() {
		g.belt.items, st.Health, st.Mana, g.regen = savedBelt, savedHP, savedMana, savedRegen
		g.regenHP, g.regenMana = 0, 0
	}()

	st.Health, st.Mana = st.MaxHealth*2/5, st.MaxMana*2/5 //nolint:gomnd // 40%
	startHP, startMana := st.Health, st.Mana

	cell, ok := d2inventory.FrontOfColumn(g.belt.kinds(), g.belt.Boxes(), col)
	if !ok {
		return []string{fmt.Sprintf("slot=%d empty", col+1)}
	}

	item := g.belt.items[cell]
	g.UseBeltColumnNoSave(col)

	afterHP, afterMana := st.Health, st.Mana

	// 60 seconds is longer than any potion lasts
	for i := 0; i < 600 && g.regen.Active(); i++ {
		g.advancePotions(0.1)
	}

	return []string{fmt.Sprintf("slot=%d cell=%d code=%s name=%q life %d/%d -> %d (instant) -> %d (after regen) mana %d/%d -> %d -> %d",
		col+1, cell, item.GetItemCode(), itemName(item), startHP, st.MaxHealth, afterHP, st.Health,
		startMana, st.MaxMana, afterMana, st.Mana)}
}

// drink applies a potion: the instant part at once, the over-time part
// through the regen that advancePotions ticks. A potion without an effect in
// the tables is used up without one.
func (g *GameControls) drink(it *diablo2item.Item) {
	e := d2inventory.PotionEffectOf(it.CommonRecord())
	st := g.hero.Stats

	before := st.Health
	beforeMana := st.Mana

	if e.InstantHPPercent > 0 {
		st.Health = d2inventory.ApplyInstant(st.Health, st.MaxHealth, e.InstantHPPercent)
	}

	if e.InstantManaPercent > 0 {
		st.Mana = d2inventory.ApplyInstant(st.Mana, st.MaxMana, e.InstantManaPercent)
	}

	g.regen.Add(e)

	g.Infof("BELT drink code=%s instant_hp=%d%% instant_mana=%d%% over_time_hp=%.0f over_time_mana=%.0f seconds=%.1f"+
		" health=%d->%d mana=%d->%d", it.GetItemCode(), e.InstantHPPercent, e.InstantManaPercent, e.HP, e.Mana, e.Seconds,
		before, st.Health, beforeMana, st.Mana)
}

// advancePotions restores the over-time part of the potions drunk.
func (g *GameControls) advancePotions(elapsed float64) {
	if !g.regen.Active() || g.hero.Stats == nil {
		return
	}

	hp, mana := g.regen.Tick(elapsed)

	st := g.hero.Stats
	g.regenHP += hp
	g.regenMana += mana

	if whole := int(g.regenHP); whole > 0 {
		st.Health += whole
		g.regenHP -= float64(whole)

		if st.Health > st.MaxHealth {
			st.Health = st.MaxHealth
		}
	}

	if whole := int(g.regenMana); whole > 0 {
		st.Mana += whole
		g.regenMana -= float64(whole)

		if st.Mana > st.MaxMana {
			st.Mana = st.MaxMana
		}
	}
}

// AutoPlaceCursor moves the cursor item to the belt when it is a potion the
// belt takes (misc.txt autobelt), else into the inventory with the original's
// auto-placement search. It returns the slot, and false (leaving the item on
// the cursor) when neither has room.
func (g *GameControls) AutoPlaceCursor() (x, y int, ok bool) {
	if cur := g.inventory.CursorItem(); cur != nil {
		if cell, onBelt := g.belt.AutoBelt(cur); onBelt {
			g.inventory.SetCursorItem(nil)
			g.Infof("BELT auto-placed %s in cell %d", cur.GetItemCode(), cell)

			return cell, 0, true
		}
	}

	x, y, ok = g.inventory.AutoPlaceCursor()
	if ok {
		g.saveHero()
	}

	return x, y, ok
}
