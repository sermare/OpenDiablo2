package d2app

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
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

	return o.a.config.Save()
}
