package d2level

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Playtest bug (Act 3): the entrance presets of the jungle and Kurast led to level 0 (Spider Cavern, Flayer
// Dungeon, Sewers, Temples were unreachable): their tile styles are the Vis slot numbers, not LvlWarp ids.
func TestAct3SlotDestination(t *testing.T) {
	for _, c := range []struct {
		level, style, want int
		ok                 bool
	}{
		{76, 1, 85, true}, // Spider Forest -> Spider Cavern
		{78, 0, 86, true}, // Flayer Jungle -> Flayer Dungeon 1
		{81, 2, 96, true}, // Upper Kurast -> Temple
		{80, 1, 92, true}, // Kurast Bazaar -> Sewers 1 (second entrance)
		{83, 0, 100, true},
		{83, 1, 0, false},
		{77, 0, 0, false}, // the Great Marsh has no dungeon
	} {
		if got, ok := Act3SlotDestination(c.level, c.style); ok != c.ok || got != c.want {
			t.Errorf("Act3SlotDestination(%d, %d) = %d %v, want %d %v", c.level, c.style, got, ok, c.want, c.ok)
		}
	}
}

// Playtest bug (Act 3): in Spider Cavern the exit tile (style 1 in the lair presets) led nowhere, so the hero
// could not leave the cave. A dungeon with a way out and no way down returns up by any low style; the other
// Act 3 dungeons resolve by the up/down rule.
func TestAct3DungeonTileDestination(t *testing.T) {
	for _, c := range []struct{ level, style, want int }{
		{85, 1, 76}, {85, 0, 76}, {84, 1, 76},
		{86, 0, 78},   // Swampy Pit 1: up to Flayer Jungle
		{86, 1, 87},   // the next stairs
		{92, 0, 80},   // Sewers 1 up
		{100, 3, 83},  // Durance 1 up to Travincal (Prev files carry the styles 2/3)
		{100, 0, 101}, // and down (Next files carry 0/1)
		{94, 0, 80},   // a temple returns to the Kurast level that leads there
	} {
		if got, ok := TileDestination(c.level, c.style); !ok || got != c.want {
			t.Errorf("TileDestination(%d, %d) = %d %v, want %d", c.level, c.style, got, ok, c.want)
		}
	}
}

// With D2_TABLES the table agrees with the Vis columns of the real Levels.txt (every slot, in order).
func TestAct3SlotsMatchLevelsTxt(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	b, err := os.ReadFile(filepath.Join(root, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	lines := strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n")
	col := map[string]int{}

	for i, h := range strings.Split(lines[0], "\t") {
		col[h] = i
	}

	for _, ln := range lines[1:] {
		f := strings.Split(ln, "\t")
		if len(f) < col["Warp7"] {
			continue
		}

		id, err := strconv.Atoi(f[col["Id"]])
		if err != nil || id < 76 || id > 83 {
			continue
		}

		for k := 0; k < 8; k++ {
			vis, _ := strconv.Atoi(f[col["Vis"+strconv.Itoa(k)]])
			warp, _ := strconv.Atoi(f[col["Warp"+strconv.Itoa(k)]])
			got, ok := Act3SlotDestination(id, k)

			// the slots that lead to a dungeon (warp >= 0) are the ones in the table
			if warp >= 0 && vis != 0 && (!ok || got != vis) {
				t.Errorf("level %d slot %d: Levels.txt says %d, table says %d %v", id, k, vis, got, ok)
			}

			if (warp < 0 || vis == 0) && ok {
				t.Errorf("level %d slot %d is in the table but empty in Levels.txt", id, k)
			}
		}
	}
}

// Numeric golden for the Durance of Hate tile rule: the special tile styles measured in the DS1 files (oracle copy of
// the game's d2data, wall layers of the Special tile types) against the Levels.txt Vis slots. Level 100 (Durance 1)
// has Prev (styles 2/3) and Next (0/1) files; level 101 the same plus a waypoint file with no special tile; MephComp
// (level 102) carries style 3.
func TestDuranceTileStyles(t *testing.T) {
	for _, c := range []struct {
		file         string
		level, style int
		want         int
	}{
		{"mephnwarpd", 100, 0, 101}, {"mephewarpd", 100, 0, 101}, {"mephswarpd", 100, 1, 101}, {"mephwwarpd", 100, 1, 101},
		{"mephnwarpu", 100, 2, 83}, {"mephewarpu", 100, 2, 83}, {"mephswarpu", 100, 3, 83}, {"mephwwarpu", 100, 3, 83},
		{"mephnwarpd", 101, 0, 102}, {"mephswarpd", 101, 1, 102},
		{"mephnwarpu", 101, 2, 100}, {"mephwwarpu", 101, 3, 100},
		{"mephcomp", 102, 3, 101},
	} {
		if got, ok := TileDestination(c.level, c.style); !ok || got != c.want {
			t.Errorf("%s in level %d: style %d -> %d %v, want %d", c.file, c.level, c.style, got, ok, c.want)
		}
	}

	if _, ok := TileDestination(102, 0); ok {
		t.Error("level 102 has no Vis0")
	}
}
