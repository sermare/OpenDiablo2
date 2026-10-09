// Package d2daynight models the real game's global day/night environment
// (Game.exe 1.14b, Env.cpp; struct pointer DAT_007986bc), notes in
// d2-re-notes/renderer.md (b4), spot-checked in Ghidra read-only:
//
//   - phase 0..5, a tick counter, ticks-per-degree (+0x28) and ambient RGB.
//   - A day is 360 "degrees" of ticks-per-degree ticks each.
//   - The REAL outdoor sun cycle is the table at 0x740da0 (verified bytes):
//     12-byte entries {startDegree, lightIdx, R, G, B}: phase0 night blue
//     (7d 90 f3) from 320, phase1 dawn orange (d0 b8 83) from 340, phase2
//     white from 0, phase3 white from 160, phase4 dusk purple (c2 98 c1)
//     from 180, phase5 night blue from 200. Ticks per degree is 128 there, so
//     a day is 46080 ticks (30.7 min at 25 Hz, rate UNVERIFIED).
//   - The table at 0x740e30 (colours ~ (0,30,243), 4 ticks per degree) is the
//     special/scripted mode (the flag argument of SetPhaseAndTick); its
//     intensity is just 32. An earlier version of this package had the two
//     tables swapped.
//   - Ambient intensity (ENVIRON_UpdateAmbientIntensity 0x61b8a0, verified in
//     the decompile): a = tick/tpd degrees; I = 128+128*cos(a) while
//     tick < tpd*180, else 128+64*cos(a); values <= 0 become 0, capped at 255
//     (170 in Act 5). Level 0x78 is fixed at 200.
//   - The colour is blended from table[phase] towards table[phase+1] by the
//     progress through the phase: UNVERIFIED (x87 in the decompile).
//   - How the per-tick advance maps onto the phase counter is modelled here
//     as "phase is the last table entry whose start degree is <= the current
//     degree", not as the original incremental code (UNVERIFIED).
package d2daynight

import "math"

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
	TicksPerDegreeReal = 128 // DAT_00740e78: real outdoor cycle
	TicksPerDegreeSpec = 4   // DAT_00740e7c: special mode
	TicksPerSecond     = 25  // assumed, unverified
)

// RGB is an ambient colour.
type RGB struct{ R, G, B uint8 }

type entry struct {
	startDegree int
	light       int
	rgb         RGB
}

// special is the table at 0x740e30 (scripted/special mode), indexed by phase.
var special = [PhaseCount]entry{
	{300, 3, RGB{0, 0x1e, 0xf3}},
	{0, 0, RGB{0, 0x1e, 0xf4}},
	{60, 1, RGB{0, 0x1e, 0xf3}},
	{120, 2, RGB{0, 0x1e, 0xf3}},
	{180, 2, RGB{0, 0x1e, 0xf4}},
	{240, 2, RGB{0, 0x1e, 0xf3}},
}

// outdoor is the table at 0x740da0: the real outdoor sun cycle.
var outdoor = [PhaseCount]entry{
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
	special        bool
}

// NewCycling returns the real outdoor environment, advancing, starting at the
// beginning of phase 0 (as 0x61c130 does).
func NewCycling() *Env {
	e := &Env{ticksPerDegree: TicksPerDegreeReal, cycling: true}
	e.SetPhase(PhaseNight0)

	return e
}

// NewFixed returns a non-advancing outdoor environment frozen at the start of
// a phase (the town can use phase 2 via 0x61c1a0).
func NewFixed(phase int) *Env {
	e := &Env{ticksPerDegree: TicksPerDegreeReal}
	if !e.SetPhase(phase) {
		e.SetPhase(PhaseDay)
	}

	return e
}

// NewSpecial returns the scripted/special-mode environment (table 0x740e30).
func NewSpecial() *Env {
	return &Env{phase: PhaseDawn, ticksPerDegree: TicksPerDegreeSpec, cycling: true, special: true}
}

// Phase returns the current phase 0..5.
func (e *Env) Phase() int { return e.phase }

// Tick returns the tick counter.
func (e *Env) Tick() int { return e.tick }

func (e *Env) table() *[PhaseCount]entry {
	if e.special {
		return &special
	}

	return &outdoor
}

// phaseAt returns the phase containing the given degree.
func (e *Env) phaseAt(degree int) int {
	t := e.table()
	best, bestStart := -1, -1
	maxP, maxStart := 0, -1

	for p := 0; p < PhaseCount; p++ {
		s := t[p].startDegree
		if s <= degree && s > bestStart {
			best, bestStart = p, s
		}

		if s > maxStart {
			maxP, maxStart = p, s
		}
	}

	if best < 0 { // before the first start: still in the last phase of the previous day
		return maxP
	}

	return best
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

// SetPhase jumps to the start of a phase (useful to force a time of day).
func (e *Env) SetPhase(phase int) bool {
	if phase < 0 || phase >= PhaseCount {
		return false
	}

	e.phase = phase
	e.tick = e.table()[phase].startDegree * e.ticksPerDegree

	return true
}

// Advance moves the clock forward by n ticks. A non-cycling environment does
// not move.
func (e *Env) Advance(n int) {
	if !e.cycling || n <= 0 {
		return
	}

	period := e.ticksPerDegree * DegreesPerDay
	e.tick = (e.tick + n) % period
	e.phase = e.phaseAt(e.tick / e.ticksPerDegree)
}

// Intensity returns the ambient light intensity 0..255 for the outdoor cycle
// (verified formula, see the package comment). The special mode is fixed 32.
func (e *Env) Intensity() int {
	if e.special {
		return 0x20
	}

	rad := float64(e.tick) / float64(e.ticksPerDegree) / 180 * math.Pi

	v := 128 + 128*math.Cos(rad)
	if e.tick >= e.ticksPerDegree*180 {
		v = 128 + 64*math.Cos(rad)
	}

	i := int(math.Round(v))
	if i < 0 {
		i = 0
	}

	if i > 255 {
		i = 255
	}

	return i
}

// Ambient returns the ambient colour: the phase colour blended towards the
// next phase's by progress through the phase (blend unverified, see package
// comment). A non-advancing environment returns the exact phase colour.
func (e *Env) Ambient() RGB {
	t := e.table()
	cur := t[e.phase]

	if !e.cycling {
		return cur.rgb
	}

	// next phase in time order
	period := DegreesPerDay * e.ticksPerDegree
	startTick := cur.startDegree * e.ticksPerDegree
	bestNext, bestDelta := e.phase, period

	for p := 0; p < PhaseCount; p++ {
		if p == e.phase {
			continue
		}

		d := (t[p].startDegree*e.ticksPerDegree - startTick + period) % period
		if d > 0 && d < bestDelta {
			bestNext, bestDelta = p, d
		}
	}

	done := (e.tick - startTick + period) % period
	nxt := t[bestNext]

	lerp := func(a, b uint8) uint8 {
		return uint8(int(a) + (int(b)-int(a))*done/bestDelta)
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
