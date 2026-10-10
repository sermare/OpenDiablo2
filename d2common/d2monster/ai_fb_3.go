package d2monster

// Faithful ports (batch 3) of Spirit, TrappedSoul, UberBaal, the Trap-* family,
// SandMaggotQueen, Sarcophagus, Tentacle, TentacleHead, FrogDemon, SiegeTower,
// SiegeBeast and ReanimatedHorde. Each function names its exe address; the
// roll order is the exe's. Scratch[0..2] are AiGeneral +0x14/+0x18/+0x1c.

func fbxReg(name string, t Think) {
	m, ok := AITargetMode(name)
	if !ok {
		m = TargetStandard
	}

	register(name, m, t)
}

func init() {
	fbxReg("UberBaal", thinkUberBaalB)
	fbxReg("Spirit", thinkSpiritB)
	fbxReg("TrappedSoul", thinkTrappedSoulB)
	fbxReg("Trap-Melee", thinkTrapMeleeB)
	fbxReg("Trap-Missile", func(c *Ctx) { fbxTrapShooter(c, false) })
	fbxReg("Trap-Poison", func(c *Ctx) { fbxTrapShooter(c, true) })
	fbxReg("Trap-Nova", func(c *Ctx) { fbxTrapShooter(c, true) })
	fbxReg("Trap-RightArrow", func(c *Ctx) { fbxTrapArrow(c, true) })
	fbxReg("Trap-LeftArrow", func(c *Ctx) { fbxTrapArrow(c, false) })
	fbxReg("SandMaggotQueen", thinkSandMaggotQueenB)
	fbxReg("Sarcophagus", thinkSarcophagusB)
	fbxReg("Tentacle", func(c *Ctx) { thinkTentacleB(c, true) })
	fbxReg("TentacleHead", func(c *Ctx) { thinkTentacleB(c, false) })
	fbxReg("FrogDemon", thinkFrogDemonB)
	fbxReg("SiegeTower", thinkSiegeTowerB)
	fbxReg("SiegeBeast", thinkSiegeBeastB)
	fbxReg("ReanimatedHorde", thinkReanimatedHordeB)
}

// FBXOwnerFinder is an optional extension: the owner/leader of a summoned or
// minion unit (MONAI_GetLeaderUnit, UNIT_FindOwnerUnitIfFlagged). Fallbacks:
// the existing Owner(b) of PetWorld/MercWorld, then Brain.Leader.
func fbxLeader(c *Ctx) (OwnerInfo, bool) {
	if l := c.B.Leader; l != nil {
		return OwnerInfo{Target: Target{ID: l.ID, X: l.X, Y: l.Y, Size: l.Size}, Mode: l.Mode, LevelID: l.LevelID}, true
	}

	if w, ok := c.W.(interface {
		Owner(b *Brain) (OwnerInfo, bool)
	}); ok {
		return w.Owner(c.B)
	}

	return OwnerInfo{}, false
}

// FBXReaper is an optional Actor extension: COMBAT_ServerHandleUnitDeath(killer,
// 1), the monster kills itself. Fallback: request mode DT.
type FBXReaper interface {
	FBXKill(b *Brain, killer *Target)
}

func fbxKill(c *Ctx, killer *Target) {
	if r, ok := c.W.(FBXReaper); ok {
		r.FBXKill(c.B, killer)
		c.acted = true

		return
	}

	c.Attack(ModeDying, fbxSelf(c.B))
}

// thinkUberBaalB: 0x5fc430 is a bare return; no event is queued.
func thinkUberBaalB(c *Ctx) { fbxStop(c) }

// thinkSpiritB is MONAI_Think_Spirit 0x5e2720 (VERIFIED from disassembly):
// after its one attack it only sleeps 50; out of range it sleeps 10.
func thinkSpiritB(c *Ctx) {
	b := c.B

	switch {
	case b.Scratch[0] != 0:
		c.Sleep(50)
	case !c.InRange || c.Target == nil:
		c.Sleep(10)
	default:
		b.Scratch[0] = 1
		c.Attack(ModeAttack1, *c.Target)
	}
}

// thinkTrappedSoulB is MONAI_Think_TrappedSoul 0x5e8fc0 (VERIFIED). Every
// think sets unit flag 0x20000. Scratch[0] = has shouted, Scratch[1] = next
// strike frame.
func thinkTrappedSoulB(c *Ctx) {
	b, now := c.B, c.W.Frame()

	fbxFlags(c, 0x20000, 0)

	if c.Target == nil || c.Dist > 4 {
		if b.Scratch[0] != 0 {
			c.Attack(ModeSkill1, fbxTargetOrSelf(c))

			return
		}

		c.Sleep(15)

		return
	}

	t := *c.Target

	if b.Scratch[0] == 0 {
		b.Scratch[0], b.Scratch[1] = 1, now
		c.Attack(ModeSkill2, t)

		return
	}

	if !c.InRange || now <= b.Scratch[1] {
		c.Attack(ModeSkill1, t)

		return
	}

	sx, sy, tx, ty := b.X, b.Y, t.X, t.Y

	if tx <= sx {
		if sy <= ty {
			c.Attack(ModeAttack1, t)
			b.Scratch[1] = now + 0x23

			return
		}

		if tx < sx {
			c.Attack(ModeSkill1, t)
			b.Scratch[1] = now + 5

			return
		}
	}

	if ty <= sy {
		c.Attack(ModeAttack2, t)
		b.Scratch[1] = now + 0x23

		return
	}

	c.Attack(ModeSkill1, t)
	b.Scratch[1] = now + 5
}

func fbxTargetOrSelf(c *Ctx) Target {
	if c.Target != nil {
		return *c.Target
	}

	return fbxSelf(c.B)
}

// thinkTrapMeleeB is MONAI_Think_TrapMelee 0x5fabd0 (VERIFIED): aip1 attack%,
// aip2 stall. Not in range: sleep 40 without a roll.
func thinkTrapMeleeB(c *Ctx) {
	b := c.B

	if !c.InRange || c.Target == nil {
		c.Sleep(0x28)

		return
	}

	if b.Roll(100) < b.AIP(1) {
		c.Attack(ModeAttack1, *c.Target)

		return
	}

	c.Sleep(b.AIP(2))
}

// fbxTrapShooter is Trap-Missile 0x5fa730 and Trap-Poison/Nova 0x5faa70 /
// 0x5fab20 (VERIFIED). aip1 max distance, aip2 shots in total, aip3 pause.
// Scratch[0] shots fired, Scratch[1] 1 while the pause is due. skill selects
// the Poison/Nova form (cast Skill1, which must be set) over a plain A1.
func fbxTrapShooter(c *Ctx, skill bool) {
	b := c.B

	if c.Target != nil && c.Dist <= b.AIP(1) && b.Scratch[0] < b.AIP(2) {
		if b.Scratch[1] == 0 && (!skill || fbxSkill(b, 0)) {
			if skill {
				c.Cast(0, *c.Target)
			} else {
				c.Attack(ModeAttack1, *c.Target)
			}

			b.Scratch[0]++
			b.Scratch[1] = 1

			return
		}

		b.Scratch[1] = 0
		c.Sleep(b.AIP(3))

		return
	}

	// spent (or nothing to shoot at): the trap removes itself
	fbxFlags(c, 0x20000, 0)
	c.Attack(ModeDying, fbxSelf(b))
}

// FBXTrapVariant is MONAI_GetCachedTrapArrowVariant: per level cached 0 below
// level id 40, else a draw from the level seed in [0,3). Variant 1 makes the
// arrow traps cast Skill1 instead of A1. Fallback 0.
type FBXTrapVariant interface {
	FBXTrapArrowVariant(b *Brain) int
}

// fbxTrapArrow is Trap-RightArrow 0x5fa830 (x axis) and Trap-LeftArrow
// 0x5fa950 (y axis), VERIFIED. aip1/aip2 distance window, aip3 reload, aip4
// Skill1 reload.
func fbxTrapArrow(c *Ctx, xAxis bool) {
	b, now := c.B, c.W.Frame()

	if c.Target == nil || c.Dist < b.AIP(1) || c.Dist > b.AIP(2) {
		c.Sleep(0x28)

		return
	}

	t := *c.Target

	delta := b.Y - t.Y
	if xAxis {
		delta = b.X - t.X
	}

	if delta < 0 {
		delta = -delta
	}

	if delta < 3 && b.Scratch[0] < now {
		b.Scratch[0] = b.AIP(3) + now

		variant := 0
		if v, ok := c.W.(FBXTrapVariant); ok {
			variant = v.FBXTrapArrowVariant(b)
		}

		if variant != 1 {
			c.Attack(ModeAttack1, t)

			return
		}

		if fbxSkill(b, 0) {
			b.Scratch[2] = b.AIP(4) + now
			c.Cast(0, t)

			return
		}
	}

	c.Sleep(0x1e)
}

// FBXMinionSpawner creates the SandMaggotQueen's young at a free cell near the
// queen (MONSTER_FindSpawnPositionForMinion + spawn with flag 0x4000000).
// Returns true when one was created. Fallback: nothing is created.
type FBXMinionSpawner interface {
	FBXSpawnMinion(b *Brain) bool
}

// thinkSandMaggotQueenB is MONAI_Think_SandMaggotQueen 0x5f8e50 (VERIFIED).
// aip1 brood limit, aip2 delay in seconds (x25 frames after a spawn).
// Scratch[0] spawned, Scratch[1] spawn animation running, Scratch[2] cooling.
func thinkSandMaggotQueenB(c *Ctx) {
	b := c.B

	if b.Scratch[2] != 0 {
		c.Sleep(b.AIP(2) * 25)
		b.Scratch[2] = 0

		return
	}

	if b.Scratch[1] == 0 {
		if b.Scratch[0] < b.AIP(1) {
			c.Attack(ModeSkill1, fbxSelf(b))
			c.Sleep(b.AIP(2))

			b.Scratch[1] = 1

			return
		}

		// brood complete: the exe returns without queueing anything
		fbxStop(c)

		return
	}

	if s, ok := c.W.(FBXMinionSpawner); ok && s.FBXSpawnMinion(b) {
		b.Scratch[0]++
	}

	c.Sleep(b.AIP(2))

	b.Scratch[1] = 0
	b.Scratch[2] = 1
}

// thinkSarcophagusB is MONAI_Think_Sarcophagus 0x5f5ab0 (VERIFIED), with the
// shared spawner Pre hook (Scratch[0] = spawn frame, Scratch[1] = 0) done on
// the first think (Scratch[2] marks it). aip1 interval, aip3 spawn limit.
func thinkSarcophagusB(c *Ctx) {
	b, now := c.B, c.W.Frame()

	if b.Scratch[2] == 0 {
		b.Scratch[2], b.Scratch[0], b.Scratch[1] = 1, now, 0
	}

	if c.Dist > 25 {
		c.Sleep(25)

		return
	}

	if b.AIP(3) < b.Scratch[1] {
		fbxFlags(c, 0x20000, 0)
		c.Attack(ModeDying, fbxSelf(b))

		return
	}

	d := now - b.Scratch[0]
	if d < 0 {
		d = -d
	}

	if fbxSkill(b, 0) && b.AIP(1) <= d && fbxSpawnCellsFree(c) {
		b.Scratch[1]++
		b.Scratch[0] = now
		c.Cast(0, fbxTargetOrSelf(c))

		return
	}

	// the exe's expression reduces to 20 + low32 % 10 with one LCG step
	c.Sleep(20 + b.Roll(10))
}

// thinkTentacleB is Tentacle 0x5f8100 (leader=true) and TentacleHead 0x5f83f0
// (VERIFIED). aip1 melee%, aip2 re-cast%, aip3 skill1 pause (s), aip4 skill2
// pause (s), aip5 stall, aip6 distance. Scratch[1] next cast frame,
// Scratch[2] phase (0 start, 1 skill1 done, 2 skill2 done). The Tentacle dies
// with its leader (the Baal throne) and the second skill targets itself.
func thinkTentacleB(c *Ctx, tentacle bool) {
	b, now := c.B, c.W.Frame()
	leaderMode := Mode(-1)

	if tentacle {
		l, ok := fbxLeader(c)
		if !ok {
			fbxKill(c, nil)

			return
		}

		leaderMode = l.Mode

		if l.Mode == ModeDead && b.Roll(100) < 40 {
			fbxKill(c, nil)

			return
		}
	}

	t := fbxTargetOrSelf(c)
	cast1 := func(wait int) {
		c.Cast(0, t)
		c.Sleep(wait)
		b.Scratch[1] = b.AIP(3)*25 + now
		b.Scratch[2] = 1
	}
	cast2 := func() {
		c.Cast(1, fbxSelf(b))
		b.Scratch[1] = b.AIP(4)*25 + now
		b.Scratch[2] = 2
	}

	if fbxSkill(b, 0) {
		if b.Scratch[2] == 0 {
			cast1(8)

			return
		}

		if b.Scratch[2] == 2 && b.Scratch[1] < now {
			again := false

			switch {
			case !tentacle:
				again = b.AIP(6) < c.Dist || (!c.InRange && b.Roll(100) < b.AIP(2))
			default:
				if b.AIP(6) < c.Dist {
					again = true
				}

				if !again && !c.InRange && b.Roll(100) < b.AIP(2) {
					again = true
				}

				if !again && leaderMode == ModeCast && b.Roll(100) < 50 {
					again = true
				}
			}

			if again {
				if tentacle {
					cast1(8)
				} else {
					cast1(0x14)
				}

				return
			}
		}
	}

	if fbxSkill(b, 1) && (b.Scratch[2] == 1 || tentacle) {
		if b.Scratch[2] == 1 && b.Scratch[1] < now {
			if c.InRange || c.Dist < b.AIP(6) {
				cast2()

				return
			}

			if !(tentacle && leaderMode == ModeCast) && tentacle && b.Roll(100) < 5 {
				cast2()

				return
			}
		}
	}

	if b.Scratch[2] == 1 {
		c.Sleep(b.AIP(5))

		return
	}

	if !tentacle {
		at, _, ok := c.W.AttackTarget(b)
		if b.Roll(100) >= b.AIP(1) || !ok {
			c.Sleep(b.AIP(5))

			return
		}

		c.Attack(ModeAttack1, at)

		return
	}

	if c.InRange && b.Roll(100) < b.AIP(1) {
		c.Attack(ModeAttack1, t)

		return
	}

	c.Sleep(b.AIP(5))
}

// FBXMonsterCell is the Frog Demon's burrow: FBXEnterBurrow is
// MONAI_EnterBurrowedState (unit flag |= 0xe, re-place the unit, 1x1 footprint
// 0x100, owner ref 1, block mask 0x3c01, sleep 12; returns false when it is
// already burrowed or cannot be placed), FBXExitBurrow the reverse (flag &=
// ^0xe, clear masks 0x1000/0x100, owner ref 5, block mask 0, sleep 12).
// Without the extension burrowing always succeeds and has no side effect.
type FBXBurrower interface {
	FBXEnterBurrow(b *Brain) bool
	FBXExitBurrow(b *Brain)
}

func fbxEnterBurrow(c *Ctx) bool {
	if w, ok := c.W.(FBXBurrower); ok {
		return w.FBXEnterBurrow(c.B)
	}

	return true
}

func fbxExitBurrow(c *Ctx) {
	if w, ok := c.W.(FBXBurrower); ok {
		w.FBXExitBurrow(c.B)
	}
}

// FBXCellFlags reads the collision flags at the unit's tile for a mask
// (COLLISION_GetCellFlagsAt). Fallback 0 (free).
type FBXCellFlags interface {
	FBXCellFlags(b *Brain, mask int) int
}

// thinkFrogDemonB is MONAI_Think_FrogDemon 0x5f73a0 (VERIFIED). Scratch[1]
// counts burrow cycles, Scratch[2] is the phase: 0 surface idle, 1 burrowed,
// 2 surfaced and fighting. Skill1 is the ranged spit, Skill2 the burrow
// attack (cast at itself). Think2 (0x5f7300) is thinkFrogDemonPhase2B.
func thinkFrogDemonB(c *Ctx) {
	b := c.B
	diff := c.Dist
	t := fbxTargetOrSelf(c)

	burrowCast := func() {
		c.Cast(1, fbxSelf(b))
		b.Scratch[2] = 2
	}

	switch b.Scratch[2] {
	case 0:
		if fbxSkill(b, 0) && diff > 12 {
			c.Cast(0, t)
			c.Sleep(8)

			b.Scratch[2] = 1

			fbxExitBurrow(c)

			return
		}

		if !fbxSkill(b, 1) || !fbxEnterBurrow(c) {
			if cf, ok := c.W.(FBXCellFlags); !ok || cf.FBXCellFlags(b, 0xc01) == 0 {
				b.Scratch[2] = 2
			}

			c.Sleep(12)

			return
		}

		burrowCast()
	case 1:
		if fbxSkill(b, 1) {
			if b.AIP(8) > diff && fbxEnterBurrow(c) {
				burrowCast()

				return
			}

			if diff <= 0x13 && b.Scratch[1] >= 0x41 && fbxEnterBurrow(c) {
				burrowCast()

				return
			}
		}

		fbxExitBurrow(c)
		c.Sleep(0x18)

		b.Scratch[1]++
		b.Scratch[2] = 1
	default:
		if c.Target == nil {
			c.Sleep(0x20)

			return
		}

		if !c.InRange {
			if b.AIP(6) <= diff {
				if b.Roll(100) >= b.AIP(4) {
					if !c.WalkTo(t, 4) {
						c.Sleep(10)
					}

					return
				}

				c.Circle(t, 3)

				return
			}

			if b.Roll(100) >= b.AIP(5) {
				if b.Roll(100) < b.AIP(4) {
					c.Circle(t, 4)

					return
				}

				c.Sleep(b.AIP(7))

				return
			}
		} else if b.Roll(100) >= b.AIP(2) {
			if b.Roll(100) < b.AIP(1) {
				c.Attack(ModeAttack1, t)

				return
			}

			if b.Roll(100) < b.AIP(3) {
				c.Circle(t, 3)

				return
			}

			c.Sleep(b.AIP(7))

			return
		}

		c.Attack(ModeAttack2, t)
	}
}

// thinkFrogDemonPhase2B is MONAI_Think2_FrogDemon 0x5f7300 (VERIFIED flow).
// When it runs is UNVERIFIED (the second think of the AI table row), so it is
// not wired into the registered AI. ai is the AiGeneral state id (+0), 10/11
// restore the state and re-think in a frame.
func thinkFrogDemonPhase2B(c *Ctx, aiState int) {
	b := c.B

	if aiState == 10 || aiState == 11 {
		c.Sleep(1)

		return
	}

	if fbxSkill(b, 0) && b.Scratch[2] > 1 {
		c.Cast(0, fbxTargetOrSelf(c))
		b.Scratch[2] = 1

		fbxExitBurrow(c)

		return
	}

	b.Scratch = [3]int{}
	c.Sleep(1)
}

// thinkSiegeTowerB is MONAI_Think_SiegeTower 0x5e06c0 (VERIFIED flow): unless
// a live owner commands it, it keeps its siege target current and always
// sleeps aip1.
func thinkSiegeTowerB(c *Ctx) {
	b := c.B

	if o, ok := fbxLeader(c); !ok || fbxDeadOwner(o) {
		if c.Target != nil {
			if r := fbxScan(c, FBXScanQuery{Kind: FBXScanSiegeTarget, Radius2: 400}); r.Found {
				fbxSiegeUpdate(c, r.T)
			}
		}
	}

	c.Sleep(b.AIP(1))
}

// fbxDeadOwner is UNIT_IsPlayerDeathMode: a dying/dead owner counts as gone.
func fbxDeadOwner(o OwnerInfo) bool { return o.Mode == ModeDying || o.Mode == ModeDead }

// FBXSiegeTargetStore is MONAI_SiegeUpdateTargetId: the siege weapon keeps the
// id of its current victim in its AI data and replaces it only when the new
// candidate is not farther. Fallback: Scratch is not touched.
type FBXSiegeTargetStore interface {
	FBXSiegeUpdate(b *Brain, t Target)
}

func fbxSiegeUpdate(c *Ctx, t Target) {
	if s, ok := c.W.(FBXSiegeTargetStore); ok {
		s.FBXSiegeUpdate(c.B, t)
	}
}

// thinkSiegeBeastB is MONAI_Think_SiegeBeast 0x5e0750 (VERIFIED flow). aip1
// scan radius, aip2 melee%, aip3 skill%, aip4 stall, aip5 ranged skill%,
// aip6 approach%, aip7 speed override.
func thinkSiegeBeastB(c *Ctx) {
	b := c.B
	if !b.Mode.IsAlive() {
		return
	}

	t := fbxTargetOrSelf(c)

	if _, hasOwner := fbxLeader(c); !hasOwner {
		if r := fbxScan(c, FBXScanQuery{Kind: FBXScanSiegeTarget, Radius2: b.AIP(1) * b.AIP(1)}); r.Found {
			fbxSiegeUpdate(c, r.T)
		}
	}

	if c.InRange {
		if fbxSkill(b, 0) && b.Roll(100) < b.AIP(3) {
			c.Cast(0, Target{})

			return
		}

		if b.Roll(100) < b.AIP(2) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(4))

		return
	}

	if fbxSkill(b, 0) && c.Dist < fbxSkillRange(c, 0) && b.Roll(100) < b.AIP(5) {
		c.Cast(0, Target{})

		return
	}

	if fbxReachable(c, t) && b.Roll(100) < b.AIP(6) {
		v := b.AIP(7)
		if v > 0x7e {
			v = 0x7f
		} else if v < 0 {
			v = 0
		}

		c.SetSpeed(v)
	}

	if !c.WalkToRange(t, 12, 0) {
		c.Sleep(10)
	}
}

// thinkReanimatedHordeB is MONAI_Think_ReanimatedHorde 0x5e03c0 (VERIFIED).
// Sets unit flags 0xe on every think. aip1 melee%, aip2 stall, aip3 skill2
// max distance, aip4 skill2%, aip5 walk%, aip6 hover%, aip7 stall.
func thinkReanimatedHordeB(c *Ctx) {
	b := c.B
	t := fbxTargetOrSelf(c)

	fbxFlags(c, 0xe, 0)

	if c.InRange {
		if b.Roll(100) < b.AIP(1) {
			c.Attack(ModeAttack1, t)

			return
		}

		c.Sleep(b.AIP(2))

		return
	}

	if fbxSkill(b, 1) && fbxReachable(c, t) && c.Dist < b.AIP(3) && c.Dist > 5 && b.Roll(100) < b.AIP(4) {
		c.Cast(1, t)

		return
	}

	if b.Roll(100) < b.AIP(5) {
		fbxWalk(c, t, 0)

		return
	}

	if b.Roll(100) < b.AIP(6) {
		if !c.WalkToRange(t, 4, 0) {
			c.Sleep(10)
		}

		return
	}

	c.Sleep(b.AIP(7))
}
