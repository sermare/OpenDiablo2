package d2monster

func init() {
	register("SkeletonBow", TargetStandard, thinkSkeletonBow)
	register("CorruptArcher", TargetStandard, thinkCorruptArcher)
	register("SkeletonMage", TargetStandard, thinkSkeletonMage)
}

// thinkSkeletonBow is MONAI_Think_SkeletonBow 0x5f5180 (VERIFIED).
// aip1 shoot%, aip2 stall, aip3 approach%, aip4 walk steps, aip5 target dist.
func thinkSkeletonBow(c *Ctx) {
	b := c.B
	pt := *c.Target

	t, d, ok := c.W.AttackTarget(b)

	if b.Aggressive {
		if ok {
			c.Attack(ModeAttack1, t)

			return
		}
	}

	if !ok || d >= 20 {
		if b.Chance(b.AIP(3)) && c.WalkToRange(pt, b.AIP(4), b.AIP(5)) {
			return
		}

		c.Sleep(20)

		return
	}

	switch {
	case b.Chance(b.AIP(1)):
		c.Attack(ModeAttack1, t)
	case b.Roll(100) < 20:
		if !c.Circle(pt, 3) {
			c.Sleep(b.AIP(2))
		}
	default:
		c.Sleep(b.AIP(2))
	}
}

// thinkCorruptArcher is MONAI_Think_CorruptArcher 0x5f4b10 (VERIFIED flow,
// 818 bytes). aip1 approach roll limit, aip2 shoot%, aip3 stall time, aip4
// run (flee-close) %, aip5 always-run distance, aip6 skill2 %, aip7 skill3 %,
// aip8 walk-toward distance. Order of the bounded rolls inside the skill
// choice is the order printed in the notes; whether the roll is skipped for an
// empty skill slot is UNVERIFIED (it is skipped here).
func thinkCorruptArcher(c *Ctx) {
	b := c.B
	pt := *c.Target

	t, d, ok := c.W.AttackTarget(b)
	if !ok {
		if b.Chance(50) && c.Circle(pt, 3) {
			return
		}

		c.Sleep(b.AIP(3))

		return
	}

	if b.Aggressive && !c.InRange {
		c.Attack(ModeAttack1, t)

		return
	}

	if d < 6 && b.Chance(b.AIP(4)) {
		c.SetSpeed(100)

		if c.RunAway(t, 12) {
			return
		}
	}

	if b.AIP(8) > 0 && d > b.AIP(8) && b.Roll(100) < b.AIP(1) {
		c.SetSpeed(10)

		if c.WalkTo(t, b.AIP(8)) {
			return
		}
	}

	if d > b.AIP(5) {
		c.SetSpeed(100)

		if c.RunTo(t, b.AIP(5)) {
			return
		}
	}

	if !b.Chance(b.AIP(2)) {
		c.Sleep(b.AIP(3))

		return
	}

	p := b.Profile

	switch {
	case p.Skills[1].Used() && b.Roll(100) < b.AIP(6):
		c.Cast(1, t)
	case p.Skills[2].Used() && b.Roll(100) < b.AIP(7):
		c.Cast(2, t)
	case p.Skills[0].Used():
		c.Cast(0, t)
	default:
		c.Attack(ModeAttack1, t)
	}
}

// thinkSkeletonMage is MONAI_Think_SkeletonMage 0x5f8830 (VERIFIED).
// aip1 shoot%, aip2 approach dist, aip3 approach%, aip4 too-close dist,
// aip5 walk-away%, aip6 fire dist, aip7 circle%, aip8 stall. The mage "casts"
// with A1; the element comes from the class (monstats MissA1).
func thinkSkeletonMage(c *Ctx) {
	b := c.B
	pt := *c.Target

	if t, d, ok := c.W.AttackTarget(b); ok {
		if d > b.AIP(2) && b.Chance(b.AIP(3)) {
			c.SetSpeed(10)

			if c.WalkTo(t, b.AIP(2)) {
				return
			}
		}

		if d <= b.AIP(4) && b.Chance(b.AIP(5)) {
			c.SetSpeed(25)

			if !c.WalkAway(t, 5) {
				c.Attack(ModeAttack1, pt)
			}

			return
		}

		if d < b.AIP(6) && b.Chance(b.AIP(1)) {
			c.Attack(ModeAttack1, t)

			return
		}
	}

	if c.Dist > b.AIP(2) && b.Chance(b.AIP(3)) {
		c.SetSpeed(10)

		if c.WalkTo(pt, b.AIP(2)) {
			return
		}
	}

	if b.Roll(100) >= b.AIP(7) {
		c.Sleep(b.AIP(8))

		return
	}

	if !c.Circle(pt, 4) {
		c.Sleep(b.AIP(8))
	}
}
