package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// A rare, personalized, ethereal item made in the game is written with its
// rare names, affix ids, properties and personalization, and nothing is
// reported as unsupported.
func TestStoredRareExportKeepsEverything(t *testing.T) {
	_, tables, _ := realSave(t)

	s := &StoredItem{
		Code: "rin", Page: PageInventory, X: 2, Y: 1, Quality: int(d2s.QualityRare), ILvl: 71, Seed: 987,
		Identified: true, Ethereal: false, Personal: "Tester",
		Facts: &diablo2item.ItemFacts{
			Code: "rin", Quality: int(d2s.QualityRare),
			Save: &diablo2item.SaveForm{
				RareNames:   [2]int{159, 121},
				RareAffixes: [6]int{300, 349, 0, 12, 0, 0},
				Props: []diablo2item.SaveWrite{
					{ID: 7, Value: 36}, {ID: 19, Value: 94}, {ID: 41, Value: 18}, {ID: 7, Value: 4},
					{ID: 17, Value: 30}, {ID: 18, Value: 30},
				},
			},
		},
	}

	it, dropped, why := D2SItemFromStored(s, tables, &AffixIDs{})
	if why != "" || dropped != 0 {
		t.Fatalf("not written: %q (%d dropped)", why, dropped)
	}

	if it.RareName1 != 159 || it.RareName2 != 121 || it.RareMask != 0b001011 || it.RareAffixes != [6]uint16{300, 349, 0, 12, 0, 0} {
		t.Errorf("rare identity %d %d %b %v", it.RareName1, it.RareName2, it.RareMask, it.RareAffixes)
	}

	if !it.Personalized || it.PersonalName != "Tester" {
		t.Errorf("personalization lost: %v %q", it.Personalized, it.PersonalName)
	}

	bits, err := d2s.EncodeItem(&it, tables)
	if err != nil {
		t.Fatal(err)
	}

	back, _, err := d2s.ParseItemList(append([]byte{'J', 'M', 1, 0}, bits...), tables)
	if err != nil || len(back) != 1 {
		t.Fatalf("parse: %v", err)
	}

	got := map[int]int64{}
	for _, p := range back[0].Properties {
		got[p.ID] += p.Value
	}

	if got[7] != 40 || got[19] != 94 || got[41] != 18 || got[17] != 30 || got[18] != 30 {
		t.Errorf("properties %v", back[0].Properties)
	}

	if back[0].RareName1 != 159 || !back[0].Personalized || back[0].PersonalName != "Tester" {
		t.Errorf("parsed back %+v", back[0])
	}
}

// Merging such an item into a save adds it and warns about nothing.
func TestMergeRareItemNoWarning(t *testing.T) {
	data, tables, state := realSave(t)
	state.Containers = containersOf(t, data, tables)

	state.Containers.Items = append(state.Containers.Items, StoredItem{
		Code: "rin", Page: PageInventory, X: 9, Y: 3, Quality: int(d2s.QualityRare), ILvl: 71, Seed: 987, Identified: true,
		Facts: &diablo2item.ItemFacts{Save: &diablo2item.SaveForm{
			RareNames: [2]int{159, 121}, RareAffixes: [6]int{300, 349, 0, 0, 0, 0},
			Props: []diablo2item.SaveWrite{{ID: 7, Value: 20}},
		}},
	})

	_, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{
		Affixes: &AffixIDs{}, Known: func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 0 {
		t.Fatalf("warnings: %v", warnings)
	}
}
