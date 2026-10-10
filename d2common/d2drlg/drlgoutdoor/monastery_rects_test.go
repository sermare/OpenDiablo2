package drlgoutdoor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

func monasteryTables(t *testing.T) *d2drlg.Tables {
	t.Helper()

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
		LvlPrest: rd("patch_d2/LvlPrest.txt"), LvlPrestBin: rd("bin/patch_d2/lvlprest.bin"), LvlSub: rd("patch_d2/LvlSub.txt")})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

// The Monastery levels linked by Warp -1 Vis slots must touch along a shared border (so that walking crosses
// them): 27 north of 26, 33 north of 32, and the Barracks on the exit side of 27 (0 west, 1 north, 2 east).
func TestMonasteryRectsTouch(t *testing.T) {
	tb := monasteryTables(t)
	want := map[int]d2level.Side{0: d2level.East, 1: d2level.South, 2: d2level.West} // side of 28 that touches 27

	for seed := uint32(1); seed <= 60; seed++ {
		for diff := d2drlg.Difficulty(0); diff <= 2; diff++ {
			rs, side, err := MonasteryRects(tb, seed, diff)
			if err != nil {
				t.Fatalf("seed %d diff %d: %v", seed, diff, err)
			}

			lv := func(r Rect) d2level.Rect { return d2level.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H} }

			for _, pair := range [][2]int{{26, 27}, {32, 33}, {28, 27}} {
				bd, ok := d2level.SharedBorder(lv(rs[pair[0]]), lv(rs[pair[1]]))
				if !ok {
					t.Errorf("seed %d diff %d side %d: %d does not touch %d: %v %v", seed, diff, side,
						pair[0], pair[1], rs[pair[0]], rs[pair[1]])

					continue
				}

				if pair[0] == 28 && bd.Side != want[side] {
					t.Errorf("seed %d side %d: barracks touch 27 on %v, want %v", seed, side, bd.Side, want[side])
				}
			}
		}
	}
}
