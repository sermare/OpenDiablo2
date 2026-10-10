package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

func TestEarStoredAsD2SEar(t *testing.T) {
	tests := []struct {
		name         string
		class, level int
		wantC, wantL uint8
	}{
		{"Bob", 1, 40, 1, 40},
		{"Over", 9, 150, 6, 99},
		{"Low", 0, 0, 0, 1},
	}

	for _, tc := range tests {
		s := &StoredItem{Code: "ear", Page: PageInventory, X: 3, Y: 2,
			Spec: &diablo2item.Spec{Code: "ear", Ear: &diablo2item.EarInfo{Name: tc.name, Class: tc.class, Level: tc.level}}}

		it, dropped, why := D2SItemFromStored(s, nil, nil)
		if why != "" || dropped != 0 || !it.Ear || it.EarInfo == nil || it.EarInfo.Name != tc.name ||
			it.EarInfo.Class != tc.wantC || it.EarInfo.Level != tc.wantL || it.X != 3 || it.Y != 2 || it.Code != "ear" {
			t.Errorf("%s: %+v %+v %q", tc.name, it, it.EarInfo, why)
		}
	}
}
