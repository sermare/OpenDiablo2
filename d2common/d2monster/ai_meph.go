package d2monster

// Mephisto's phase bodies: the five targets of the jump table at 0x5f7000
// (0x5f6c10 idle, 0x5f6c1c back-off, 0x5f6c83 burst, 0x5f6edc melee, 0x5f6f7d
// approach), read from the disassembly (Ghidra has no function for them).
// Every body stores its "next phase" in AiGeneral +0x1c (Scratch[2]).
//
// Slots are monstats Skill1..Skill6 at +0x170 + 2*(n-1) with the mode byte at
// +0x180 + (n-1): 1 PrimeLightning, 2 PrimeBolt, 3 PrimePoisonNova, 4
// MephistoMissile, 5 MephFrostNova, 6 Blizzard.

// MephScanner is MONAI_FindBestUnitByModeScan 0x5f6850: a scan of the units
// near Mephisto (squared distance <= 0x400, alive, flag 4 set in the unit
// flags, hostile by 0x552270) that reports the one with the lowest current
// life and, when Mephisto's own unit is a monster, the nearest one. UNVERIFIED
// which units those are in practice (the unit-mode list it walks was not
// identified); absent, the burst never retargets and draws no extra roll.
type MephScanner interface {
	MephScan(b *Brain) (weakest *Target, nearest *Target)
}

// mephRetarget is 0x5f6850 given the burst's current target: when the scan
// found a weakest unit, a roll below aip2 (monstats +0x5c) returns it; then,
// when it found a nearest unit, a roll below aip3 (+0x62) returns "the
// weakest" again (the decompile returns the same variable both times, which
// looks like a slip of the original but is kept); otherwise the target stays.
func mephRetarget(c *Ctx, cur Target) Target {
	s, ok := c.W.(MephScanner)
	if !ok {
		return cur
	}

	weak, near := s.MephScan(c.B)
	b := c.B

	if weak != nil && b.Roll(100) < b.AIP(2) {
		return *weak
	}

	if near != nil && b.Roll(100) < b.AIP(3) {
		if weak != nil {
			return *weak
		}

		return cur
	}

	return cur
}

// mephRunPhase runs the body of the selected phase.
func mephRunPhase(c *Ctx, t Target, phase, k int) {
	b := c.B

	switch phase {
	case mephIdle: // 0x5f6c10: store phase 4, wait 5
		b.Scratch[2] = mephApproach
		c.Sleep(5)
	case mephBackOff: // 0x5f6c1c
		b.Scratch[2] = mephIdle
		c.SetSpeed(50)

		if c.WalkAway(t, 8) {
			return
		}

		if c.InRange {
			mephCast(c, slot3, t)

			return
		}

		// out of reach and unable to retreat: one burst step with next = idle
		mephBurstStep(c, t, k, mephIdle)
	case mephBurst: // 0x5f6c83
		mephBurstStep(c, t, k, mephBurst)
	case mephMelee: // 0x5f6edc
		b.Scratch[2] = mephIdle

		if b.Roll(100) >= 80+k {
			// retreat: strafe (CircleOrStrafe with speed 3 and the extra flag),
			// and wander 12 when that was not queued
			c.SetSpeed(50)

			if !c.Circle(t, 3) {
				c.Wander(12)
			}

			return
		}

		if b.Roll(100) < 80-k {
			c.Attack(ModeAttack1, t)

			return
		}

		mephCast(c, slot3, t)
	case mephApproach: // 0x5f6f7d
		b.Scratch[2] = mephIdle
		c.SetSpeed(50)

		if c.WalkNearTarget(&t, 6) {
			return
		}

		// (the exe cancels its pending events of type 2 here: no Go equivalent)
		if !c.WalkToRange(t, 12, 6) {
			c.Wander(12)
		}
	}
}

// mephCast casts a monstats slot at the target. The exe does not test that the
// slot is defined; an undefined one is a no-op here.
func mephCast(c *Ctx, slot int, t Target) {
	if !c.B.Profile.Skills[slot].Used() {
		c.Sleep(10)

		return
	}

	c.Cast(slot, t)
}

// mephBurstStep is the burst body 0x5f6c83. next is the phase stored at the
// end (the burst keeps its phase until the counter in Scratch[0] hits exactly
// zero; the counter is decremented first, and a counter that was already 0 goes
// negative and never ends the burst on its own).
func mephBurstStep(c *Ctx, t Target, k, next int) {
	b := c.B
	p := b.Profile

	b.Scratch[0]--
	if b.Scratch[0] == 0 {
		next = mephIdle
	}

	b.Scratch[2] = next

	if b.Scratch[1] == 0 {
		// first step: remember it (2) and strafe; a strafe that could not be
		// queued turns into a short wander, and either way the tick ends
		b.Scratch[1] = 2

		if !c.Circle(t, 3) {
			c.Wander(12)
		}

		return
	}

	b.Scratch[1]++

	t = mephRetarget(c, t)

	used := 0

	for i := 0; i < 8; i++ {
		if !p.Skills[i].Used() {
			break
		}

		used++
	}

	if used <= 0 {
		c.Wander(6)
	} else {
		share := 100 / used
		r := b.Roll(100)
		hard := b.Diff > Normal

		switch {
		case hard && (c.lineBlocked(t, 4) || c.Dist > 30):
			mephCast(c, slot6, t) // Blizzard
		case hard && c.Dist < 15 && r < share:
			mephCast(c, slot5, t) // Frost Nova
		case r < 2*share:
			mephCast(c, slot4, t)
		case r < 3*share:
			mephCast(c, slot2, t)
		default:
			mephCast(c, slot1, t)
		}
	}

	// after any action: a roll under 50-k forgets the first-step flag, so the
	// next step strafes again
	if b.Roll(100) < 50-k {
		b.Scratch[1] = 0
	}
}
