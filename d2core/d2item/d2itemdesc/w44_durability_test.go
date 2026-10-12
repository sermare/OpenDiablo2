package d2itemdesc

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestSplitDurabilityLine(t *testing.T) {
	tb := testTables()
	it := &d2s.Item{Code: "cap", MaxDurability: 24, Durability: 12}
	text := tb.tr("ItemStats1d") + " 12 " + tb.tr("ItemStats1j") + " 24"

	head, mx, ok := tb.SplitDurabilityLine(it, Line{text, White})
	if !ok || mx != "24" || head+mx != text {
		t.Errorf("split: %q %q %v", head, mx, ok)
	}

	if _, _, ok := tb.SplitDurabilityLine(it, Line{"Defense: 24", White}); ok {
		t.Error("a defense line must not split")
	}
}

// INV_BuildDurabilityLine 0x480eb0: the maximum is blue only when stat 0x4b contributes.
func TestDurabilityMaxBlue(t *testing.T) {
	tb := testTables()
	for _, id := range []int{statMaxDurPercent} {
		if tb.Stats[id] == nil {
			d := &StatDef{ID: id, Name: "item_maxdurability_percent"}
			tb.Stats[id], tb.StatByName[d.Name] = d, d
		}
	}

	tests := []struct {
		name  string
		props []d2s.Property
		want  bool
	}{
		{"none", nil, false},
		{"other stat", []d2s.Property{{ID: 7, Value: 10}}, false},
		{"maxdur percent", []d2s.Property{{ID: statMaxDurPercent, Value: 20}}, true},
	}

	for _, tc := range tests {
		it := &d2s.Item{Code: "cap", Identified: true, Properties: tc.props}
		if got := tb.DurabilityMaxBlue(it); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
