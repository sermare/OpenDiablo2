package d2records

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// TestRealMonsterResists pins the resist columns of the effective (patch_d2)
// monstats.txt: ResDm, ResMa, ResFi, ResLi, ResCo, ResPo and their (N) / (H)
// variants. The d2data/d2exp copies use the older FireResist style headers
// and are overridden by patch_d2, so only patch_d2 is read. Skipped without D2_TABLES.
func TestRealMonsterResists(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	rm := &RecordManager{Logger: d2util.NewLogger()}
	if err = monsterStatsLoader(rm, d2txt.LoadDataDictionary(buf)); err != nil {
		t.Fatal(err)
	}

	// {phys, magic, fire, light, cold, poison} per difficulty, read off the table
	type row [3][6]int

	tests := []struct {
		id   string
		want row
	}{
		{"skeleton1", row{{0, 0, 0, 0, 0, 50}, {0, 0, 0, 0, 0, 75}, {33, 0, 0, 100, 0, 75}}},
		{"andariel", row{{0, 0, -50, 50, 50, 80}, {0, 0, -50, 50, 50, 50}, {66, 0, -50, 66, 66, 66}}},
		{"duriel", row{{0, 0, 20, 20, 50, 20}, {0, 0, 50, 50, 75, 50}, {50, 33, 75, 75, 95, 75}}},
		{"izual", row{{30, 30, 30, 30, 75, 30}, {30, 30, 30, 30, 75, 30}, {30, 30, 30, 30, 75, 30}}},
		{"corruptrogue1", row{{0, 0, 0, 0, 0, 0}, {0, 0, 0, 0, 0, 0}, {20, 20, 33, 33, 100, 20}}},
	}

	for _, tt := range tests {
		r := rm.Monster.Stats[tt.id]
		if r == nil {
			t.Errorf("%s: no monstats row", tt.id)

			continue
		}

		got := row{
			{r.ResistancePhysicalNormal, r.ResistanceMagicNormal, r.ResistanceFireNormal, r.ResistanceLightningNormal,
				r.ResistanceColdNormal, r.ResistancePoisonNormal},
			{r.ResistancePhysicalNightmare, r.ResistanceMagicNightmare, r.ResistanceFireNightmare,
				r.ResistanceLightningNightmare, r.ResistanceColdNightmare, r.ResistancePoisonNightmare},
			{r.ResistancePhysicalHell, r.ResistanceMagicHell, r.ResistanceFireHell, r.ResistanceLightningHell,
				r.ResistanceColdHell, r.ResistancePoisonHell},
		}
		if got != tt.want {
			t.Errorf("%s: got %v want %v", tt.id, got, tt.want)
		}
	}
}
