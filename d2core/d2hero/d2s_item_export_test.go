package d2hero

import (
	"bytes"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

func containersOf(t *testing.T, data []byte, tables *d2s.ItemTables) *HeroContainers {
	t.Helper()

	c, err := d2s.Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	out := &HeroContainers{Items: []StoredItem{}}

	for i := range c.Items {
		if s, skip := StoredFromD2S(&c.Items[i], func(string) bool { return true }); skip == "" {
			out.Items = append(out.Items, s)
		}
	}

	return out
}

func TestMergeContainerItemsUnchangedKeepsBits(t *testing.T) {
	data, tables, state := realSave(t)
	state.Containers = containersOf(t, data, tables)

	out, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{
		Affixes: &AffixIDs{}, Known: func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 0 || !bytes.Equal(out, data) {
		t.Fatalf("unchanged containers must export the original bits: warnings %v, equal %v", warnings, bytes.Equal(out, data))
	}
}

func TestMergeContainerItemsAddMoveRemove(t *testing.T) {
	data, tables, state := realSave(t)
	state.Containers = containersOf(t, data, tables)

	before, _ := d2s.Parse(data, tables)
	nBefore := len(before.Items)

	// remove the first stored item, move the second
	gone := state.Containers.Items[0]
	moved := state.Containers.Items[1]
	state.Containers.Items = state.Containers.Items[1:]
	state.Containers.Items[0].X, state.Containers.Items[0].Y = 9, 7

	// a magic cap picked up in the game, a bought stack of keys, and an item
	// with an affix the tables do not know (skipped with a warning)
	state.Containers.Items = append(state.Containers.Items,
		StoredItem{
			Code: "cap", Page: PageInventory, X: 8, Y: 3, Quality: int(d2s.QualityMagic), ILvl: 12, Seed: 4242,
			Prefixes: []string{"Sturdy"}, Identified: true,
			Facts: &diablo2item.ItemFacts{
				Defense: 6, Durability: 12, MaxDurability: 12,
				Affix: []diablo2item.ExportStat{{ID: 0, Value: 3}, {ID: 16, Value: 20}, {ID: 0, Value: 1}, {ID: 204, Value: 9, HasParams: true}},
			},
		},
		StoredItem{
			Code: "key", Page: PageInventory, X: 7, Y: 3, Quantity: 3, ILvl: 1, Seed: 77,
			Facts: &diablo2item.ItemFacts{},
		},
		StoredItem{
			Code: "cap", Page: PageInventory, X: 6, Y: 3, Quality: int(d2s.QualityMagic), Prefixes: []string{"Nope"},
			Facts: &diablo2item.ItemFacts{},
		},
	)

	out, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{
		Affixes: &AffixIDs{Prefix: map[string]int{"Sturdy": 5}},
		Known:   func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}

	back, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	if want := nBefore - 1 + 2; len(back.Items) != want {
		t.Fatalf("%d items, want %d", len(back.Items), want)
	}

	var cap, key, movedBack *d2s.Item

	for i := range back.Items {
		it := &back.Items[i]

		switch {
		case it.ID == 4242:
			cap = it
		case strings.TrimSpace(it.Code) == "key" && it.Quantity == 3 && it.Page == uint8(PageInventory) && it.X == 7:
			key = it
		case it.ID == moved.D2S.ID && it.Code == moved.D2S.Code:
			movedBack = it
		case it.ID == gone.D2S.ID && it.Code == gone.D2S.Code && it.Location == d2s.LocationStored:
			t.Errorf("removed item %q is still there", it.Code)
		}
	}

	if cap == nil || cap.Quality != d2s.QualityMagic || cap.MagicPrefix != 5 || cap.Level != 12 || !cap.Identified ||
		cap.Defense != 6 || cap.MaxDurability != 12 || cap.X != 8 || cap.Y != 3 {
		t.Fatalf("new cap: %+v", cap)
	}

	// duplicates are summed, the id order is the game's, the parameter stat is left out
	if len(cap.Properties) != 2 || cap.Properties[0].ID != 0 || cap.Properties[0].Value != 4 || cap.Properties[1].ID != 16 {
		t.Errorf("cap properties: %+v", cap.Properties)
	}

	if key == nil {
		t.Error("new key stack missing")
	}

	if movedBack == nil || movedBack.X != 9 || movedBack.Y != 7 {
		t.Errorf("moved item: %+v", movedBack)
	}

	reported := false

	for _, w := range warnings {
		if strings.Contains(w, "Nope") {
			reported = true
		}
	}

	if !reported {
		t.Errorf("the item with an unknown affix was not reported: %v", warnings)
	}
}

func TestD2SItemFromStoredRejects(t *testing.T) {
	_, tables, _ := realSave(t)
	ids := &AffixIDs{}

	for name, s := range map[string]StoredItem{
		"no facts": {Code: "cap"},
		"rare":     {Code: "cap", Quality: int(d2s.QualityRare), Facts: &diablo2item.ItemFacts{}},
		"unique":   {Code: "cap", Quality: int(d2s.QualityUnique), Unique: "x", Facts: &diablo2item.ItemFacts{}},
		"unknown":  {Code: "zzz", Facts: &diablo2item.ItemFacts{}},
	} {
		s := s
		if _, _, why := D2SItemFromStored(&s, tables, ids); why == "" {
			t.Errorf("%s: accepted", name)
		}
	}
}
