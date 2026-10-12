package d2hero

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// TestMercGearMadeInGameReachesJF: a magic cap picked up in the game and given to the merc is written into the 'jf' section (w44),
// and an item without rolled values is still reported instead of written.
func TestMercGearMadeInGameReachesJF(t *testing.T) {
	data, tables, state, c := mercSample(t)

	for state.Merc.TakeMercItem(d2equip.LocHead) != nil {
	}

	before := len(c.MercItems)
	if before > 0 {
		before-- // the head item (if the sample has one) is gone now
	}

	state.Merc.Items = append(state.Merc.Items,
		StoredItem{
			Code: "cap", Page: PageEquipped, X: int(d2equip.LocHead), Quality: int(d2s.QualityMagic), ILvl: 12, Seed: 4242,
			Prefixes: []string{"Sturdy"}, Identified: true,
			Facts: &diablo2item.ItemFacts{
				Defense: 6, Durability: 12, MaxDurability: 12,
				Affix: []diablo2item.ExportStat{{ID: 0, Value: 3}, {ID: 16, Value: 20}, {ID: 0, Value: 1}, {ID: 204, Value: 9, HasParams: true}},
			},
		})
	state.Merc.ItemsDirty = true

	out, _, err := ExportD2SWithOptions(state, data, tables, ExportOptions{
		Affixes: &AffixIDs{Prefix: map[string]int{"Sturdy": 5}},
		Known:   func(string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := d2s.Parse(out, tables)
	if err != nil {
		t.Fatal(err)
	}

	found := false

	for _, it := range got.MercItems {
		if strings.TrimSpace(it.Code) == "cap" && it.Location == d2s.LocationEquipped && it.Equipped == uint8(d2equip.LocHead) {
			found = true
		}
	}

	if !found {
		t.Fatalf("the cap made in the game is not in the jf section: %d items", len(got.MercItems))
	}

	// no rolled values: reported, not written
	state.Merc.Items[len(state.Merc.Items)-1].Facts = nil

	var warned []string

	_, _, err = ExportD2SWithOptions(state, data, tables, ExportOptions{
		Affixes: &AffixIDs{Prefix: map[string]int{"Sturdy": 5}},
		Known:   func(string) bool { return true },
		Warn:    func(s string) { warned = append(warned, s) },
	})
	if err != nil {
		t.Fatal(err)
	}

	hit := false

	for _, w := range warned {
		if strings.Contains(w, "merc item") && strings.Contains(w, "not written") {
			hit = true
		}
	}

	if !hit {
		t.Errorf("an item without rolled values was not reported: %v", warned)
	}
}
