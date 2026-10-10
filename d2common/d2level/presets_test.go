package d2level

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPresetTileDestinations(t *testing.T) {
	for _, c := range []struct{ level, style, want int }{
		{40, 2, 47}, {40, 3, 47}, {40, 4, 50}, {50, 0, 40}, {50, 2, 51}, {50, 3, 51},
	} {
		if got, ok := PresetTileDestination(c.level, c.style); !ok || got != c.want {
			t.Errorf("level %d style %d: %d %v, want %d", c.level, c.style, got, ok, c.want)
		}
	}

	for _, c := range []struct{ level, style int }{{40, 0}, {40, 30}, {40, 31}, {50, 1}, {51, 0}} {
		if got, ok := PresetTileDestination(c.level, c.style); ok {
			t.Errorf("level %d style %d must not lead anywhere, got %d", c.level, c.style, got)
		}
	}

	if !IsActPresetLevel(50) || IsActPresetLevel(40) {
		t.Error("level 50 is a preset level, the town is not counted")
	}
}

// With D2_TABLES set the table of Act 2 Vis slots equals the Vis0-7 columns of the real Levels.txt.
func TestAct2VisSlotsMatchLevelsTxt(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(root, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(string(data), "\n")
	head := strings.Split(strings.TrimRight(lines[0], "\r"), "\t")
	col := map[string]int{}

	for i, h := range head {
		col[h] = i
	}

	seen := 0

	for _, l := range lines[1:] {
		f := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(f) <= col["Vis7"] {
			continue
		}

		level, err := strconv.Atoi(f[col["Id"]])
		if err != nil || level < 40 || level > 74 {
			continue
		}

		for slot := 0; slot < 8; slot++ {
			vis, _ := strconv.Atoi(f[col["Vis"+strconv.Itoa(slot)]])
			got, ok := Act2VisDestination(level, slot)

			if got != vis || ok != (vis != 0) {
				t.Errorf("level %d slot %d: Levels.txt says %d, the table %d %v", level, slot, vis, got, ok)
			}
		}

		seen++
	}

	if seen < 30 {
		t.Errorf("only %d Act 2 levels compared", seen)
	}
}

// The two entrance tiles of the Burial Grounds lead to the Crypt and the Mausoleum (found by walkto:exit=18 failing).
func TestBurialGroundsEntrances(t *testing.T) {
	for _, c := range []struct{ style, want int }{{0, 18}, {1, 19}} {
		if got, ok := TileDestination(17, c.style); !ok || got != c.want {
			t.Errorf("style %d: %d %v, want %d", c.style, got, ok, c.want)
		}
	}

	if _, ok := TileDestination(17, 5); ok {
		t.Error("style 5 of the Burial Grounds must not lead anywhere")
	}
}
