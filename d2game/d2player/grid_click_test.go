package d2player

import "testing"

func TestClickWith(t *testing.T) {
	// grid origin (100,200), 29 px cells, 10x4
	cell := func(x, y int) (int, int) { return 100 + x*29 + 5, 200 + y*29 + 5 }

	a := &fakeGridItem{w: 1, h: 1, x: 3, y: 1}
	b := &fakeGridItem{w: 1, h: 1, x: 4, y: 1}
	g := newTestGrid(10, 4, a, b)

	// empty cursor on an item picks it up
	mx, my := cell(3, 1)

	held, act, _, _ := g.ClickWith(nil, mx, my, false)
	if act != ClickPickup || held != InventoryItem(a) || len(g.items) != 1 {
		t.Fatalf("pickup: %v %v", act, held)
	}

	// empty cursor on empty cells does nothing
	if _, act, _, _ := g.ClickWith(nil, mx, my, false); act != ClickNoTarget {
		t.Errorf("empty click: %v", act)
	}

	// held item into free cells is placed
	held, act, x, y := g.ClickWith(a, mx, my, false)
	if act != ClickPlace || held != nil || x != 3 || y != 1 || len(g.items) != 2 {
		t.Fatalf("place: %v (%d,%d)", act, x, y)
	}

	// a different item onto exactly one item swaps
	c := &fakeGridItem{w: 1, h: 1}
	bx, by := cell(4, 1)

	held, act, _, _ = g.ClickWith(c, bx, by, false)
	if act != ClickSwap || held != InventoryItem(b) || len(g.items) != 2 {
		t.Fatalf("swap: %v %v", act, held)
	}

	// a 2x1 item over two items is refused and stays on the cursor
	wide := &fakeGridItem{w: 2, h: 1}

	held, act, _, _ = g.ClickWith(wide, 100+29*4, 200+29*1+5, false)
	if act != ClickRefused || held != InventoryItem(wide) {
		t.Fatalf("refuse: %v", act)
	}

	// ctrl auto-places
	d := &fakeGridItem{w: 1, h: 1}

	held, act, _, _ = g.ClickWith(d, mx, my, true)
	if act != ClickAuto || held != nil || len(g.items) != 3 {
		t.Fatalf("auto: %v", act)
	}
}
