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

// Create flags of MISSILE_CreateServerMissile (0x59d5d0) that affect timing.
const (
	CreateExplicitVel uint32 = 0x4    // velocity given in the create struct (+0x28)
	CreateSubLoopLife uint32 = 0x8    // add the sub-loop length to the lifetime
	CreateVelFixed    uint32 = 0x10   // the given velocity is already 8.8 (no shift)
	CreateLifeOffset  uint32 = 0x200  // subtract Param40 from the lifetime
	CreateLob         uint32 = 0x400  // lifetime from distance / velocity
	CreateLifeAdjust  uint32 = 0x800  // second reduction is Param44 instead of record byte +0x136
	CreateRangeGiven  uint32 = 0x8000 // lifetime given in the create struct (+0x4c)
)

// MissileCreateIn carries the numbers MISSILE_CreateServerMissile uses to set a
// server missile's speed and lifetime. Record fields are missiles.txt values
// (Vel, VelLev, Range, LevRange, MaxVel, Accel, SubLoop, SubStart, SubStop and
// the record byte +0x136).
type MissileCreateIn struct {
	Flags                uint32
	Level                int
	VelOverride          int // create struct +0x28
	RangeOverride        int // +0x4c
	SubLoops             int // +0x34
	Param40, Param44     int
	Vel, VelLev          uint8
	Range, LevRange      int16
	MaxVel               uint8
	Accel                int16
	SubLoop              bool // record byte +0x180 non-zero
	SubStart, SubStop    uint8
	Byte136              uint8
	CanSlow, OwnerSlowed bool
	SlowPct              int // owner stat 0xa1 (used when slowed)
	LobDist              int // distance helper 0x642c30 result
}

// MissileCreateOut is what the exe passes to its setters.
type MissileCreateOut struct {
	Velocity int // path velocity set by 0x649a50, 8.8 after the 75% step (0 when none)
	Life     int // 0x64b580 and the first 0x64b5e0 argument
	LobLife  int // second 0x64b5e0 argument when CreateLob is set
	Life3    int // 0x64b850 argument
	Accel    int // 0x649ac0
	MaxVel   int // 0x649a90 (record byte << 8)
	Offset   int // stat 0x44 value (Param40<<8) when CreateLifeOffset is set
}

// MissileCreateTiming computes the verified speed and lifetime of a created
// server missile (golden dm_mis). Order in the exe: velocity (explicit or
// (VelLev*level)/8+Vel, <<8; the slow stat scales it when the record can slow
// and the owner is slowed; then x75/100 if non-zero), lifetime (given, or
// LevRange*level+Range (+(SubStop-SubStart)*SubLoops with flag 8 and a
// sub-loop), minus Param40 with flag 0x200), then the reduction by
// Param44 (flag 0x800) or the record byte +0x136.
func MissileCreateTiming(in MissileCreateIn) MissileCreateOut {
	var o MissileCreateOut

	var v int32

	if in.Flags&CreateExplicitVel != 0 {
		v = int32(in.VelOverride)
		if in.Flags&CreateVelFixed == 0 {
			v <<= 8
		}
	} else {
		v = (int32(in.VelLev)*int32(in.Level))/8 + int32(in.Vel)
		v <<= 8
	}

	if in.CanSlow && in.OwnerSlowed {
		v = (v * int32(in.SlowPct)) / 100
	}

	if v != 0 {
		v = (v * 75) / 100
	}

	o.Velocity = int(v)

	var life int32

	switch {
	case in.Flags&CreateRangeGiven != 0:
		life = int32(in.RangeOverride)
	default:
		life = int32(in.LevRange)*int32(in.Level) + int32(in.Range)
		if in.Flags&CreateSubLoopLife != 0 && in.SubLoop {
			life += (int32(in.SubStop) - int32(in.SubStart)) * int32(in.SubLoops)
		}
	}

	if in.Flags&CreateLifeOffset != 0 {
		life -= int32(in.Param40)
		o.Offset = in.Param40 << 8
	}

	o.Life = int(life)
	o.Life3 = int(life)

	if in.Flags&CreateLifeAdjust != 0 {
		o.Life3 = int(life) - in.Param44
	} else {
		o.Life3 = int(life) - int(in.Byte136)
	}

	if in.Flags&CreateLob != 0 {
		d := uint32(in.LobDist)
		if d == 0 {
			d = 1
		}

		if den := uint32(v) << 4; den != 0 {
			o.LobLife = int((d << 16) / den)
		}
	}

	o.Accel = int(in.Accel)
	o.MaxVel = int(in.MaxVel) << 8

	return o
}
