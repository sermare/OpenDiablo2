package d2monster

// Second half of the pet ports (see ai_fb_pet.go).

// fbnPetMoveToward is MONAI_PetMoveTowardTarget(leader, n) (VERIFIED draw
// order): one LCG step whose low bit decides which axis gets the full n (even:
// x gets the bounded roll, y gets n; odd: the reverse), one bounded roll(n)
// for the other axis, then one step per axis whose low bit negates that axis.
// The destination is around the monster's own position, walk mode, in spite
// of the name.
func fbnPetMoveToward(c *Ctx, n int) bool {
	b := c.B
	s := b.Seed.Step()
	r := int(b.Seed.Roll(int32(n)))

	dx, dy := r, n
	if s&1 != 0 {
		dx, dy = n, r
	}

	if b.Seed.Step()&1 != 0 {
		dx = -dx
	}

	if b.Seed.Step()&1 != 0 {
		dy = -dy
	}

	return c.move(Point{b.X + dx, b.Y + dy}, nil, 0, false)
}

// fbnEnemy is MONAI_FindNearestPlayerTarget[WithinDistance]: the nearest
// enemy of the pet, within radius when radius > 0. Friendly units (flag
// 0x40000000) are excluded by the host.
func fbnEnemy(c *Ctx, radius int) (Target, int, bool) {
	if w, ok := c.W.(interface {
		OwnerEnemy(b *Brain, radius int) (Target, int, bool)
	}); ok {
		r := radius
		if r <= 0 {
			r = 1 << 20
		}

		return w.OwnerEnemy(c.B, r)
	}

	t, d, ok := c.W.Nearest(c.B)
	if ok && radius > 0 && d > radius {
		return Target{}, 0, false
	}

	return t, d, ok
}

// fbnSelectPet is MONAI_SelectPetActionByRange: chooses the FollowLeaderStep
// mode and returns true when it moved the pet. p6 is the leash (7 or 8), flag
// the "stay put when adjacent" bit. The exe's path-history and region checks
// (previous path points, DRLG region ids) are not visible to the AI: they are
// UNVERIFIED and omitted, so mode 2 stays the default.
func fbnSelectPet(c *Ctx, target *Target, l OwnerInfo, flag bool, p6 int) bool {
	b := c.B
	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)

	maxDist := p6
	if pc, ok := c.W.(FBNPetCounter); ok {
		maxDist += pc.FBNPetCount(b) >> 1
	}

	if maxDist > 0x24 {
		maxDist = 0x24
	}

	if d < 2 && l.Mode == ModeNeutral && !flag {
		return fbnFollow(c, l, 5, 0, maxDist)
	}

	if target != nil && !fbnInTown(c) {
		if d < 0x51 {
			return false
		}

		return fbnFollow(c, l, 3, 0, maxDist)
	}

	mode := 2
	if l.Mode == ModeWalk || l.Mode == ModeGetHit || l.Mode == 3 {
		mode = 0
	}

	if maxDist < d {
		mode = 1
	}

	if d > 0x32 {
		mode = 3
	}

	return fbnFollow(c, l, mode, 0, maxDist)
}

// fbnPetRoll draws the leash roll: above 14 the pet keeps a leash of 7, else 8
// and the "stay put" flag.
func fbnPetRoll(c *Ctx) (flag bool, p6 int) {
	if c.B.Roll(100) > 14 {
		return false, 7
	}

	return true, 8
}

// fbnPetGone handles a missing leader: with a remembered leader id the pet
// follows it by teleport, otherwise it waits 10.
func fbnPetGone(c *Ctx) {
	if c.B.Scratch[2] != 0 && fbnFollow(c, OwnerInfo{}, 3, 0, 0) {
		return
	}

	c.Sleep(10)
}

// thinkNecroPetB is MONAI_Think_NecroPet 0x5e3c30: cancel the pending think
// event and dispatch on Scratch[0] (casters: skeleton mage, Valkyrie, golems
// with a skill) between the caster and melee thinkers.
func thinkNecroPetB(c *Ctx) {
	if c.B.Scratch[0] != 0 {
		fbnPetCaster(c)

		return
	}

	fbnPetMelee(c)
}

// fbnPetMelee is MONAI_Pet_ThinkMelee (VERIFIED flow).
func fbnPetMelee(c *Ctx) {
	b := c.B

	l, ok := fbnLeader(c)
	if !ok {
		fbnPetGone(c)

		return
	}

	b.Scratch[2] = int(l.ID)
	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)

	if d > 0x32 {
		fbnFollow(c, l, 3, 0, 0)

		return
	}

	boost := fbnRunSpeed(b.Profile)

	if d >= 0x1d {
		fbnFollow(c, l, 0, boost, 0)

		return
	}

	n1, nd, have1 := fbnEnemy(c, 0)
	e24, d24, have24 := fbnEnemy(c, 0x18)

	var tgt *Target

	inRange := false

	if have24 {
		inRange = c.W.InRange(b, e24, d24)
	}

	switch {
	case have24 && d24 <= 6:
		tgt = &e24
	case have1 && nd < 0x24:
		tgt = &n1
	}

	if tgt != nil && !fbnReachable(c, *tgt) {
		tgt = nil
	}

	flag, p6 := fbnPetRoll(c)

	if fbnSelectPet(c, tgt, l, flag, p6) {
		return
	}

	if tgt != nil && !fbnInTown(c) {
		if inRange {
			if b.Roll(100) > 0x4f {
				c.Sleep(10)

				return
			}

			c.Attack(ModeAttack1, *tgt)

			return
		}

		c.SetSpeed(0)

		if !c.WalkTo(*tgt, 7) {
			c.Sleep(10)
		}

		return
	}

	fbnPetMoveToward(c, 4)
}

// fbnPetCaster is MONAI_Pet_ThinkCaster (VERIFIED flow): casts Skill1 at its
// attack target.
func fbnPetCaster(c *Ctx) {
	b := c.B

	l, ok := fbnLeader(c)
	if !ok {
		fbnPetGone(c)

		return
	}

	b.Scratch[2] = int(l.ID)

	n1, _, have1 := fbnEnemy(c, 0)
	t, d, found := c.W.AttackTarget(b)

	var tgt *Target

	if found {
		tgt = &t
	}

	if !found || d > 15 {
		tgt = nil

		if have1 && !fbnBlocked(c, n1) && EdgeDistance(b.X-n1.X, b.Y-n1.Y, b.Size) < 0x14 {
			tgt = &n1
		}
	}

	flag, p6 := fbnPetRoll(c)

	if fbnSelectPet(c, tgt, l, flag, p6) {
		return
	}

	if tgt != nil && !fbnInTown(c) {
		if b.Roll(100) > 0x4f {
			if b.Roll(100) > 0x4a {
				c.Circle(*tgt, 3)

				return
			}

			c.Sleep(10)

			return
		}

		c.Cast(0, *tgt)

		return
	}

	fbnPetMoveToward(c, 4)
}

// thinkVinesB is MONAI_Think_Vines 0x5eb7c0 (VERIFIED flow). aip1 cast pause
// (frames, added to Scratch[1] = last cast frame), aip2 reach, aip3 idle
// sleep, aip4 retreat step, aip5 leash.
func thinkVinesB(c *Ctx) {
	b := c.B

	l, ok := fbnLeader(c)
	if !ok {
		c.Sleep(0x19)

		return
	}

	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)
	if d >= b.AIP(5) && fbnFollow(c, l, 3, 0, 6) {
		return
	}

	if fbnInTown(c) {
		if !fbnSelectPet(c, nil, l, false, 6) {
			c.Sleep(b.AIP(3))
		}

		return
	}

	t, dist, found := c.W.AttackTarget(b)

	var tgt *Target

	if found && dist < b.AIP(2) {
		tgt = &t
	}

	if fbnSelectPet(c, tgt, l, false, 6) {
		return
	}

	if tgt != nil {
		avoid := false

		if s, ok := c.W.(FBNTargetStates); ok {
			avoid = s.FBNTargetHasState(b, *tgt, 2) || s.FBNTargetStat(b, *tgt, 0x2d) == 100
		}

		if avoid {
			if !c.WalkAway(*tgt, b.AIP(4)) {
				c.Sleep(10)
			}

			return
		}

		if !c.W.InRange(b, *tgt, dist) {
			if !c.WalkTo(*tgt, 7) {
				c.Sleep(10)
			}

			return
		}

		if fbnSkill(b, 0) && b.AIP(1)+b.Scratch[1] < c.W.Frame() {
			c.Cast(0, *tgt)
			b.Scratch[1] = c.W.Frame()

			return
		}
	}

	c.Sleep(b.AIP(3))
}

// thinkTotemB is MONAI_Think_Totem 0x5ecb10 (VERIFIED flow). aip1 flee%, aip2
// forget-target%, aip3 recall distance, aip4 follow distance.
func thinkTotemB(c *Ctx) {
	b := c.B

	l, ok := fbnLeader(c)
	if !ok {
		c.Sleep(10)

		return
	}

	tgt, tdist, have := fbnEnemy(c, 0x18)
	inRange := false

	if have {
		inRange = c.W.InRange(b, tgt, tdist)
	}

	if inRange && b.Roll(100) < b.AIP(1) && c.WalkAway(tgt, 6) {
		return
	}

	if b.Roll(100) < b.AIP(2) {
		have = false
	}

	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)

	if b.AIP(3) < d {
		if r, ok := c.W.(FBNLeaderTeleporter); ok && r.FBNTeleportToLeader(b) {
			c.Sleep(0x19)

			return
		}
	}

	if b.AIP(4) < d {
		if (l.Mode == ModeWalk || l.Mode == ModeRun) && fbnFollow(c, l, 0, 0, 0) {
			return
		}

		if l.Mode == 3 && fbnFollow(c, l, 0, 0x3c, 0) {
			return
		}
	}

	var tp *Target
	if have {
		tp = &tgt
	}

	if fbnSelectPet(c, tp, l, false, 6) {
		return
	}

	c.Sleep(0x19)
}

// thinkRavenB is MONAI_Think_Raven 0x5ebd20 (VERIFIED flow). aip1/aip2 the
// orbit band around the Druid, aip3 pause between pecks (x10 frames), aip4
// peck %, aip5 peck distance. Scratch[0] pecks left (-1 = not yet counted;
// the zero state of a fresh brain with no peck yet is treated the same),
// Scratch[1] next peck frame, Scratch[2] orbit direction.
func thinkRavenB(c *Ctx) {
	b, now := c.B, c.W.Frame()

	l, ok := fbnLeader(c)
	if !ok {
		c.Sleep(10)

		return
	}

	switch {
	case b.Scratch[0] == -1 || (b.Scratch[0] == 0 && b.Scratch[1] == 0):
		b.Scratch[0] = 3

		if fbnSkill(b, 0) {
			if s, ok := c.W.(FBNSkillValue); ok {
				b.Scratch[0] = s.FBNRavenPecks(b)
			}
		}
	case b.Scratch[0] == 0:
		c.Attack(ModeDying, Target{ID: b.ID, X: b.X, Y: b.Y})

		return
	}

	d := EdgeDistance(b.X-l.X, b.Y-l.Y, b.Size)
	if d > 0x32 {
		fbnFollow(c, l, 3, 0, 0)

		return
	}

	if d > 0x1c {
		fbnFollow(c, l, 0, fbnRunSpeed(b.Profile), 0)

		return
	}

	if c.Target != nil && b.Scratch[1] < now && b.Roll(100) < b.AIP(4) && c.Dist < b.AIP(5) {
		if c.InRange {
			c.Attack(ModeAttack1, *c.Target)

			b.Scratch[0]--
			b.Scratch[1] = now + b.AIP(3)*10

			return
		}

		if !c.WalkTo(*c.Target, 0) {
			c.Sleep(10)
		}

		return
	}

	if d <= b.AIP(1) && d >= b.AIP(2) {
		c.SetSpeed(0)

		if c.WalkTo(l.Target, 4) {
			return
		}

		b.Scratch[2] = 1 - b.Scratch[2]

		c.SetSpeed(0)

		if c.WalkTo(l.Target, 4) {
			return
		}
	}

	// MONAI_RavenMoveAroundTarget: the point (aip1+aip2)/2 from the Druid on
	// the line towards the raven
	step := (b.AIP(1) + b.AIP(2)) / 2
	if d != 0 && c.move(Point{l.X + (b.X-l.X)*step/d, l.Y + (b.Y-l.Y)*step/d}, nil, 0, false) {
		return
	}

	fbnFollow(c, l, 1, 0, 0)
}

// fbnRunSpeed is the pet speed boost: run*100/walk - 100 (record shorts
// +0x34/+0x32, UNVERIFIED columns), 100 when walk < 1 or the value exceeds 99.
func fbnRunSpeed(p *Profile) int {
	if p.Walk < 1 {
		return 100
	}

	v := p.Run*100/p.Walk - 100
	if v > 99 {
		return 100
	}

	return v
}

func fbnSkill(b *Brain, i int) bool { return b.Profile.Skills[i].Used() }

func fbnReachable(c *Ctx, t Target) bool {
	if r, ok := c.W.(TargetReachability); ok {
		return r.Reachable(c.B, t)
	}

	return true
}

// fbnBlocked is COLLISION_CanTraceLineBetweenUnits(self, t, 4) == 0
// (MONAI_IsNotImmune): true when the line is blocked. Optional host
// extension, fallback false.
func fbnBlocked(c *Ctx, t Target) bool {
	if r, ok := c.W.(interface {
		FBNBlocked(b *Brain, t Target, mask int) bool
	}); ok {
		return r.FBNBlocked(c.B, t, 4)
	}

	return false
}
