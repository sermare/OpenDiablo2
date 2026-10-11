package d2summon

import "testing"

func TestTargetsOwnerAndEntries(t *testing.T) {
	r := NewTargets()

	a, _ := r.AddPlayer(100)
	b, _ := r.AddPlayer(101)

	if a != 0 || b != 1 {
		t.Fatalf("slots %d %d", a, b)
	}

	if _, err := r.AddPlayer(100); err != ErrInBucket {
		t.Error("double add")
	}

	if r.BucketOf(7) != NoBucket {
		t.Error("unknown unit")
	}

	_ = r.AddEntry(1, "a", a)
	_ = r.AddEntry(2, "b", a)

	if m := r.Members(a); len(m) != 2 || m[0].Unit != 2 {
		t.Errorf("order %+v", m)
	}

	if r.AddEntry(3, "x", 5) != ErrNoOwner || r.AddEntry(1, "x", b) != ErrInBucket {
		t.Error("add errors")
	}

	if r.AddGlobal(4, "x", 3) != ErrBucket || r.AddGlobal(4, "x", 9) != nil || r.AddGlobal(5, "x", 9) != nil {
		t.Error("global add")
	}

	if m := r.Members(9); m[0].Unit != 5 || r.BucketOf(4) != 9 {
		t.Errorf("global LIFO %+v", m)
	}

	if !r.Remove(1) || r.BucketOf(1) != NoBucket || len(r.Members(a)) != 1 || r.Remove(1) {
		t.Error("remove")
	}

	rel := r.OwnerDied(100)
	if len(rel) != 1 || r.BucketOf(2) != NoBucket || r.BucketOf(100) != NoBucket || len(r.Members(a)) != 0 {
		t.Errorf("owner death %+v", rel)
	}

	// the freed slot is reused
	if s, _ := r.AddPlayer(102); s != 0 {
		t.Errorf("slot reuse %d", s)
	}

	if n := r.FreeAll(); n != 4|| r.BucketOf(4) != NoBucket {
		t.Errorf("free all %d", n)
	}
}

func TestTargetsPlayerLimit(t *testing.T) {
	r := NewTargets()

	for i := 0; i < MaxPlayers; i++ {
		if _, err := r.AddPlayer(uint32(i + 1)); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := r.AddPlayer(99); err != ErrNoSlot {
		t.Error("ninth player")
	}
}
