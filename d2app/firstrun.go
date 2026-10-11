package d2app

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/OpenDiablo2/OpenDiablo2/d2app/d2setup"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2display"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2config"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

const (
	maxWindowScale = 4
	// windowChromeHeight is the room kept for the menu bar and title bar when the start window is
	// limited to the monitor height.
	windowChromeHeight = 80
)

// firstRunSetup makes sure the Diablo II files are configured (asking the
// user through native dialogs when they are not) and tells the hero factory
// where real .d2s characters can be imported from. OD2_NO_SETUP=1 skips the
// dialogs and keeps the old behaviour (an error screen for missing files).
func (a *App) firstRunSetup() error {
	home, _ := os.UserHomeDir()

	if os.Getenv("OD2_NO_SETUP") == "" {
		ui := d2setup.DefaultUI()

		if err := d2setup.EnsureGameFiles(a.config, ui, home); err != nil {
			if errors.Is(err, d2setup.ErrNoGameFiles) {
				ui.Alert("OpenDiablo2", "OpenDiablo2 needs the game files from your own copy of "+
					"Diablo II and Lord of Destruction (d2data.mpq, d2char.mpq, patch_d2.mpq and so on).\n\n"+
					"Open OpenDiablo2 again and pick the folder that contains them.")
			} else {
				ui.Alert("OpenDiablo2", "Could not save the settings: "+err.Error())
			}

			return fmt.Errorf("%w: %v", d2setup.ErrReported, err)
		}
	}

	dirs := d2setup.ImportDirs(a.config, home)
	d2hero.SetImportDirs(dirs)

	for _, d := range dirs {
		a.Infof("importing real Diablo II characters from %s (originals are only read)", d)
	}

	return nil
}

// windowScale is the start window size multiplier from config.json (1..4).
// displaySize resolves the logical start size and the integer UI scale: OD2_DISPLAY=WxH and
// OD2_UI_SCALE=1..4 win over the config (Display section, UIScale option, WindowScale), which wins over the
// default 1512x982 limited to the monitor. It stores both in the display model.
func (a *App) displaySize() (d2display.Size, int) {
	var screen d2display.Size

	if r, ok := a.renderer.(interface{ ScreenSize() (int, int) }); ok {
		screen.W, screen.H = r.ScreenSize()
		// keep room for the menu bar and the title bar of a window
		if screen.H > 0 {
			screen.H -= windowChromeHeight
		}
	}

	size, err := d2display.Resolve(os.Getenv("OD2_DISPLAY"), d2display.Size{
		W: a.config.Display.Width, H: a.config.Display.Height}, screen)
	if err != nil {
		a.Warning(err.Error())
	}

	scale := a.windowScale()
	if v, err := strconv.Atoi(os.Getenv("OD2_UI_SCALE")); err == nil && v >= 1 && v <= d2display.MaxUIScale {
		scale = v
	}

	d2display.Set(size)
	d2display.SetScale(scale)
	// an explicit OD2_DISPLAY pins the logical size whatever the window turns out to be
	d2display.SetFixed(err == nil && os.Getenv("OD2_DISPLAY") != "")
	a.Infof("display %dx%d, ui scale %d", size.W, size.H, scale)

	return size, scale
}

func (a *App) windowScale() int {
	// the accessibility option (Esc -> Options -> Accessibility) wins once it was chosen
	if i, set := a.config.Options[d2config.OptUIScale]; set && i >= 0 && i < 3 {
		return i + 1
	}

	if s := a.config.WindowScale; s >= 1 && s <= maxWindowScale {
		return s
	}

	return 1
}

func (a *App) saveConfig() {
	if err := a.config.Save(); err != nil {
		a.terminal.Errorf("could not save settings: %v", err)
	}
}

func (a *App) setVolume(args []string, bgm bool) error {
	v, err := strconv.ParseFloat(args[0], 64)
	if err != nil || v < 0 || v > 1 {
		a.terminal.Errorf("volume must be a number from 0 to 1")
		return nil
	}

	if bgm {
		a.config.BgmVolume = v
	} else {
		a.config.SfxVolume = v
	}

	a.audio.SetVolumes(a.config.BgmVolume, a.config.SfxVolume)
	a.saveConfig()
	a.terminal.Infof("music volume %.2f, sound volume %.2f (saved)", a.config.BgmVolume, a.config.SfxVolume)

	return nil
}

func (a *App) setWindowScale(args []string) error {
	s, err := strconv.Atoi(args[0])
	if err != nil || s < 1 || s > maxWindowScale {
		a.terminal.Errorf("window scale must be a whole number from 1 to %d", maxWindowScale)
		return nil
	}

	a.config.WindowScale = s
	a.saveConfig()
	a.terminal.Infof("window scale %d saved; it applies the next time the game starts", s)

	return nil
}
