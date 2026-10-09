package d2app

import (
	"os"
	"strconv"
)

// OD2_AUTOSHOT=<file>.png saves one screenshot of the window a few seconds after
// the game starts and, with OD2_AUTOEXIT=1, quits afterwards. OD2_AUTOSHOT_DELAY
// sets the wait in seconds (default 6). It is for checking a layout without a
// person at the keyboard. A scenario that needs panels open first should use the
// `shot:<file>` step of OD2_AUTOSCRIPT instead, which takes the shot at that point.
const autoShotDefaultDelay = 6.0

type autoShotState struct {
	elapsed   float64
	requested bool
	done      bool
}

var autoShot autoShotState

// advanceAutoShot runs the OD2_AUTOSHOT timer; it is called every frame.
func (a *App) advanceAutoShot(elapsed float64) {
	path := os.Getenv("OD2_AUTOSHOT")
	if path == "" || autoShot.done {
		return
	}

	// OD2_AUTOSCRIPT scenarios take their own shots with the shot: step
	if os.Getenv("OD2_AUTOSCRIPT") != "" {
		autoShot.done = true
		return
	}

	if autoShot.requested {
		// renderCapture clears the state once the frame has been written
		if a.captureState == captureStateNone {
			autoShot.done = true

			a.Infof("AUTOSHOT saved %s", path)

			if os.Getenv("OD2_AUTOEXIT") != "" {
				os.Exit(0)
			}
		}

		return
	}

	autoShot.elapsed += elapsed

	delay := autoShotDefaultDelay

	if d, err := strconv.ParseFloat(os.Getenv("OD2_AUTOSHOT_DELAY"), 64); err == nil && d >= 0 {
		delay = d
	}

	if autoShot.elapsed >= delay {
		autoShot.requested = true
		a.captureState = captureStateFrame
		a.capturePath = path
		a.captureFrames = nil
	}
}
