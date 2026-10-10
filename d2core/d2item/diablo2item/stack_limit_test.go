package diablo2item

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2stats/diablo2stats"
)

// realStatRows reads the Stat/ID columns of the real ItemStatCost.txt into
// records (skipped when D2_TABLES is unset).
func realStatRows(t *testing.T) d2records.ItemStatCosts {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(dir, "ItemStatCost.txt"))
	if err != nil {
		t.Skipf("no ItemStatCost.txt: %v", err)
	}

	d := d2txt.LoadDataDictionary(buf)
	out := make(d2records.ItemStatCosts)

	for d.Next() {
		name := d.String("Stat")
		out[name] = &d2records.ItemStatCostRecord{Name: name, Index: d.Number("ID")}
	}

	return out
}

func fakeStackItem(stats d2records.ItemStatCosts, rec *d2records.ItemCommonRecord) (*Item, *diablo2stats.StatFactory) {
	am := &d2asset.AssetManager{Records: &d2records.RecordManager{}}
	am.Records.Item.Stats = stats
	am.Records.Item.All = map[string]*d2records.ItemCommonRecord{rec.Code: rec}

	sf, _ := diablo2stats.NewStatFactory(am)
	f := &ItemFactory{asset: am, stat: sf}

	return &Item{factory: f, CommonCode: rec.Code}, sf
}

func TestExtraStackStatFoundByID(t *testing.T) {
	stats := realStatRows(t)

	name := extraStackStatName(stats)
	if name == "" {
		t.Fatalf("no ItemStatCost row with id %d", extraStackStatID)
	}

	if stats[name].Index != 254 {
		t.Errorf("row %q has id %d", name, stats[name].Index)
	}

	if name != "item_extra_stack" {
		t.Errorf("stat 254 is %q, expected item_extra_stack", name)
	}
}

func TestItemStackLimit(t *testing.T) {
	stats := realStatRows(t)
	name := extraStackStatName(stats)

	if name == "" {
		t.Fatal("stat 254 missing from ItemStatCost")
	}

	rec := &d2records.ItemCommonRecord{Code: "tbk", Stackable: true, MaxStack: 40}
	plain := &d2records.ItemCommonRecord{Code: "hax", Stackable: false}

	tests := []struct {
		name  string
		extra float64
		with  bool
		want  int
	}{
		{"no stat list", 0, false, 40},
		{"extra +60", 60, true, 100},
		{"clamped to 511", 900, true, 511},
		{"negative extra", -10, true, 30},
	}

	for _, tc := range tests {
		it, sf := fakeStackItem(stats, rec)
		if tc.with {
			it.statList = sf.NewStatList(sf.NewStat(name, tc.extra))
		}

		if got := it.StackLimit(); got != tc.want {
			t.Errorf("%s: StackLimit %d, want %d", tc.name, got, tc.want)
		}

		if !it.IsStackable() {
			t.Errorf("%s: should be stackable", tc.name)
		}
	}

	it, _ := fakeStackItem(stats, plain)
	if it.IsStackable() {
		t.Error("axe must not be stackable")
	}

	// A table without id 254 must add nothing (and not panic).
	delete(stats, name)

	it, sf := fakeStackItem(stats, rec)
	it.statList = sf.NewStatList()

	if got := it.StackLimit(); got != 40 {
		t.Errorf("missing stat row: %d, want 40", got)
	}
}
