package d2gamescreen

import "testing"

func TestTakeHirelingPopup(t *testing.T) {
	seen, marks := false, 0

	SetHirelingPopupFlag(func() bool { return seen }, func() { seen = true; marks++ })
	t.Cleanup(func() { SetHirelingPopupFlag(nil, nil); hirelingPopupLive = false })

	if !takeHirelingPopup() || marks != 1 {
		t.Fatalf("first hire: marks=%d", marks)
	}

	if takeHirelingPopup() || marks != 1 {
		t.Fatalf("second hire showed it again: marks=%d", marks)
	}
}
