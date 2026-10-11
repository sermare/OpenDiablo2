package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Client side monster skills driven by animation frames or effects (cltdofunc
// 59, 62, 64, 75). The pure rules are in d2monster/clientskill.go; this file binds
// them to the monster entity and the shake hook. Server side outcomes (damage,
// movement of the Director) are unchanged.

func skillParams(rec *d2records.SkillRecord) [7]int {
	return [7]int{0, rec.Param1, rec.Param2, rec.Param3, rec.Param4, rec.Param5, rec.Param6}
}

// mosquitoBites is CLTST_042_Mosquito (0x4df2b0): calc1 plus a roll below calc2 -
// calc1, at least 1. The exe seeds the roll from the two unit ids; the cosmetic
// generator is used here.
func mosquitoBites(c1, c2, roll int) int {
	n := c1
	if c2 > c1 {
		n += roll % (c2 - c1)
	}

	if n < 1 {
		n = 1
	}

	return n
}

// bindClientSkill attaches the client script of the skill a monster is starting.
func (d *Director) bindClientSkill(u *unit) {
	u.m.ClearClientSkill()

	if u.skill == nil || u.skill.rec == nil {
		return
	}

	rec := u.skill.rec

	switch rec.Cltdofunc {
	case d2monster.CltDoDiabRun:
		u.m.SetClientSkill(rec.Cltdofunc, skillParams(rec), 0)
	case d2monster.CltDoMosquito:
		c1, c2 := 1, 1
		if !rec.Calc1.Empty() {
			c1 = rec.Calc1.Eval(zeroCalcEnv{})
		}

		if !rec.Calc2.Empty() {
			c2 = rec.Calc2.Eval(zeroCalcEnv{})
		}

		u.m.SetClientSkill(rec.Cltdofunc, skillParams(rec), mosquitoBites(c1, c2, d.snd.Intn(1<<16)))
	}
}

// bindDeathClientSkill binds the Queen death sequence (cltdofunc 64) when one of
// the monster's skills carries it.
func (d *Director) bindDeathClientSkill(u *unit) {
	if u.b.Profile == nil {
		return
	}

	for slot := 0; slot < d2monster.NumSkills; slot++ {
		if !u.b.Profile.Skills[slot].Used() {
			continue
		}

		if sk := d.resolveSkill(u.b.Profile, slot); sk != nil && sk.rec != nil && sk.rec.Cltdofunc == d2monster.CltDoQueenDeath {
			u.m.SetClientSkill(sk.rec.Cltdofunc, skillParams(sk.rec), 0)

			return
		}
	}
}

// stompShake starts the screen shake of a Siege Beast stomp at the action frame.
func (d *Director) stompShake(u *unit) {
	if d.opt.OnShake == nil || u.skill == nil || u.skill.rec == nil || u.skill.rec.Cltdofunc != d2monster.CltDoStomp {
		return
	}

	r := u.skill.rec
	d.opt.OnShake(d2monster.StompShake(r.Param1, r.Param2, r.Param3, r.Param4))
}

// targetWithin says whether the unit's current target is within dist subtiles
// (table distance, edge to edge).
func (d *Director) targetWithin(u *unit, dist int) bool {
	tx, ty, ok := d.targetPos(u, u.attackTarget)
	if !ok {
		return false
	}

	sx, sy := u.m.SubtilePos()

	return d2monster.EdgeDistance(sx-tx, sy-ty, u.b.Size) <= dist
}
