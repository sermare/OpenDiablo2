package d2monsters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

var _ d2monster.FBXMinionSpawner = (*Director)(nil)

func TestQueenHostCap(t *testing.T) {
	// the queen is self-limited (aip1 is 7..9 in the tables), held at the host bound only
	if c := HostSummonCap(SummonNest, "SandMaggotQueen"); c != selfLimitedSummonCap {
		t.Errorf("cap %d, want %d", c, selfLimitedSummonCap)
	}

	for _, tc := range []struct {
		live int
		want bool
	}{{0, true}, {selfLimitedSummonCap - 1, true}, {selfLimitedSummonCap, false}, {selfLimitedSummonCap + 5, false}} {
		if got := queenMayLay(tc.live, "SandMaggotQueen"); got != tc.want {
			t.Errorf("live %d: %v", tc.live, got)
		}
	}

	// an unknown AI is held to the small generic bound
	if queenMayLay(nestSummonCap, "Other") {
		t.Error("generic AI above the nest cap")
	}
}

func TestFBXSpawnMinionWithoutWorld(t *testing.T) {
	queen := petUnit(1, 0, 0, "SandMaggotQueen", "")
	queen.m.Stat = &d2records.MonStatRecord{Key: "maggotqueen1", AiKey: "SandMaggotQueen", SpawnKey: "sandmaggot1"}
	d := testDirector(queen)

	if d.FBXSpawnMinion(queen.b) {
		t.Error("laid an egg with no path map or assets")
	}

	if d.FBXSpawnMinion(&d2monster.Brain{ID: 99}) {
		t.Error("laid an egg for an unknown brain")
	}
}

// TestRealQueenSpawn checks every queen of the real table names a SandMaggot
// as her spawn class, with the spawn offset the table gives.
func TestRealQueenSpawn(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	rm, err := d2records.NewRecordManager(d2util.LogLevelNone)
	if err != nil {
		t.Fatal(err)
	}

	if err = rm.Load(d2resource.MonStats, d2txt.LoadDataDictionary(buf)); err != nil {
		t.Fatal(err)
	}

	queens := 0

	for _, st := range rm.Monster.Stats {
		if !strings.EqualFold(st.AiKey, "SandMaggotQueen") {
			continue
		}

		queens++

		young := rm.Monster.Stats[st.SpawnKey]
		if young == nil || !strings.EqualFold(young.AiKey, "SandMaggot") {
			t.Errorf("%s: spawn %q is not a SandMaggot row", st.Key, st.SpawnKey)
		}

		if st.SpawnOffsetX != 8 {
			t.Errorf("%s: spawnx %d, want 8", st.Key, st.SpawnOffsetX)
		}
	}

	if queens != 5 {
		t.Errorf("%d queens, want 5", queens)
	}
}
