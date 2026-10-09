package d2monster

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// TestRealSpawnData checks the group planner against the real tables: the
// Countess is corruptrogue3 with 6 followers drawn from its minion column
// (monster-ai-2.md section 2), and ordinary groups respect MinGrp/MaxGrp.
// Skipped without D2_TABLES.
func TestRealSpawnData(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	ms, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	su, err := os.ReadFile(filepath.Join(dir, "itemgen", "patch_d2", "SuperUniques.txt"))
	if err != nil {
		t.Skip(err)
	}

	classes, err := LoadTxtClasses(ms)
	if err != nil {
		t.Fatal(err)
	}

	supers, err := LoadTxtSuperUniques(su)
	if err != nil {
		t.Fatal(err)
	}

	var countess *SuperUnique

	for i := range supers {
		if supers[i].Key == "The Countess" {
			countess = &supers[i]
		}
	}

	if countess == nil || countess.Class != "corruptrogue3" || countess.MinGrp != 6 || countess.MaxGrp != 6 {
		t.Fatalf("countess row: %+v", countess)
	}

	info, ok := classes.ByID(countess.Class)
	if !ok || info.Class != 45 {
		t.Fatalf("corruptrogue3: %+v", info)
	}

	// SURPRISE vs monster-ai-2.md section 2: in the real patch_d2 monstats.txt
	// corruptrogue3 has NO minion1/minion2, so its followers fall back to its own class.
	if info.Minion1 != -1 || info.Minion2 != -1 {
		t.Errorf("corruptrogue3 minions = %d,%d; the notes' minion1/2 do not exist in patch_d2", info.Minion1, info.Minion2)
	}

	p := PlanSuperUnique(d2rand.New(3), countess.Key, info, countess.MinGrp, countess.MaxGrp)
	if len(p.Members) != 7 {
		t.Errorf("countess pack has %d members, want 7", len(p.Members))
	}

	fallen, _ := classes.ByID("fallen1")
	if fallen.MinGrp < 1 || fallen.MaxGrp < fallen.MinGrp || !fallen.IsSpawn {
		t.Errorf("fallen1 spawn data: %+v", fallen)
	}

	// fallen1 is a SetBoss/BossXfer class whose minion1 is itself, party 2..3
	if !fallen.SetBoss || !fallen.BossXfer || fallen.Minion1 != fallen.Class || fallen.PartyMin != 2 || fallen.PartyMax != 3 {
		t.Errorf("fallen1 leader data: %+v", fallen)
	}

	for seed := uint32(1); seed < 40; seed++ {
		n := len(PlanGroup(d2rand.New(seed), fallen).Members)
		if n < fallen.MinGrp+fallen.PartyMin || n > fallen.MaxGrp+fallen.PartyMax {
			t.Fatalf("fallen group of %d outside %d..%d", n, fallen.MinGrp+fallen.PartyMin, fallen.MaxGrp+fallen.PartyMax)
		}
	}
}
