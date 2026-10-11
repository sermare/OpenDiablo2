package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

func TestBonePrisonCellsAreTheExeTable(t *testing.T) {
	seen := map[[2]int]bool{}

	for i := range d2skill.BonePrisonOffsets {
		x, y := wallCell(0, 0, 100, 50, i, len(d2skill.BonePrisonOffsets), "ring")
		o := d2skill.BonePrisonOffsets[i]

		if x != 100+o[0] || y != 50+o[1] {
			t.Errorf("piece %d at (%d,%d)", i, x, y)
		}

		seen[[2]int{x, y}] = true
	}

	if len(seen) != 12 {
		t.Errorf("%d distinct cells, want 12", len(seen))
	}
}
