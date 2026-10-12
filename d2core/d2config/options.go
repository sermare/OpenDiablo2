package d2config

import (
	"fmt"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
)

// Option keys of the in-game options menu (Esc -> Options). The menu rows and
// their value lists follow the original Diablo II menus (sound, video and
// automap pages). Sound and music are stored as SfxVolume / BgmVolume, every
// other choice as an index into its value list in Configuration.Options.
const (
	OptSound            = "sound"
	OptMusic            = "music"
	Opt3DBias           = "3dbias"
	OptHardwareAccel    = "hwaccel"
	OptEnvEffects       = "enveffects"
	OptNpcSpeech        = "npcspeech"
	OptResolution       = "resolution"
	OptLightingQuality  = "lighting"
	OptBlendedShadows   = "shadows"
	OptPerspective      = "perspective"
	OptGamma            = "gamma"
	OptContrast         = "contrast"
	OptAutomapSize      = "automapsize"
	OptAutomapFade      = "automapfade"
	OptAutomapCenter    = "automapcenter"
	OptAutomapShowParty = "automapparty"
	OptAutomapNames     = "automapnames"

	// Accessibility (Esc -> Options -> Accessibility).
	OptUIScale     = "uiscale"     // window size multiplier, 1X..3X
	OptColorblind  = "colorblind"  // colour-blind friendly item colours
	OptSubtitleLog = "subtitlelog" // NPC speech subtitles are also written to the log

	// OptFullscreen is the FULL SCREEN row of the video page. The game starts full screen by default; choosing OFF is
	// remembered (index 1) so the next start opens a window.
	OptFullscreen = "fullscreen"
	// Cinematics: movies play on their own (intro, act changes, Baal) when ON; the Cinematics menu always plays.
	OptCinematics = "cinematics"
	// OptIntroSeen is not shown in a menu: the opening movie plays on the first launch only (0 = not yet).
	OptIntroSeen = "introseen"
	// OptPopupHireling is the registry value PopupHireling of the exe ("Diablo II" key, written by UI_DrawHirelingPortraitStrip 0x4904f0
	// after the first hire opened the mercenary screen by itself): 0 = not yet shown.
	OptPopupHireling = "popuphireling"
)

// OptionDef describes one row of the options menu.
type OptionDef struct {
	Key     string
	Values  []string
	Default int // index into Values
}

// OptionLevels is the number of steps of a slider option (0..10 = 0%..100%).
const OptionLevels = 11

func levels() []string {
	v := make([]string, OptionLevels)
	for i := range v {
		v[i] = strconv.Itoa(i*10) + "%"
	}

	return v
}

var (
	onOff = []string{"ON", "OFF"}
	yesNo = []string{"YES", "NO"}
)

// OptionDefs lists every option in menu order. The slider options (volumes,
// 3D bias, gamma, contrast) are shown in 10% steps. UNVERIFIED: the original
// draws these rows as slider bars; the step count is our choice.
func OptionDefs() []OptionDef {
	defs := baseOptionDefs()
	defs = append(defs,
		OptionDef{OptUIScale, []string{"1X", "2X", "3X"}, 0},
		OptionDef{OptColorblind, []string{"OFF", "ON"}, 0},
		OptionDef{OptSubtitleLog, []string{"OFF", "ON"}, 0},
		OptionDef{OptFullscreen, onOff, 0},
		OptionDef{OptCinematics, onOff, 0},
		OptionDef{OptIntroSeen, []string{"NO", "YES"}, 0},
		OptionDef{OptPopupHireling, []string{"NO", "YES"}, 0},
	)

	// one remap row per gamepad button: the value is the action name
	def := d2gamepad.DefaultMapping()
	names := d2gamepad.ActionNames()

	for _, b := range d2gamepad.Buttons() {
		defs = append(defs, OptionDef{d2gamepad.OptionKey(b), names, int(def[b])})
	}

	return defs
}

func baseOptionDefs() []OptionDef {
	return []OptionDef{
		{OptSound, levels(), 10},
		{OptMusic, levels(), 3},
		{Opt3DBias, levels(), 5},
		{OptHardwareAccel, onOff, 1},
		{OptEnvEffects, onOff, 0},
		{OptNpcSpeech, []string{"AUDIO AND TEXT", "AUDIO ONLY", "TEXT ONLY"}, 0},
		{OptResolution, []string{"800X600", "1024X768"}, 0},
		{OptLightingQuality, []string{"LOW", "HIGH"}, 1},
		{OptBlendedShadows, onOff, 0},
		{OptPerspective, onOff, 1},
		{OptGamma, levels(), 5},
		{OptContrast, levels(), 5},
		{OptAutomapSize, []string{"FULL SCREEN", "MINI"}, 0},
		{OptAutomapFade, yesNo, 0},
		{OptAutomapCenter, yesNo, 0},
		{OptAutomapShowParty, yesNo, 0},
		{OptAutomapNames, yesNo, 0},
	}
}

// OptionDefFor returns the definition of a key.
func OptionDefFor(key string) (OptionDef, bool) {
	for _, d := range OptionDefs() {
		if d.Key == key {
			return d, true
		}
	}

	return OptionDef{}, false
}

func volumeToIndex(v float64) int {
	i := int(v*float64(OptionLevels-1) + 0.5)
	if i < 0 {
		return 0
	}

	if i > OptionLevels-1 {
		return OptionLevels - 1
	}

	return i
}

// OptionIndex returns the chosen index of an option (the default when it was
// never set or the stored value is out of range).
func (c *Configuration) OptionIndex(key string) int {
	def, ok := OptionDefFor(key)
	if !ok {
		return 0
	}

	switch key {
	case OptSound:
		return volumeToIndex(c.SfxVolume)
	case OptMusic:
		return volumeToIndex(c.BgmVolume)
	}

	if i, set := c.Options[key]; set && i >= 0 && i < len(def.Values) {
		return i
	}

	return def.Default
}

// OptionValue returns the displayed value of an option.
func (c *Configuration) OptionValue(key string) string {
	def, ok := OptionDefFor(key)
	if !ok {
		return ""
	}

	return def.Values[c.OptionIndex(key)]
}

// SetOption stores a choice (it does not save the file).
func (c *Configuration) SetOption(key string, index int) error {
	def, ok := OptionDefFor(key)
	if !ok {
		return fmt.Errorf("unknown option %q", key)
	}

	if index < 0 || index >= len(def.Values) {
		return fmt.Errorf("option %q: index %d out of range", key, index)
	}

	switch key {
	case OptSound:
		c.SfxVolume = float64(index) / float64(OptionLevels-1)
	case OptMusic:
		c.BgmVolume = float64(index) / float64(OptionLevels-1)
	default:
		if c.Options == nil {
			c.Options = make(map[string]int)
		}

		c.Options[key] = index
	}

	return nil
}

// PadMapping is the gamepad button mapping chosen in the controls pages.
func (c *Configuration) PadMapping() d2gamepad.Mapping {
	return d2gamepad.MappingFromChoices(c.OptionValue)
}

// WantFullscreen reports whether the game should start full screen: the default, until the video page (or Cmd+Enter,
// Cmd+F, the console) chose a window.
func (c *Configuration) WantFullscreen() bool { return c.OptionIndex(OptFullscreen) == 0 }

// SetFullscreenChoice remembers the choice in both the FullScreen field and the option (it does not save the file).
func (c *Configuration) SetFullscreenChoice(full bool) {
	c.FullScreen = full

	i := 1
	if full {
		i = 0
	}

	_ = c.SetOption(OptFullscreen, i)
}

// UIScale is the chosen window size multiplier (1..3).
func (c *Configuration) UIScale() int { return c.OptionIndex(OptUIScale) + 1 }

// ColorBlind reports the colour-blind friendly item colours option.
func (c *Configuration) ColorBlind() bool { return c.OptionIndex(OptColorblind) == 1 }

// SubtitleLog reports the NPC speech subtitle log option.
func (c *Configuration) SubtitleLog() bool { return c.OptionIndex(OptSubtitleLog) == 1 }

// Cinematics reports whether the movies play on their own (default ON).
func (c *Configuration) Cinematics() bool { return c.OptionIndex(OptCinematics) == 0 }

// IntroSeen reports that the opening movie was already shown on an earlier launch.
func (c *Configuration) IntroSeen() bool { return c.OptionIndex(OptIntroSeen) == 1 }

// HirelingPopupSeen reports that the first-hire mercenary screen was already shown once (PopupHireling).
func (c *Configuration) HirelingPopupSeen() bool { return c.OptionIndex(OptPopupHireling) == 1 }
