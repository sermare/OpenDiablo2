package d2mapgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// Playtest bug (Act 3): the seamless borders of Kurast Docks, the three jungle levels and the Kurast column
// were not in the link table, so no hero could leave the town. For every seed there is a chain of touching
// rectangles from the town to Travincal, and each hop is found by EdgeExit from both sides.
func TestAct3WorldBordersForEverySeed(t *testing.T) {
	tb := act2Tables(t)

	for seed := uint32(1); seed <= 40; seed++ {
		rects := act23Rects(tb, d2level.KurastDocks, seed*7919, d2drlg.Normal)
		if rects == nil {
			t.Fatalf("seed %d: no Act 3 world", seed)
		}

		// breadth first over the edge links from the town
		seen := map[int]bool{d2level.KurastDocks: true}
		queue := []int{d2level.KurastDocks}

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			for _, n := range d2level.EdgeNeighbors(cur) {
				bd, ok := d2level.SharedBorder(rects[cur], rects[n])
				if !ok || seen[n] {
					continue
				}

				// the crossing is found from the side of cur
				if to, ok := d2level.EdgeExit(rects, cur, midOf(bd, rects[cur]).x, midOf(bd, rects[cur]).y, 1.5); !ok || (to != n && !touches(rects, cur, to)) {
					t.Errorf("seed %d: no exit from %d towards %d at its border (%v %d)", seed, cur, n, ok, to)
				}

				seen[n] = true
				queue = append(queue, n)
			}
		}

		for id := 75; id <= 83; id++ {
			if !seen[id] {
				t.Errorf("seed %d: level %d cannot be reached over the borders from Kurast Docks (rects %v)", seed, id, rects)
			}
		}
	}
}

func touches(rects map[int]d2level.Rect, a, b int) bool {
	_, ok := d2level.SharedBorder(rects[a], rects[b])

	return ok
}

// Playtest bug (Act 3): Spider Cavern, the Flayer Dungeons, the Sewers, the Temples and the Durance of Hate
// were not built (the maze provider stopped at level 72, and the temples are DrlgType 2 presets). Every
// Act 3 maze level generates, every other one is a preset level the preset provider covers.
func TestAct3DungeonsAreBuildable(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rd := func(p string) []byte {
		b, err := os.ReadFile(filepath.Join(root, "drlg", p))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: rd("patch_d2/Levels.txt"), LvlMaze: rd("patch_d2/LvlMaze.txt"),
		LvlPrest: rd("patch_d2/LvlPrest.txt"), LvlPrestBin: rd("bin/patch_d2/lvlprest.bin"), LvlTypes: rd("patch_d2/LvlTypes.txt"),
		LvlSub: rd("patch_d2/LvlSub.txt")})
	if err != nil {
		t.Fatal(err)
	}

	base := uint32(0x101d574a)

	for id := 84; id <= 102; id++ {
		rec, ok := tb.Level(id)
		if !ok {
			t.Fatalf("level %d missing", id)
		}

		if rec.DrlgType == 2 {
			if !isPresetLevel(id) {
				t.Errorf("level %d is a preset level but the preset provider does not cover it", id)
			}

			continue
		}

		if id > maxMazeLevel {
			t.Errorf("level %d lies beyond the last maze level %d", id, maxMazeLevel)
		}

		if _, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: id, BaseSeed: base}); err != nil {
			t.Errorf("level %d (%s): %v", id, rec.Name, err)
		}
	}

	for _, id := range []int{90, 91, 93, 94, 95, 96, 97, 98, 99, 102} {
		if !isAct3Preset(id) {
			t.Errorf("level %d should be an Act 3 preset level", id)
		}
	}

	if isAct3Preset(92) || isAct3Preset(100) || isAct3Preset(85) {
		t.Errorf("a maze level is listed as a preset level")
	}
}
