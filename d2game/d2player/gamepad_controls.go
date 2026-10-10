package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
)

// BindGamepad connects the gamepad controller to this game: key actions use
// the keys the player bound (the key map), the left stick moves the cursor
// instead of walking while a panel or menu is open, and the bumpers cycle the
// left and right skills. UnbindGamepad undoes it when the game screen unloads.
func (g *GameControls) BindGamepad() {
	pad := d2gamepad.Default()

	pad.SetKeyResolver(func(e d2enum.GameEvent) (d2enum.Key, bool) {
		if g.keyMap == nil {
			return 0, false
		}

		b := g.keyMap.GetKeysForGameEvent(e)
		if b == nil {
			return 0, false
		}

		for _, k := range []d2enum.Key{b.Primary, b.Secondary} {
			if k >= d2enum.KeyMin && k <= d2enum.KeyMax {
				return k, true
			}
		}

		return 0, false
	})
	pad.SetUIMode(g.gamepadUIMode)
	pad.SetActionHandler(g.onGamepadAction)
}

// UnbindGamepad removes the hooks of BindGamepad.
func (g *GameControls) UnbindGamepad() {
	pad := d2gamepad.Default()
	pad.SetKeyResolver(nil)
	pad.SetUIMode(nil)
	pad.SetActionHandler(nil)
}

// gamepadUIMode reports a panel or menu is in the way of walking.
func (g *GameControls) gamepadUIMode() bool {
	return g.hasOpenPanels() || g.escapeMenu.IsOpen() || g.HelpOverlay.IsOpen() ||
		g.NPCMenu.IsOpen() || g.Waypoints.IsOpen() || g.PTrade.IsOpen()
}

func (g *GameControls) onGamepadAction(a d2gamepad.Action) {
	left, dir := false, 1

	switch a {
	case d2gamepad.ActionCycleLeftSkill:
		left = true
	case d2gamepad.ActionPrevLeftSkill:
		left, dir = true, -1
	case d2gamepad.ActionPrevRightSkill:
		dir = -1
	case d2gamepad.ActionCycleRightSkill:
	default:
		return
	}

	id, ok := d2hero.CycleSkill(g.hero.Skills, g.activeSkillID(left), left, dir)
	if !ok {
		g.Infof("GAMEPAD cycle %s skill: nothing to select", handName(left))
		return
	}

	g.Infof("GAMEPAD cycle %s skill dir=%d -> %q", handName(left), dir, g.skillName(id))

	_ = g.SelectSkill(left, id)
}

// AutoPadState describes what the gamepad changed, for the pad:state step of
// the autoscript harness: the open panels, the automap and the active skills.
func (g *GameControls) AutoPadState() string {
	return fmt.Sprintf("inventory=%v character=%v skilltree=%v quests=%v automap=%v ui=%v left=%q right=%q",
		g.inventory.IsOpen(), g.heroStatsPanel.IsOpen(), g.skilltree.IsOpen(), g.questLog.IsOpen(), g.automap.On(),
		g.gamepadUIMode(), g.skillName(g.activeSkillID(true)), g.skillName(g.activeSkillID(false)))
}
