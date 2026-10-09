package d2monster

// Hireable is MONAI_Think_Hireable (0x5e4240, monster-ai-2.md 6.2): the AI of
// mercenaries (rogue 271, desert guard 338, Iron Wolf 359, barbarians
// 560/561). It is registered under "Hireable"; the engine assigns it to merc
// units directly because their monstats AI column is not "Hireable".

// Distances of the follow logic (VERIFIED defaults, notes 6.2).
const (
	mercTeleportDist = 100 // edge distance above which the merc teleports to the owner
	mercOuterDefault = 0x18
	mercFarDefault   = 0x10
	mercInnerDefault = 5
	mercSeekRadius   = 25 // hostile scan radius around the merc
	mercThrCap       = 0x5f
	mercThrFixed     = 0x62
)

// OwnerInfo is the snapshot of a merc's owner.
type OwnerInfo struct {
	Target
	Mode    Mode
	LevelID int
}

// MercWorld extends World with what the Hireable AI needs.
type MercWorld interface {
	World
	// Owner returns the owning player; ok is false when there is none.
	Owner(b *Brain) (OwnerInfo, bool)
	// Teleport is FollowHelper(3): move to a free tile next to the owner.
	Teleport(b *Brain) bool
	// MercLevel is the merc's level (stat 0xc).
	MercLevel(b *Brain) int
	// IsRanged is true for archers and mages (rogue, Iron Wolf).
	IsRanged(b *Brain) bool
	// ChooseAndCast is MERC_ChooseAndQueueSkill: weighted skill pick, or the
	// default attack. It reports whether an action was started.
	ChooseAndCast(b *Brain, t Target, dist int) bool
}

func init() {
	register("Hireable", TargetNone, thinkHireable)
}

// mercFixedThreshold lists the classes with the fixed attack threshold 0x62
// (desert guard, barbarians; VERIFIED).
func mercFixedThreshold(class int) bool { return class == 338 || class == 560 || class == 561 }

func thinkHireable(c *Ctx) {
	w, ok := c.W.(MercWorld)
	if !ok {
		c.Sleep(25)

		return
	}

	b := c.B

	owner, ok := w.Owner(b)
	if !ok {
		c.Sleep(10) // the original switches to a forced-state AI; no owner means idle here

		return
	}

	outer, far, inner := mercOuterDefault, mercFarDefault, mercInnerDefault

	if band := b.Scratch[0]; band >= 0x11 && band <= 0x13 {
		outer, far, inner = 2*band, band, band>>1
	}

	d := EdgeDistance(b.X-owner.X, b.Y-owner.Y, b.Size)

	switch {
	case d > mercTeleportDist:
		if w.Teleport(b) {
			c.Sleep(5)

			return
		}
	case d > outer:
		c.SetSpeed(0x3c)

		if c.RunTo(owner.Target, inner) {
			return
		}
	case d > far:
		switch owner.Mode {
		case ModeWalk:
			if c.WalkTo(owner.Target, inner) {
				return
			}
		case ModeRun:
			if c.RunTo(owner.Target, inner) {
				return
			}
		}
	}

	if b.Mode == ModeNeutral {
		if t, dist, ok := w.AttackTarget(b); ok && dist <= mercSeekRadius {
			mercDecideAttack(c, w, t, dist)

			return
		}

		switch {
		case b.LevelID != owner.LevelID:
			if c.RunTo(owner.Target, inner) {
				return
			}
		case d < 2:
			if c.WalkNearRandom(owner.Target, inner-1) {
				return
			}
		case b.Roll(100) < 5:
			if c.WalkNearRandom(owner.Target, far) {
				return
			}
		}
	}

	c.Sleep(5)
}

// mercDecideAttack is MERC_DecideAttack (0x5e3fa0, VERIFIED): a pressure
// counter in Scratch[0] raises the chance to hold back; ranged mercs keep the
// target within reach and shoot, melee mercs shuffle when too close.
func mercDecideAttack(c *Ctx, w MercWorld, t Target, dist int) {
	b := c.B

	thr := b.Scratch[0] + 0x28 + 2*w.MercLevel(b)
	if thr > mercThrCap {
		thr = mercThrCap
	}

	if mercFixedThreshold(b.Class) {
		thr = mercThrFixed
	}

	bump := b.Roll(100) >= thr
	if bump {
		b.Scratch[0] += 10
	} else {
		b.Scratch[0] = 0
	}

	inRange := w.InRange(b, t, dist)

	if w.IsRanged(b) {
		switch {
		case dist <= 3 && inRange:
			if bump {
				c.Sleep(10)

				return
			}
		case inRange: // UNVERIFIED: the exe walks to range with FUN_005ddae0; shooting from where it stands is the safe reading
		default:
			if c.WalkTo(t, meleeReach) {
				return
			}

			c.Sleep(10)

			return
		}

		if !w.ChooseAndCast(b, t, dist) {
			c.Sleep(10)
		} else {
			c.busy()
		}

		return
	}

	if dist < 4 && b.Chance(50) {
		// UNVERIFIED which of "near" and "away" is taken: alternate on the RNG
		if b.Chance(50) {
			if c.WalkNearRandom(t, 4) {
				return
			}
		} else if c.WalkAway(t, 4) {
			return
		}
	}

	if bump {
		c.Sleep(10)

		return
	}

	if !w.ChooseAndCast(b, t, dist) {
		c.Sleep(10)

		return
	}

	c.busy()
}

// WalkNearRandom is MONAI_WalkNearTargetRandom(target, n): walk to a random
// point within n subtiles of the target (U: the notes only say the offsets
// are rolled like Wander).
func (c *Ctx) WalkNearRandom(t Target, n int) bool {
	if n < 1 {
		n = 1
	}

	b := c.B
	dx := b.Roll(2*n+1) - n
	dy := b.Roll(2*n+1) - n

	return c.move(Point{t.X + dx, t.Y + dy}, nil, 0, false)
}
