package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Skill fire callback for sentries, vines and other stationary summons.
//
// thinkSentry / thinkStationary (d2common/d2monster/ai_pet.go) end in
// Actor.Cast. For a hostile monster Director.Cast plays the monstats mode and
// the strike is a monstats melee hit; for a hero's sentry that would lose the
// hero-level skill damage and the missile the skill engine fires today
// (d2skills registerTrap -> Pipeline.CastTrap). A unit armed with ArmTrap
// instead hands its Cast to the SkillFirer, which the skill engine implements
// with the same CastTrap call.
//
// STATUS (UNVERIFIED): the callback and the Director side are in place and
// tested with a fake, but the skill engine still drives sentries itself
// (trapsTick). Switching was NOT done: the AI path picks targets with
// monstats aggro/InRange (melee 7 or ranged line of sight) and paces with
// aip1/aip2, while trapsTick uses a fixed 20 frame period and a 20 subtile
// Chebyshev range, so firing cadence and target set would change. The fire
// itself (missile, level, damage, start position) is shared code in d2skills
// and covered by a parity test there.

// TrapShot is one request to fire a trap's skill.
type TrapShot struct {
	// Trap is the sentry or vine firing.
	Trap *d2mapentity.Monster
	// Owner is the hero that placed it.
	Owner *d2mapentity.Player
	// Spec is what ArmTrap registered.
	Spec TrapSpec
	// FromX, FromY is the trap's subtile.
	FromX, FromY int
	// Target is the monster aimed at.
	Target *d2mapentity.Monster
}

// TrapSpec names the skill a trap fires; the skill engine resolves it.
type TrapSpec struct {
	SkillID   int
	SkillName string
	Missile   string
}

// SkillFirer is implemented by the skill engine. It returns whether a missile
// was fired.
type SkillFirer interface {
	FireTrap(TrapShot) bool
}

// SetSkillFirer injects the skill engine side. nil disables the callback.
func (d *Director) SetSkillFirer(f SkillFirer) { d.firer = f }

// ArmTrap makes m (a unit made by SpawnMinion) fire spec through the
// SkillFirer whenever its AI casts. It reports false if m is not a minion unit.
func (d *Director) ArmTrap(m *d2mapentity.Monster, spec TrapSpec) bool {
	u := d.byEntity[m.ID()]
	if u == nil || u.ally == nil {
		return false
	}

	u.ally.trap = &spec

	return true
}

// fireTrap routes a Cast of an armed unit to the SkillFirer. handled is false
// when the unit is not armed (or no firer is set): the caller does its normal
// monstats cast.
func (d *Director) fireTrap(u *unit, t d2monster.Target) (handled, ok bool) {
	if d.firer == nil || u == nil || u.ally == nil || u.ally.trap == nil {
		return false, false
	}

	tu := d.units[t.ID]
	if tu == nil || !tu.m.Alive() {
		return true, false
	}

	x, y := u.m.SubtilePos()

	return true, d.firer.FireTrap(TrapShot{Trap: u.m, Owner: u.ally.owner, Spec: *u.ally.trap, FromX: x, FromY: y, Target: tu.m})
}

// trapTick runs the ported AI of an armed trap; false means it is not armed.
func (d *Director) trapTick(u *unit) bool {
	if d.firer == nil || u.ally.trap == nil || u.b.Def == nil || !u.b.Def.Implemented {
		return false
	}

	d2monster.Tick(d, u.b)

	return true
}
