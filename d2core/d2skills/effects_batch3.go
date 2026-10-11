package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Engine consumers of the effects of skills batch 3 (see
// d2common/d2skill/skills_batch3.go).

// formStates are the shape shift states (Wearwolf, Wearbear, Delirium change).
var formStates = []string{"wolf", "bear", "delerium"}

// Shapeshifted implements d2skill.StateHolder: the hero is in a form.
func (h *heroUnit) Shapeshifted() bool {
	set := h.e.setOf(h.p.ID())

	for _, s := range formStates {
		if set.Active(h.e.frame, s) {
			return true
		}
	}

	return false
}

var _ d2skill.StateHolder = (*heroUnit)(nil)

// toggleForm is the first step of SRVDO_116 (0x5c4e80): the States.txt group of
// the form state is cleared with the include-self flag (0x56a480); when that
// ended anything the cast is only a switch back to the human shape and the
// new state is not applied.
func toggleForm(set *d2state.Set, frame int, state string) (applied bool) {
	return !set.ClearGroup(frame, state)
}

// knockArea throws back the monsters inside a circle around a point, away from
// the hero (Leap: result flags 9 with no damage, 0x5d8e60). The circle is the
// exe's (squared distance <= radius squared).
func (e *Engine) knockArea(u *heroUnit, name string, cx, cy, radius, dist int) int {
	n := 0

	for _, m := range e.monstersWithin(cx, cy, radius) {
		if !m.Alive() {
			continue
		}

		e.target(m)
		e.pushAway(u, m, dist, name)

		n++
	}

	return n
}

// unitClearState removes a state from the one monster the effect names and
// lets go of the hold it caused (Leap Attack removes the stun, state 21).
func (e *Engine) unitClearState(ef *d2skill.Effect) {
	mt, _ := ef.Target.(*monsterTarget)
	if mt == nil || !mt.m.Alive() {
		return
	}

	e.setOf(mt.m.ID()).Remove(ef.State)

	if ef.State == d2state.Stun {
		e.monsters.ClearStun(mt.m)
	}

	e.emit("state", "STATE clear unit=%s state=%s", mt.m.Label(), ef.State)
}

// inAuraRange is the exe's aura circle: squared distance <= radius squared.
func inAuraRange(ax, ay, bx, by, radius int) bool {
	dx, dy := ax-bx, ay-by

	return dx*dx+dy*dy <= radius*radius
}

// canConvertMonster is the type test of SKILL_IsValidMonsterSkillTarget
// (0x56c0a0 -> 0x5dc1e0, VERIFIED): a unique (0x8) or super unique (0x2)
// monster, the +0x16 mask 0xa tested by 0x59dd60, is not a valid target.
// Champions, minions and ordinary monsters pass.
func canConvertMonster(typeFlags uint16) bool {
	return typeFlags&(d2mapentity.MonTypeUnique|d2mapentity.MonTypeSuperUnique) == 0
}
