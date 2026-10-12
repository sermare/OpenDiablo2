package d2party

import "testing"

func TestCorpseLootPermission(t *testing.T) {
	r := game()

	if !r.CanLootCorpse("a", "a") || r.CanLootCorpse("a", "b") {
		t.Fatal("only the owner by default")
	}

	_ = r.Invite("a", "b")
	_, _ = r.Accept("b")

	if r.CanLootCorpse("a", "b") {
		t.Fatal("a party does not grant loot by itself")
	}

	if err := r.SetLootAllowed("a", "b", true); err != nil || !r.CanLootCorpse("a", "b") {
		t.Fatalf("allowed: %v", err)
	}

	if r.CanLootCorpse("b", "a") {
		t.Fatal("the permission is one-way")
	}

	_ = r.SetLootAllowed("a", "b", false)

	if r.CanLootCorpse("a", "b") || r.SetLootAllowed("a", "a", true) != ErrSelf || r.SetLootAllowed("a", "x", true) != ErrUnknownPlayer {
		t.Fatal("withdrawal / errors")
	}
}
