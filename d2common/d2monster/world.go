package d2monster

// Target is a snapshot of a unit the AI can target.
type Target struct {
	ID       uint32
	X, Y     int // subtiles
	Size     int
	IsPlayer bool
}

// Point is a position in subtiles.
type Point struct{ X, Y int }

// Senses is what the AI can observe.
type Senses interface {
	// Frame is the current game frame (25 per second).
	Frame() int
	// Nearest returns the nearest hostile target on the monster's level
	// regardless of aggro radius, with the edge distance to it (the AI
	// applies aidist itself).
	Nearest(b *Brain) (t Target, dist int, ok bool)
	// AttackTarget is MONAI_GetAttackTargetAndDistance, used by ranged units
	// and casters instead of the tick's target (UNVERIFIED filter details).
	AttackTarget(b *Brain) (t Target, dist int, ok bool)
	// InRange reports whether the monster can attack t right now (the
	// "inRange" flag of the tick parameters: reach and line of sight).
	InRange(b *Brain, t Target, dist int) bool
	// DyingNear reports a unit in mode DT within radius of the monster (the
	// Fallen "something nearby died" scan).
	DyingNear(b *Brain, radius int) bool
	// HasState reports whether the unit has the given state id.
	HasState(b *Brain, state int) bool
}

// Actor is how the AI changes the world. Every call is one request; the
// return value is false when the request could not be submitted (the
// original then falls back to a short sleep or another action).
type Actor interface {
	// Attack starts an attack/skill animation mode (A1, A2, S1..S4) on t.
	Attack(b *Brain, mode Mode, t Target) bool
	// Cast casts the monstats skill slot (0..7) at t.
	Cast(b *Brain, slot int, t Target) bool
	// MoveTo requests a path to dest, or to the unit target if non-nil,
	// stopping within reach subtiles; run selects mode RN instead of WL.
	MoveTo(b *Brain, dest Point, target *Target, reach int, run bool) bool
	// SetSpeed is MONAI_SetMoveSpeedOverride; its argument semantics are
	// UNVERIFIED, so engines may ignore it.
	SetSpeed(b *Brain, v int)
}

// World is everything a Tick needs.
type World interface {
	Senses
	Actor
}

// Shouter is an optional extension of an Actor: FUN_005513c0, the monster
// "shout" a Fallen makes when it starts to flee (UNVERIFIED meaning).
type Shouter interface {
	Shout(b *Brain)
}
