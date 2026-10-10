package d2player

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
)

// OptionsBackend is what the options pages of the escape menu read and write:
// the OD2 configuration plus the application of a change (volumes) and saving
// it. d2app provides it; without one the pages keep their defaults and nothing
// is persisted.
type OptionsBackend interface {
	Config() *d2config.Configuration
	// Change stores the choice, applies it where the engine supports it and
	// saves config.json.
	Change(key string, index int) error
}

var optionsBackend OptionsBackend

// SetOptionsBackend registers the backend used by every escape menu.
func SetOptionsBackend(b OptionsBackend) { optionsBackend = b }

var optionKeys = map[optionID]string{
	optAudioSoundVolume:          d2config.OptSound,
	optAudioMusicVolume:          d2config.OptMusic,
	optAudio3dSound:              d2config.Opt3DBias,
	optAudioHardwareAcceleration: d2config.OptHardwareAccel,
	optAudioEnvEffects:           d2config.OptEnvEffects,
	optAudioNpcSpeech:            d2config.OptNpcSpeech,
	optVideoResolution:           d2config.OptResolution,
	optVideoLightingQuality:      d2config.OptLightingQuality,
	optVideoBlendedShadows:       d2config.OptBlendedShadows,
	optVideoPerspective:          d2config.OptPerspective,
	optVideoGamma:                d2config.OptGamma,
	optVideoContrast:             d2config.OptContrast,
	optAutomapSize:               d2config.OptAutomapSize,
	optAutomapFade:               d2config.OptAutomapFade,
	optAutomapCenterWhenCleared:  d2config.OptAutomapCenter,
	optAutomapShowParty:          d2config.OptAutomapShowParty,
	optAutomapShowNames:          d2config.OptAutomapNames,
}

func optionValues(id optionID) []string {
	def, _ := d2config.OptionDefFor(optionKeys[id])
	return def.Values
}

func optionIndex(id optionID) int {
	if optionsBackend != nil {
		return optionsBackend.Config().OptionIndex(optionKeys[id])
	}

	def, _ := d2config.OptionDefFor(optionKeys[id])

	return def.Default
}

func (m *EscapeMenu) applyOption(id optionID, value string) {
	if optionsBackend == nil {
		return
	}

	for i, v := range optionValues(id) {
		if v == value {
			if err := optionsBackend.Change(optionKeys[id], i); err != nil {
				m.Errorf("OPTIONS could not save %s: %v", optionKeys[id], err)
			}

			return
		}
	}
}

// syncOptionLabels refreshes every option row from the configuration (called
// when the menu opens, so changes made elsewhere, e.g. the console, show).
func (m *EscapeMenu) syncOptionLabels() {
	for _, l := range m.layouts {
		for _, el := range l.actionableElements {
			if e, ok := el.(*enumLabel); ok {
				e.current = optionIndex(e.optionID)
				_ = e.textChangingLabel.SetText(e.values[e.current])
			}
		}
	}
}

// automapSizeOption is the size the Tab key shows the automap in.
func automapSizeOption() d2automap.Size {
	if optionsBackend != nil && optionsBackend.Config().OptionIndex(d2config.OptAutomapSize) == 1 {
		return d2automap.SizeMini
	}

	return d2automap.SizeFull
}

// automapOptionOn reads one of the yes/no automap options of the options menu
// (key: fade, center, party, names).
func automapOptionOn(key string) bool {
	id := map[string]string{
		"fade": d2config.OptAutomapFade, "center": d2config.OptAutomapCenter,
		"party": d2config.OptAutomapShowParty, "names": d2config.OptAutomapNames,
	}[key]

	return optionsBackend != nil && id != "" && optionsBackend.Config().OptionIndex(id) == 1
}

// RunOptionsAutoTest is the scripted run behind OD2_AUTOOPTIONS: it opens the
// menu, walks the sound, video and automap pages, "clicks" rows until they
// show chosen values, checks config.json on disk and the automap size, and
// closes the menu. No mouse input.
func (g *GameControls) RunOptionsAutoTest() {
	m := g.escapeMenu
	m.open()
	m.Infof("OPTIONS autotest open=%v backend=%v", m.IsOpen(), optionsBackend != nil)

	// wanted index per option (differs from the defaults on purpose)
	want := map[optionID]int{
		optAudioSoundVolume: 6, optAudioMusicVolume: 8, optAudio3dSound: 2, optAudioNpcSpeech: 2,
		optVideoGamma: 7, optVideoContrast: 3, optVideoLightingQuality: 0, optVideoPerspective: 0,
		optAutomapSize: 1, optAutomapFade: 1, optAutomapShowNames: 1,
	}

	for _, page := range []layoutID{soundOptionsLayoutID, videoOptionsLayoutID, automapOptionsLayoutID} {
		m.setLayout(page)

		for _, el := range m.layouts[page].actionableElements {
			e, ok := el.(*enumLabel)
			if !ok {
				continue
			}

			target, chosen := want[e.optionID]
			for chosen && e.current != target {
				e.Trigger()
			}

			m.Infof("OPTIONS row %s=%s", optionKeys[e.optionID], e.values[e.current])
		}
	}

	// the automap uses the chosen size when Tab shows it
	g.automap.SetOn(false)
	g.automap.Toggle()
	m.Infof("OPTIONS automap size=%v (1=mini)", g.automap.Size())

	// what a restart would read
	if optionsBackend == nil {
		return
	}

	cfg := optionsBackend.Config()
	data, err := os.ReadFile(cfg.Path())
	disk := &d2config.Configuration{}

	if err == nil {
		err = json.Unmarshal(data, disk)
	}

	ok := err == nil

	var parts []string

	for _, d := range d2config.OptionDefs() {
		if disk.OptionIndex(d.Key) != cfg.OptionIndex(d.Key) {
			ok = false
		}

		parts = append(parts, d.Key+"="+disk.OptionValue(d.Key))
	}

	m.Infof("OPTIONS persisted ok=%v err=%v %s", ok, err, strings.Join(parts, " "))

	m.close()
}
