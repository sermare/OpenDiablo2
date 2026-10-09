package d2gamescreen

import "testing"

// fakeGround is a ground-item sink: piles keyed by id, all at one spot.
type fakeGround struct {
	piles map[int]int // id -> amount
	next  int
}

func newFakeGround() *fakeGround { return &fakeGround{piles: map[int]int{}} }

func (f *fakeGround) add(amount int) int {
	f.next++
	f.piles[f.next] = amount

	return f.next
}

func (f *fakeGround) pile(id int) goldPileFuncs {
	return goldPileFuncs{
		remove: func() { delete(f.piles, id) },
		leave:  func(amount int) { f.add(amount) },
	}
}

func (f *fakeGround) total() (sum int) {
	for _, a := range f.piles {
		sum += a
	}

	return sum
}

// fakePurse caps like GameControls.PickUpGold.
type fakePurse struct{ gold, limit int }

func (p *fakePurse) PickUpGold(amount int) int {
	room := p.limit - p.gold
	if room < 0 {
		room = 0
	}

	if amount <= room {
		p.gold += amount
		return 0
	}

	p.gold += room

	return amount - room
}

func TestPickUpGoldPile(t *testing.T) {
	// under the cap: pile consumed, nothing left
	g := newFakeGround()
	p := &fakePurse{gold: 1000, limit: 10000}
	id := g.add(500)

	if !pickUpGoldPile(p, g.pile(id), 500) || p.gold != 1500 || len(g.piles) != 0 {
		t.Fatalf("under cap: gold %d piles %v", p.gold, g.piles)
	}

	// overflow: remainder left as one new pile
	g = newFakeGround()
	p = &fakePurse{gold: 9000, limit: 10000}
	id = g.add(3000)

	if !pickUpGoldPile(p, g.pile(id), 3000) || p.gold != 10000 {
		t.Fatalf("overflow: gold %d", p.gold)
	}

	if len(g.piles) != 1 || g.total() != 2000 {
		t.Fatalf("overflow: piles %v, want one pile of 2000", g.piles)
	}

	if _, old := g.piles[id]; old {
		t.Error("the original pile must be gone")
	}

	rest := id + 1 // the remainder pile

	// purse already full: the pile is untouched (same id, same amount)
	if pickUpGoldPile(p, g.pile(rest), 2000) || g.piles[rest] != 2000 || p.gold != 10000 || len(g.piles) != 1 {
		t.Fatalf("full purse: gold %d piles %v", p.gold, g.piles)
	}

	// spend some gold, pick the remainder up: a second overflow
	p.gold = 9500

	if !pickUpGoldPile(p, g.pile(rest), 2000) || p.gold != 10000 {
		t.Fatalf("second overflow: gold %d", p.gold)
	}

	if len(g.piles) != 1 || g.total() != 1500 {
		t.Fatalf("second overflow: piles %v, want one pile of 1500", g.piles)
	}

	// total gold is conserved
	if p.gold+g.total() != 11500 {
		t.Errorf("gold not conserved: purse %d + ground %d", p.gold, g.total())
	}
}
