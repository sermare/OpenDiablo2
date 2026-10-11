package d2automap

import "testing"

// The panel shift is a quarter of the 800 pixel interface column, not of the screen width.
func TestPanelShiftIsColumnQuarter(t *testing.T) {
	for _, w := range []int{800, 1512, 1920, 3440, 5120} {
		h := 600
		none := ComputeLayoutOffsets(SizeFull, w, h, 100, 100, PanelNone, false, 0, 0)
		right := ComputeLayoutOffsets(SizeFull, w, h, 100, 100, PanelRight, false, 0, 0)
		left := ComputeLayoutOffsets(SizeFull, w, h, 100, 100, PanelLeft, false, 0, 0)

		if d := right.OriginX - none.OriginX; d != 200 {
			t.Errorf("w=%d right panel: origin moved by %d, want 200", w, d)
		}

		if d := left.OriginX - none.OriginX; d != -200 {
			t.Errorf("w=%d left panel: origin moved by %d, want -200", w, d)
		}
	}

	if PanelNone.Pixels() != 0 || PanelRight.Pixels() != -200 || PanelLeft.Pixels() != 200 {
		t.Error("Pixels")
	}
}
