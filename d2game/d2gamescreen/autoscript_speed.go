package d2gamescreen

import (
	"math"
	"os"
	"strconv"
	"sync"
)

const (
	// maxAutoTimeScale is the fastest OD2_AUTOSPEED.
	maxAutoTimeScale = 32.0
	// maxAutoSimStep is the longest simulated span of one update step: one 25 Hz game tick. The skill
	// engine and the monster director clamp a call to 0.25 s, and the hero's walking and collision
	// were made for short steps.
	maxAutoSimStep = 1.0 / 25.0
	// maxAutoSubsteps bounds the work of one rendered frame (a very slow frame at a high speed).
	maxAutoSubsteps = 48
)

var (
	autoTimeScaleOnce  sync.Once
	autoTimeScaleValue = 1.0
)

// autoTimeScale is OD2_AUTOSPEED=<1..32>: the game clock of a scripted run
// (movement, fights, timers, the script's own waits) runs that many times
// faster than real time, so a long playthrough fits the time a verification
// run may take. Without the variable the game runs at normal speed.
func autoTimeScale() float64 {
	autoTimeScaleOnce.Do(func() {
		if f, err := strconv.ParseFloat(os.Getenv("OD2_AUTOSPEED"), 64); err == nil && f > 1 {
			if f > maxAutoTimeScale {
				f = maxAutoTimeScale
			}

			autoTimeScaleValue = f
		}
	})

	return autoTimeScaleValue
}

// autoSubsteps is how many update steps a frame of total simulated seconds is split into: steps of at
// most maxAutoSimStep, at least one, at most maxAutoSubsteps.
func autoSubsteps(total float64) int {
	n := int(math.Ceil(total / maxAutoSimStep))
	if n < 1 {
		n = 1
	}

	if n > maxAutoSubsteps {
		n = maxAutoSubsteps
	}

	return n
}
