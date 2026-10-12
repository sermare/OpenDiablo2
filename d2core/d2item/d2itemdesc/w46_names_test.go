package d2itemdesc

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Name colours read from INV_BuildPanelAndItemTooltips 0x48a090 (w46).
func TestNameColourW46(t *testing.T) {
	tb := describeTables()
	tb.Bases["ceh"] = &BaseItem{Code: "ceh", Name: "Charged Essence of Hatred", Type: "ques"}

	first := func(it d2s.Item) Line {
		it.Identified = true
		if it.Quality == 0 {
			it.Quality = d2s.QualityNormal
		}

		return tb.Describe(&it, nil)[0]
	}

	cases := []struct {
		name string
		it   d2s.Item
		want Color
	}{
		{"healthy armor", d2s.Item{Code: "cap", Defense: 4, MaxDurability: 12, Durability: 5}, White},
		{"broken armor is red", d2s.Item{Code: "cap", Defense: 4, MaxDurability: 12, Durability: 0}, Red},
		{"weapon at 0 durability keeps its colour", d2s.Item{Code: "ssd", MaxDurability: 24, Durability: 0}, White},
		{"essence is orange", d2s.Item{Code: "ceh"}, Orange},
	}

	for _, c := range cases {
		if got := first(c.it); got.Color != c.want {
			t.Errorf("%s: %s is %v, want %v", c.name, got.Text, got.Color, c.want)
		}
	}
}
