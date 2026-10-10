package d2player

import "testing"

func TestGameMenuLayout(t *testing.T) {
	// 800x600: centre (600-80)/2 = 260; 5 entries -> 260-125 = 135
	for i, want := range []int{135, 185, 235} {
		if got := Mode800.GameMenuRowTop(5, i); got != want {
			t.Errorf("row %d: got %d want %d", i, got, want)
		}
	}

	if got := Mode640.GameMenuRowTop(3, 0); got != 200-75 {
		t.Errorf("640 row 0: %d", got)
	}

	l, r, y := Mode800.GameMenuPentagrams(135, 54)
	if l != 400-54-249 || r != 649 || y != 186 {
		t.Errorf("pents %d %d %d", l, r, y)
	}

	for tick, want := range [][2]int{{0, 0}, {7, 1}, {6, 2}, {5, 3}, {4, 4}, {3, 5}, {2, 6}, {1, 7}, {0, 0}, {7, 1}} {
		if a, b := GameMenuPentFrames(tick); a != want[0] || b != want[1] {
			t.Errorf("tick %d: %d %d", tick, a, b)
		}
	}
}
