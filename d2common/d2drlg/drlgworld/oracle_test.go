package drlgworld

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
)

type oracleLevel struct {
	ID    int `json:"id"`
	X     int `json:"x"`
	Y     int `json:"y"`
	W     int `json:"w"`
	H     int `json:"h"`
	Data4 int `json:"data4"`
}

type oracleSeed struct {
	Seed   uint32                 `json:"seed"`
	Base   uint32                 `json:"base"`
	DrlgLo uint32                 `json:"drlgLo"`
	DrlgHi uint32                 `json:"drlgHi"`
	Levels map[string]oracleLevel `json:"levels"`
}

// TestOracleWorld compares Generate against golden layout data produced by
// emulating the real Game.exe DRLG (CreateDrlg, act 0) with unicorn. The golden
// file holds only derived numbers (see ~/git/drlg-oracle).
func TestOracleWorld(t *testing.T) {
	lv := levels(t)

	gp := os.Getenv("ORACLE_WORLD")
	if gp == "" {
		gp = filepath.Join("..", "testdata", "world_act1_normal.json")
	}

	b, err := os.ReadFile(gp)
	if err != nil {
		t.Skip(err)
	}

	var gold []oracleSeed
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	bad := map[string]int{}

	for _, g := range gold {
		l, err := Generate(lv, g.Seed, d2drlg.Normal)
		if err != nil {
			t.Fatal(err)
		}

		for _, id := range []int{4, 3, 2, 1, 0x11, 0x27, 0x1a, 7, 6, 5} {
			want := g.Levels[itoa(id)]
			got := l.Levels[id].Rect

			if got.X != want.X || got.Y != want.Y || got.W != want.W || got.H != want.H {
				bad["rect"]++
				t.Errorf("seed %#x level %d: go %+v oracle (%d,%d,%d,%d)", g.Seed, id, got, want.X, want.Y, want.W, want.H)
			}
		}

		if l.TownFile != g.Levels["1"].Data4 {
			bad["townfile"]++
			t.Errorf("seed %#x town file: go %d oracle %d", g.Seed, l.TownFile, g.Levels["1"].Data4)
		}

		if l.BarracksExitSide >= 0 && l.BarracksExitSide != g.Levels["27"].Data4 {
			bad["l27"]++
			t.Errorf("seed %#x level 27 exit side: go %d oracle %d", g.Seed, l.BarracksExitSide, g.Levels["27"].Data4)
		}

		if lo, hi := l.DrlgSeed.Lo, l.DrlgSeed.Hi; lo != g.DrlgLo || hi != g.DrlgHi {
			bad["seed"]++
			t.Errorf("seed %#x drlg seed after search: go %#x:%#x oracle %#x:%#x", g.Seed, lo, hi, g.DrlgLo, g.DrlgHi)
		}
	}

	t.Logf("mismatch counts over %d seeds: %v", len(gold), bad)
}

func itoa(i int) string {
	b, _ := json.Marshal(i)

	return string(b)
}
