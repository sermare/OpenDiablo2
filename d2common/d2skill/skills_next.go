package d2skill

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
)

// Skills batch "next": Psychic Hammer (33), Cloak of Shadows (47), Mind Blast
// (51), Charge (67), Corpse Explosion (55), Smite (150) and the Dragon Talon
// (42) knockback. Each handler cites the exe function it follows; VERIFIED
// marks what was read in the decompilation (Game.exe 1.14b, read-only),
// U what is inferred. Notes with the addresses: skills-batch-next.md in the
// d2-re-notes folder.

// ReasonNoShield is the refusal of Smite without a shield in a hand.
const ReasonNoShield = "no_shield"

// KnockClass is the class of a knockback victim; the skills pick one of
// calc1..calc4 by it.
type KnockClass int

// The classes in the order the exe tests them (0x5d17f0, 0x5d4370).
const (
	KnockNormal KnockClass = iota // a plain monster
	KnockSmall                    // monster data flag 16 bit 8 (U meaning)
	KnockBoss                     // MONSTER_IsBossMonster
	KnockPlayer                   // a player or a hireling
)

// KnockClassed is optionally implemented by a Target to say which knockback
// class it is; a Target that is a player is KnockPlayer, any other KnockNormal.
type KnockClassed interface{ KnockClass() KnockClass }

// ShieldDamager is optionally implemented by a Unit: the damage range of the
// shield in a hand (items.txt mindam / maxdam of the type 33 item), ok false
// without a shield. Smite deals that range.
type ShieldDamager interface {
	ShieldDamage() (min, max int, ok bool)
}

// KnockDistance is how far a knockback pushes a monster, in path steps. VERIFIED
// (0x5a51e0, the monster mode 0xd handler reached through MONAI_ExecuteAiCommand
// from COMBAT_ServerHandleUnitHit 0x57ae50): it switches the path to type 8 and
// sets the step counter (path byte 0x90) to 10, or to 5 for monster base class
// 78; the path is then computed away from the attacker. U: one step is one
// subtile (the path-type table 0x6ecc40 was not decoded).
const KnockDistance = 10

// KnockDistanceSmall is the step count for monster base class 78 (VERIFIED the
// branch at 0x5a5264; which monster that class is, was not looked up).
const KnockDistanceSmall = 5

// KnockDistanceFor picks the knockback step count by the monster's base class
// id (monstats baseid), 78 getting the short one.
func KnockDistanceFor(baseClass int) int {
	if baseClass == 78 {
		return KnockDistanceSmall
	}

	return KnockDistance
}

// StateSkiller is optionally implemented by a Unit: the skill id and level
// that put a timed state on it (the stat list's skill fields, STATS_GetStatListSkill
// and ...SkillLevel), ok false when the state is not active.
type StateSkiller interface {
	StateSkill(state string) (skillID, level int, ok bool)
}

// holyShieldState is the state name of Holy Shield (state 0x65 in the exe).
const holyShieldState = "holyshield"

// OverlayBash is the overlay.txt name of overlay 147 (BABash), the mark Charge
// shows on its target (UNIT_AddOverlayEffectStat(target, 0x93) at 0x5cde00,
// VERIFIED).
const OverlayBash = "bash"

// Diminishing is SKILL_CalcDiminishingReturn (0x646ed0, VERIFIED): the
// skills.txt dm56 / dm34 style curve, lo + (lvl*110/(lvl+6)) * (hi-lo) / 100,
// capped at hi.
func Diminishing(lvl, lo, hi int) int {
	v := (lvl*110/(lvl+6))*(hi-lo)/100 + lo
	if hi < v {
		v = hi
	}

	return v
}

func classOf(t d2missile.Target) KnockClass {
	if t.IsPlayer() {
		return KnockPlayer
	}

	if k, ok := t.(KnockClassed); ok {
		return k.KnockClass()
	}

	return KnockNormal
}

// knockRoll decides a knockback: the chance (percent) is calc4 for a player,
// calc3 for a boss, calc2 for the small class and normal otherwise (VERIFIED
// 0x5d17f0 and 0x5d4370: Psychic Hammer passes calc1 as the normal chance,
// Dragon Talon 100); a chance below 1 skips the roll, otherwise the caster's
// generator rolls 0..99 and a result below the chance knocks the target back.
func (c *cast) knockRoll(t d2missile.Target, normal int) bool {
	chance := normal

	switch classOf(t) {
	case KnockPlayer:
		chance = c.calc(4)
	case KnockBoss:
		chance = c.calc(3)
	case KnockSmall:
		chance = c.calc(2)
	}

	return chance > 0 && c.rollN(100) < chance
}

func (c *cast) knock(t d2missile.Target) {
	c.effect(Effect{Kind: "knockback", Target: t, Dist: KnockDistance})
}

// doPsychicHammerFn is SRVDO_033_PsychicHammer (0x5d17f0, VERIFIED): the
// skill's physical and elemental damage hits the one target unit (not an
// area); then the knockback chance picked by the target's class decides
// whether the hit carries the knockback bit. U: without a target unit the
// nearest enemy within 3 subtiles of the aim is hit (the exe has no such
// fallback; the engine often aims at the ground).
func doPsychicHammerFn(c *cast) {
	t := c.tgt.Unit

	if t == nil || !t.Alive() {
		ax, ay := c.aim()

		for _, f := range c.p.foes(ax, ay, 3) {
			if t == nil || !t.Alive() || cheb(f.X-ax, f.Y-ay) < cheb(c.tgt.UX-ax, c.tgt.UY-ay) {
				t, c.tgt.UX, c.tgt.UY = f.Target, f.X, f.Y
			}
		}

		if t == nil || !t.Alive() {
			c.fail(ReasonTarget)
			return
		}
	}

	c.effect(Effect{Kind: "hit_unit", Target: t, Desc: c.desc()})

	if c.knockRoll(t, c.calc(1)) {
		c.knock(t)
	}
}

// doMindBlastFn is SRVDO_051_MindBlast (0x5d6120 with the per-unit callbacks
// 0x5d60c0 / 0x5d5e50, VERIFIED): every unit within the radius (the skill's
// prgcalc1 or state value, else aurarangecalc) of the aim point is either
// converted or damaged. A plain monster (no player, no hireling, alignment 0)
// is converted when a 0..99 roll is at most Diminishing(level, Param5, Param6)
// percent: the conversion state lasts Param3 + roll(Param4) frames and nothing
// else happens to it; every other unit takes the skill's damage (EType stun
// adds its stun length). The engine's conversion scaling (monsters above the
// caster's level) is the one of Conversion.
func doMindBlastFn(c *cast) {
	ax, ay := c.aim()
	radius := c.env.eval(c.sk.AuraRangeCalc)
	chance := Diminishing(c.lvl, c.sk.Params[5], c.sk.Params[6])
	d := c.desc()

	for _, f := range c.p.foes(ax, ay, radius) {
		convertible := !f.Target.IsPlayer()
		if cv, has := f.Target.(Convertible); has {
			convertible = convertible && cv.CanConvert()
		}

		if convertible && c.rollN(100) <= chance {
			frames := c.sk.Params[3] + c.rollN(int32(c.sk.Params[4]))
			c.effect(Effect{Kind: "convert", State: "conversion", Frames: frames, Target: f.Target, CasterLevel: c.u.Level()})

			continue
		}

		c.effect(Effect{Kind: "hit_unit", Target: f.Target, Desc: d})
	}
}

// doCloakFn is SRVDO_047_CloakOfShadows (0x5d4fe0 with the per-unit callback
// 0x5d4ec0, VERIFIED): refused when the caster already holds the state. The
// caster gets aurastate for auralencalc frames with the stats of passivestat2..5
// (passivecalc2..5, zero values left out); every unit within aurarangecalc
// gets auratargetstate with the aurastat1..6 stats, and a monster that is a
// valid skill target is forced into AI state 10 (blind, the same as Dim
// Vision). The existing "ends when the cloaked hero is hit" rule of the engine
// stays.
func doCloakFn(c *cast) {
	sk := c.sk
	if sk.AuraState == "" {
		c.fail(ReasonNoSkill)
		return
	}

	frames := c.env.eval(sk.AuraLenCalc)

	var self []StatMod

	for i := 2; i <= 5; i++ {
		if sk.PassiveStat[i] == "" {
			continue
		}

		if v := c.env.eval(sk.PassiveCalc[i]); v != 0 {
			self = append(self, StatMod{Stat: sk.PassiveStat[i], Value: v})
		}
	}

	c.effect(Effect{Kind: "self_state", State: sk.AuraState, Frames: frames, Stats: self})

	if sk.AuraTargetState == "" {
		return
	}

	stats := append(statsOf(c.env, sk), StatMod{d2state.StatBlind, 1})
	c.effect(Effect{Kind: "area_state", Origin: "self", Radius: c.env.eval(sk.AuraRangeCalc), Filter: sk.AuraFilter,
		State: sk.AuraTargetState, Frames: frames, Stats: stats})
}

// doSmiteFn is SRVDO_150_Smite (0x5ccdd0, VERIFIED): needs a target unit in
// melee range. A player rolls the damage range of the shield in a hand
// (items.txt mindam / maxdam; no shield: the cast does nothing) raised by
// calc1 percent plus the damage percent stat; the hit always lands, and
// calc2 is the stun length in frames (the damage struct's stun field). The
// Holy Shield state adds its skill's damage range to the shield's (not
// modelled). A Unit without ShieldDamager keeps the older weapon-based stand-in.
func doSmiteFn(c *cast) {
	if !c.needTarget() {
		return
	}

	sd, has := c.u.(ShieldDamager)
	if !has {
		o := c.meleeOpt()
		o.pct = c.calc(1)
		m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)

		if m.Hit {
			m.Damage.StunLen = int32(c.calc(2))
		}

		c.addMelee(m)

		return
	}

	lo, hi, ok := sd.ShieldDamage()
	if !ok {
		c.fail(ReasonNoShield)
		return
	}

	lo8, hi8 := int32(lo)<<8, int32(hi)<<8

	// VERIFIED (0x5ccdd0): with the Holy Shield state on, the damage range of
	// the skill that made the state (at its level) is added to the shield's.
	if ss, has := c.u.(StateSkiller); has {
		if id, lvl, on := ss.StateSkill(holyShieldState); on && c.p.Skills != nil {
			if hs := c.p.Skills.ByID(id); hs != nil {
				he := c.env.sub(hs, lvl)
				lo8 += hs.PhysMin(he, lvl, 0, false)
				hi8 += hs.PhysMax(he, lvl, 0, false)
			}
		}
	}

	ph := rollRange(c.u.Roller(), lo8, hi8)
	ph += int32(mulDiv(int(ph), c.calc(1)+c.u.Stat("damagepercent"), 100))
	dmg := d2combat.Damage{Result: d2combat.ResultHit, HitClass: int32(c.sk.HitClass), Physical: ph, StunLen: int32(c.calc(2))}
	c.addMelee(&MeleeResult{Target: c.tgt.Unit, Hit: true, Chance: 100, Damage: dmg, Total: dmg.SumTotal(false)})
}

// doChargeFn is SRVDO_067_Charge (0x5cde00, VERIFIED flow): the paladin rushes
// at the target unit of the skill, or, with none, the nearest unit within 3
// subtiles of himself; once in melee range he strikes it with calc1 percent
// damage and the skill's elemental part, and the overlay stat 0x93 is shown
// on the target. The rush ends next to the target (not halfway: the
// animation loop of the exe repeats until the target is within melee range),
// which is what the "move" effect now says.
func doChargeFn(c *cast) {
	hx, hy := c.u.Pos()
	t, tx, ty := c.tgt.Unit, c.tgt.UX, c.tgt.UY

	if t == nil || !t.Alive() {
		best := -1

		for _, f := range c.p.foes(hx, hy, 3) {
			if d := cheb(f.X-hx, f.Y-hy); best < 0 || d < best {
				t, tx, ty, best = f.Target, f.X, f.Y, d
			}
		}

		if t == nil {
			c.fail(ReasonTarget)
			return
		}
	}

	c.effect(Effect{Kind: "move", Mode: "charge", X: tx - sgn(tx-hx), Y: ty - sgn(ty-hy)})

	o := c.meleeOpt()
	o.pct = c.calc(1)
	m := c.p.strike(c.u, c.sk, c.lvl, t, c.env, o)
	c.addMelee(m)

	// VERIFIED: the strike is followed by overlay 147 on the target (only when
	// the rush ended in melee range, i.e. the blow was made).
	c.effect(Effect{Kind: "overlay", Overlay: OverlayBash, Target: t, X: tx, Y: ty})
}

func sgn(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}

	return 0
}

// splitCorpseDamage divides a Corpse Explosion total (whole points) into the
// fire part and the physical rest: calc3 percent of it is the skill's element
// (EType), VERIFIED the branch exists (0x5c2c60: calc3 above 0 and a skill
// element), U that the rest stays physical.
func splitCorpseDamage(total int32, pct int, etype string) (phys int32, elem d2missile.Elem) {
	if pct <= 0 || etype == "" {
		return total, d2missile.Elem{}
	}

	if pct > 100 {
		pct = 100
	}

	e := int32(mulDiv(int(total), pct, 100))

	return total - e, d2missile.Elem{Min: e, Max: e}
}

// doTauntFn is SRVDO_071_Taunt (0x5d6fe0, VERIFIED): the curse goes on ONE
// unit: the skill's target when it is valid and hostile, else the nearest
// valid unit within 20 subtiles of the caster (SKILL_FindTauntTarget); no
// unit, no cast. The target gets auratargetstate for auralencalc frames with
// the aurastat stats, then is sent after the caster (its dynamic path target
// is the caster), which the engine does through the taunted marker stat the
// curse carries.
func doTauntFn(c *cast) {
	if c.sk.AuraTargetState == "" {
		c.fail(ReasonNoSkill)
		return
	}

	t := c.tgt.Unit
	if t == nil || !t.Alive() || t.IsPlayer() {
		hx, hy := c.u.Pos()
		t = nil
		best := -1

		for _, f := range c.p.foes(hx, hy, tauntSearchRadius) {
			if d := cheb(f.X-hx, f.Y-hy); best < 0 || d < best {
				t, best = f.Target, d
			}
		}

		if t == nil {
			c.fail(ReasonTarget)
			return
		}
	}

	c.effect(Effect{Kind: "unit_state", Target: t, State: c.sk.AuraTargetState, Frames: c.env.eval(c.sk.AuraLenCalc),
		Stats: append(statsOf(c.env, c.sk), StatMod{d2state.StatTaunted, 1})})
}

// tauntSearchRadius is the radius of SKILL_FindTauntTarget's scan (0x14).
const tauntSearchRadius = 20
