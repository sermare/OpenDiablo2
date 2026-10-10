package d2app

import (
	"os"
	"path/filepath"
	"strings"

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
