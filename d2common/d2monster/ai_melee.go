package d2monster

// meleeReach is the stop distance (subtiles) of "walk to melee" (VERIFIED, 7).
const meleeReach = 7

// zombieGraveyardLevel is the levels.txt id (17) at which Zombies always
// chase once they have a target (VERIFIED constant in the decompile).
const zombieGraveyardLevel = 17

func init() {
	register("None", TargetNone, func(*Ctx) {})
	register("Idle", TargetNone, thinkIdle)
	register("Skeleton", TargetStandard, thinkSkeleton)
	register("Goatman", TargetStandard, thinkGoatman)
	register("Swarm", TargetStandard, thinkSwarm)
	register("Wraith", TargetStandard, thinkWraith)
	register("Zombie", TargetStandard, thinkZombie)
}

// thinkIdle is AI 1 (VERIFIED): wait 200 ticks. (The original does so only
// if the unit has a room; the owner simply does not tick monsters without
// one.) Idle monsters never wander: wandering comes from tick acquisition.
func thinkIdle(c *Ctx) { c.Sleep(200) }

// thinkSkeleton is MONAI_Think_Skeleton 0x5eee10 (VERIFIED).
// aip1 approach%, aip2 stall ticks, aip3 attack%, aip4 A1-vs-A2 %.
func thinkSkeleton(c *Ctx) {
	b, t := c.B, *c.Target

	if !c.InRange {
		if b.Chance(b.AIP(1)) && c.WalkTo(t, meleeReach) {
			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if b.Chance(b.AIP(3)) {
		attackA1orA2(c, t, b.AIP(4))

		return
	}

	c.Sleep(b.AIP(2))
}

// attackA1orA2 rolls "chance(pctA1) ? A1 : A2".
func attackA1orA2(c *Ctx, t Target, pctA1 int) {
	if c.B.Chance(pctA1) {
		c.Attack(ModeAttack1, t)
	} else {
		c.Attack(ModeAttack2, t)
	}
}

// thinkGoatman is MONAI_Think_Goatman 0x5f0360 (VERIFIED): Skeleton, except
// that in range it always uses A1.
func thinkGoatman(c *Ctx) {
	b, t := c.B, *c.Target

	if !c.InRange {
		if b.Chance(b.AIP(1)) && c.WalkTo(t, meleeReach) {
			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if b.Chance(b.AIP(3)) {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(b.AIP(2))
}

// thinkSwarm is MONAI_Think_Swarm 0x5f14b0 (VERIFIED): like Skeleton, wait
// aip2, attack always A1.
func thinkSwarm(c *Ctx) { thinkGoatman(c) }

// thinkWraith is MONAI_Think_Wraith 0x5efb30 (VERIFIED). In range:
// chance(aip3) -> A1 else sleep(aip2). Out of range: chance(aip1) -> walk to
// range (step 12, desired 0) else sleep(aip2).
func thinkWraith(c *Ctx) {
	b, t := c.B, *c.Target

	if c.InRange {
		if b.Chance(b.AIP(3)) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if b.Chance(b.AIP(1)) && c.WalkToRange(t, 12, 0) {
		return
	}

	c.Sleep(b.AIP(2))
}

// thinkZombie is MONAI_Think_Zombie 0x5eef40 (VERIFIED flow). In range:
// chance(aip4) ? A1 : A2 (no aip3 gate). Out of range: unless the unit is
// "aggressive" (flag VERIFIED as pUnitData+0x54 in {3,0x13}, see Brain.Aggressive), a unit
// closer than aip2 chases with chance aip1; otherwise it wanders 3, except in
// the Graveyard (level 17) where it always chases. The chase is a run.
func thinkZombie(c *Ctx) {
	b, t := c.B, *c.Target

	if c.InRange {
		attackA1orA2(c, t, b.AIP(4))

		return
	}

	if !b.Aggressive {
		chase := c.Dist < b.AIP(2) && b.Chance(b.AIP(1))

		if !chase && b.LevelID != zombieGraveyardLevel {
			if c.Wander(3) {
				return
			}

			c.Sleep(10)

			return
		}
	}

	c.SetSpeed(100)

	if !c.RunTo(t, 0) {
		c.Sleep(10)
	}
}
