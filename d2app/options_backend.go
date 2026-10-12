package d2app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// optionsBackend connects the escape menu's option pages to config.json.
type optionsBackend struct{ a *App }

func (o optionsBackend) Config() *d2config.Configuration { return o.a.config }

// Change stores a choice, applies the ones the engine supports now (sound and
// music volume) and saves the configuration. UNVERIFIED/not applied yet: gamma,
// contrast, lighting, shadows, perspective, resolution, 3D bias, environment
// effects and NPC speech are saved and shown but do not change rendering.
func (o optionsBackend) Change(key string, index int) error {
	if err := o.a.config.SetOption(key, index); err != nil {
		return err
	}

	if key == d2config.OptSound || key == d2config.OptMusic {
		o.a.audio.SetVolumes(o.a.config.BgmVolume, o.a.config.SfxVolume)
	}

	o.a.applyAccessibility(key)

	if key == d2config.OptFullscreen {
		// the FULL SCREEN row: switch now, keep the FullScreen field in step with the option
		full := o.a.config.WantFullscreen()
		o.a.config.SetFullscreenChoice(full)
		o.a.renderer.SetFullScreen(full)
	}

	if key == d2config.OptNpcSpeech {
		o.a.applySpeechOption()
	}

	return o.a.config.Save()
}

// windowScaler is implemented by renderers that can resize the window at run time.
type windowScaler interface{ SetWindowScale(scale int) }

// applyAccessibility applies the accessibility and gamepad options that take
// effect at once; key=="" applies all of them (at start up).
func (a *App) applyAccessibility(key string) {
	all := key == ""

	if all || key == d2config.OptColorblind {
		d2ui.SetColorBlind(a.config.ColorBlind())
	}

	if all || strings.HasPrefix(key, "pad.") {
		d2gamepad.Default().SetMapping(a.config.PadMapping())
	}

	if key == d2config.OptUIScale {
		if r, ok := a.renderer.(windowScaler); ok {
			r.SetWindowScale(a.config.UIScale())
		}
	}
}

// loadGamepadProfiles reads gamepad-profiles.json next to config.json, which
// can add or replace the raw button layouts of controllers (see docs/gamepad.md).
func (a *App) loadGamepadProfiles() {
	path := filepath.Join(filepath.Dir(a.config.Path()), "gamepad-profiles.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	if err := d2gamepad.DefaultProfiles().Override(data); err != nil {
		a.Warningf("%s: %v", path, err)
		return
	}

	a.Infof("gamepad profiles loaded from %s", path)
}

// applySpeechOption tells the audio provider whether NPC speech is played
// (the NPC SPEECH option: AUDIO AND TEXT, AUDIO ONLY or TEXT ONLY).
func (a *App) applySpeechOption() {
	if sp, ok := a.audio.(interface{ SetSpeechAudio(bool) }); ok {
		const textOnly = 2

		sp.SetSpeechAudio(a.config.OptionIndex(d2config.OptNpcSpeech) != textOnly)
	}
}

// monitorSize is the size of the monitor (device independent pixels), 0x0 when unknown.
func (a *App) monitorSize() d2display.Size {
	if r, ok := a.renderer.(interface{ ScreenSize() (int, int) }); ok {
		w, h := r.ScreenSize()
		return d2display.Size{W: w, H: h}
	}

	return d2display.Size{}
}

// ResolutionChoices lists the VIDEO RESOLUTION row: window sizes that fit the monitor (the monitor size itself
// means full screen) and the index of the one the window has now.
func (o optionsBackend) ResolutionChoices() ([]string, int) {
	mon := o.a.monitorSize()
	fit := mon

	if fit.H > 0 {
		fit.H -= windowChromeHeight // a window needs room for the menu bar and title bar
	}

	cur := d2display.WindowSize()
	if o.a.renderer.IsFullScreen() && mon.W > 0 {
		cur = mon
	}

	sizes := d2display.ResolutionChoices(fit, cur)
	if mon.W > 0 {
		sizes = d2display.ResolutionChoices(fit, cur)
		sizes = append(sizes, mon)
		sizes = uniqueSizes(sizes)
	}

	cur = d2display.Clamp(cur)
	values := make([]string, len(sizes))
	idx := 0

	for i, s := range sizes {
		values[i] = s.String()

		if s == cur {
			idx = i
		}
	}

	return values, idx
}

func uniqueSizes(in []d2display.Size) []d2display.Size {
	seen := map[d2display.Size]bool{}
	out := in[:0]

	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}

	return out
}

type windowedSizer interface{ SetWindowedSize(w, h int) }

// SetResolution applies a row value now: the monitor size is full screen, any other size a window of that size
// (this leaves full screen). The logical size is saved as the start size ("Display" of config.json).
func (o optionsBackend) SetResolution(value string) error {
	size, err := d2display.ParseRow(value)
	if err != nil {
		return err
	}

	a := o.a
	if mon := a.monitorSize(); mon.W > 0 && size == mon {
		a.config.SetFullscreenChoice(true)
		a.renderer.SetFullScreen(true)

		return a.config.Save()
	}

	ws, ok := a.renderer.(windowedSizer)
	if !ok {
		return fmt.Errorf("renderer cannot resize the window")
	}

	a.config.SetFullscreenChoice(false)
	ws.SetWindowedSize(size.W, size.H)

	logical := d2display.Logical(size.W, size.H, d2display.Scale())
	a.config.Display.Width, a.config.Display.Height = logical.W, logical.H

	return a.config.Save()
}

// FullScreenNow is the real state of the window, which the FULL SCREEN row shows.
func (o optionsBackend) FullScreenNow() bool { return o.a.renderer.IsFullScreen() }
