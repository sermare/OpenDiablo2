package d2records

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

func loadStates(t *testing.T, data []byte) States {
	t.Helper()

	r := &RecordManager{}
	r.Logger = d2util.NewLogger()

	if err := statesLoader(r, d2txt.LoadDataDictionary(data)); err != nil {
		t.Fatal(err)
	}

	return r.States
}

// TestStatesLoaderColumns pins the column names the loader reads: a wrong name
// silently reads column 0 (the state name), which used to fill PgOverlay with
// the state name and leave Cureable, RemFunc and CltActiveFunc at zero.
func TestStatesLoaderColumns(t *testing.T) {
	const tbl = "state\tid\tgroup\tcurable\tcurse\tpgsvoverlay\tremfunc\tcltactivefunc\n" +
		"none\t\t\t\t\t\t\n" +
		"poison\t\t2\t1\t1\tpg\t7\t9\n"

	st := loadStates(t, []byte(tbl))
	p := st["poison"]

	if p == nil {
		t.Fatal("no poison row")
	}

	if p.ID != 1 || p.Group != 2 || !p.Cureable || !p.Curse || p.PgOverlay != "pg" || p.RemFunc != 7 || p.CltActiveFunc != 9 {
		t.Errorf("poison = %+v", *p)
	}

	if st["none"].PgOverlay != "" {
		t.Errorf("none.PgOverlay = %q, want empty", st["none"].PgOverlay)
	}
}

// TestRealStates checks the shipped table (D2_TABLES/states/patch_d2/States.txt).
func TestRealStates(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(root, "states", "patch_d2", "States.txt"))
	if err != nil {
		t.Skip(err)
	}

	st := loadStates(t, data)

	for name, id := range map[string]int{"none": 0, "freeze": 1, "poison": 2, "cold": 11, "stunned": 21, "burning": 115,
		"uninterruptable": 54, "skilldelay": 121, "blood_mana": 114, "slowmissiles": 87, "holyshield": 101} {
		r := st[name]
		if r == nil {
			t.Errorf("state %q missing", name)
			continue
		}

		if r.ID != id {
			t.Errorf("%s id = %d, want %d", name, r.ID, id)
		}
	}

	if st["sanctuary"].ID != 47 {
		t.Errorf("sanctuary id = %d, want 47 (0x2f)", st["sanctuary"].ID)
	}

	if !st["amplifydamage"].Cureable || st["amplifydamage"].PgOverlay != "" {
		t.Errorf("amplifydamage = %+v", *st["amplifydamage"])
	}
}
