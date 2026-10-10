package d2path

// Movement speeds, ported from missiles-pathing.md section (c)
// (PATH_AdvanceUnitOneFrame / PATH_ApplyAccelAndScaleStep / PATH_SetVelocity /
// PATH_UpdateUnitVelocityAndAnimRate). Re-checked against Game.exe:
// 0x651750 (step and acceleration) and 0x624150 (velocity from stats) are
// VERIFIED below; what is still open is marked UNVERIFIED.

// Class ids in charstats.txt order.
const (
	ClassAmazon = iota
	ClassSorceress
	ClassNecromancer
	ClassPaladin
	ClassBarbarian
	ClassDruid
	ClassAssassin
)

// MoveMode is how a unit travels along its path.
type MoveMode int

// Movement modes.
const (
	ModeWalk MoveMode = iota
	ModeRun
	// ModeTeleport jumps straight to the destination (path type 9: one node).
	// The range limit is a skill property (UNVERIFIED, not modelled).
	ModeTeleport
	// ModeLeap is the leap/charge mode (player mode 0x13, monster mode 0xd)
	// whose velocity is fixed at 0x1000 (VERIFIED).
	ModeLeap
)

const (
	// TickRate is the logic rate (VERIFIED, 25 Hz).
	TickRate = 25
	// StepScale is the scale PATH_AdvanceUnitOneFrame passes (0x400).
	StepScale = 0x400
	// LeapVelocity is the fixed 8.8 leap/charge velocity (VERIFIED).
	LeapVelocity = 0x1000
	// MinSpeedPercent is the floor of the movement percentage (VERIFIED).
	MinSpeedPercent = 25
	// AccelEvery: acceleration is applied every 5th frame (VERIFIED).
	AccelEvery = 5
)

// BaseVelocity is the table velocity of a unit, in the units of
// charstats WalkVelocity/RunVelocity and monstats Velocity/Run.
type BaseVelocity struct{ Walk, Run int }

// PlayerBase returns charstats.txt WalkVelocity/RunVelocity. All seven
// shipped classes use 6/9 (checked against CharStats.txt). Re-check of
// 624150 (via 621550): for players it reads only the byte at charstats
// record +0x40 (stride 0xc4), shifted left 8, for both the walk (2) and run
// (3) modes (FUN_00621690 accepts both); a separate RunVelocity offset was not
// found there, so which of the two columns +0x40 holds and where the other
// is applied stay UNVERIFIED. Monsters read the short at monstats +0x32
// (stride 0x1a8). Callers with real data should prefer the table values.
func PlayerBase(class int) BaseVelocity {
	if class < ClassAmazon || class > ClassAssassin {
		return BaseVelocity{}
	}

	return BaseVelocity{Walk: 6, Run: 9}
}

// Base picks the table velocity for a mode (teleport and leap have none).
func (b BaseVelocity) Base(m MoveMode) int {
	switch m {
	case ModeWalk:
		return b.Walk
	case ModeRun:
		return b.Run
	}

	return 0
}

// FRWDiminishK is the constant of the diminishing-returns row for faster
// run/walk (stat 0x60) in the stat table at 6ea3d4 (entry 4: flag 1, k=150,
// stat 96; VERIFIED by reading the table).
const FRWDiminishK = 150

// Diminish is the table's diminishing-returns formula k*v/(k+v) (621930), in
// C integer division; zero and sums that would divide by zero return v.
func Diminish(k, v int) int {
	if v == 0 || k+v == 0 {
		return v
	}

	return k * v / (k + v)
}

// SpeedPercent is the movement percentage of PATH_UpdateUnitVelocityAndAnimRate
// for walk and run (VERIFIED, 624150): max(25, stat 0x43 + Diminish(150, FRW
// stat 0x60)). stat43 is the raw movement-velocity stat, which therefore must
// carry the 100% base itself (no +100 appears in the formula, so a unit with
// no bonuses has 100; cold slow lowers it). That the base is 100 is inferred
// (the stat's initialiser was not located); frw is the item FRW stat total.
func SpeedPercent(stat43, frw int) int {
	if p := stat43 + Diminish(FRWDiminishK, frw); p > MinSpeedPercent {
		return p
	}

	return MinSpeedPercent
}

// Velocity returns the 8.8 path velocity of a unit: base<<8 scaled by the
// percentage. VERIFIED (624150): the base is shifted first (621550 returns
// base<<8) and then multiplied by the percentage and divided by 100
// (truncating), before PATH_SetVelocity.
// Leap ignores table velocity and percentage; teleport has no velocity.
func Velocity(b BaseVelocity, m MoveMode, percent int) int {
	switch m {
	case ModeLeap:
		return LeapVelocity
	case ModeTeleport:
		return 0
	}

	return (b.Base(m) << 8) * percent / 100
}

// Step is one tick of displacement in 16.16 fixed point for an 8.8 velocity
// and a 4.12 direction component (VERIFIED, 651750: the product vel*scale is
// shifted right 6 first, then multiplied by dir and shifted right 12; a scale
// below 1 is replaced by 0x400).
func Step(vel, scale, dir int) int {
	if scale < 1 {
		scale = StepScale
	}

	return ((vel * scale) >> 6) * dir >> 12
}

// SubtilesPerTick is the straight-line distance covered per 25 Hz tick for an
// 8.8 velocity (V<<8 gives V/16 subtile, derived in the notes).
func SubtilesPerTick(vel int) float64 {
	return float64(Step(vel, StepScale, 1<<12)) / 65536
}

// SubtilesPerSecond is SubtilesPerTick at 25 Hz: walk 6 gives 9.375, run 9
// gives 14.0625.
func SubtilesPerSecond(vel int) float64 { return SubtilesPerTick(vel) * TickRate }

// Accelerate applies path acceleration exactly as PATH_ApplyAccelAndScaleStep
// does (VERIFIED, 651750): nothing happens while accel (+0x88) is zero;
// otherwise the counter (+0x8c) is incremented and, once it exceeds 4 (every
// 5th frame), vel (+0x7c) += accel. If vel then exceeds maxVel (+0x84) it is
// capped and accel is zeroed; otherwise a negative vel is clamped to 0 (accel
// kept, so deceleration stops at rest). The counter resets only in that
// branch. It returns the updated vel, accel and counter.
func Accelerate(vel, maxVel, accel, counter int) (newVel, newAccel, newCounter int) {
	if accel == 0 {
		return vel, accel, counter
	}

	counter++
	if counter <= AccelEvery-1 {
		return vel, accel, counter
	}

	vel += accel

	switch {
	case vel > maxVel:
		vel, accel = maxVel, 0
	case vel < 0:
		vel = 0
	}

	return vel, accel, 0
}

// Advance moves a 16.16 position one tick along a 4.12 direction vector.
func Advance(x, y, vel, dirX, dirY int) (nx, ny int) {
	return x + Step(vel, StepScale, dirX), y + Step(vel, StepScale, dirY)
}

// TeleportTo returns the destination if free for the mask, otherwise the
// nearest free cell within radius (as PATH_ComputePathForUnit does with flag
// 0x1000 via FindNearestFreeCell).
func TeleportTo(g Grid, mask uint16, dest Point, radius int) (Point, bool) {
	return NearestFree(g, mask, dest, radius)
}
