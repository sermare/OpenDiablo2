package d2setup

import (
	"path/filepath"
	"testing"
)

func TestFindInstallsSteamAndBottles(t *testing.T) {
	home := t.TempDir()
	steam := filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "common", "Diablo II")
	whisky := filepath.Join(home, "Library", "Containers", "com.isaacmarovitz.Whisky", "Bottles", "ABC",
		"drive_c", "Program Files (x86)", "Diablo II")
	wine := filepath.Join(home, ".wine", "drive_c", "Program Files (x86)", "Diablo II")

	for _, d := range []string{steam, whisky, wine} {
		fillGame(t, d, CoreMPQs)
	}

	if got := FindInstalls(home); len(got) != 3 {
		t.Fatalf("want 3 installs, got %+v", got)
	}
}
