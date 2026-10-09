package d2input

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestResolveWindowShortcut(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		cmd, alt bool
		key      d2enum.Key
		want     WindowAction
	}{
		{"cmd+enter", "darwin", true, false, d2enum.KeyEnter, WindowToggleFullscreen},
		{"option+enter", "darwin", false, true, d2enum.KeyEnter, WindowToggleFullscreen},
		{"plain enter (chat)", "darwin", false, false, d2enum.KeyEnter, WindowNone},
		{"cmd+q", "darwin", true, false, d2enum.KeyQ, WindowQuit},
		{"cmd+w", "darwin", true, false, d2enum.KeyW, WindowQuit},
		{"q alone is the quest log", "darwin", false, false, d2enum.KeyQ, WindowNone},
		{"w alone is weapon swap", "darwin", false, false, d2enum.KeyW, WindowNone},
		{"option+q does not quit", "darwin", false, true, d2enum.KeyQ, WindowNone},
		{"cmd+i is not ours", "darwin", true, false, d2enum.KeyI, WindowNone},
		{"alt+enter elsewhere", "linux", false, true, d2enum.KeyEnter, WindowToggleFullscreen},
		{"cmd does not exist elsewhere", "linux", true, false, d2enum.KeyQ, WindowNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveWindowShortcut(tt.goos, tt.cmd, tt.alt, tt.key); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
