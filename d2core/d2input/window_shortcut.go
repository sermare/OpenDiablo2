package d2input

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"

// WindowAction is something the window (not the game) does for a key combination.
type WindowAction int

// Window actions.
const (
	WindowNone WindowAction = iota
	// WindowToggleFullscreen switches between a window and full screen.
	WindowToggleFullscreen
	// WindowQuit closes the game (the hero is saved on the way out).
	WindowQuit
)

// ResolveWindowShortcut maps a key that was just pressed, with the state of the
// Command and Option/Alt keys, to a window action. On macOS: Cmd+Enter and
// Option+Enter toggle full screen, Cmd+Q and Cmd+W quit. Elsewhere only
// Alt+Enter toggles full screen (Cmd does not exist).
func ResolveWindowShortcut(goos string, cmd, alt bool, key d2enum.Key) WindowAction {
	if key == d2enum.KeyEnter && (alt || (goos == "darwin" && cmd)) {
		return WindowToggleFullscreen
	}

	if goos == "darwin" && cmd && (key == d2enum.KeyQ || key == d2enum.KeyW) {
		return WindowQuit
	}

	return WindowNone
}
