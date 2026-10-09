package diablo2item

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func testDropFactory() *ItemFactory {
	rec := &d2records.RecordManager{}

	cap1 := &d2records.ItemCommonRecord{Code: "cap", Type: "helm", Level: 3, Rarity: 1, Spawnable: true}
	cap2 := &d2records.ItemCommonRecord{Code: "skp", Type: "helm", Level: 4, Rarity: 1, Spawnable: true}
	rec.Item.All = d2records.CommonItems{"cap": cap1, "skp": cap2}
	rec.Item.Equivalency = d2records.ItemEquivalenceMap{
		"helm": {cap1, cap2},
		"armo": {cap1, cap2},
	}
	rec.Item.Types = d2records.ItemTypes{
		"helm": {Code: "helm", Rare: true},
		"armo": {Code: "armo", Rare: true, TreasureClass: 1},
	}
	rec.Item.Ratios = d2records.ItemRatios{
		"x": {Version: true, UniqueDropInfo: d2records.DropRatioInfo{Frequency: 400, Divisor: 1, DivisorMin: 6400}},
	}
	rec.Item.Treasure.Expansion = d2records.TreasureClass{
		"Boss": {Name: "Boss", NumPicks: 1, FreqUnique: 1024, Treasures: []*d2records.Treasure{{Code: "armo6", Probability: 1}}},
		// names sort against the file order on purpose
		"Z1": {Name: "Z1", Index: 1, Group: 3, Level: 0, NumPicks: 1},
		"Y2": {Name: "Y2", Index: 2, Group: 3, Level: 10, NumPicks: 1},
		"X3": {Name: "X3", Index: 3, Group: 3, Level: 30, NumPicks: 1},
	}

	return &ItemFactory{asset: &d2asset.AssetManager{Records: rec}}
}

func TestDropTablesAdapter(t *testing.T) {
	f := testDropFactory()
	tab := f.dropTables()

	// generated armo6 holds items in (3, 6] only: skp (4), not cap (3)
	armo6, ok := tab.tcs.TreasureClass("armo6")
	if !ok || len(armo6.Entries) != 1 || armo6.Entries[0].Code != "skp" {
		t.Fatalf("armo6 = %+v", armo6)
	}

	// level groups survive the map -> table conversion
	g1, _ := tab.tcs.TreasureClass("Z1")
	for lv, want := range map[int]string{0: "Z1", 9: "Z1", 10: "Y2", 15: "Y2", 30: "X3", 99: "X3"} {
		if got := tab.tcs.Upgrade(g1, lv).Name; got != want {
			t.Errorf("upgrade at %d = %s, want %s", lv, got, want)
		}
	}

	if _, ok := tab.ItemRatio(false, false); !ok {
		t.Error("ItemRatio lookup")
	}

	// A boss class with a full unique modifier always makes the item unique;
	// the dropper uses the dropper's level as item level.
	drops, err := tab.dropper.Roll(&d2drop.Context{RNG: d2rand.New(5), ILvl: 70}, "Boss")
	if err != nil || len(drops) != 1 {
		t.Fatalf("drops %v err %v", drops, err)
	}

	if drops[0].Quality != d2drop.QualityUnique || drops[0].ILvl != 70 || drops[0].Code != "skp" {
		t.Errorf("drop %+v", drops[0])
	}
}
