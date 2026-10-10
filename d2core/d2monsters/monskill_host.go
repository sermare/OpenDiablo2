package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Monster Jump (srvdofunc 89) and Diablo Run (103). Both are classified as
// melee do-functions (d2monster.ClassifyEffect), so the hit frame resolves the
// skill's damage; this file moves the caster to its target first.
//
//   - Jump (SRVDO_089_Jump 0x5c9be0, VERIFIED shape): the caster flies over
//     obstacles to the stored destination and the strike resolves on landing.
//     Here the unit lands next to its target at the hit frame (instant, no
//     flight arc: UNVERIFIED simplification).
//   - DiabRun (SRVDO_103_DiabRun 0x5cb680, VERIFIED shape): the caster runs at
//     MulDiv(calc1, stat 0x43, 100) and strikes once per action frame while
//     its target is in melee range. Here every hit frame first closes the gap
//     by up to diabRunStride subtiles, then the strike resolves if in reach
//     (UNVERIFIED stride).

const diabRunStride = 8

// closeInDest is the cell a caster that advances adv subtiles toward a target
// stands on (pure): the point on the line from (sx,sy) to (tx,ty), measured in
// the AI distance metric.
func closeInDest(sx, sy, tx, ty, adv int) (int, int) {
	dist := d2monster.Distance(tx-sx, ty-sy)
	if dist <= 0 || adv <= 0 {
		return sx, sy
	}

	if adv > dist {
		adv = dist
	}

	return sx + (tx-sx)*adv/dist, sy + (ty-sy)*adv/dist
}

// moveSkillAdvance is how many subtiles the cast closes at a hit frame.
func moveSkillAdvance(srvdofunc, dist, reach int) int {
	switch srvdofunc {
	case d2monster.DoJump:
		if dist > reach {
			return dist - reach + 1
		}
	case d2monster.DoDiabRun:
		return d2monster.RunAdvance(dist, reach, diabRunStride)
	}

	return 0
}

// closeIn moves a Jump or Diablo Run caster toward its target before the
// strike. It does nothing for other skills.
func (d *Director) closeIn(u *unit) {
	if u.skill == nil || u.skill.rec == nil || d.fp == nil {
		return
	}

	do := u.skill.rec.Srvdofunc
	if do != d2monster.DoJump && do != d2monster.DoDiabRun {
		return
	}

	tx, ty, ok := d.targetPos(u, u.attackTarget)
	if !ok {
		return
	}

	sx, sy := u.m.SubtilePos()
	dist := d2monster.EdgeDistance(sx-tx, sy-ty, u.b.Size)

	adv := moveSkillAdvance(do, dist, attackReach(false)+heroReach/2)
	if adv <= 0 {
		return
	}

	x, y := closeInDest(sx, sy, tx, ty, adv)

	p, ok := d2path.NearestFree(d.fp, d2path.MaskMonster, d2path.Point{X: x, Y: y}, 2)
	if !ok {
		return
	}

	u.m.TeleportTo(p.X, p.Y)
	u.b.X, u.b.Y = p.X, p.Y
	d.footprint(u)
	d.emit("attack", "MONSTER %s %s closes to (%d,%d)", u.m.Label(), u.skill.name, p.X, p.Y)
}
