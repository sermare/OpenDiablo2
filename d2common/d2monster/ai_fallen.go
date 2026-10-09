package d2monster

func init() {
	register("Fallen", TargetStandard, thinkFallen)
}

// Fallen constants (VERIFIED from the decompile unless noted).
const (
	fallenFleeRadius   = 15 // a dying unit closer than this scares the Fallen
	fallenFleeDistance = 12
	fallenRallyRadius  = 15 // leader shouts only when the target is closer
)

// thinkFallen is MONAI_Think_Fallen 0x5ef3d0 (VERIFIED structure, 980 bytes).
// aip1 leader rally %, aip2 chase distance, aip3 attack%, aip4 A1%.
//
// Packs: the leader's S2 "shout" animation broadcasts a CmdAlert to every
// minion's command queue; a minion holding the command chases and attacks
// without needing the usual approach rolls. Commands are consumed per attack
// (Count), which is an UNVERIFIED reading of the +0x14 field.
func thinkFallen(c *Ctx) {
	b, t := c.B, *c.Target

	if !b.Mode.IsAlive() {
		return
	}

	// Flee when something nearby dies.
	if c.W.DyingNear(b, fallenFleeRadius) {
		b.Scratch[0] = 1

		b.PopCommand()
		c.SetSpeed(50)

		if c.WalkAway(t, fallenFleeDistance) {
			if b.Roll(20) == 0 {
				if s, ok := c.W.(Shouter); ok {
					s.Shout(b)
				}
			}

			return
		}
	}

	if b.Mode == ModeNeutral {
		if fallenNeutral(c, t) {
			return
		}
	}

	c.Sleep(10)
}

// fallenNeutral runs the mode==NU part; it reports whether it acted.
func fallenNeutral(c *Ctx, t Target) bool {
	b := c.B

	cmd := b.PeekCommand()

	switch {
	case cmd == nil:
		if !c.InRange && b.Aggressive {
			return c.WalkTo(t, 0)
		}

		if c.Dist < fallenRallyRadius && b.IsGroupLeader() && b.Roll(100) < b.AIP(1) {
			alert := Command{Type: CmdAlert, Count: 1}
			b.Broadcast(alert)
			b.PushCommand(alert)

			return c.Attack(ModeSkill2, t)
		}

		if !c.InRange {
			if c.Dist <= b.AIP(2) {
				return c.WalkTo(t, meleeReach)
			}

			if b.Roll(100) < 30 {
				return c.Wander(3)
			}

			return false
		}

		if b.Scratch[0] != 0 || b.Roll(100) < b.AIP(3) {
			b.Scratch[0] = 0

			attackA1orA2(c, t, b.AIP(4))

			return true
		}

		if b.Roll(100) < 30 {
			return c.Attack(ModeSkill2, t)
		}

		return false

	case cmd.Type == CmdAlert:
		if !c.InRange {
			if c.WalkTo(t, 0) {
				return true
			}

			// The notes print "PopAiCommand; return" here; sleeping after the
			// pop keeps the unit scheduled (UNVERIFIED which the exe does).
			b.PopCommand()

			return false
		}

		if b.Roll(100) >= b.AIP(3) {
			c.Sleep(5)

			return true
		}

		cmd.Count--
		if cmd.Count <= 0 {
			b.PopCommand()
		}

		attackA1orA2(c, t, b.AIP(4))

		return true

	default:
		b.PopCommand()

		return false
	}
}
