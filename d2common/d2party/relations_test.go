package d2party

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func relRoster() *Roster {
	r := New()
	r.Add(Member{ID: "a", Name: "A", Level: d2enum.PlayersHostileLevel})
	r.Add(Member{ID: "b", Name: "B", Level: d2enum.PlayersHostileLevel})
	r.Add(Member{ID: "c", Name: "C", Level: d2enum.PlayersHostileLevel})

	return r
}

func TestRelationsPerDirection(t *testing.T) {
	r := relRoster()

	// a hostile declaration only creates a's node for b
	if err := r.SetHostile("a", "b", true); err != nil {
		t.Fatal(err)
	}

	if _, ok := r.RelationOf("a", "b"); !ok {
		t.Fatal("a->b missing")
	}

	if _, ok := r.RelationOf("b", "a"); ok {
		t.Fatal("b->a must not exist (per direction)")
	}
}

func TestRelationPartyBit(t *testing.T) {
	r := relRoster()

	// c is in a party with b; a then relates to c: bit set
	if err := r.Invite("b", "c"); err != nil {
		t.Fatal(err)
	}

	if _, err := r.Accept("c"); err != nil {
		t.Fatal(err)
	}

	if err := r.AddRelation("a", "c"); err != nil {
		t.Fatal(err)
	}

	if n, _ := r.RelationOf("a", "c"); !n.OtherInParty {
		t.Error("c is in a party: bit expected")
	}

	if n, _ := r.RelationOf("a", "b"); n.OtherInParty {
		t.Error("no node a->b yet")
	}

	if err := r.AddRelation("a", "b"); err != nil {
		t.Fatal(err)
	}

	if n, _ := r.RelationOf("c", "b"); !n.OtherInParty {
		t.Error("c->b: b is in the party")
	}

	// leaving dissolves the party and clears the bits
	r.Leave("c")

	if n, _ := r.RelationOf("a", "c"); n.OtherInParty {
		t.Error("bit must clear after the party dissolved")
	}
}

func TestRelationRefusedForState7(t *testing.T) {
	for _, dead := range []string{"a", "b"} {
		r := relRoster()
		r.SetPlayerBody(dead, true)

		if err := r.AddRelation("a", "b"); !errors.Is(err, ErrPlayerBody) {
			t.Errorf("%s dead: AddRelation %v", dead, err)
		}

		if err := r.Invite("a", "b"); !errors.Is(err, ErrPlayerBody) {
			t.Errorf("%s dead: Invite %v", dead, err)
		}

		if err := r.SetHostile("a", "b", true); !errors.Is(err, ErrPlayerBody) {
			t.Errorf("%s dead: SetHostile %v", dead, err)
		}

		if _, ok := r.RelationOf("a", "b"); ok {
			t.Errorf("%s dead: node created", dead)
		}
	}

	r := relRoster()
	if err := r.AddRelation("a", "a"); !errors.Is(err, ErrSelf) {
		t.Errorf("self: %v", err)
	}
}

func TestRemoveDropsRelations(t *testing.T) {
	r := relRoster()
	_ = r.AddRelation("a", "b")
	_ = r.AddRelation("b", "a")
	r.Remove("b")

	if _, ok := r.RelationOf("a", "b"); ok {
		t.Error("a node outlived its target")
	}
}
