package d2skills

import (
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Player versus player for skills. The caster's engine finds the heroes that
// are hostile to it (Engine.Rivals: declared hostile, not in its party, not in
// town) and treats them as one more kind of missile / area target. A hit is
// rolled like a hit on a monster (to-hit against the rival's defense for
// missiles), then every damage component is scaled by the PvP percent
// (d2combat.PvPPercent, the pre-step of COMBAT_ApplyResistsToDamageStruct) and
// handed to Engine.OnPvPHit, which the game sends to the defender's client:
// that client applies its own resists (d2combat.PvPReceiveParts).
//
// UNVERIFIED simplifications: poison and burn over time, stun/freeze and
// curse states do not cross the network (the caster's local state only); the
// defender applies its resists after the scale (order not verified).

// PvPHit is one skill hit on a rival, already scaled.
type PvPHit struct {
	Source, Target *d2mapentity.Player
	Skill          string
	Raw            int // whole points before the PvP scale and the defender's resists
	Parts          d2combat.PvPParts
}

// playerTarget adapts a rival hero to d2missile.Target.
type playerTarget struct {
	e *Engine
	p *d2mapentity.Player
}

func (e *Engine) rivalTarget(p *d2mapentity.Player) *playerTarget {
	t := e.rivalTargets[p.ID()]
	if t == nil || t.p != p {
		t = &playerTarget{e: e, p: p}
		if e.rivalTargets == nil {
			e.rivalTargets = map[string]*playerTarget{}
		}

		e.rivalTargets[p.ID()] = t
	}

	return t
}

func (t *playerTarget) ID() string     { return t.p.ID() }
func (t *playerTarget) IsPlayer() bool { return true }
func (t *playerTarget) Alive() bool    { return !t.p.IsDead() && t.p.Stats.Health > 0 }
func (t *playerTarget) Level() int     { return t.p.Stats.Level }

func (t *playerTarget) Defense(bool) int {
	if tot := t.p.Stats.Totals; tot != nil {
		return tot.Defense
	}

	return d2combat.Defense(0, t.p.Stats.Dexterity, 0)
}

// SubPos implements d2missile.Positioned.
func (t *playerTarget) SubPos() (float64, float64) {
	return float64(int(t.p.Position.X())) + 0.5, float64(int(t.p.Position.Y())) + 0.5
}

// isRival says whether the id belongs to a hero the local caster may hurt.
func (e *Engine) isRival(id string) bool {
	for _, p := range e.rivals() {
		if p.ID() == id {
			return true
		}
	}

	return false
}

// rivals lists the heroes hostile to the caster, sorted by id.
func (e *Engine) rivals() []*d2mapentity.Player {
	if e.Rivals == nil {
		return nil
	}

	out := e.Rivals()
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })

	return out
}

// rivalsNear lists the living rivals within a Chebyshev radius of a subtile.
func (e *Engine) rivalsNear(x, y, r int) []*d2mapentity.Player {
	var out []*d2mapentity.Player

	for _, p := range e.rivals() {
		if p.IsDead() || p.Stats.Health <= 0 {
			continue
		}

		if chebyshev(int(p.Position.X())-x, int(p.Position.Y())-y) <= r {
			out = append(out, p)
		}
	}

	return out
}

// hurtPlayer scales a rolled damage struct for PvP and sends it to the
// defender through OnPvPHit.
func (e *Engine) hurtPlayer(dst, src *d2mapentity.Player, d *d2combat.Damage, what string) {
	if src == nil || dst == nil {
		return
	}

	raw := d2combat.PvPParts{
		Phys: whole(d.Physical), Fire: whole(d.Fire), Light: whole(d.Lightning), Magic: whole(d.Magic), Cold: whole(d.Cold),
	}

	// The scale runs on the 8.8 values, with the fraction of a point carried to the
	// next hit on the same defender: a burning ground tick (4.88 fire) is 0.83 life
	// at 17 percent, which whole points would round to nothing (d2combat.PvPCarry).
	if e.pvpCarry == nil {
		e.pvpCarry = map[string]*d2combat.PvPCarry{}
	}

	carry := e.pvpCarry[dst.ID()]
	if carry == nil {
		carry = &d2combat.PvPCarry{}
		e.pvpCarry[dst.ID()] = carry
	}

	scaled := carry.Scale([5]int32{d.Physical, d.Fire, d.Lightning, d.Magic, d.Cold}, d2combat.PvPPercent())

	e.Counters.PvPHits++
	e.Counters.Damage += scaled.Total()
	// exact= is the unrounded 8.8 damage per type (raw= is rounded to whole points per type): scenarios check the
	// charged total against d2combat.PvPScaledBounds with it
	e.emit("damage", "PVPSKILL skill=%q target=%s raw=%d scaled=%d pct=%d parts=%v exact=%v", what, dst.Name(), raw.Total(),
		scaled.Total(), d2combat.PvPPercent(), scaled.Slice(), [5]int32{d.Physical, d.Fire, d.Lightning, d.Magic, d.Cold})

	// a tick that is still under one point is only carried, nothing to send
	if scaled.Total() == 0 {
		return
	}

	if e.OnPvPHit != nil {
		e.OnPvPHit(PvPHit{Source: src, Target: dst, Skill: what, Raw: raw.Total(), Parts: scaled})
	}
}

// whole converts an 8.8 damage value to whole points.
func whole(v int32) int {
	if v <= 0 {
		return 0
	}

	return (int(v) + 128) >> 8
}

// hitRivals damages the rivals within radius of a point with a rolled
// descriptor-style damage.
func (e *Engine) hitRivals(src *d2mapentity.Player, x, y, radius int, roll func() d2combat.Damage, what string) int {
	n := 0

	for _, p := range e.rivalsNear(x, y, radius) {
		d := roll()
		n++

		e.hurtPlayer(p, src, &d, what)
	}

	return n
}
