package d2itemdesc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2tbl"
)

// The real-data tests read the game tables from D2_TABLES (the extracted
// tables folder: the plain tables at its top, and the layered extras in
// itemdesc/{d2data,d2exp,patch_d2}) and the sample save from
// D2S_SAMPLE_BODY. They skip when those are not set. No game data is
// committed.

var layers = []string{"patch_d2", "d2exp", "d2data"}

func realTables(t *testing.T) *Tables {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	src := func(name string) ([]byte, bool) {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return b, true
		}

		for _, l := range layers {
			if b, err := os.ReadFile(filepath.Join(dir, "itemdesc", l, strings.ToLower(name))); err == nil {
				return b, true
			}
		}

		return nil, false
	}

	if _, ok := src("uniqueitems.txt"); !ok {
		t.Skip("itemdesc tables not extracted under D2_TABLES/itemdesc")
	}

	strs := d2tbl.TextDictionary{}

	for _, f := range []string{"d2data/string.tbl", "d2exp/expansionstring.tbl", "patch_d2/patchstring.tbl"} {
		b, err := os.ReadFile(filepath.Join(dir, "itemdesc", f))
		if err != nil {
			t.Skipf("string table %s missing", f)
		}

		d, err := d2tbl.LoadTextDictionary(b)
		if err != nil {
			t.Fatal(err)
		}

		for k, v := range d {
			strs[k] = v
		}
	}

	tb, err := Load(src, func(k string) string { return strs[k] })
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func sampleChar(t *testing.T) *d2s.Character {
	t.Helper()

	path, dir := os.Getenv("D2S_SAMPLE_BODY"), os.Getenv("D2_TABLES")
	if path == "" || dir == "" {
		t.Skip("set D2_TABLES and D2S_SAMPLE_BODY")
	}

	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		t.Skipf("none of %v in %s", names, dir)

		return nil
	}

	tables, err := d2s.NewItemTables(read("itemstatcost.bin", "ItemStatCost.txt"), read("armor.txt"),
		read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestDebugSample(t *testing.T) {
	if os.Getenv("D2_ITEMDESC_DEBUG") == "" {
		t.Skip("debug output only")
	}

	tb := realTables(t)
	c := sampleChar(t)

	for i := range c.Items {
		it := &c.Items[i]
		t.Logf("---- %s q%d id%d uid%d sid%d rw%d r1=%d r2=%d", it.Code, it.Quality, it.ID, it.UniqueID, it.SetID, it.RunewordID, it.RareName1, it.RareName2)

		for _, l := range tb.Describe(it, nil) {
			t.Logf("   [%s] %s", l.Color, l.Text)
		}
	}
}
