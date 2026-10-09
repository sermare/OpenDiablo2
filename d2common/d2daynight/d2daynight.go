// Package d2daynight models the real game's global day/night environment
// (Game.exe 1.14b, Env.cpp; struct pointer DAT_007986bc), spot-checked in
// Ghidra read-only:
//
//   - phase 0..5, a tick counter, ticks-per-degree (+0x28) and ambient RGB.
//   - A day is 360 "degrees". In cycling mode (outdoor towns/fields) the
//     ticks-per-degree constant is 4 (DAT_00740e7c), so a day is 1440 ticks.
//     Phase start degrees (table 0x740e30): phase1 0, phase2 60, phase3 120,
//     phase4 180, phase5 240, phase0 300 (verified from memory dump).
//   - Per tick (FUN_0061bc10): tick++; wrap to 0 at 360*tpd; when tick is
//     greater than the next phase's start*tpd the phase advances and the
//     tick snaps to that start. Because phase 1 starts at 0, phase 0 lasts a
//     single step and the wrap never fires (cycle = 1206 steps, as read). (Non-cycling mode, ticks-per-degree 128,
//     uses table 0x740da0 and extra act-dependent tick bumps; modelled here
//     only as "frozen", which is how the town sets it up via 0x61c1a0.)
//   - Ambient RGB (FUN_0061ba00): current phase colour moved towards the next
//     phase colour by the progress through the phase. The decompiler output
//     for the interpolation is garbled (x87), so the linear blend is an
//     UNVERIFIED reading; the table colours themselves are verified bytes.
//     The cycling table's colours are all within one unit of (0,30,243/244),
//     which looks odd for an ambient tint and may be a different quantity.
//   - What real time one tick lasts is not established (the advance runs
//     once per server/game tick; 25 per second is assumed, UNVERIFIED).
package d2daynight

// Phases of the day. The names are interpretive; only the numbers and the
// greeting classification are verified.
const (
	PhaseNight0 = 0
	PhaseDawn   = 1
	PhaseDay    = 2
	PhaseDusk   = 3
	PhaseNight4 = 4
	PhaseNight5 = 5

	PhaseCount = 6

	DegreesPerDay      = 360 // 0x168
	TicksPerDegreeCyc  = 4   // DAT_00740e7c
	TicksPerDegreeFixd = 128 // DAT_00740e78
	TicksPerSecond     = 25  // assumed, unverified
)

// RGB is an ambient colour.
type RGB struct{ R, G, B uint8 }

type entry struct {
	startDegree int
	light       int
	rgb         RGB
}

// cycling is the table at 0x740e30 (stride 12: start degree, light index,
// RGB bytes), indexed by phase.
var cycling = [PhaseCount]entry{
	{300, 3, RGB{0, 0x1e, 0xf3}},
	{0, 0, RGB{0, 0x1e, 0xf4}},
	{60, 1, RGB{0, 0x1e, 0xf3}},
	{120, 2, RGB{0, 0x1e, 0xf3}},
	{180, 2, RGB{0, 0x1e, 0xf4}},
	{240, 2, RGB{0, 0x1e, 0xf3}},
}

// fixed is the table at 0x740da0 used when cycling is off.
var fixed = [PhaseCount]entry{
	{0x140, 3, RGB{0x7d, 0x90, 0xf3}},
	{0x154, 3, RGB{0xd0, 0xb8, 0x83}},
	{0, 0, RGB{0xff, 0xff, 0xff}},
	{0xa0, 1, RGB{0xff, 0xff, 0xff}},
	{0xb4, 1, RGB{0xc2, 0x98, 0xc1}},
	{0xc8, 2, RGB{0x7d, 0x90, 0xf3}},
}

// Env is the day/night state.
type Env struct {
	phase          int
	tick           int
	ticksPerDegree int
	cycling        bool
}

// NewCycling returns an outdoor environment that starts at the beginning of
// phase 1 (as 0x61c130 does, up to its phase-0 start quirk).
func NewCycling() *Env {
	return &Env{phase: PhaseDawn, ticksPerDegree: TicksPerDegreeCyc, cycling: true}
}

// NewFixed returns a non-advancing environment frozen at a phase (the town
// uses phase 2 via 0x61c1a0).
func NewFixed(phase int) *Env {
	if phase < 0 || phase >= PhaseCount {
		phase = PhaseDay
	}

	return &Env{phase: phase, ticksPerDegree: TicksPerDegreeFixd}
}

// Phase returns the current phase 0..5.
func (e *Env) Phase() int { return e.phase }

// Tick returns the tick counter.
func (e *Env) Tick() int { return e.tick }

func (e *Env) table() *[PhaseCount]entry {
	if e.cycling {
		return &cycling
	}

	return &fixed
}

// SetPhaseAndTick mirrors ENVIRON_SetPhaseAndTick's validation: phase must be
// 0..5 and tick 0..ticksPerDegree*360 (larger ticks reset to 0). It returns
// false and changes nothing for invalid input.
func (e *Env) SetPhaseAndTick(phase, tick int) bool {
	if phase < 0 || phase >= PhaseCount || tick < 0 {
		return false
	}

	if tick > e.ticksPerDegree*DegreesPerDay {
		tick = 0
	}

	e.phase, e.tick = phase, tick

	return true
}

// Advance moves the clock forward by n ticks. A non-cycling environment does
// not move.
func (e *Env) Advance(n int) {
	if !e.cycling {
		return
	}

	for ; n > 0; n-- {
		e.step()
	}
}

func (e *Env) step() {
	e.tick++

	if e.tick >= e.ticksPerDegree*DegreesPerDay {
		e.tick = 0
	}

	next := (e.phase + 1) % PhaseCount
	start := e.table()[next].startDegree * e.ticksPerDegree

	if e.tick > start {
		e.phase = next
		e.tick = start
	}
}

// Ambient returns the ambient colour: the phase colour blended towards the
// next phase's by progress through the phase (blend unverified, see package
// comment).
func (e *Env) Ambient() RGB {
	t := e.table()
	cur := t[e.phase]

	if !e.cycling {
		return cur.rgb // frozen: no blending
	}

	nxt := t[(e.phase+1)%PhaseCount]

	span := (nxt.startDegree - cur.startDegree) * e.ticksPerDegree
	if span <= 0 {
		span += DegreesPerDay * e.ticksPerDegree
	}

	done := e.tick - cur.startDegree*e.ticksPerDegree
	if done < 0 {
		done += DegreesPerDay * e.ticksPerDegree
	}

	lerp := func(a, b uint8) uint8 {
		return uint8(int(a) + (int(b)-int(a))*done/span)
	}

	return RGB{lerp(cur.rgb.R, nxt.rgb.R), lerp(cur.rgb.G, nxt.rgb.G), lerp(cur.rgb.B, nxt.rgb.B)}
}

// TimeOfDay is the classification NPC greetings use (verified from
// SOUND_PickNpcGreeting 0x4dd4c0).
type TimeOfDay int

// Greeting time classes; the values index GREETING_TIME_1..3.
const (
	Morning TimeOfDay = iota // phase 1
	Day                      // phases 2, 3
	Evening                  // phases 0, 4, 5
)

// Classify maps a phase to its greeting class.
func Classify(phase int) TimeOfDay {
	switch phase {
	case PhaseDawn:
		return Morning
	case PhaseDay, PhaseDusk:
		return Day
	default:
		return Evening
	}
}

// TimeOfDay classifies the current phase.
func (e *Env) TimeOfDay() TimeOfDay { return Classify(e.phase) }
