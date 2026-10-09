package d2app

import (
	"os"
	"strconv"
)

const autoShotDefaultSeconds = 10.0

// autoShotState implements OD2_AUTOSHOT=<path.png>: after
// OD2_AUTOSHOT_SECONDS (default 10) of run time it saves the rendered frame
// (what the player sees, UI included) through the same capture facility as
// the console command "capframe", so a headless-ish autotest can look at the
// result. With OD2_AUTOEXIT=1 and no OD2_AUTOSCRIPT / OD2_AUTOMONSTER running
// the process quits after the shot. More shots during a scripted scenario can
// be taken with the script step say:capframe <path.png>.
type autoShotState struct {
	path      string
	at        float64
	elapsed   float64
	requested bool
}

func newAutoShot() *autoShotState {
	path := os.Getenv("OD2_AUTOSHOT")
	if path == "" {
		return nil
	}

	s := &autoShotState{path: path, at: autoShotDefaultSeconds}
	if v, err := strconv.ParseFloat(os.Getenv("OD2_AUTOSHOT_SECONDS"), 64); err == nil && v >= 0 {
		s.at = v
	}

	return s
}

// advanceAutoShot is called once per frame with the unscaled elapsed seconds.
func (a *App) advanceAutoShot(elapsed float64) {
	s := a.autoShot
	if s == nil {
		return
	}

	if s.requested {
		// the capture happens in the render pass and resets the state when done
		if a.captureState != captureStateNone {
			return
		}

		if fi, err := os.Stat(s.path); err == nil {
			a.Infof("AUTOSHOT saved %s (%d bytes)", s.path, fi.Size())
		} else {
			a.Errorf("AUTOSHOT failed: %v", err)
		}

		a.autoShot = nil

		if os.Getenv("OD2_AUTOEXIT") != "" && os.Getenv("OD2_AUTOSCRIPT") == "" && os.Getenv("OD2_AUTOMONSTER") == "" {
			os.Exit(0)
		}

		return
	}

	s.elapsed += elapsed
	if s.elapsed < s.at {
		return
	}

	s.requested = true
	a.captureState = captureStateFrame
	a.capturePath = s.path
	a.captureFrames = nil
}
