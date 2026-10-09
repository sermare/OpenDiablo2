package d2monster

// Params are the AiTickParams a think function receives (VERIFIED layout:
// target, edge distance, in-range flag).
type Params struct {
	Target  *Target // nil when the AI's target mode does not acquire one
	Dist    int
	InRange bool
}

// Ctx is passed to a think function. Its methods are the primitives of
// monster-ai-2.md section 3; each requests one action and marks the brain as
// busy until WakeNow, or sleeps.
type Ctx struct {
	B *Brain
	W World
	Params
	acted bool
}

// Acted reports whether the think function issued an action or sleep.
func (c *Ctx) Acted() bool { return c.acted }

func (c *Ctx) busy() {
	c.acted = true
	c.B.Wake = waitForever
}

// Sleep is MONAI_ScheduleWaitTicks: the next think runs n frames from now
// (n==0 becomes 1). The original also forces mode neutral if needed; the
// engine does that when the monster is not already idle.
func (c *Ctx) Sleep(n int) {
	if n < 1 {
		n = 1
	}

	c.acted = true
	c.B.Wake = c.W.Frame() + n
}

// Attack requests an attack/skill mode; on failure the original sleeps 10.
func (c *Ctx) Attack(mode Mode, t Target) bool {
	if !c.W.Attack(c.B, mode, t) {
		c.Sleep(10)
		return false
	}

	c.busy()

	return true
}

// Cast requests the monstats skill slot at t.
func (c *Ctx) Cast(slot int, t Target) bool {
	if !c.W.Cast(c.B, slot, t) {
		c.Sleep(10)
		return false
	}

	c.busy()

	return true
}

// SetSpeed is MONAI_SetMoveSpeedOverride (UNVERIFIED semantics).
func (c *Ctx) SetSpeed(v int) { c.W.SetSpeed(c.B, v) }

// WalkTo is MONAI_WalkToTarget(target, reach): reach 7 is the usual melee
// value, 0 is tile exact. Returns false when the request failed (no sleep is
// scheduled so the caller can fall back, as the original does).
func (c *Ctx) WalkTo(t Target, reach int) bool { return c.move(Point{t.X, t.Y}, &t, reach, false) }

// RunTo is MONAI_RunToTarget (mode RN).
func (c *Ctx) RunTo(t Target, reach int) bool { return c.move(Point{t.X, t.Y}, &t, reach, true) }

func (c *Ctx) move(p Point, t *Target, reach int, run bool) bool {
	if !c.W.MoveTo(c.B, p, t, reach, run) {
		return false
	}

	c.busy()

	return true
}

// WalkToRange is MONAI_WalkToRangeOfTarget(target, maxStep, desiredDist): the
// kiting primitive. With d the distance to the target, it moves toward the
// target by min(|d-desired|, maxStep) along the (dx,dy) ratio when
// d >= desired, and away from it by the same amount when d < desired
// (VERIFIED description; the rounding of the ratio is not recorded so it is
// truncation toward zero here).
func (c *Ctx) WalkToRange(t Target, maxStep, desired int) bool {
	return c.toRange(t, maxStep, desired, false)
}

// RunToRange is MONAI_MoveToRangeOfTarget with run mode.
func (c *Ctx) RunToRange(t Target, maxStep, desired int) bool {
	return c.toRange(t, maxStep, desired, true)
}

func (c *Ctx) toRange(t Target, maxStep, desired int, run bool) bool {
	b := c.B
	dx, dy := t.X-b.X, t.Y-b.Y
	d := EdgeDistance(dx, dy, b.Size)

	step := d - desired
	if step < 0 {
		step = -step
	}

	if step > maxStep {
		step = maxStep
	}

	full := Distance(dx, dy)
	if full == 0 || step == 0 {
		return c.move(Point{b.X, b.Y}, nil, 0, run)
	}

	mx, my := dx*step/full, dy*step/full
	if d < desired {
		mx, my = -mx, -my
	}

	return c.move(Point{b.X + mx, b.Y + my}, nil, 0, run)
}

// WalkAway is MONAI_WalkAwayFromTarget(target, n): the destination is the
// monster's position plus n*sign(own-target) per axis (VERIFIED: a diagonal
// flee). It returns false when the request fails.
func (c *Ctx) WalkAway(t Target, n int) bool { return c.away(t, n, false) }

// RunAway is MONAI_RunAwayFromTarget.
func (c *Ctx) RunAway(t Target, n int) bool { return c.away(t, n, true) }

func (c *Ctx) away(t Target, n int, run bool) bool {
	b := c.B

	return c.move(Point{b.X + n*sign(b.X-t.X), b.Y + n*sign(b.Y-t.Y)}, nil, 0, run)
}

// Wander is MONAI_WanderRandomNearby(n). The notes describe it only as "two
// LCG steps decide magnitude/sign per axis: the first step's parity picks
// which axis gets n and which a bounded roll, then the sign flips by parity".
// UNVERIFIED reading used here: step 1 parity selects the axis that gets the
// full n, step 1 bit 1 its sign; step 2 is a bounded roll in [0,n] for the
// other axis whose sign comes from step 2's low bit.
func (c *Ctx) Wander(n int) bool {
	b := c.B
	s1 := b.Seed.Step()
	other := b.Roll(n + 1)

	major, minor := n, other
	if s1&2 != 0 {
		major = -major
	}

	if b.Seed.Lo&1 != 0 {
		minor = -minor
	}

	dx, dy := major, minor
	if s1&1 == 0 {
		dx, dy = minor, major
	}

	return c.move(Point{b.X + dx, b.Y + dy}, nil, 0, false)
}

// WalkNearTarget is MONAI_WalkNearTargetRandom(target, n): the Wander offset
// scheme applied around the target instead of the monster (VERIFIED
// description, the offsets share Wander's UNVERIFIED reading). Without a
// target it falls back to Wander(2).
func (c *Ctx) WalkNearTarget(t *Target, n int) bool {
	if t == nil {
		return c.Wander(2)
	}

	b := c.B
	s1 := b.Seed.Step()
	other := b.Roll(n + 1)

	major, minor := n, other
	if s1&2 != 0 {
		major = -major
	}

	if b.Seed.Lo&1 != 0 {
		minor = -minor
	}

	dx, dy := major, minor
	if s1&1 == 0 {
		dx, dy = minor, major
	}

	return c.move(Point{t.X + dx, t.Y + dy}, nil, 0, false)
}

// Circle is MONAI_CircleOrStrafeTarget. The notes only say it consumes one
// LCG step and queues a walk with no coordinates (UNVERIFIED), so this port
// consumes the step and strafes n subtiles perpendicular to the target
// direction, the side chosen by the step's parity.
func (c *Ctx) Circle(t Target, n int) bool {
	b := c.B
	s := b.Seed.Step()
	dx, dy := t.X-b.X, t.Y-b.Y
	px, py := -sign(dy), sign(dx)

	if s&1 == 0 {
		px, py = -px, -py
	}

	return c.move(Point{b.X + px*n, b.Y + py*n}, nil, 0, false)
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}

	return 0
}
