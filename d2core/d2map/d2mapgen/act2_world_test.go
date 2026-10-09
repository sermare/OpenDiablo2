package d2mapgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// act2Tables reads the DRLG tables from D2_TABLES (the extracted game tables; skipped when unset).
func act2Tables(t *testing.T) *d2drlg.Tables {
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

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: rd("patch_d2/Levels.txt"), LvlPrest: rd("patch_d2/LvlPrest.txt"),
		LvlPrestBin: rd("bin/patch_d2/lvlprest.bin"), LvlSub: rd("patch_d2/LvlSub.txt")})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

// Playtest bug: the Act 2 levels had no world rectangles, so the desert levels could not be walked from one
// to the next. For every seed the chain Lut Gholein .. Valley of Snakes shares a border segment, and the
// town can leave through the side Rocky Waste lies on (LutW: west, LutN: north).
func TestAct2WorldBordersForEverySeed(t *testing.T) {
	tb := act2Tables(t)

	for seed := uint32(1); seed <= 40; seed++ {
		rects := act23Rects(tb, d2level.LutGholein, seed*7919, d2drlg.Normal)
		if rects == nil {
			t.Fatalf("seed %d: no Act 2 world", seed)
		}

		for _, a := range []int{40, 41, 42, 43, 44} {
			b := a + 1

			bd, ok := d2level.SharedBorder(rects[a], rects[b])
			if !ok {
				t.Errorf("seed %d: levels %d %v and %d %v share no border", seed, a, rects[a], b, rects[b])
				continue
			}

			// the crossing is found from both sides and arrives on the same world line
			if to, ok := d2level.EdgeExit(rects, a, midOf(bd, rects[a]).x, midOf(bd, rects[a]).y, 1.5); !ok || to != b {
				t.Errorf("seed %d: no exit from %d towards %d at its border (%v %d)", seed, a, b, ok, to)
			}
		}
	}
}

type pt struct{ x, y float64 }

// midOf is a point of rect a just inside the middle of the border bd.
func midOf(bd d2level.Border, a d2level.Rect) pt {
	mid := float64(bd.From+bd.To) / 2

	switch bd.Side {
	case d2level.West:
		return pt{float64(bd.Pos) + 0.5, mid}
	case d2level.East:
		return pt{float64(bd.Pos) - 0.5, mid}
	case d2level.North:
		return pt{mid, float64(bd.Pos) + 0.5}
	default:
		return pt{mid, float64(bd.Pos) - 0.5}
	}
}
