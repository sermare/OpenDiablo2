package d2monster

// Summoned-minion AIs: NecroPet (Necromancer skeletons, mages, golems,
// Valkyrie; monster-ai-2.md 6.2, VERIFIED for the melee path), plus simplified
// Raven, Vines/CycleOfLife and AssassinSentry thinkers (UNVERIFIED: the notes
// only give their table slots, so the behaviour below is the observable
// minimum - follow the owner, hit enemies near it, stationary units fire
// their first monstats skill at the nearest enemy).

// Distances of the NecroPet think (FUN_5e3770).
const (
	petTeleportDist = 50 // VERIFIED: edge distance from the owner above which the pet teleports
	petEngageDist   = 29 // VERIFIED: below it the pet looks for enemies
	petEnemyRadius  = 24 // VERIFIED: enemies of the owner are searched within this radius
	petMeleeReach   = 6  // edge distance at which the pet stops walking up to a foe
	petRunPercent   = 14 // VERIFIED: roll > this -> run, else walk
	petAttackChance = 80 // VERIFIED: percent chance to attack when in range
	petIdleFollow   = 8  // UNVERIFIED: closer than this to the owner the pet stands still
)

// PetWorld extends World with what summoned minions need.
type PetWorld interface {
	World
	// Owner returns the summoner; ok is false when it is gone.
	Owner(b *Brain) (OwnerInfo, bool)
	// Teleport is FollowHelper(3): move to a free tile next to the owner.
	Teleport(b *Brain) bool
	// OwnerEnemy is FUN_005dcc20: the nearest enemy of the owner within
	// radius of the pet, with its edge distance. Friendly units (flag
	// 0x40000000) are never returned.
	OwnerEnemy(b *Brain, radius int) (t Target, dist int, ok bool)
}

func init() {
	register("NecroPet", TargetNone, thinkNecroPet)
	register("Raven", TargetNone, thinkRaven)
	register("Vines", TargetNone, thinkStationary)
	register("CycleOfLife", TargetNone, thinkStationary)
	register("AssassinSentry", TargetNone, thinkSentry)
}

// thinkNecroPet is MONAI_Think_NecroPet. Scratch[0] != 0 marks a casting pet
// (Valkyrie, Skeleton Mage, golems with a skill): FUN_5e3a00 is not ported, so
// it shares the melee flow and casts slot 0 instead of the A1 attack
// (UNVERIFIED).
func thinkNecroPet(c *Ctx) {
	w, ok := c.W.(PetWorld)
	if !ok {
		c.Sleep(25)

		return
	}

	b := c.B

	owner, ok := w.Owner(b)
	if !ok {
		// the remembered owner id (Scratch[2], AiGeneral+0x1c) still allows a
		// teleport back; otherwise wait
		if b.Scratch[2] != 0 && w.Teleport(b) {
			c.Sleep(5)

			return
		}

		c.Sleep(10)

		return
	}

	b.Scratch[2] = int(owner.ID)

	d := EdgeDistance(b.X-owner.X, b.Y-owner.Y, b.Size)

	if d > petTeleportDist && w.Teleport(b) {
		c.Sleep(5)

		return
	}

	if d >= petEngageDist {
		if !c.WalkTo(owner.Target, petIdleFollow) {
			c.Sleep(10)
		}

		return
	}

	petEngage(c, w, owner, d, b.Scratch[0] != 0)
}

// petEngage fights an enemy of the owner or stays close to the owner.
func petEngage(c *Ctx, w PetWorld, owner OwnerInfo, d int, caster bool) {
	b := c.B

	enemy, dist, ok := w.OwnerEnemy(b, petEnemyRadius)
	if !ok {
		if d > petIdleFollow {
			if b.Roll(100) > petRunPercent {
				if c.RunTo(owner.Target, petIdleFollow) {
					return
				}
			} else if c.WalkTo(owner.Target, petIdleFollow) {
				return
			}
		}

		c.Sleep(5)

		return
	}

	if w.InRange(b, enemy, dist) {
		if !b.Chance(petAttackChance) {
			c.Sleep(10)

			return
		}

		if caster {
			c.Cast(0, enemy)
		} else {
			c.Attack(ModeAttack1, enemy)
		}

		return
	}

	// out of range: walk or run up to the enemy
	var moved bool

	if b.Roll(100) > petRunPercent {
		moved = c.RunTo(enemy, petMeleeReach)
	} else {
		moved = c.WalkTo(enemy, petMeleeReach)
	}

	if !moved {
		c.Sleep(10)
	}
}

// thinkRaven (UNVERIFIED): ravens stay around the owner and peck enemies of
// the owner; monstats aip1 is the percent chance to attack when in range, aip2
// the sleep in frames when idle.
func thinkRaven(c *Ctx) {
	w, ok := c.W.(PetWorld)
	if !ok {
		c.Sleep(25)

		return
	}

	b := c.B

	owner, ok := w.Owner(b)
	if !ok {
		c.Sleep(10)

		return
	}

	d := EdgeDistance(b.X-owner.X, b.Y-owner.Y, b.Size)
	if d > petTeleportDist && w.Teleport(b) {
		c.Sleep(5)

		return
	}

	enemy, dist, found := w.OwnerEnemy(b, petEnemyRadius)

	switch {
	case found && d < petEngageDist:
		if w.InRange(b, enemy, dist) {
			if b.Chance(b.AIP(1)) {
				c.Attack(ModeAttack1, enemy)

				return
			}
		} else if c.RunTo(enemy, 1) {
			return
		}
	case d > petIdleFollow:
		if c.RunTo(owner.Target, petIdleFollow) {
			return
		}
	}

	c.Sleep(maxInt(b.AIP(2), 1))
}

// thinkSentry (UNVERIFIED): a trap fires monstats skill 1 at the nearest enemy
// it can reach; aip1 is the percent chance per think (0 = always), aip2 the
// sleep between thinks.
func thinkSentry(c *Ctx) {
	b := c.B

	if t, dist, ok := c.W.AttackTarget(b); ok && c.W.InRange(b, t, dist) {
		if p := b.AIP(1); p == 0 || b.Chance(p) {
			c.Cast(0, t)

			return
		}
	}

	c.Sleep(maxInt(b.AIP(2), 5))
}

// thinkStationary (UNVERIFIED): rooted vine plants strike the nearest enemy
// with skill 1 whenever one is in reach.
func thinkStationary(c *Ctx) {
	b := c.B

	if t, dist, ok := c.W.AttackTarget(b); ok && c.W.InRange(b, t, dist) {
		c.Cast(0, t)

		return
	}

	c.Sleep(maxInt(b.AIP(2), 5))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
