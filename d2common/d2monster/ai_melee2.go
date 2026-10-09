package d2monster

func init() {
	register("Brute", TargetStandard, thinkBrute)
	register("Mummy", TargetStandard, thinkMummy)
	register("Scarab", TargetStandard, thinkScarab)
	register("Bighead", TargetStandard, thinkBighead)
	register("CorruptRogue", TargetStandard, thinkCorruptRogue)
}

func clamp(v, lo, hi int) int {
	switch {
	case v < lo:
		return lo
	case v > hi:
		return hi
	}

	return v
}

// thinkBrute is MONAI_Think_Brute 0x5eecb0 (VERIFIED code, including a quirk).
//
// Out of range: the move speed override is 100-clamp(hp%,40,100) (the more
// hurt, the more it "rages"; the override's meaning is UNVERIFIED so engines
// ignore it), then it walks to the target. aip1 is unused.
//
// In range, as OBSERVED in the decompile (reproduced on purpose): the first
// test is chance(aip3); if it passes a second chance(aip3) (the same column
// again, where the monai comments suggest aip4) picks A1 or A2; if the first
// test fails a third chance(aip3) circles the target for 4, else the unit
// sleeps 15. The A1/A2 split is therefore by aip3 twice.
func thinkBrute(c *Ctx) {
	b, t := c.B, *c.Target

	if !c.InRange {
		c.SetSpeed(100 - clamp(b.HPPercent, 40, 100))

		// reach 7 is UNVERIFIED for this primitive (WalkToTargetResetSpeed)
		if !c.WalkTo(t, meleeReach) {
			c.Sleep(10)
		}

		return
	}

	if b.Chance(b.AIP(3)) {
		if b.Chance(b.AIP(3)) { // sic: aip3 again, see above
			c.Attack(ModeAttack1, t)
		} else {
			c.Attack(ModeAttack2, t)
		}

		return
	}

	if b.Chance(b.AIP(3)) && c.Circle(t, 4) {
		return
	}

	c.Sleep(15)
}

// thinkMummy is MONAI_Think_Mummy 0x5f1870 (VERIFIED). aip1 awake distance,
// aip2 wander%, aip3 attack%, aip4 A1%, aip5 stall.
//
// As in the decompile, after a failed attack roll in range the unit sleeps
// aip5 and then falls straight into the walk-to-target request; the notes
// call this probably intended ("stalls, then lunges") and it is kept: the
// walk request replaces the sleep when it can be submitted, and when the unit
// already stands within reach the request fails and the sleep stands.
func thinkMummy(c *Ctx) {
	b, t := c.B, *c.Target

	if b.Aggressive && !c.InRange && c.WalkTo(t, meleeReach) {
		return
	}

	if c.Dist <= b.AIP(1) {
		if c.InRange {
			if b.Chance(b.AIP(3)) {
				attackA1orA2(c, t, b.AIP(4))

				return
			}

			c.Sleep(b.AIP(5))
		}

		c.WalkTo(t, meleeReach)

		return
	}

	if b.Chance(b.AIP(2)) && c.Wander(3) {
		return
	}

	c.Sleep(b.AIP(5))
}

// scarabJab is monstats skill slot 1 ("Jab"), VERIFIED to be Skill1.
const scarabJab = 0

// thinkScarab is MONAI_Think_Scarab 0x5f1570. aip1 attack%, aip2 A1-vs-A2 %,
// aip3 stall, aip4 jab%, aip5 rally% (sample 75/50/15/35/20). The notes
// describe the structure but the exact rolls are marked partly uncertain
// ("aip4-code uses +0x6e = aip5 'cmd?'"), so the group rally and the toggle
// are UNVERIFIED readings:
//
//   - a group leader closer than 20 broadcasts the alert (type 1) with chance
//     aip5; followers holding the alert chase the target and jab/attack;
//   - without a command, an out-of-range scarab alternates between circling
//     (scratch14=1) and walking to the target (VERIFIED: "scratch14 toggles
//     circle vs walk");
//   - in range: chance(aip1) failing sleeps aip3 (VERIFIED), Jab with chance
//     aip4 (VERIFIED skill), else A1 with chance aip2, else A2 (VERIFIED).
func thinkScarab(c *Ctx) {
	b, t := c.B, *c.Target

	cmd := b.PeekCommand()
	if cmd != nil && cmd.Type != CmdAlert {
		b.PopCommand()

		cmd = nil
	}

	if cmd == nil && b.IsGroupLeader() && c.Dist < 20 && b.Roll(100) < b.AIP(5) {
		alert := Command{Type: CmdAlert, Count: 1}
		b.Broadcast(alert)
		b.PushCommand(alert)
	}

	if !c.InRange {
		if cmd == nil && b.Scratch[0] == 0 {
			b.Scratch[0] = 1

			if c.Circle(t, 3) {
				return
			}
		}

		b.Scratch[0] = 0

		if !c.WalkTo(t, meleeReach) {
			c.Sleep(b.AIP(3))
		}

		return
	}

	if !b.Chance(b.AIP(1)) && cmd == nil {
		c.Sleep(b.AIP(3))

		return
	}

	if cmd != nil {
		cmd.Count--
		if cmd.Count <= 0 {
			b.PopCommand()
		}
	}

	if b.Profile.Skills[scarabJab].Used() && b.Chance(b.AIP(4)) {
		c.Cast(scarabJab, t)

		return
	}

	attackA1orA2(c, t, b.AIP(2))
}

// thinkBighead is MONAI_Think_Bighead 0x5ef060 (VERIFIED). aip1 hurt%, aip2
// circle%, aip3 fire-while-healthy%, aip4 fire-while-hurt%. Healthy (hp% >=
// aip1) it fights in melee and sometimes casts A2; hurt it kites and casts
// A2. The wounded MonTeleport of MONAI_PostTargetChecks is not ported
// (TODO: needs the skill pipeline).
func thinkBighead(c *Ctx) {
	b, t := c.B, *c.Target

	if !c.InRange && b.Aggressive {
		c.Attack(ModeAttack2, t)

		return
	}

	if b.HPPercent >= b.AIP(1) {
		bigheadHealthy(c, t)

		return
	}

	switch {
	case c.Dist < 3:
		c.SetSpeed(50)

		if !c.WalkAway(t, 5) {
			c.Attack(ModeAttack2, t)
		}
	case c.Dist > 15:
		if !c.WalkTo(t, 6) {
			c.Sleep(10)
		}
	default:
		if at, _, ok := c.W.AttackTarget(b); ok && b.Roll(100) < b.AIP(4) {
			c.Attack(ModeAttack2, at)

			return
		}

		if b.Roll(100) >= b.AIP(2) {
			c.Sleep(10)

			return
		}

		if !c.Circle(t, 3) {
			c.Sleep(10)
		}
	}
}

func bigheadHealthy(c *Ctx, t Target) {
	b := c.B

	switch {
	case c.InRange:
		c.Attack(ModeAttack1, t)
	case c.Dist < 15:
		if at, _, ok := c.W.AttackTarget(b); ok && b.Roll(100) < b.AIP(3) {
			c.Attack(ModeAttack2, at)

			return
		}

		fallthrough
	default:
		if !c.WalkTo(t, meleeReach) {
			c.Sleep(10)
		}
	}
}

// corruptRogueRunDistance is the distance beyond which a Corrupt Rogue runs
// at the target. The exe subtracts 3*something from 20 using the group info of
// FUN_00571760 (UNVERIFIED); that term is taken as 0 here.
const corruptRogueRunDistance = 20

// thinkCorruptRogue is MONAI_Think_CorruptRogue 0x5efbf0 (VERIFIED flow,
// the group-info term UNVERIFIED). aip1 approach%, aip2 stall, aip3 attack%,
// aip4 run velocity, aip5 run%. Also drives The Countess (class 45) whose six
// followers come from the spawn plan, not from this function.
func thinkCorruptRogue(c *Ctx) {
	b, t := c.B, *c.Target

	if c.Dist > corruptRogueRunDistance {
		c.SetSpeed(b.AIP(4))

		if c.RunTo(t, 3) {
			return
		}
	}

	if c.InRange {
		if b.Chance(b.AIP(3)) {
			c.Attack(ModeAttack1, t)
		} else {
			c.Sleep(b.AIP(2))
		}

		return
	}

	if !b.Chance(b.AIP(1)) {
		c.Sleep(b.AIP(2))

		return
	}

	if b.Roll(100) >= b.AIP(5) {
		if !c.WalkTo(t, meleeReach) {
			c.Sleep(b.AIP(2))
		}

		return
	}

	c.SetSpeed(b.AIP(4))

	if !c.RunTo(t, 3) {
		c.Sleep(b.AIP(2))
	}
}
