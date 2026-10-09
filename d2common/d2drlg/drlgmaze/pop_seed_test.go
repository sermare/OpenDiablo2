package drlgmaze

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type popSeedRoom struct {
	X, Y, W, H int
	Seed       [2]uint32 `json:"seed"`
}

type popSeedLevel struct {
	Act   int           `json:"act"`
	Seed  uint32        `json:"seed"`
	Diff  int           `json:"diff"`
	Level int           `json:"level"`
	Rooms []popSeedRoom `json:"rooms"`
}

// TestChunkRoomSeeds checks the room seed of every maze chunk (the seed the
// DS1 filter of a lazily loaded preset and the room's population start from)
// against the DrlgRooms of the emulated game.
func TestChunkRoomSeeds(t *testing.T) {
	tb := realTables(t)

	raw, err := os.ReadFile(filepath.Join("..", "testdata", "pop_presets.json"))
	if err != nil {
		t.Skip(err)
	}

	var gold []popSeedLevel
	if err := json.Unmarshal(raw, &gold); err != nil {
		t.Fatal(err)
	}

	checked, bad := 0, 0

	for _, lv := range gold {
		rec, ok := tb.Level(lv.Level)
		if !ok || rec.DrlgType != 1 || lv.Level == 28 {
			continue // mazes only (the barracks joint needs the world layout)
		}

		base, _ := d2rand.DrlgBaseSeed(lv.Seed)

		ex := d2drlg.DrawActExtras(lv.Seed, lv.Act)

		res, err := Generate(tb, Params{LevelID: lv.Level, Difficulty: 0, BaseSeed: base, TombA: ex.TombA, TombB: ex.TombB})
		if err != nil {
			continue
		}

		want := map[string][2]uint32{}
		for _, r := range lv.Rooms {
			want[fmt.Sprintf("%d,%d,%d,%d", r.X, r.Y, r.W, r.H)] = r.Seed
		}

		for _, c := range res.Chunks {
			key := fmt.Sprintf("%d,%d,%d,%d", c.X, c.Y, c.W, c.H)
			if w, ok := want[key]; ok {
				checked++

				if got := [2]uint32{c.Seed.Lo, c.Seed.Hi}; got != w {
					bad++

					if bad < 10 {
						t.Errorf("level %d chunk %s: seed %v, want %v", lv.Level, key, got, w)
					}
				}
			}
		}
	}

	if checked == 0 {
		t.Skip("no maze level in the golden")
	}

	t.Logf("%d chunk seeds compared, %d mismatches", checked, bad)
}
