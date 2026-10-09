package d2drlg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type oracleAct struct {
	Lo, Hi       uint32
	TombA, TombB int
	Flip         int
}

// TestOracleActExtras checks DrawActExtras against golden values produced by
// running the real DRLG_CreateDrlg in an emulator (numbers only). The seed
// state afterwards is not compared: the game keeps drawing from the DRLG seed
// in DRLG_InitActLevelLinks (act-specific level placement), which is not ported
// for acts 2-5.
func TestOracleActExtras(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "acts.json"))
	if err != nil {
		t.Skip(err)
	}

	var gold []struct {
		Seed                   uint32
		Act1, Act2, Act3, Act4 oracleAct
	}

	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	for _, g := range gold {
		for act, want := range []oracleAct{{}, g.Act1, g.Act2, g.Act3, g.Act4} {
			if act == 0 {
				continue
			}

			got := DrawActExtras(g.Seed, act)
			if got.TombA != want.TombA || got.TombB != want.TombB || got.Flip != want.Flip {
				t.Errorf("seed %#x act index %d: go %+v oracle %+v", g.Seed, act, got, want)
			}
		}
	}
}
