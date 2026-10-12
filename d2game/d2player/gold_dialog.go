package d2player

import (
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The gold dialogs and the stash gold of the hero (w48, Game.exe 1.14b):
//
//	inventory coin button  UI_OpenPanelByIndex(3) 0x44f7b0: "deposit", the edit box prefilled with the carried gold,
//	                       OK sends packet 0x4f action 0x14 (stash open) -- VERIFIED. With the stash closed the original
//	                       has no other caller; the "drop gold" dialog (kinds 0/1, packet 0x50) is U, this build opens it
//	                       when the stash is closed, which is what the 1.14 game does in play.
//	stash coin button      UI_OpenPanelByIndex(4): "withdraw", limit = the stash gold, packet 0x4f action 0x13.
//
// The dialogs need an empty cursor (otherwise the exe resets the overlays instead of opening).

// stashCoin is the coin button and the amount line of the stash panel.
type stashCoin struct {
	group  *d2ui.WidgetGroup
	button *d2ui.Button
	label  *d2ui.Label
}

// loadStashCoin builds the stash coin button (UI_DrawGoldAmountAndTooltip 0x484250, GoldStash variant).
func (g *GameControls) loadStashCoin() {
	lay := Mode800.Gold(GoldStash, true) // the stash is open, so UI_GetOpenPanelMask != 0: the amount stands in the box under the title (w51)

	c := &stashCoin{group: g.ui.NewWidgetGroup(d2ui.RenderPriorityInventory)}

	c.button = g.ui.NewButton(d2ui.ButtonTypeGoldCoin, "")
	c.button.SetPosition(lay.ButtonX, lay.ButtonY-lay.HitH)
	c.button.OnActivated(func() { g.OpenGoldDialog(d2inventory.GoldDialogWithdraw) })
	c.group.AddWidget(c.button)

	c.label = g.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	c.label.Alignment = d2ui.HorizontalAlignLeft
	c.label.SetPosition(lay.TextX, lay.TextY-14)
	c.group.AddWidget(c.label)

	c.group.SetVisible(false)
	g.stashCoin = c
}

// advanceStashCoin shows the coin and the stash gold while the stash is open.
func (g *GameControls) advanceStashCoin() {
	if g.stashCoin == nil {
		return
	}

	open := g.stash != nil && g.stash.IsOpen()
	g.stashCoin.group.SetVisible(open)

	if open {
		g.stashCoin.label.SetText(strconv.Itoa(g.hero.StashGold))
	}
}

// heroGoldLimit is the carry limit of the hero's level (PLAYER_GetMaxGoldCarry 0x623050).
func (g *GameControls) heroGoldLimit() int {
	level := 1
	if g.hero.Stats != nil {
		level = g.hero.Stats.Level
	}

	return d2inventory.InventoryGoldLimit(level)
}

// OpenGoldDialog opens the gold dialog of a kind; it does nothing with an item on the cursor.
func (g *GameControls) OpenGoldDialog(kind d2inventory.GoldDialogKind) bool {
	if g.inventory.CursorItem() != nil {
		return false
	}

	limit := d2inventory.GoldDialogLimit(kind, g.hero.Gold, g.hero.StashGold)
	g.inventory.moveGoldPanel.OpenKind(kind, limit, 0)

	return true
}

// onInventoryGoldClick is the click on the inventory coin button.
func (g *GameControls) onInventoryGoldClick() {
	if g.inventory.moveGoldPanel.IsOpen() {
		g.inventory.moveGoldPanel.Close()

		return
	}

	kind := d2inventory.GoldDialogDrop
	if g.stash != nil && g.stash.IsOpen() {
		kind = d2inventory.GoldDialogDeposit
	}

	g.OpenGoldDialog(kind)
}

// saveGold writes the hero after a bank move (the stash gold travels with the save packet).
func (g *GameControls) saveGold() {
	if g.inputListener != nil {
		g.saveHero()
	}
}

// GoldMove applies a confirmed gold dialog (the server side of packets 0x50 and 0x4f) and reports whether it
// changed anything.
func (g *GameControls) GoldMove(kind d2inventory.GoldDialogKind, amount int) bool {
	switch kind {
	case d2inventory.GoldDialogDrop:
		left, dropped, ok := d2inventory.DropGold(g.hero.Gold, amount, g.heroGoldLimit())
		if !ok || dropped == 0 {
			return false
		}

		g.AddGold(left - g.hero.Gold)

		if g.OnDropGold != nil {
			g.OnDropGold(dropped)
		}

		g.Infof("GOLD drop amount=%d gold=%d", dropped, g.hero.Gold)

		return true
	case d2inventory.GoldDialogDeposit:
		c, s, ok := d2inventory.DepositGold(g.hero.Gold, g.hero.StashGold, amount, d2inventory.StashGoldLimit)
		if !ok {
			return false
		}

		g.AddGold(c - g.hero.Gold)
		g.hero.StashGold = s
		g.Infof("GOLD deposit amount=%d gold=%d stash=%d", amount, c, s)
		g.saveGold()

		return true
	case d2inventory.GoldDialogWithdraw:
		c, s, ok := d2inventory.WithdrawGold(g.hero.Gold, g.hero.StashGold, amount, g.heroGoldLimit())
		if !ok {
			g.sayHeroVoice(0x13) // the error ping: UNITSND 0x13 as for every refused action
			g.Infof("GOLD withdraw amount=%d refused gold=%d stash=%d", amount, g.hero.Gold, g.hero.StashGold)

			return false
		}

		g.AddGold(c - g.hero.Gold)
		g.hero.StashGold = s
		g.Infof("GOLD withdraw amount=%d gold=%d stash=%d", amount, c, s)
		g.saveGold()

		return true
	}

	return false
}
