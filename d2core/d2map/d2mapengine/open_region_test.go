package d2mapengine

import "testing"

// Playtest bug (Act 3, Upper Kurast): the edge arrival lay in a 5x5 alcove closed on all sides by building
// walls; the scripted hero could not take a step ("EXIT no way found"). The arrival moves out of pockets.
func TestNearestOpenCellSkipsClosedPockets(t *testing.T) {
	const w, h = 60, 40

	// a wall ring around a 5x5 alcove at (10..14, 10..14), open ground everywhere else
	blocked := func(x, y int) bool {
		inRing := x >= 9 && x <= 15 && y >= 9 && y <= 15
		inside := x >= 10 && x <= 14 && y >= 10 && y <= 14

		return inRing && !inside
	}

	cases := []struct {
		name     string
		sx, sy   int
		wantFree bool
		inPocket bool
	}{
		{"inside the alcove", 12, 12, true, false},
		{"open ground", 30, 30, true, false},
		{"on the wall", 9, 12, true, false},
	}

	for _, c := range cases {
		x, y, ok := nearestOpenCell(blocked, w, h, c.sx, c.sy, 50, 100)
		if ok != c.wantFree {
			t.Errorf("%s: ok = %v", c.name, ok)
			continue
		}

		if blocked(x, y) || (x >= 10 && x <= 14 && y >= 10 && y <= 14) {
			t.Errorf("%s: arrival (%d,%d) is blocked or inside the alcove", c.name, x, y)
		}
	}

	// the open ground itself stays where it is
	if x, y, _ := nearestOpenCell(blocked, w, h, 30, 30, 50, 100); x != 30 || y != 30 {
		t.Errorf("open ground moved to (%d,%d)", x, y)
	}

	// no free region of that size within the radius
	if _, _, ok := nearestOpenCell(blocked, w, h, 12, 12, 3, 100); ok {
		t.Errorf("found a cell although none lies within 3 sub-tiles")
	}
}
