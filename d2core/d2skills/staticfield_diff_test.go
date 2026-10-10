package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Static Field stops at the StaticFieldMin percent of the monster's own
// difficulty (DifficultyLevels.txt: 0, 33, 50), not always the Normal row.
func TestStaticFieldFloorFollowsMonsterDifficulty(t *testing.T) {
	recs := d2records.DifficultyLevels{
		d2enum.DifficultyNormal:    {StaticFieldMin: 0},
		d2enum.DifficultyNightmare: {StaticFieldMin: 33},
		d2enum.DifficultyHell:      {StaticFieldMin: 50},
	}
	e := &Engine{asset: &d2asset.AssetManager{Records: &d2records.RecordManager{DifficultyLevels: recs}}}

	for diff, want := range map[d2monster.Difficulty]int{d2monster.Normal: 0, d2monster.Nightmare: 33, d2monster.Hell: 50} {
		m := &d2mapentity.Monster{}
		m.Vitals.Difficulty = diff

		if got := e.staticFieldFloor(m, 7); got != want {
			t.Errorf("%v: floor %d, want %d", diff, got, want)
		}
	}

	// no table: the caller's default
	if got := (&Engine{}).staticFieldFloor(&d2mapentity.Monster{}, 7); got != 7 {
		t.Errorf("no table: floor %d, want 7", got)
	}
}
