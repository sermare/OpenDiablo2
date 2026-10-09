package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

func newTestGrid(w, h int, placed ...*fakeGridItem) *ItemGrid {
	// a nil sprite entry makes Load skip the UI
	g := &ItemGrid{width: w, height: h, originX: 100, originY: 200, slotSize: 29,
		sprites: map[string]*d2ui.Sprite{"fake": nil}}

	for _, p := range placed {
		g.items = append(g.items, p)
	}

	return g
}

func TestCursorAnchor(t *testing.T) {
	// grid origin (100,200), 29 px cells, 10x4 cells
	tests := []struct {
		name         string
		mx, my       int
		w, h         int
		wantX, wantY int
	}{
		{"1x1 inside cell", 100 + 29*3 + 5, 200 + 29*2 + 5, 1, 1, 3, 2},
		{"2x3 held by the middle", 100 + 29*3 + 5, 200 + 29*2 + 5, 2, 3, 2, 1},
		{"2x2 between cells rounds", 100 + 29*4, 200 + 29*2, 2, 2, 3, 1},
		{"clamped at the left/top", 100, 200, 2, 3, 0, 0},
		{"clamped at the right/bottom", 100 + 29*9 + 10, 200 + 29*3 + 10, 2, 2, 8, 2},
	}

	for _, tt := range tests {
		x, y := CursorAnchor(tt.mx, tt.my, 100, 200, 29, tt.w, tt.h, 10, 4)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("%s: got (%d,%d) want (%d,%d)", tt.name, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestItemGridAutoPlace(t *testing.T) {
	g := newTestGrid(10, 4)

	// the original's player search takes the best-scoring slot, so an item in an
	// empty grid lands in a corner (all-edge neighbours score highest)
	first := &fakeGridItem{w: 1, h: 1}

	x, y, ok := g.AutoPlace(first)
	if !ok {
		t.Fatal("empty grid refused an item")
	}

	if (x != 0 && x != 9) || (y != 0 && y != 3) {
		t.Errorf("first item at (%d,%d), want a grid corner", x, y)
	}

	// fill the grid with 2x2 items; every placement must stay inside and not overlap
	for i := 0; i < 5; i++ {
		it := &fakeGridItem{w: 2, h: 2}
		if _, _, ok := g.AutoPlace(it); !ok {
			t.Fatalf("2x2 item %d should fit", i)
		}
	}

	for i, a := range g.items {
		ax, ay := a.InventoryGridSlot()
		aw, ah := a.InventoryGridSize()

		if ax < 0 || ay < 0 || ax+aw > 10 || ay+ah > 4 {
			t.Errorf("item %d outside the grid at (%d,%d)", i, ax, ay)
		}

		for j, b := range g.items {
			if i < j && len(g.Overlapping(ax, ay, aw, ah)) > 1 {
				bx, by := b.InventoryGridSlot()
				t.Errorf("items %d (%d,%d) and %d (%d,%d) overlap", i, ax, ay, j, bx, by)
			}
		}
	}

	// a full grid refuses
	full := newTestGrid(2, 2, &fakeGridItem{w: 2, h: 2})
	if _, _, ok := full.AutoPlace(&fakeGridItem{w: 1, h: 1}); ok {
		t.Error("full grid accepted an item")
	}
}

func TestItemGridOverlappingAndContains(t *testing.T) {
	a := &fakeGridItem{w: 2, h: 2, x: 0, y: 0}
	b := &fakeGridItem{w: 1, h: 1, x: 3, y: 0}
	g := newTestGrid(10, 4, a, b)

	if n := len(g.Overlapping(1, 1, 1, 1)); n != 1 {
		t.Errorf("one item under (1,1), got %d", n)
	}

	if n := len(g.Overlapping(1, 0, 3, 1)); n != 2 {
		t.Errorf("two items under the wide footprint, got %d", n)
	}

	if n := len(g.Overlapping(5, 3, 2, 1)); n != 0 {
		t.Errorf("no item under an empty footprint, got %d", n)
	}

	if !g.Contains(100, 200) || g.Contains(99, 200) || g.Contains(100+29*10, 200) || !g.Contains(100+29*10-1, 200+29*4-1) {
		t.Error("Contains boundaries wrong")
	}
}
