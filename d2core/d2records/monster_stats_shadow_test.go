package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// The shipped monstats.txt repeats the Id "cr_lancer8" in two rows (classes
// 617 and 723). The key map keeps the last row, the earlier one stays
// reachable as a shadowed record so its class index still resolves.
func TestMonsterStatsDuplicateKeyIsShadowed(t *testing.T) {
	data := []byte("Id\thcIdx\ncr_lancer8\t617\nother\t5\ncr_lancer8\t723\n")
	r := &RecordManager{}
	r.Logger = d2util.NewLogger()

	if err := monsterStatsLoader(r, d2txt.LoadDataDictionary(data)); err != nil {
		t.Fatal(err)
	}

	if got := r.Monster.Stats["cr_lancer8"]; got == nil || got.ID != 723 {
		t.Errorf("key map record %+v, want class 723", got)
	}

	if len(r.Monster.Shadowed) != 1 || r.Monster.Shadowed[0].ID != 617 {
		t.Errorf("shadowed %v, want only class 617", r.Monster.Shadowed)
	}
}
