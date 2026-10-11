package d2skill

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// Batch 3 of the exe re-reads (docs: d2-re-notes/skills-batch3.md). Every
// function below replaces a table entry of class.go and re-implements the rule
// of the original handler; the addresses are Game.exe 1.14b. VERIFIED marks a
// rule read in the decompilation, U an inferred one.
//
//	Feral Rage, Maul   SRVDO_120 0x5c57c0
//	Leap               SRVDO_077 0x5d8e60 (+ 0x5d8a70, 0x5d8c20)
//	Leap Attack        SRVDO_078 0x5d9320 (+ 0x5d8f90 target, 0x5d9170 hit)
//	Dragon Claw        SRVDO_046 0x5d4cf0 (+ 0x5d4ba0)
//	Dragon Tail        SRVDO_050 0x5d5b90
//	Dragon Flight      SRVDO_052 0x5d6290
//	Wearwolf, Wearbear SRVDO_116 0x5c4e80
//	Thunder Storm      SRVDO_029 0x5c8650
func init() {
	doTable[29] = doThunderStormFn
	doTable[46] = doDragonClawFn
	doTable[50] = doDragonTailFn
	doTable[52] = doDragonFlightExeFn
	doTable[77] = doLeapExeFn
	doTable[78] = doLeapAttackExeFn
	doTable[116] = doFormFn
	doTable[120] = doStackBuffFn
}

// StateHolder is implemented by a Unit that can say it is shapeshifted. The
// affordability test of the exe (0x569ee0) treats srvdofunc 116 (Wearwolf,
// Wearbear, Delirium change) as free while the unit already is in a form
// (VERIFIED, the test calls 0x63b510); skills.txt names the same rule
// nocostinstate.
type StateHolder interface {
	Shapeshifted() bool
}

// costOf is the 8.8 mana cost of a cast: the table cost, or 0 for a shape
// shift form skill cast from inside a form (the cast then turns the form off).
func costOf(u Unit, sk *Skill, lvl int) int {
	if sk.SrvDoFunc == 116 {
		if h, ok := u.(StateHolder); ok && h.Shapeshifted() {
			return 0
		}
	}

	return sk.ManaCost(lvl)
}

// ---- Wearwolf, Wearbear ----

// doFormFn is SRVDO_116 (0x5c4e80, VERIFIED): the skill needs a valid
// aurastate. It first ends every state of the aurastate's States.txt group,
// the state itself included (0x56a480 with the include-self flag). If that
// ended something the form is NOT applied again: the cast only starts the
// skill's cooldown, so casting the same form again (or the other one) takes
// the shape back. Otherwise the state lasts auralencalc frames with the
// aurastat columns, carries the casting skill and level (stats 0x15e / 0x15f).
// The engine does the group clear and the toggle (Effect.Toggle).
func doFormFn(c *cast) {
	c.p.doState(c.sk, c.env, c.res, "self_state")
	e := &c.res.Effects[len(c.res.Effects)-1]
	e.Level, e.SkillID, e.SkillName = c.lvl, c.sk.ID, c.sk.Name
	e.Toggle = true
}

// ---- Feral Rage, Maul ----

// doStackBuffFn is SRVDO_120 (0x5c57c0, VERIFIED). The skill must be the unit's
// current skill and have a target. The melee blow resolves as usual (calc1
// percent here); then, hit or miss, the aurastate is put on the caster for
// auralencalc frames and its stack grows: count = min(calc2, count + 1), with
// calc2 the stack limit (lvl/par7 + par8). The aurastat columns are evaluated
// with the STACK COUNT in place of the skill level (the exe hands the count to
// SKILL_ApplyAuraStatsToStatList as the level), so Feral Rage's life steal
// (par2 * lvl) and Maul's damage (lvl * par3) grow per hit; the real skill level
// goes to stat 0x15f. Unlike the form skills it does NOT end the other states
// of its group (wolf and bear are in the same group as feralrage and maul).
func doStackBuffFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))

	limit := c.calc(2)
	n := c.u.Stat(c.sk.AuraState) + 1

	if n > limit {
		n = limit
	}

	if n < 1 {
		n = 1
	}

	stats := statsOf(c.p.env(c.sk, n, c.u), c.sk)
	stats = append(stats, StatMod{Stat: c.sk.AuraState, Value: n})

	frames := c.env.eval(c.sk.AuraLenCalc)
	if frames < 1 {
		frames = 1
	}

	c.effect(Effect{Kind: "self_state", State: c.sk.AuraState, Frames: frames, Stats: stats, Stack: maxInt(limit, 1), NoGroup: true})
}

// ---- Leap, Leap Attack ----

// leapMeleeRange is the reach in which the skill's own target counts as in
// melee range after the landing (COMBAT_IsTargetWithinMeleeRange; subtiles, U).
const leapMeleeRange = 3

// leapScanRadius is the radius of the search for another victim, "melee range
// plus 4" in the exe (0x5d8f90; the plus-4 helper's value is not read, U).
const leapScanRadius = 7

// doLeapExeFn is SRVDO_077 (0x5d8e60, VERIFIED shape): the jump itself is a
// path of type 9 at the run speed (0x5d8a70) that ends on the aim point (0x5d8c20).
// When a player lands, the area function is run at the unit with an empty
// damage struct whose result flags are 9 (hit + knockback bit 8) and the
// radius calc1: no damage, but everything hostile within the radius is thrown
// back. Monsters that leap (unit type 1) skip the landing hit. U: the area is
// centred on the landing point (the x / y arguments are 0 in the exe, which
// the area helper takes as the unit's own position); the radius is in
// subtiles.
func doLeapExeFn(c *cast) {
	x, y, ok := c.clampMove(leapDistance)
	if !ok {
		c.fail(ReasonLOS)
		return
	}

	c.effect(Effect{Kind: "move", Mode: "leap", X: x, Y: y})

	if r := c.calc(1); r > 0 && c.u.IsPlayer() {
		c.effect(Effect{Kind: "knock_area", X: x, Y: y, Radius: r, Dist: KnockDistance})
	}
}

// doLeapAttackExeFn is SRVDO_078 (0x5d9320, VERIFIED). The jump is the same.
// On landing ONE victim is chosen (0x5d8f90): the skill's target if it is in
// melee range, else the one remembered from the previous stage, else the
// nearest unit found by a scan (filter 0x20003). Nothing is hit when none is
// found. The blow (0x5d9170) rolls to hit with the skill's bonus; a hit gets
// result flag 8 (knockback), the always-set damage flag 0x20, calc1 percent
// damage, the elemental part only when EType is set and calc4 > 0, SrcDam
// (128 when 0), then the skill's srvoverlay (bash) is shown on the victim and
// its stun state (state 21) is removed.
func doLeapAttackExeFn(c *cast) {
	x, y, ok := c.clampMove(leapDistance)
	if !ok {
		c.fail(ReasonLOS)
		return
	}

	c.effect(Effect{Kind: "move", Mode: "leap", X: x, Y: y})

	t, tx, ty := c.leapVictim(x, y)
	if t == nil {
		return
	}

	o := c.meleeOpt()
	o.pct = c.calc(1)
	o.skillElem = c.sk.EType != "" && c.calc(4) > 0
	m := c.p.strike(c.u, c.sk, c.lvl, t, c.env, o)
	c.addMelee(m)

	if !m.Hit {
		return
	}

	c.knock(t)
	c.effect(Effect{Kind: "overlay", Overlay: OverlayBash, Target: t, X: tx, Y: ty})
	c.effect(Effect{Kind: "unit_clear_state", State: "stunned", Target: t})
}

// leapVictim picks the unit a Leap Attack hits after landing at (x, y).
func (c *cast) leapVictim(x, y int) (d2missile.Target, int, int) {
	if t := c.tgt.Unit; t != nil && t.Alive() && cheb(c.tgt.UX-x, c.tgt.UY-y) <= leapMeleeRange {
		return t, c.tgt.UX, c.tgt.UY
	}

	var (
		best  d2missile.Target
		bx    int
		by    int
		bestD = -1
	)

	for _, f := range c.p.foes(x, y, leapScanRadius) {
		if d := cheb(f.X-x, f.Y-y); f.Target != nil && f.Target.Alive() && (bestD < 0 || d < bestD) {
			best, bx, by, bestD = f.Target, f.X, f.Y, d
		}
	}

	return best, bx, by
}

// ---- Dragon Claw, Dragon Tail, Dragon Flight ----

// progressiveToHit is stat 0x145 (325) "progressive_tohit": the charge-up
// states of the assassin add it to the attack rating of the finishing moves
// (SKILL_DragonClawBuildHit adds it to the skill's to-hit bonus before the
// roll, VERIFIED at 0x5d4ba0 and 0x5d6290).
const progressiveToHit = "progressive_tohit"

// doDragonClawFn is SRVDO_046 (0x5d4cf0, VERIFIED): the finisher fires once per
// action frame of the animation (two frames, the claws strike left and right);
// each blow rolls to hit with the skill bonus plus stat 0x145, takes calc1 as
// its damage bonus and the skill's SrcDam, and then the held charges are
// released (0x5d3ae0). This port does both blows in one cast.
func doDragonClawFn(c *cast) {
	if !c.needTarget() {
		return
	}

	for i := 0; i < 2; i++ {
		o := c.meleeOpt()
		o.pct = c.calc(1)
		o.toHitPct += c.u.Stat(progressiveToHit)
		o.skillElem = false
		c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
	}

	c.releaseCharges()
}

// dragonTailRadiusMin keeps the old fallback radius when the table gives none.
const dragonTailRadiusMin = 4

// DragonTailFire is the explosion of Dragon Tail (SRVDO_050 0x5d5b90,
// VERIFIED): the damage of the kick (its entry in the queued hit sequence,
// field +0xc of the copied entry) times (calc1 + stat 0x149) percent, with
// stat 0x149 = passive_fire_mastery. The product goes into the damage struct
// at +0x10 (U: the fire field; skills.txt gives the skill EType fire). All in
// 8.8 fixed point.
func DragonTailFire(kick int32, calc1, fireMastery int) int32 {
	if kick <= 0 {
		return 0
	}

	return int32(int64(kick) * int64(calc1+fireMastery) / 100)
}

// doDragonTailFn is SRVDO_050 (0x5d5b90, VERIFIED). A target must exist. The
// kick resolves as the ordinary kick of the hit sequence (calc1 is NOT a bonus
// on the kick), the held charges are released, and unless the caster is dying
// the area function runs around the TARGET with radius aurarangecalc (par3)
// and a damage struct of result flags 9 (hit + knockback): everything hostile
// in the circle, the kicked unit included, takes the fire explosion and is
// thrown back.
func doDragonTailFn(c *cast) {
	if !c.needTarget() {
		return
	}

	o := c.meleeOpt()
	o.skillElem = false
	m := c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o)
	c.addMelee(m)

	fire := DragonTailFire(m.Damage.Physical, c.calc(1), c.u.Stat("passive_fire_mastery"))
	if fire > 0 {
		c.effect(Effect{Kind: "area_hit", Origin: "aim", X: c.tgt.UX, Y: c.tgt.UY,
			Radius: maxInt(c.env.eval(c.sk.AuraRangeCalc), dragonTailRadiusMin), Knock: true,
			Desc: &d2missile.DamageDesc{Fire: d2missile.Elem{Min: fire, Max: fire}}})
	}

	c.releaseCharges()
}

// doDragonFlightExeFn is SRVDO_052 (0x5d6290, VERIFIED). The handler needs a
// target unit. Which half runs depends on the animation frame (bit 8 of the
// sequence frame): the first action frame teleports to the target's position
// when the level's levels.txt Teleport column is non-zero (2: refused when the
// line is blocked for mask 0x804), the second delivers the kick: a to-hit roll
// with the skill bonus plus stat 0x145, the finisher damage, SrcDam (128 when
// 0), and the charge release. A cast makes both halves in turn.
func doDragonFlightExeFn(c *cast) {
	if !c.needTarget() {
		return
	}

	flag := 1
	if c.p.TeleportFlag != nil {
		flag = c.p.TeleportFlag()
	}

	hx, hy := c.u.Pos()
	x, y := c.tgt.UX, c.tgt.UY

	if flag == 0 {
		c.fail(ReasonLOS)
		return
	}

	if flag == 2 && c.p.Grid != nil && !c.traceClear(hx, hy, x, y) {
		c.fail(ReasonLOS)
		return
	}

	// the exe moves the caster to the target's own position (VERIFIED, batch 4), no side offset
	if x != hx || y != hy {
		c.effect(Effect{Kind: "move", Mode: "teleport", X: x, Y: y})
	}

	o := c.meleeOpt()
	o.autoHit = false
	o.toHitPct += c.u.Stat(progressiveToHit)
	c.addMelee(c.p.strike(c.u, c.sk, c.lvl, c.tgt.Unit, c.env, o))
	c.releaseCharges()
}

// ---- Thunder Storm ----

// doThunderStormFn is SRVDO_029 (0x5c8650, VERIFIED shape). The first do puts
// the aurastate on the caster for auralencalc frames; later periodic calls
// scan the radius Param7 (SKILL_GetParam7, filter 3) for the next target AFTER
// the one struck last, make the srvmissilea at a strike position near it and
// process its hit on that target at once. The period is the skill's perdelay
// ((100 - dm56) * par4 / 100 + par3 frames in the table), not a fixed number.
// The engine rotates through the units in range (NextAfter, VERIFIED).
func doThunderStormFn(c *cast) {
	doStormFn(c)

	if n := len(c.res.Effects); n > 0 {
		e := &c.res.Effects[n-1]

		if r := c.sk.Params[7]; r > 0 {
			e.Radius = r
		}

		if c.sk.PerDelay != nil {
			e.Interval = maxInt(c.env.eval(c.sk.PerDelay), 1)
		}
	}
}

// traceClear tests the line from (x0, y0) to (x1, y1) for the collision mask
// 0x804 (walls and objects) that levels.txt Teleport = 2 forbids crossing.
func (c *cast) traceClear(x0, y0, x1, y1 int) bool {
	clear, _ := d2path.TraceLine(c.p.Grid, 0x804, d2path.Point{X: x0, Y: y0}, d2path.Point{X: x1, Y: y1})

	return clear
}
