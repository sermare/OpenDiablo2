package d2app

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
)

// optionsBackend connects the escape menu's option pages to config.json.
type optionsBackend struct{ a *App }

func (o optionsBackend) Config() *d2config.Configuration { return o.a.config }

// Change stores a choice, applies the ones the engine supports now (sound and
// music volume, gamma and contrast) and saves the configuration. UNVERIFIED/not
// applied yet: lighting, shadows, perspective, resolution, 3D bias, environment
// effects and NPC speech are saved and shown but do not change rendering.
func (o optionsBackend) Change(key string, index int) error {
	if err := o.a.config.SetOption(key, index); err != nil {
		return err
	}

	if key == d2config.OptSound || key == d2config.OptMusic {
		o.a.audio.SetVolumes(o.a.config.BgmVolume, o.a.config.SfxVolume)
	}

	if key == d2config.OptGamma || key == d2config.OptContrast {
		o.a.applyColorAdjust()
	}

	return o.a.config.Save()
}

// colorAdjuster is the renderer's gamma / contrast pass.
type colorAdjuster interface {
	SetColorAdjust(gamma, contrast int)
	AdjustColors(target d2interface.Surface)
}

// applyColorAdjust passes the Gamma and Contrast options to the renderer.
func (a *App) applyColorAdjust() {
	adj, ok := a.renderer.(colorAdjuster)
	if !ok || a.config == nil {
		return
	}

	g, c := a.config.OptionIndex(d2config.OptGamma), a.config.OptionIndex(d2config.OptContrast)
	adj.SetColorAdjust(g, c)
	a.Infof("OPTIONS video gamma step=%d contrast step=%d applied to rendering", g, c)
}
