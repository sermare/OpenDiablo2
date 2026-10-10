package d2mapgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
)

// TestEveryLevelHasAGenerator: each level of Levels.txt (2..132, the Acts 1-5 rows) is built by one of the
// providers: DrlgType 1 by the maze generator, 2 by the preset provider, 3 by the outdoor provider. The
// town levels are built by their own providers. Found by the 9s walk scenarios (Forgotten Tower, level 20,
// DrlgType 2, was refused). Needs D2_TABLES (the drlg tables).
func TestEveryLevelHasAGenerator(t *testing.T) {
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

	towns := map[int]bool{1: true, 40: true, 75: true, 103: true, 109: true}
	base := uint32(0x101d574a)

	for id := 2; id <= 132; id++ {
		rec, ok := tb.Level(id)
		if !ok || towns[id] {
			continue
		}

		switch rec.DrlgType {
		case 1:
			if _, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: id, BaseSeed: base}); err != nil {
				t.Errorf("level %d (%s): maze generator: %v", id, rec.Name, err)
			}
		case 2:
			if !isPresetLevel(id) {
				if _, ok := actPresetPrest[id]; !ok {
					t.Errorf("level %d (%s) is DrlgType 2 but no provider builds it", id, rec.Name)
				}
			}
		case 3:
			if !isOutdoorLevel(id) {
				t.Errorf("level %d (%s) is DrlgType 3 but the outdoor provider does not build it", id, rec.Name)
			}
		}
	}
}
