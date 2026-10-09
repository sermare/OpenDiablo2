package d2monster

func init() {
	register("PantherJavelin", TargetStandard, thinkPantherJavelin)
	register("Andariel", TargetStandard, thinkAndariel)
	register("Smith", TargetStandard, thinkSmith)
	register("Griswold", TargetStandard, thinkGriswold)
	register("BloodRaven", TargetStandard, thinkBloodRaven)
}

// thinkPantherJavelin is MONAI_Think_PantherJavelin 0x5dfed0 (VERIFIED).
// aip1 approach%, aip2 throw%, aip3 group distance, aip4 walk-away%, aip5
// stall, aip6 throw distance. When the attack-target scan finds nothing the
// tick's distance stands in for it (UNVERIFIED).
func thinkPantherJavelin(c *Ctx) {
	b, pt := c.B, *c.Target

	at, d, ok := c.W.AttackTarget(b)
	if !ok {
		d = c.Dist
	}

	if d < 8 && b.Chance(b.AIP(4)) && c.WalkAway(pt, 16) {
		return
	}

	if d > b.AIP(6)-6 && b.Chance(b.AIP(1)) && c.WalkNearTarget(&pt, 4) {
		return
	}

	if ok && d < b.AIP(6) {
		if b.Chance(b.AIP(2)) {
			c.Attack(ModeAttack1, at)
		} else {
			c.Sleep(b.AIP(5))
		}

		return
	}

	// Regroup with the nearest ally of the same class when it is farther than
	// aip3 (the notes compare squared distances).
	if f, isF := c.W.(AllyFinder); isF {
		if ally, ad, found := f.NearestAlly(b); found && ad > b.AIP(3) && c.WalkTo(ally, meleeReach) {
			return
		}
	}

	c.Sleep(b.AIP(5))
}

// Skill slots used by the Act 1 bosses (VERIFIED: Skill1/Skill2 offsets).
const (
	slot1 = 0
	slot2 = 1
)

// thinkAndariel is MONAI_Think_Andariel 0x5f4960 (VERIFIED, 432 bytes). aip1
// spray-in-melee%, aip2 stall%, aip3 fire-or-engage%, aip4 spray-from-
// distance%. Skill1 is AndrialSpray, Skill2 AndyPoisonBolt. No phases, no
// summoning, never flees.
func thinkAndariel(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	if c.InRange {
		if p.Skills[slot1].Used() && b.Chance(b.AIP(1)) {
			c.Cast(slot1, t)

			return
		}

		c.Attack(ModeAttack1, t)

		return
	}

	if b.Chance(b.AIP(2)) {
		c.Sleep(5)

		return
	}

	if b.Chance(b.AIP(3)) {
		if p.Skills[slot1].Used() && b.Roll(100) < b.AIP(4) {
			c.Cast(slot1, t)

			return
		}

		if p.Skills[slot2].Used() {
			c.Cast(slot2, t)

			return
		}
	}

	c.SetSpeed(0)

	if !c.WalkTo(t, meleeReach) {
		c.Sleep(10)
	}
}

// thinkSmith is MONAI_Think_Smith 0x5e2770 (VERIFIED): A1 in range; out of
// range the speed override is (100-clamp(hp%,0,100))>>1 (he hurries as he
// loses hit points; the override itself is UNVERIFIED) and he walks to melee.
func thinkSmith(c *Ctx) {
	b, t := c.B, *c.Target

	if c.InRange {
		c.Attack(ModeAttack1, t)

		return
	}

	c.SetSpeed((100 - clamp(b.HPPercent, 0, 100)) >> 1)

	if !c.WalkTo(t, meleeReach) {
		c.Sleep(10)
	}
}

// thinkGriswold is MONAI_Think_Griswold 0x5e4a70 (VERIFIED, reads no
// monstats parameter): out of range 50% walk else sleep 10; in range 80% A1
// else sleep 10.
func thinkGriswold(c *Ctx) {
	b, t := c.B, *c.Target

	if !c.InRange {
		if b.Roll(100) < 50 && c.WalkTo(t, meleeReach) {
			return
		}

		c.Sleep(10)

		return
	}

	if b.Roll(100) < 80 {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(10)
}

// Blood Raven scratch registers (AiGeneral +0x14, +0x18, +0x1c).
const (
	brCharge   = 0 // +0x14: shot probability, +3 per tick
	brShots    = 1 // +0x18: arrows fired
	brFleeHome = 2 // +0x1c: returning to the anchor
)

// Blood Raven constants (VERIFIED from the decompile).
const (
	brIgnoreDist   = 45 // targets farther than this are ignored (sleep 5)
	brLeash        = 49 // farther than this from the anchor: run home
	brHomeDist     = 5  // stop running home within this
	brCloseInDist  = 20 // target farther than this: close in
	brAnchorTarget = 50 // ... only if the target is within this of the anchor
)

// thinkBloodRaven is MONAI_Think_BloodRaven 0x5e5260 (VERIFIED flow, 1226
// bytes) with MONAI_Pre_BloodRaven 0x5e5250 (fleeing flag cleared, which a
// fresh Brain already has). She is an archer who leashes to her spawn point
// (a type 10 anchor command created on the first tick), fires Skill1 at a
// random spot near the player with a probability that grows 3 per tick, and
// uses Skill2 with (diff+4)*10 percent.
//
// UNVERIFIED details: the offset range of the random shot spot (5..10
// subtiles here), the "desired distance" of the close-in move (0), resetting
// the arrow counter whenever she moves (the notes show a cap of 8+2*diff
// without a reset, which would silence her for good), and the target of the
// melee label.
func thinkBloodRaven(c *Ctx) {
	b, t := c.B, *c.Target
	p := b.Profile

	anchor := b.FindCommand(CmdAnchor)
	if anchor == nil {
		b.AppendCommand(Command{Type: CmdAnchor, X: b.X, Y: b.Y})
		anchor = b.FindCommand(CmdAnchor)
	}

	distHome := b.DistanceTo(anchor.X, anchor.Y)

	if c.Dist > brIgnoreDist {
		c.Sleep(5)

		return
	}

	if distHome > brLeash || (b.Scratch[brFleeHome] != 0 && distHome > brHomeDist) {
		b.Scratch[brFleeHome] = 1
		b.Scratch[brShots] = 0

		c.SetSpeed(100)
		c.runHome(anchor)

		return
	}

	b.Scratch[brFleeHome] = 0

	targetFromHome := EdgeDistance(t.X-anchor.X, t.Y-anchor.Y, 0)
	if c.Dist > brCloseInDist && targetFromHome < brAnchorTarget {
		step := c.Dist / 2
		if step < 12 {
			step = 12
		}

		b.Scratch[brShots] = 0

		c.SetSpeed(100)

		if c.RunToRange(t, step, 0) {
			return
		}
	}

	b.Scratch[brCharge] += 3

	if p.Skills[slot1].Used() && !c.InRange && b.Scratch[brShots] < int(b.Diff)*2+8 &&
		b.Roll(100) < b.Scratch[brCharge] {
		spot := b.randomSpotNear(t)
		if c.Cast(slot1, spot) {
			b.Scratch[brShots]++
			b.Scratch[brCharge] = 0

			return
		}
	}

	if c.brMidRange(t, distHome) {
		return
	}

	if b.Roll(100) < 30 && c.Dist < 12 {
		c.SetSpeed(100)

		if c.RunAway(t, 12-c.Dist) {
			return
		}
	}

	c.Attack(ModeAttack1, t)
}

// runHome runs to the anchor point.
func (c *Ctx) runHome(a *Command) {
	if !c.move(Point{a.X, a.Y}, nil, 0, true) {
		c.Sleep(10)
	}
}

// brMidRange is the part of Blood Raven's tick for a target more than 5 away;
// it reports whether it acted.
func (c *Ctx) brMidRange(t Target, distHome int) bool {
	b := c.B

	if c.Dist <= 5 {
		return false
	}

	if b.Roll(100) < 5 && distHome < brAnchorTarget {
		c.SetSpeed(100)

		if c.RunToRange(t, 12, 0) {
			return true
		}
	}

	if at, _, ok := c.W.AttackTarget(b); ok && !b.Aggressive && b.Roll(100) < 80 {
		if b.Profile.Skills[slot2].Used() && b.Roll(100) < (int(b.Diff)+4)*10 {
			return c.Cast(slot2, at)
		}

		return c.Attack(ModeAttack1, at) // "goto melee"
	}

	c.SetSpeed(50)

	return c.Circle(t, 4)
}

// randomSpotNear picks a ground point 5..10 subtiles from t (UNVERIFIED
// ranges; the notes say two random offsets with the sign taken from the LCG
// parity). The returned target has ID 0, meaning a ground point.
func (b *Brain) randomSpotNear(t Target) Target {
	ox, oy := 5+b.Roll(6), 5+b.Roll(6)

	if b.Seed.Step()&1 != 0 {
		ox = -ox
	}

	if b.Seed.Step()&1 != 0 {
		oy = -oy
	}

	return Target{X: t.X + ox, Y: t.Y + oy, Size: 1}
}
