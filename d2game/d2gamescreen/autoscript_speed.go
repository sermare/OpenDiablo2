package d2gamescreen

import (
	"os"
	"strconv"
	"sync"
)

// maxAutoTimeScale keeps a game step short enough for the walking and the
// collision of the hero (no tunnelling through a wall in one frame).
const maxAutoTimeScale = 4.0

var (
	autoTimeScaleOnce  sync.Once
	autoTimeScaleValue = 1.0
)

// autoTimeScale is OD2_AUTOSPEED=<1..4>: the game clock of a scripted run
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
