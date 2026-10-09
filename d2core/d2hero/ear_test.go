package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestEarD2S(t *testing.T) {
	tests := []struct {
		name        string
		level       int
		class       d2s.Class
		wantLvl     uint8
		wantClass   uint8
		wantEarName string
	}{
		{"Bob", 40, d2s.Necromancer, 40, 2, "Bob"},
		{"Zed", 0, d2s.Amazon, 1, 0, "Zed"},
		{"Max", 120, d2s.Assassin, 99, 6, "Max"},
	}

	for _, tc := range tests {
		it := EarD2S(tc.name, tc.level, tc.class)
		if !it.Ear || it.EarInfo == nil || it.EarInfo.Name != tc.wantEarName || it.EarInfo.Level != tc.wantLvl ||
			it.EarInfo.Class != tc.wantClass || it.Code != "ear" {
			t.Errorf("%s: %+v %+v", tc.name, it, it.EarInfo)
		}
	}
}

func TestExportKeepsEar(t *testing.T) {
	ear := EarD2S("Bob", 40, d2s.Sorceress)
	c := &HeroContainers{Items: []StoredItem{{Code: "ear", Page: PageInventory, X: 3, Y: 2, D2S: &ear}}}

	out, skipped := ExportD2SItems(c)
	if len(out) != 1 || len(skipped) != 0 {
		t.Fatalf("exported %d skipped %d", len(out), len(skipped))
	}

	if !out[0].Ear || out[0].X != 3 || out[0].Y != 2 || out[0].EarInfo.Name != "Bob" || out[0].Page != uint8(PageInventory) {
		t.Fatalf("exported ear %+v", out[0])
	}
}
