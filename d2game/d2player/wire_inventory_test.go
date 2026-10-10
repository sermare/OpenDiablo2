package d2player

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// stackItem is a stackable fake: code, quantity, limit and an optional durability.
type stackItem struct {
	fakeGridItem
	code        string
	qty, limit  int
	noStack     bool
	dur, durMax int
}

func (s *stackItem) GetItemCode() string    { return s.code }
func (s *stackItem) IsStackable() bool      { return !s.noStack }
func (s *stackItem) Quantity() int          { return s.qty }
func (s *stackItem) SetQuantity(n int)      { s.qty = n }
func (s *stackItem) StackLimit() int        { return s.limit }
func (s *stackItem) Durability() (int, int) { return s.dur, s.durMax }
func (s *stackItem) SetDurability(n int)    { s.dur = n }

func newStack(code string, qty, limit int) *stackItem {
	return &stackItem{fakeGridItem: fakeGridItem{w: 1, h: 1}, code: code, qty: qty, limit: limit}
}

func TestStackOnDropMerge(t *testing.T) {
	mx, my := 100+3*29+5, 200+1*29+5 // cell (3,1)

	tests := []struct {
		name       string
		grid, held int
		limit      int
		wantGrid   int
		wantRest   int // 0 = cursor consumed
		wantAct    ClickAction
		heldCode   string
		heldNoStk  bool
	}{
		{"fits", 10, 5, 20, 15, 0, ClickMerge, "arrow", false},
		{"exact", 10, 10, 20, 20, 0, ClickMerge, "arrow", false},
		{"surplus stays on the cursor", 18, 5, 20, 20, 3, ClickMerge, "arrow", false},
		{"grid stack above the limit is set to it", 25, 5, 20, 20, 10, ClickMerge, "arrow", false},
		{"different base item swaps", 10, 5, 20, 10, 5, ClickSwap, "bolt", false},
		{"not stackable swaps", 10, 5, 20, 10, 5, ClickSwap, "arrow", true},
	}

	for _, tc := range tests {
		onGrid := newStack("arrow", tc.grid, tc.limit)
		onGrid.x, onGrid.y = 3, 1
		g := newTestGrid(10, 4)
		g.sprites["arrow"], g.sprites["bolt"] = nil, nil
		g.items = []InventoryItem{onGrid}

		held := newStack(tc.heldCode, tc.held, tc.limit)
		held.noStack = tc.heldNoStk

		cursor, act, _, _ := g.ClickWith(held, mx, my, false)
		if act != tc.wantAct {
			t.Errorf("%s: act = %v, want %v", tc.name, act, tc.wantAct)
			continue
		}

		if act == ClickSwap {
			if onGrid.qty != tc.grid || held.qty != tc.held || cursor != InventoryItem(onGrid) {
				t.Errorf("%s: swap changed quantities (%d,%d)", tc.name, onGrid.qty, held.qty)
			}

			continue
		}

		if onGrid.qty != tc.wantGrid || len(g.items) != 1 {
			t.Errorf("%s: grid stack = %d (%d items), want %d", tc.name, onGrid.qty, len(g.items), tc.wantGrid)
		}

		switch {
		case tc.wantRest == 0 && cursor != nil:
			t.Errorf("%s: cursor should be empty", tc.name)
		case tc.wantRest > 0 && (cursor != InventoryItem(held) || held.qty != tc.wantRest):
			t.Errorf("%s: cursor rest %d, want %d", tc.name, held.qty, tc.wantRest)
		}
	}
}

func TestStackMergeDurabilityTakesLower(t *testing.T) {
	a, b := newStack("jav", 5, 20), newStack("jav", 5, 20)
	a.dur, a.durMax, b.dur, b.durMax = 30, 40, 12, 40

	if _, merged := mergeIntoStack(a, b); !merged || a.dur != 12 || a.qty != 10 {
		t.Fatalf("merged=%v dur=%d qty=%d", merged, a.dur, a.qty)
	}

	// a higher cursor durability leaves the grid one alone
	c, d := newStack("jav", 5, 20), newStack("jav", 5, 20)
	c.dur, c.durMax, d.dur, d.durMax = 10, 40, 35, 40
	mergeIntoStack(c, d)

	if c.dur != 10 {
		t.Errorf("dur = %d, want 10", c.dur)
	}
}

// The original has no split (packet 0x22 is a stub): picking up with ctrl held
// never divides a stack.
func TestNoSplitOnCtrlClick(t *testing.T) {
	onGrid := newStack("arrow", 20, 30)
	onGrid.x, onGrid.y = 3, 1
	g := newTestGrid(10, 4)
	g.items = []InventoryItem{onGrid}

	cursor, act, _, _ := g.ClickWith(nil, 100+3*29+5, 200+1*29+5, true)
	if act != ClickPickup || cursor != InventoryItem(onGrid) || onGrid.qty != 20 {
		t.Fatalf("act %v qty %d", act, onGrid.qty)
	}
}

func TestPickUpGoldCap(t *testing.T) {
	tests := []struct {
		name                string
		level, have, pickup int
		wantHave, wantOver  int
	}{
		{"fits", 10, 1000, 500, 1500, 0},
		{"exactly full", 10, 99000, 1000, 100000, 0},
		{"overflow stays on the ground", 10, 99000, 5000, 100000, 4000},
		{"already full", 10, 100000, 300, 100000, 300},
		{"level 1", 1, 0, 25000, 10000, 15000},
		{"level 94", 94, 900000, 100000, 940000, 60000},
	}

	for _, tc := range tests {
		hero := &d2mapentity.Player{Gold: tc.have, Stats: &d2hero.HeroStatsState{Level: tc.level}}
		g := &GameControls{hero: hero, inventory: &Inventory{moveGoldPanel: &MoveGoldPanel{gold: tc.have}}}

		over := g.PickUpGold(tc.pickup)
		if over != tc.wantOver || hero.Gold != tc.wantHave || g.inventory.Gold() != tc.wantHave {
			t.Errorf("%s: over=%d gold=%d panel=%d, want %d/%d", tc.name, over, hero.Gold, g.inventory.Gold(),
				tc.wantOver, tc.wantHave)
		}
	}
}

func TestBeltBoxesFor(t *testing.T) {
	tests := []struct {
		name string
		worn bool
		typ  int
		want int
	}{
		{"no belt", false, 5, 4},
		{"plain belt", true, 0, 12},
		{"sash", true, 1, 8},
		{"girdle", true, 3, 16},
		{"light belt", true, 4, 8},
		{"heavy belt", true, 5, 12},
		{"exceptional/elite belt", true, 6, 16},
		{"unknown type falls back to the default row", true, 99, 4},
	}

	for _, tc := range tests {
		if got := beltBoxesFor(tc.worn, tc.typ); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.name, got, tc.want)
		}
	}
}
