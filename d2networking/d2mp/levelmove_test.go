package d2mp

import "testing"

func TestLevelMoveLimit(t *testing.T) {
	var l levelMoves

	roll := func(v int) func() int { return func() int { return v } }

	// four moves in quick succession are ordinary
	for i := uint32(1); i <= 4; i++ {
		if n := l.protectionFrames(i*100, roll(0)); n != protectionDefault {
			t.Fatalf("move %d: %d", i, n)
		}
	}

	// the fifth within the window is limited; the roll picks the row
	for _, tc := range []struct{ roll, want int }{{0, 1500}, {49, 1500}, {50, 3000}, {74, 3000}, {75, 4500}, {99, 4500}} {
		c := l
		if n := c.protectionFrames(500, roll(tc.roll)); n != tc.want {
			t.Errorf("roll %d: got %d want %d", tc.roll, n, tc.want)
		}
	}

	// moves older than the window free their slots
	if n := l.protectionFrames(100+levelMoveWindow+1, roll(0)); n != protectionDefault {
		t.Errorf("after the window: %d", n)
	}
}

func TestLevelMoveWindowEdge(t *testing.T) {
	var l levelMoves
	for i := 0; i < 5; i++ {
		l.record(10)
	}

	if !l.limited(10 + levelMoveWindow) {
		t.Error("2251 frames later is still inside")
	}

	if l.limited(10 + levelMoveWindow + 1) {
		t.Error("2252 frames later is outside")
	}
}
