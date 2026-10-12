package d2app

import (
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2client/d2clientconnectiontype"
)

// windowResizer is implemented by renderers that can resize the window at run time.
type windowResizer interface{ SetWindowPixels(w, h int) }

// autoFlowCmd performs the screen-independent steps of OD2_AUTOFLOW (display QA, docs/DISPLAY.md): goto:<screen>,
// resize:WxH, fullscreen:on|off|toggle and uiscale:N. A step that cannot be done is logged and skipped.
func (a *App) autoFlowCmd(verb, arg string) {
	a.Infof("AUTOFLOW cmd %s %s", verb, arg)

	switch verb {
	case "goto":
		switch arg {
		case "credits":
			a.ToCredits()
		case "cinematics":
			a.ToCinematics()
		case "mainmenu":
			a.ToMainMenu()
		case "charselect":
			a.ToCharacterSelect(d2clientconnectiontype.Local, "")
		default:
			a.Warningf("AUTOFLOW goto: unknown screen %q", arg)
		}
	case "videores": // the VIDEO RESOLUTION row of the options page (saves into the config of the run)
		if err := (optionsBackend{a}).SetResolution(strings.ToUpper(arg)); err != nil {
			a.Warningf("AUTOFLOW videores %q: %v", arg, err)
		}
	case "resize":
		w, h, ok := strings.Cut(arg, "x")
		wi, err1 := strconv.Atoi(w)
		hi, err2 := strconv.Atoi(h)

		r, can := a.renderer.(windowResizer)
		if !ok || err1 != nil || err2 != nil || !can {
			a.Warningf("AUTOFLOW resize: cannot resize to %q", arg)

			return
		}

		r.SetWindowPixels(wi, hi)
	case "fullscreen":
		full := !a.renderer.IsFullScreen()

		switch arg {
		case "on":
			full = true
		case "off":
			full = false
		}

		a.renderer.SetFullScreen(full) // not saved: a QA run never changes the config
	case "uiscale":
		n, err := strconv.Atoi(arg)

		r, can := a.renderer.(windowScaler)
		if err != nil || !can {
			a.Warningf("AUTOFLOW uiscale: cannot set %q", arg)

			return
		}

		r.SetWindowScale(n)
	}
}
