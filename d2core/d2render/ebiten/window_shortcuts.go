package ebiten

import (
	"errors"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input"
)

// errQuit ends ebiten.RunGame from Update (ebiten 2.0 has no Termination value);
// Run turns it into a clean exit.
var errQuit = errors.New("quit requested from the keyboard")

// SetFullscreenHook registers a function called with the new state after the
// keyboard toggled full screen (so the choice can be saved).
func (r *Renderer) SetFullscreenHook(f func(fullscreen bool)) { r.fullscreenHook = f }

// handleWindowShortcuts runs once per tick: Cmd+Enter / Option+Enter toggle
// full screen, Cmd+Q / Cmd+W quit (macOS). See d2input.ResolveWindowShortcut.
func (r *Renderer) handleWindowShortcuts() error {
	alt := ebiten.IsKeyPressed(ebiten.KeyAlt)
	cmd := commandHeld()

	if !alt && !cmd {
		return nil
	}

	for _, k := range []struct {
		ebiten ebiten.Key
		key    d2enum.Key
	}{{ebiten.KeyEnter, d2enum.KeyEnter}, {ebiten.KeyQ, d2enum.KeyQ}, {ebiten.KeyW, d2enum.KeyW}} {
		if !inpututil.IsKeyJustPressed(k.ebiten) {
			continue
		}

		switch d2input.ResolveWindowShortcut(runtime.GOOS, cmd, alt, k.key) {
		case d2input.WindowToggleFullscreen:
			full := !ebiten.IsFullscreen()
			ebiten.SetFullscreen(full)

			if r.fullscreenHook != nil {
				r.fullscreenHook(full)
			}
		case d2input.WindowQuit:
			return errQuit
		case d2input.WindowNone:
		}
	}

	return nil
}
