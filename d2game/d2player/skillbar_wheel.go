package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// onWheelSkill is the mouse wheel (or whatever the user bound to "select previous / next skill"): it steps the right
// skill through the skills the hero can use (Shift: the left skill). OD2-ONLY: Game.exe 1.14b has no wheel skill
// cycling that we found (the 13 callers of UI_IsMouseWheelActive 0x4489f0 are the dialog stack, the key config tables and
// menu scrolling), so which hand it changes is our own design; the order is the skill id order shared with the gamepad
// bumpers (d2hero.CycleSkill).
func (g *GameControls) onWheelSkill(dir int, mod d2enum.KeyMod) {
	if g.escapeMenu.IsOpen() || g.NPCMenu.IsOpen() {
		return
	}

	left := mod&d2enum.KeyModShift != 0
	from := g.activeSkillID(left)

	id, ok := d2hero.CycleSkill(g.hero.Skills, from, left, dir)
	if !ok || id == from {
		g.Infof("SKILLBAR wheel hand=%s dir=%d: nothing to select", handName(left), dir)
		return
	}

	g.Infof("SKILLBAR wheel hand=%s dir=%d %q -> %q", handName(left), dir, g.skillName(from), g.skillName(id))

	_ = g.SelectSkill(left, id)
}
