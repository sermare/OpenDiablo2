package d2monster

// Batch 4 of faithful ports (notes: d2-re-notes/ai-faithful-batch4.md):
// AssassinSentry 0x5e9480 (pre 0x5e9340, helper 0x5e9360 = sentryFollow),
// CycleOfLife 0x5eb9c0 (pre 0x5eb7b0) and JarJar 0x5e64b0. These were the
// last monai entries still on the unverified stand-ins (ai_pet.go thinkSentry
// and thinkStationary, ai_rest.go thinkInert); the rest of the table already
// had ports in ai_faithful_a.go / ai_fb_*.go / ai_faithful_3.go.
//
// Scratch[0..2] are AiGeneral +0x14/+0x18/+0x1c. Everything the engine owns
// (quest node, skill calc, unit stats) is an optional host interface with a
// documented fallback.

// ArcaneNpcQuest is the Act 2 Arcane Sanctuary quest node (id 0xb) that JarJar
// reads: QUEST_A2_ArcaneSanctuary_IsNpcIdleState2 0x599500, _IsNpcStateTwo
// 0x598b20 and _GetNpcTargetCoords 0x599540. Without it the quest node is
// treated as absent: idle2 true, stateTwo false, no coordinates.
type ArcaneNpcQuest interface {
	// ArcaneNpcState: idle2 is true when the node is absent, or when neither
	// of its two flag bytes is set and its state word is 2; stateTwo is true
	// when the node exists and its state word is 2; coords is the quest's
	// walk point (flag byte A: the stored point, flag byte B: the stored point
	// with y reduced by 4), has false when neither flag is set.
	ArcaneNpcState(b *Brain) (idle2, stateTwo bool, coords Point, has bool)
}

// LeaderVitals gives the current and maximum life (which 0) or mana (which 1)
// of the pack leader, for CycleOfLife's "does the owner need it" gate.
// Absent means the owner always needs it.
type LeaderVitals interface {
	LeaderVitals(b *Brain, which int) (cur, max int)
}

// AuraRanger evaluates the aura range calc (skills.txt aurarangecalc) of a
// monstats skill slot at a level (SKILL_EvalCalcBytecode).
type AuraRanger interface {
	AuraRange(b *Brain, slot, level int) int
}

// DeathModeChecker is UNIT_IsPlayerDeathMode for a target.
type DeathModeChecker interface {
	TargetInDeathMode(b *Brain, t Target) bool
}

const (
	cycleClassLife   = 0x1aa // gate: leader life below max (stat 6 vs max hp)
	cycleClassMana   = 0x1ab // gate: leader mana below max (stat 8 vs max mana)
	cycleAuraMin     = 5
	cycleAuraMax     = 0x32
	cycleWalkAwayPct = 0x19
	cycleLeash       = 6
	cycleNoLeader    = 0x19 // not in the exe (it returns); see the notes
	jarJarWaitIdle   = 0x14
	jarJarWaitLong   = 0x78
	jarJarWaitNear   = 200
	jarJarSpacing    = 200 // frames between anchor walks
	jarJarNearDist   = 12
	jarJarRollMax    = 1000
	jarJarGreetBelow = 100
	jarJarWanderBel  = 50
	jarJarYOffset    = 3
	sentryStateBusy  = 0xc
)

func init() {
	fbnReg("AssassinSentry", thinkAssassinSentry)
	fbnReg("CycleOfLife", thinkCycleOfLife)
	fbnReg("JarJar", thinkJarJar)
}

// FaithfulBatch4 lists the names ported in this file.
func FaithfulBatch4() []string {
	return []string{"AssassinSentry", "CycleOfLife", "JarJar"}
}

// thinkAssassinSentry is MONAI_Think_AssassinSentry (VERIFIED). Pre 0x5e9340:
// S0 = frame, S1 = -1 (S1 is the life counter of sentryFollow). aip1 fire
// percent, aip2 wait after a missed roll, aip3 wait with no target, aip4 range.
func thinkAssassinSentry(c *Ctx) {
	b := c.B
	s := &b.Scratch

	c.pre(func() { s[0], s[1] = c.frame(), -1 })

	if sentryFollow(c, false) {
		return
	}

	if !c.skill(slotA) {
		c.setMode(ModeDying)

		return
	}

	if c.W.HasState(b, sentryStateBusy) {
		c.setState(sentryStateBusy, false)
	}

	t, d, ok := c.W.AttackTarget(b)
	if !ok || b.AIP(4) <= d {
		c.Sleep(b.AIP(3))

		return
	}

	if b.Roll(100) >= b.AIP(1) {
		c.Sleep(b.AIP(2))

		return
	}

	if sentryFollow(c, true) {
		return
	}

	// The exe cancels pending events of type 2 here (EVENT_CancelByTypeAndData
	// (2,0)); the Go brain has no event list.
	c.Cast(slotA, t)
}

// thinkCycleOfLife is MONAI_Think_CycleOfLife (VERIFIED flow; pre 0x5eb7b0 is
// S0 = 0). S1 is the frame of the last cast. A summoned healer that stays near
// its owner: aip1 cast gap, aip2 reach to an ally, aip3 idle sleep, aip4
// walk-away step, aip5 leash before it follows the owner.
func thinkCycleOfLife(c *Ctx) {
	b := c.B
	s := &b.Scratch

	c.pre(func() { s[0] = 0 })

	l, ok := fbnLeader(c)
	if !ok {
		// The exe simply returns here; the port waits so the tick does not spin.
		c.Sleep(cycleNoLeader)

		return
	}

	target := c.Target
	inRange := c.InRange

	if EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size) >= b.AIP(5) && fbnFollow(c, l, 3, 0, cycleLeash) {
		return
	}

	if target != nil {
		if d, ok := c.W.(DeathModeChecker); ok && d.TargetInDeathMode(b, *target) {
			target, inRange = nil, false
		}
	}

	var (
		cand  *Target
		cdist int
	)

	if b.Profile.Skills[slotA].Used() {
		lv := c.skillLevel(slotA)
		if lv > 0 {
			r := cycleAuraMin

			if a, ok := c.W.(AuraRanger); ok {
				r = a.AuraRange(b, slotA, lv)
			}

			if r < cycleAuraMin+1 {
				r = cycleAuraMin
			} else if r > cycleAuraMax-1 {
				r = cycleAuraMax
			}

			if f, ok := c.W.(SkillTargetSearcher); ok {
				if u, found := f.NearbySkillTarget(b, r); found {
					cand = &u
					cdist = EdgeDistance(b.X-u.X, b.Y-u.Y, b.Size)
				}
			}
		}
	}

	melee := false

	if cdist < b.AIP(2) {
		if cand != nil {
			melee = c.W.InRange(b, *cand, cdist)
		}
	} else {
		cand = nil
	}

	if fbnSelectPet(c, cand, l, false, cycleLeash) {
		return
	}

	needs := true

	if v, ok := c.W.(LeaderVitals); ok {
		switch b.Class {
		case cycleClassLife:
			cur, max := v.LeaderVitals(b, 0)
			needs = cur < max
		case cycleClassMana:
			cur, max := v.LeaderVitals(b, 1)
			needs = cur < max
		}
	}

	if cand != nil && melee && needs && b.AIP(1)+s[1] < c.frame() {
		c.Cast(slotA, *cand)
		s[1] = c.frame()

		return
	}

	if inRange && target != nil && b.Roll(100) < cycleWalkAwayPct {
		c.WalkAway(*target, b.AIP(4))

		return
	}

	if cand == nil {
		c.Sleep(b.AIP(3))

		return
	}

	c.WalkTo(*cand, 7)
}

// thinkJarJar is MONAI_Think_JarJar (VERIFIED flow). The state lives in the
// anchor command node (type 10): X/Y the home point, Delay (+0x18) the frame
// of the last anchor walk. While the Arcane Sanctuary quest node is "active"
// (not idle2) the NPC walks to the quest point and waits there; otherwise it
// stays near home (three tiles above the anchor) and sometimes greets or
// shuffles when a player is near.
func thinkJarJar(c *Ctx) {
	b := c.B

	if fbnStartAnchorWalk(c) {
		return
	}

	node := b.FindCommand(CmdAnchor)

	idle2, stateTwo, coords, has := true, false, Point{}, false
	if q, ok := c.W.(ArcaneNpcQuest); ok {
		idle2, stateTwo, coords, has = q.ArcaneNpcState(b)
	}

	if !idle2 {
		jarJarQuestActive(c, node, coords, has)

		return
	}

	if node == nil {
		c.Sleep(jarJarWaitLong)

		return
	}

	h, hasHost := fbnHost(c)

	var (
		p     Target
		found bool
	)

	if hasHost {
		p, _, found = h.NearestPlayerNearby(b)
	}

	d := 0
	if found {
		d = EdgeDistance(b.X-p.X, b.Y-p.Y, b.Size)
	}

	ty := node.Y - jarJarYOffset
	tiles := unitTileDist(b, node.X, ty)

	if (stateTwo && tiles > 2) || tiles > 7 {
		fbnWalkPoint(c, Point{node.X, ty})

		return
	}

	if fbnReact(c) {
		c.Sleep(jarJarWaitLong)

		return
	}

	if found && p.ID != b.ID {
		if p.IsPlayer {
			if d < jarJarNearDist {
				if b.Roll(jarJarRollMax) < jarJarGreetBelow {
					c.WalkNearTarget(&p, 2)
				}

				c.Sleep(jarJarWaitNear)

				return
			}

			c.Sleep(jarJarWaitIdle)
		}

		if b.Roll(jarJarRollMax) < jarJarWanderBel {
			c.Wander(3)
		}
	}

	c.Sleep(jarJarWaitLong)
}

// jarJarQuestActive is the branch where the quest node is not idle2. The walk
// target is the anchor, or the quest point when the node has one; the quest
// point disables the player reaction.
func jarJarQuestActive(c *Ctx, node *Command, coords Point, has bool) {
	b := c.B
	frame := c.frame()

	tx, ty := node.X, node.Y
	react := true

	if has {
		tx, ty = coords.X, coords.Y
		react = false
	}

	// A walk is only (re)issued once the last one is 200 frames old.
	if node.Delay < frame && frame-node.Delay < jarJarSpacing {
		c.Sleep(jarJarWaitIdle)

		return
	}

	if unitTileDist(b, tx, ty) > 1 {
		fbnWalkPoint(c, Point{tx, ty})
		node.Delay = frame

		return
	}

	if !react {
		c.Sleep(jarJarWaitIdle)

		return
	}

	if fbnReact(c) {
		node.Delay = frame

		return
	}

	c.Sleep(jarJarWaitIdle)
}

// unitTileDist is UNIT_IsWithinTileDistance 0x5db310 (VERIFIED arithmetic):
// with dx, dy the absolute coordinate differences, the larger plus half the
// smaller. The coordinates are the path's own units, which the Go brain
// shares (subtiles); the function's name suggests tiles but the exe does no
// division by 5 (UNVERIFIED whether the path position is in subtiles).
func unitTileDist(b *Brain, x, y int) int {
	dx, dy := b.X-x, b.Y-y
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dy < dx {
		return (dy + dx*2) / 2
	}

	return (dx + dy*2) / 2
}
