package d2combat

// MissileVelocity returns a missile's velocity in 8.8 fixed point. Verified in
// MISSILE_CreateServerMissile (0x59d5d0):
//
//	vel = ((velLev*level)/8 + vel) << 8   (division truncates toward zero)
//
// If the missile has the CanSlow flag and the owner is slowed (state 0x57),
// pass slowed=true and slowPct (stat 0xa1): vel = vel*slowPct/100.
func MissileVelocity(vel, velLev uint8, level int, slowed bool, slowPct int) int {
	v := ((int(velLev)*level)/8 + int(vel)) << 8

	if slowed {
		v = v * slowPct / 100
	}

	return v
}

// MissileStep returns the per-frame movement for a velocity: velocity*75/100,
// or 0 for zero velocity. Verified.
func MissileStep(velocity int) int {
	if velocity == 0 {
		return 0
	}

	return velocity * 75 / 100
}

// MissileRange returns a missile's lifetime in frames: LevRange*level + Range
// (both int16). Verified. When extendSubLoop (create flag 8) and the missile
// has SubLoop set, add (SubStop-SubStart)*loops.
func MissileRange(rng, levRange int16, level int, subLoop bool, subStart, subStop uint8, loops int, extendSubLoop bool) int {
	r := int(levRange)*level + int(rng)

	if extendSubLoop && subLoop {
		r += (int(subStop) - int(subStart)) * loops
	}

	return r
}
