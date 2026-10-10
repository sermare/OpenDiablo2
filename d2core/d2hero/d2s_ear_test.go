package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

func TestEarD2SItemRanges(t *testing.T) {
	ok := &StoredItem{Code: "ear", Page: PageInventory, X: 3, Y: 1, Spec: &diablo2item.Spec{
		Code: "ear", Ear: &diablo2item.EarInfo{Name: "AVeryLongHeroNameIndeed", Class: 3, Level: 41}}}

	it, _, why := earD2SItem(ok)
	if why != "" || !it.Ear || it.EarInfo.Name != "AVeryLongHeroNa" || it.EarInfo.Class != 3 || it.EarInfo.Level != 41 ||
		it.Location != d2s.LocationStored || it.Page != PageInventory {
		t.Fatalf("ear item %+v (%s)", it, why)
	}

	for _, bad := range []diablo2item.EarInfo{{Name: "x", Class: 7, Level: 5}, {Name: "x", Class: 0, Level: 0},
		{Name: "x", Class: -1, Level: 5}, {Name: "x", Class: 0, Level: 100}} {
		bad := bad
		if _, _, why := earD2SItem(&StoredItem{Spec: &diablo2item.Spec{Ear: &bad}}); why == "" {
			t.Errorf("ear %+v was written", bad)
		}
	}
}

// An ear made in the game is written by the d2s item encoder and read back
// as the same player; an ear the save held is imported and stays.
func TestEarExportsAndImports(t *testing.T) {
	data, tables, state := realSave(t)
	state.Containers = containersOf(t, data, tables)

	spec := diablo2item.Spec{Code: "ear", ILvl: 41, Identified: true, Durability: -1,
		Ear: &diablo2item.EarInfo{Name: "Doomguy", Class: 3, Level: 41}}
	state.Containers.Items = append(state.Containers.Items, StoredItem{
		Code: "ear", Page: PageInventory, X: 9, Y: 0, ILvl: 41, Identified: true, Spec: &spec,
	})

	out, warnings, err := ExportD2SWithOptions(state, data, tables, ExportOptions{
		Affixes: &AffixIDs{}, Known: func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, w := range warnings {
		if w != "" {
			t.Logf("warning: %s", w)
		}
	}

	c, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	var ear *d2s.Item

	for i := range c.Items {
		if c.Items[i].Ear {
			ear = &c.Items[i]
		}
	}

	if ear == nil || ear.EarInfo == nil || ear.EarInfo.Name != "Doomguy" || ear.EarInfo.Class != 3 ||
		ear.EarInfo.Level != 41 || ear.X != 9 {
		t.Fatalf("the written ear is %+v", ear)
	}

	// importing the saved ear gives it back as a stored item with its player
	s, skip := StoredFromD2S(ear, func(string) bool { return true })
	if skip != "" || s.Spec == nil || s.Spec.Ear == nil || s.Spec.Ear.Name != "Doomguy" || s.D2S == nil {
		t.Fatalf("imported ear %+v (%s)", s, skip)
	}
}
