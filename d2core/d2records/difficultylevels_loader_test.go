package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// The freeze and cold divisor columns (record +0x14 and +0x18 in the exe) are
// read by header name; the real file has Freeze before Cold.
func TestDifficultyLevelsDivisors(t *testing.T) {
	data := []byte("Name\tMonsterFreezeDivisor\tMonsterColdDivisor\nNormal\t1\t1\nNightmare\t2\t3\nHell\t4\t5\n")
	r := &RecordManager{}
	r.Logger = d2util.NewLogger()

	if err := difficultyLevelsLoader(r, d2txt.LoadDataDictionary(data)); err != nil {
		t.Fatal(err)
	}

	for diff, want := range map[d2enum.DifficultyType][2]int{
		d2enum.DifficultyNormal: {1, 1}, d2enum.DifficultyNightmare: {2, 3}, d2enum.DifficultyHell: {4, 5},
	} {
		rec := r.DifficultyLevels[diff]
		if rec.MonsterFreezeDivisor != want[0] || rec.MonsterColdDivisor != want[1] {
			t.Errorf("diff %d: freeze %d cold %d, want %v", diff, rec.MonsterFreezeDivisor, rec.MonsterColdDivisor, want)
		}
	}
}
