package d2statlist

import "testing"

func TestTreeAttachDetachSkip(t *testing.T) {
	const strength, armor = 0, StatArmorClass

	skip := func(id int) bool { return id == armor }
	root := NewTree(Owner{Type: 0, ID: 1}, FlagBaseUnit)
	root.Add(strength, 0, 10)

	item := NewTree(Owner{Type: 4, ID: 7}, 0)
	item.Add(strength, 0, 5)
	item.Add(armor, 0, 30)
	item.Attach(root, skip)

	if item.Flags&FlagApplied == 0 || root.Children() != 1 {
		t.Fatalf("flags %#x children %d", item.Flags, root.Children())
	}

	if g := root.Get(strength); g != 15 {
		t.Errorf("strength after attach = %d, want 15", g)
	}

	if g := root.Get(armor); g != 0 {
		t.Errorf("a stat that does not apply to the parent reached it: %d", g)
	}

	if g := item.Get(armor); g != 30 {
		t.Errorf("the item's own read = %d, want 30", g)
	}

	// a change to an attached list reaches the parent
	item.Add(strength, 0, 2)

	if g := root.Get(strength); g != 17 {
		t.Errorf("strength after a change = %d, want 17", g)
	}

	// attaching twice does nothing
	item.Attach(root, skip)

	if root.Get(strength) != 17 || root.Children() != 1 {
		t.Error("a second attach changed the parent")
	}

	item.Detach()

	if g := root.Get(strength); g != 10 || root.Children() != 0 || item.Flags&FlagApplied != 0 {
		t.Errorf("after detach: strength %d children %d flags %#x", g, root.Children(), item.Flags)
	}
}

func TestTreeNested(t *testing.T) {
	root := NewTree(Owner{ID: 1}, FlagBaseUnit)
	mid := NewTree(Owner{Type: 4}, 0)
	leaf := NewTree(Owner{Type: 4}, FlagSetState)

	leaf.Add(3, 0, 4)
	leaf.Attach(mid, nil)
	mid.Add(3, 0, 1)
	mid.Attach(root, nil)

	if g := root.Get(3); g != 5 {
		t.Errorf("nested sum = %d, want 5", g)
	}

	leaf.Add(3, 0, 1)

	if g := root.Get(3); g != 6 {
		t.Errorf("change in a grandchild = %d, want 6", g)
	}

	mid.Detach()

	if g := root.Get(3); g != 0 {
		t.Errorf("after detaching the subtree = %d, want 0", g)
	}
}

// Partial bonuses switch on at 2 to 5 worn pieces and the full bonus with the
// whole set; the totals come from the tree.
func TestComputeSetTiers(t *testing.T) {
	const set = 3

	defs := SetDefs{set: {
		Pieces:  5,
		Partial: [][]Prop{{{ID: StatStrength, Value: 1}}, {{ID: StatStrength, Value: 10}}, {{ID: StatStrength, Value: 100}}, {{ID: StatStrength, Value: 1000}}},
		Full:    []Prop{{ID: StatStrength, Value: 10000}},
	}}
	slots := []int{SlotHead, SlotTorso, SlotRightHand, SlotLeftHand, SlotGloves}
	want := []int64{0, 0, 1, 11, 111, 11111}

	for n := 0; n <= 5; n++ {
		var items []Item
		for i := 0; i < n; i++ {
			items = append(items, Item{Code: "x", Slot: slots[i], SetID: set})
		}

		tot := Compute(Hero{Level: 1}, items, &Env{Sets: defs})

		if g := tot.Stats.Get(StatStrength); g != want[n] {
			t.Errorf("%d pieces: strength bonus %d, want %d", n, g, want[n])
		}

		if tot.SetPieces[set] != n && !(n == 0 && tot.SetPieces[set] == 0) {
			t.Errorf("%d pieces: SetPieces = %d", n, tot.SetPieces[set])
		}
	}
}
