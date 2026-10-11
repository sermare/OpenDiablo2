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
// In range (VERIFIED again in batch 7): chance(aip3) gates the attack and the
// A1/A2 split is chance(aip4) (monstats +0x68; batch 4 read +0x62 twice by
// mistake); if the first test fails, chance(aip3) circles the target for 4,
// else the unit sleeps 15.
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
		if b.Chance(b.AIP(4)) {
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

// thinkScarab is MONAI_Think_Scarab 0x5f1570 (VERIFIED in batch 7 from the
// decompile; batch 4 had the alert path and the roll order wrong). aip1
// attack gate %, aip2 A1 %, aip3 stall, aip4 jab %, aip5 rally %.
//
//   - A group leader closer than 20 with no command rolls aip5 and, on a pass,
//     broadcasts an alert (type 1) and keeps its own copy. A command of another
//     type is NOT popped: the unit simply ignores it.
//   - Holding an alert: in reach with Skill1 defined it pops the alert and
//     jabs at once (no roll); otherwise speed override 100 and an exact-tile
//     walk to the target, popping the alert when the walk cannot be queued.
//   - Without an alert, out of reach: Scratch[0] set means a walk to 7 (speed
//     override 0) after which a roll above 10 clears the flag; clear means a
//     strafe and the flag is set. In reach: roll(100) >= aip1 stalls aip3,
//     then Jab with aip4 (only when defined), then A1 with aip2 else A2.
func thinkScarab(c *Ctx) {
	b, t := c.B, *c.Target

	cmd := b.PeekCommand()
	if cmd == nil && c.Dist < 20 && b.IsGroupLeader() && b.Roll(100) < b.AIP(5) {
		alert := Command{Type: CmdAlert, Count: 1}
		b.Broadcast(alert)
		b.PushCommand(alert)

		cmd = b.PeekCommand()
	}

	if cmd != nil && cmd.Type == CmdAlert {
		if c.InRange && b.Profile.Skills[scarabJab].Used() {
			b.PopCommand()
			c.Cast(scarabJab, t)

			return
		}

		c.SetSpeed(100)

		if !c.WalkTo(t, 0) {
			b.PopCommand()
		}

		return
	}

	if !c.InRange {
		if b.Scratch[0] != 0 {
			c.SetSpeed(0)
			c.WalkTo(t, meleeReach)

			if b.Roll(100) > 10 {
				b.Scratch[0] = 0
			}

			return
		}

		c.Circle(t, 3)

		b.Scratch[0] = 1

		return
	}

	if !b.Chance(b.AIP(1)) {
		c.Sleep(b.AIP(3))

		return
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
		// batch 7: the exe queues the shot at the TICK target, the attack target
		// only gates it
		if _, _, ok := c.W.AttackTarget(b); ok && b.Roll(100) < b.AIP(4) {
			c.Attack(ModeAttack2, t)

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
		if _, _, ok := c.W.AttackTarget(b); ok && b.Roll(100) < b.AIP(3) {
			c.Attack(ModeAttack2, t)

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
// at the target: 20 - 3*s where s is the second output of
// MONSTER_CalcPlayerCountScaling (0x571760), the PlayerScaler host (VERIFIED
// in batch 7: the exe computes 0x14 - 3*s; s is taken as 0 without a host).
const corruptRogueRunDistance = 20

// thinkCorruptRogue is MONAI_Think_CorruptRogue 0x5efbf0 (VERIFIED flow,
// the group-info term UNVERIFIED). aip1 approach%, aip2 stall, aip3 attack%,
// aip4 run velocity, aip5 run%. Also drives The Countess (class 45) whose six
// followers come from the spawn plan, not from this function.
func thinkCorruptRogue(c *Ctx) {
	b, t := c.B, *c.Target

	scale := 0
	if ps, ok := c.W.(PlayerScaler); ok {
		scale = ps.PlayerScaling(b)
	}

	if c.Dist > corruptRogueRunDistance-3*scale {
		c.SetSpeed(b.AIP(4))

		// the exe ends the tick here whether or not the run was queued
		if !c.RunTo(t, 3) {
			c.Sleep(b.AIP(2))
		}

		return
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
