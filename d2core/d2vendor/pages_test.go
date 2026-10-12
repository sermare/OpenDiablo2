package d2vendor

import "testing"

func TestStockPagesAreSeparateGrids(t *testing.T) {
	s := NewStock()
	a := &Item{Code: "a", W: 2, H: 3, Page: 0}
	b := &Item{Code: "b", W: 2, H: 3, Page: 1}
	n := &Item{Code: "n", W: 1, H: 1, Page: -1}

	if !s.Place(a) || !s.Place(b) || s.Place(n) {
		t.Fatal("placement")
	}

	if a.X != 0 || a.Y != 0 || b.X != 0 || b.Y != 0 {
		t.Fatalf("each page starts at the origin: %+v %+v", a, b)
	}

	if s.ItemAtPage(1, 1, 2) != b || s.ItemAtPage(0, 1, 2) != a || s.ItemAtPage(2, 0, 0) != nil {
		t.Fatal("ItemAtPage")
	}

	s.Remove(a)

	c := &Item{Code: "c", W: 2, H: 3, Page: 0}
	if !s.Place(c) || c.X != 0 || c.Y != 0 {
		t.Fatal("page 0 cell not freed")
	}
}

// The vendor's pages are 10 x 10 grids: a weapon sold to a vendor whose weapons page is full goes to page 2,
// every other page overflow is lost, an item without a store page is never kept (0x577320).
func TestPlaceSoldOverflowsWeaponsToMagicPage(t *testing.T) {
	s := NewStock()

	for i := 0; i < GridCols*GridRows; i++ {
		if !s.Place(&Item{Code: "f", W: 1, H: 1, Page: 1}) {
			t.Fatalf("filling the weapons page failed at %d", i)
		}
	}

	w := &Item{Code: "w", W: 1, H: 1, Page: 1}
	if !s.PlaceSold(w) || w.Page != 2 {
		t.Fatalf("weapon should land on page 2: %+v", w)
	}

	for i := 0; i < GridCols*GridRows; i++ {
		s.Place(&Item{Code: "a", W: 1, H: 1, Page: 0})
	}

	if a := (&Item{Code: "x", W: 1, H: 1, Page: 0}); s.PlaceSold(a) {
		t.Fatal("a full armor page has no overflow")
	}

	if n := (&Item{Code: "n", W: 1, H: 1, Page: -1}); s.PlaceSold(n) {
		t.Fatal("an item without a store page is not kept")
	}

	for i := 0; i < GridCols*GridRows-1; i++ {
		s.Place(&Item{Code: "m", W: 1, H: 1, Page: 2})
	}

	if full := (&Item{Code: "z", W: 1, H: 1, Page: 1}); s.PlaceSold(full) || full.Page != 1 {
		t.Fatalf("both pages full: refused, page restored, got page %d", full.Page)
	}
}

func TestLeaveTownRestock(t *testing.T) {
	tests := []struct {
		empty   bool
		elapsed int64
		want    bool
	}{
		{true, 0, true}, {true, RestockInterval, true}, {false, RestockInterval, false}, {false, RestockInterval + 1, true},
	}

	for _, tt := range tests {
		if got := LeaveTownRestock(tt.empty, tt.elapsed); got != tt.want {
			t.Errorf("LeaveTownRestock(%v,%d)=%v want %v", tt.empty, tt.elapsed, got, tt.want)
		}
	}
}
