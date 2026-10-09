package d2player

import "testing"

type fakeGridItem struct {
	w, h, x, y int
}

func (f *fakeGridItem) InventoryGridSize() (width, height int) { return f.w, f.h }
func (f *fakeGridItem) GetItemCode() string                    { return "fake" }
func (f *fakeGridItem) InventoryGridSlot() (x, y int)          { return f.x, f.y }
func (f *fakeGridItem) SetInventoryGridSlot(x, y int)          { f.x, f.y = x, y }
func (f *fakeGridItem) GetItemDescription() []string           { return nil }

func TestItemGridCanFit(t *testing.T) {
	tests := []struct {
		name   string
		placed []*fakeGridItem
		x, y   int
		w, h   int
		want   bool
	}{
		{"empty", nil, 0, 0, 2, 2, true},
		{"touching right neighbour", []*fakeGridItem{{w: 2, h: 2, x: 0, y: 0}}, 2, 0, 2, 2, true},
		{"touching left neighbour", []*fakeGridItem{{w: 2, h: 2, x: 4, y: 0}}, 2, 0, 2, 2, true},
		{"touching below neighbour", []*fakeGridItem{{w: 2, h: 2, x: 0, y: 2}}, 0, 0, 2, 2, true},
		{"touching above neighbour", []*fakeGridItem{{w: 2, h: 2, x: 0, y: 0}}, 0, 2, 2, 2, true},
		{"diagonal touch", []*fakeGridItem{{w: 1, h: 1, x: 1, y: 1}}, 0, 0, 1, 1, true},
		{"overlap", []*fakeGridItem{{w: 2, h: 2, x: 0, y: 0}}, 1, 1, 2, 2, false},
		{"same cell", []*fakeGridItem{{w: 1, h: 1, x: 3, y: 3}}, 3, 3, 1, 1, false},
		{"outside right", nil, 9, 0, 2, 1, false},
		{"outside bottom", nil, 0, 3, 1, 2, false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			g := &ItemGrid{width: 10, height: 4}
			for _, p := range tt.placed {
				g.items = append(g.items, p)
			}

			if got := g.canFit(tt.x, tt.y, &fakeGridItem{w: tt.w, h: tt.h}); got != tt.want {
				t.Errorf("canFit=%v want %v", got, tt.want)
			}
		})
	}
}

func TestItemGridAddPacksTightly(t *testing.T) {
	// add() itself loads sprites (needs the UI); exercise the same scan via canFit.
	g := &ItemGrid{width: 4, height: 2}

	place := func(it *fakeGridItem) bool {
		for y := 0; y < g.height; y++ {
			for x := 0; x < g.width; x++ {
				if g.canFit(x, y, it) {
					it.x, it.y = x, y
					g.items = append(g.items, it)

					return true
				}
			}
		}

		return false
	}

	for i := 0; i < 4; i++ {
		if !place(&fakeGridItem{w: 2, h: 1}) {
			t.Fatalf("2x1 item %d should fit in a 4x2 grid", i)
		}
	}

	if place(&fakeGridItem{w: 1, h: 1}) {
		t.Error("grid should be full")
	}
}
