package d2path

// Movement speeds, ported from missiles-pathing.md section (c)
// (PATH_AdvanceUnitOneFrame / PATH_ApplyAccelAndScaleStep / PATH_SetVelocity /
// PATH_UpdateUnitVelocityAndAnimRate). The Ghidra re-check of 0x651750 timed
// out (shared window busy), so everything is as VERIFIED in the notes, not
// re-verified here.

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
// shipped classes use 6/9 (checked against CharStats.txt); the run offset in
// the exe record is UNVERIFIED, so callers with real data should prefer
// values read from the table.
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

// SpeedPercent converts the movement-speed stat (0x43) to the percentage
// applied to velocity and animation rate, floored at 25%. The notes say
// `max(25, stat0x43 + diminishing-returns term)`; that stat 0x43 is an offset
// from a 100% base (so cold slow is negative) is UNVERIFIED.
func SpeedPercent(stat43 int) int {
	if p := 100 + stat43; p > MinSpeedPercent {
		return p
	}

	return MinSpeedPercent
}

// Velocity returns the 8.8 path velocity of a unit: base<<8 scaled by the
// percentage (integer division; the order of shift and scale is UNVERIFIED).
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
// and a 4.12 direction component (VERIFIED: ((vel*scale)>>6)*dir>>12).
func Step(vel, scale, dir int) int {
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

// Accelerate applies path acceleration: every AccelEvery-th frame vel grows by
// accel up to maxVel (VERIFIED shape; deceleration is UNVERIFIED). counter is
// the +0x8c tick counter; the updated one is returned.
func Accelerate(vel, maxVel, accel, counter int) (newVel, newCounter int) {
	counter++
	if counter < AccelEvery {
		return vel, counter
	}

	vel += accel
	if vel > maxVel {
		vel = maxVel
	}

	return vel, 0
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
