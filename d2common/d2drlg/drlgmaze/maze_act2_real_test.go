package drlgmaze

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Every Act 2 maze level generates for many game seeds (with the two special tombs the game seed draws), and the
// real Tal Rasha tomb (TombA) is the only tomb that gets the chamber with the orifice. Needs D2_TABLES.
func TestAct2MazeLevelsGenerate(t *testing.T) {
	tb := realTables(t)

	levels := []int{47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 74}

	for seed := uint32(1); seed <= 12; seed++ {
		base, _ := d2rand.DrlgBaseSeed(seed)
		ex := d2drlg.DrawActExtras(seed, 1)

		for _, id := range levels {
			for diff := d2drlg.Difficulty(0); diff <= 2; diff++ {
				res, err := Generate(tb, Params{LevelID: id, Difficulty: diff, BaseSeed: base, TombA: ex.TombA, TombB: ex.TombB})
				if err != nil {
					t.Errorf("seed %d level %d difficulty %d: %v", seed, id, diff, err)

					continue
				}

				if len(res.Rooms) == 0 {
					t.Errorf("seed %d level %d: no rooms", seed, id)
				}
			}
		}
	}
}
