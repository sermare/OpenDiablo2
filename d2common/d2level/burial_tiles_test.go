package d2level

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Numeric golden of the exit tiles measured in the DS1 files (wall layers, tile type special, style = bits 20..25
// of the cell) and the rule of D2MOO DRLGPRESET_BuildPresetArea / sub_6FD77BB0: style k < 8 -> Vis[k] of Levels.txt.
//
//	gravey.ds1 (Burial Grounds, LvlPrest 108): style 1 at (11,6) the north one, style 0 at (12,27) the south one
//	tempEnter.ds1 (Temple Entrance, LvlPrest 1088): style 1 at (11,6) and (12,6); style 9 at (9,4) and (14,13) and
//	style 30 at (13,25) are doors, not exits.
func TestBurialAndTempleExitTilesAreVisSlots(t *testing.T) {
	for _, c := range []struct {
		file         string
		level, style int
		want         int
		ok           bool
	}{
		{"gravey (12,27)", 17, 0, 18, true},
		{"gravey (11,6)", 17, 1, 19, true},
		{"tempEnter (11,6)", 121, 1, 122, true},
		{"tempEnter (12,6)", 121, 1, 122, true},
		{"tempEnter door (9,4)", 121, 9, 0, false},
		{"tempEnter (13,25)", 121, 30, 0, false},
		{"tempEnter slot 0 is empty", 121, 0, 0, false},
	} {
		got, ok := TileDestination(c.level, c.style)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: level %d style %d -> %d %v, want %d %v", c.file, c.level, c.style, got, ok, c.want, c.ok)
		}
	}
}

// With D2_TABLES set, TileDestination of levels 17 and 121 equals the Vis column of Levels.txt for every slot.
func TestBurialAndTempleVisMatchLevelsTxt(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(root, "drlg", "d2exp", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(string(data), "\n")
	col := map[string]int{}

	for i, h := range strings.Split(strings.TrimRight(lines[0], "\r"), "\t") {
		col[h] = i
	}

	checked := 0

	for _, l := range lines[1:] {
		f := strings.Split(strings.TrimRight(l, "\r"), "\t")
		if len(f) <= col["Vis7"] {
			continue
		}

		level, err := strconv.Atoi(f[col["Id"]])
		if err != nil || (level != 17 && level != 121) {
			continue
		}

		for slot := 0; slot < 8; slot++ {
			if level == 17 && slot > 5 {
				continue // 6 and 7 are LvlWarp ids of the cave rule, not DS1 styles (see level_test.go)
			}

			vis, _ := strconv.Atoi(f[col["Vis"+strconv.Itoa(slot)]])
			got, ok := TileDestination(level, slot)

			if ok != (vis != 0) || got != vis {
				t.Errorf("level %d slot %d: Levels.txt Vis %d, TileDestination %d %v", level, slot, vis, got, ok)
			}
		}

		checked++
	}

	if checked != 2 {
		t.Errorf("checked %d levels, want 2", checked)
	}
}
