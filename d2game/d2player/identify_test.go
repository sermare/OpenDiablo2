package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

type codeOnlyItem struct{ code string }

func (c codeOnlyItem) InventoryGridSize() (w, h int) { return 1, 1 }
func (c codeOnlyItem) GetItemCode() string           { return c.code }
func (c codeOnlyItem) InventoryGridSlot() (x, y int) { return 0, 0 }
func (c codeOnlyItem) SetInventoryGridSlot(_, _ int) {}
func (c codeOnlyItem) GetItemDescription() []string  { return nil }

func TestIdentifyCostFor(t *testing.T) {
	var done, other d2s.QuestRecord

	done.Set(4, 0)
	other.Set(5, 0)

	tests := []struct {
		name  string
		n     int
		quest *d2s.QuestRecord
		want  int
	}{
		{"none", 0, nil, 0},
		{"one", 1, nil, 100},
		{"five", 5, nil, 500},
		{"quest done is free", 5, &done, 0},
		{"other quest bit does not count", 3, &other, 300},
	}

	for _, tt := range tests {
		if got := IdentifyCostFor(tt.n, tt.quest); got != tt.want {
			t.Errorf("%s: %d want %d", tt.name, got, tt.want)
		}
	}
}

func TestIsIdentifyScroll(t *testing.T) {
	for code, want := range map[string]bool{"isc": true, "ibk": true, "tsc": false, "cm1": false} {
		if got := IsIdentifyScroll(codeOnlyItem{code}); got != want {
			t.Errorf("%s: %v want %v", code, got, want)
		}
	}
}
