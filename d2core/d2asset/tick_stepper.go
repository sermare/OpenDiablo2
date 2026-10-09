package d2asset

import "math"

const (
	// engineTickRate is the original's fixed game tick (25 Hz, renderer.md b5, verified).
	engineTickRate = 25.0
	// fixedPointOne is 1.0 in the 8.8 fixed point animation accumulator.
	fixedPointOne = 256
)

// tickStepper reproduces the original's frame advance: once per 25 Hz tick an 8.8
// fixed point accumulator grows by the animation rate (the AnimData speed, 256 = one frame per
// tick) and every overflow of 256 advances one frame. Over time this equals
// 1/(speed*25/256) seconds per frame, but frames only change on tick boundaries.
type tickStepper struct {
	rate     int     // 8.8 frames per tick
	acc      int     // 8.8 accumulator, always < 256 between ticks
	tickTime float64 // seconds not yet consumed by a whole tick
}

// rateFromFrameSeconds converts a frame length in seconds into the 8.8 rate (0 if not positive).
func rateFromFrameSeconds(seconds float64) int {
	if seconds <= 0 || math.IsInf(seconds, 0) || math.IsNaN(seconds) {
		return 0
	}

	return int(math.Round(fixedPointOne / (seconds * engineTickRate)))
}

// advance consumes elapsed seconds and returns how many frames to advance.
func (t *tickStepper) advance(elapsed float64) int {
	if t.rate <= 0 || elapsed <= 0 {
		return 0
	}

	t.tickTime += elapsed

	ticks := int(t.tickTime * engineTickRate)
	t.tickTime -= float64(ticks) / engineTickRate

	frames := 0

	for i := 0; i < ticks; i++ {
		t.acc += t.rate
		frames += t.acc / fixedPointOne
		t.acc %= fixedPointOne
	}

	return frames
}

func (t *tickStepper) reset() { t.acc, t.tickTime = 0, 0 }
