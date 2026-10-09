package d2playertrade

import (
	"errors"
	"testing"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

func size(code string) (int, int, bool) {
	switch code {
	case "rin", "amu", "hp1":
		return 1, 1, true
	case "lsd":
		return 1, 3, true
	case "plt":
		return 2, 3, true
	}

	return 0, 0, false
}

func inv(items ...d2hero.StoredItem) *d2hero.HeroContainers {
	return &d2hero.HeroContainers{Items: items}
}

func at(code string, x, y int) d2hero.StoredItem {
	return d2hero.StoredItem{Code: code, Page: d2hero.PageInventory, X: x, Y: y}
}

func codes(items []d2hero.StoredItem) map[string]int {
	m := map[string]int{}
	for _, it := range items {
		m[it.Code]++
	}

	return m
}

func TestCommitSwapsItemsAndGold(t *testing.T) {
	ga, gb := 500, 40
	a := Hand{inv(at("rin", 0, 0), at("lsd", 3, 0), at("hp1", 9, 3)), &ga}
	b := Hand{inv(at("amu", 5, 1)), &gb}

	res, err := Commit(a, b,
		Offer{Items: []d2hero.StoredItem{at("rin", 0, 0)}, Gold: 100},
		Offer{Items: []d2hero.StoredItem{at("amu", 5, 1)}, Gold: 10}, size)
	if err != nil {
		t.Fatal(err)
	}

	if got := codes(a.Containers.Items); got["rin"] != 0 || got["amu"] != 1 || got["lsd"] != 1 || len(a.Containers.Items) != 3 {
		t.Fatalf("a has %v", got)
	}

	if got := codes(b.Containers.Items); got["rin"] != 1 || got["amu"] != 0 || len(b.Containers.Items) != 1 {
		t.Fatalf("b has %v", got)
	}

	if ga != 500-100+10 || gb != 40+100-10 {
		t.Fatalf("gold a=%d b=%d", ga, gb)
	}

	if res[0].GoldBefore != 500 || res[0].GoldAfter != 410 || res[1].GoldBefore != 40 || res[1].GoldAfter != 130 {
		t.Fatalf("result gold: %+v", res)
	}

	if len(res[0].Gave) != 1 || res[0].Gave[0].Code != "rin" || len(res[0].Got) != 1 || res[0].Got[0].Code != "amu" {
		t.Fatalf("result items: %+v", res)
	}

	// the received item does not overlap anything already there
	for _, h := range []Hand{a, b} {
		occ := map[[2]int]bool{}

		for _, it := range h.Containers.Items {
			w, hh, _ := size(it.Code)
			for x := it.X; x < it.X+w; x++ {
				for y := it.Y; y < it.Y+hh; y++ {
					if occ[[2]int{x, y}] || x >= InvCols || y >= InvRows {
						t.Fatalf("item %s overlaps or is outside at %d,%d", it.Code, x, y)
					}

					occ[[2]int{x, y}] = true
				}
			}
		}
	}
}

func TestCommitIsAtomic(t *testing.T) {
	tests := []struct {
		name string
		a, b func() (Hand, Offer)
		want error
	}{
		{"missing item",
			func() (Hand, Offer) {
				g := 10
				return Hand{inv(at("rin", 0, 0)), &g}, Offer{Items: []d2hero.StoredItem{at("rin", 1, 1)}}
			},
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{Gold: 1} }, ErrMissingItem},
		{"wrong code at the position",
			func() (Hand, Offer) {
				g := 10
				return Hand{inv(at("rin", 0, 0)), &g}, Offer{Items: []d2hero.StoredItem{at("amu", 0, 0)}}
			},
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{} }, ErrMissingItem},
		{"not enough gold",
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{Gold: 11} },
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{} }, ErrGold},
		{"empty trade",
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{} },
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{} }, ErrEmpty},
		{"no inventory",
			func() (Hand, Offer) { g := 10; return Hand{nil, &g}, Offer{Gold: 1} },
			func() (Hand, Offer) { g := 10; return Hand{inv(), &g}, Offer{} }, ErrNoContainers},
	}

	for _, tc := range tests {
		a, oa := tc.a()
		b, ob := tc.b()
		ga, gb := *a.Gold, *b.Gold

		var before int
		if a.Containers != nil {
			before = len(a.Containers.Items)
		}

		if _, err := Commit(a, b, oa, ob, size); !errors.Is(err, tc.want) {
			t.Errorf("%s: err %v want %v", tc.name, err, tc.want)
		}

		if *a.Gold != ga || *b.Gold != gb || (a.Containers != nil && len(a.Containers.Items) != before) {
			t.Errorf("%s: a failed trade must not change anything", tc.name)
		}
	}
}

func TestCommitNoSpace(t *testing.T) {
	// b's inventory is completely full of 1x1 items, a gives a ring: no room
	var full []d2hero.StoredItem

	for x := 0; x < InvCols; x++ {
		for y := 0; y < InvRows; y++ {
			full = append(full, at("hp1", x, y))
		}
	}

	ga, gb := 0, 0
	a := Hand{inv(at("rin", 0, 0)), &ga}
	b := Hand{inv(full...), &gb}

	if _, err := Commit(a, b, Offer{Items: []d2hero.StoredItem{at("rin", 0, 0)}}, Offer{}, size); !errors.Is(err, ErrNoSpace) {
		t.Fatalf("err %v", err)
	}

	if len(a.Containers.Items) != 1 || len(b.Containers.Items) != 40 {
		t.Fatal("nothing may move")
	}

	// when b gives one back the freed cell is used
	if _, err := Commit(a, b, Offer{Items: []d2hero.StoredItem{at("rin", 0, 0)}},
		Offer{Items: []d2hero.StoredItem{at("hp1", 4, 2)}}, size); err != nil {
		t.Fatal(err)
	}

	var got d2hero.StoredItem

	for _, it := range b.Containers.Items {
		if it.Code == "rin" {
			got = it
		}
	}

	if got.X != 4 || got.Y != 2 {
		t.Fatalf("the ring must take the freed cell, got %d,%d", got.X, got.Y)
	}
}

func TestSessionFlow(t *testing.T) {
	now := time.Unix(1000, 0)
	s := NewSession("a", "b")
	s.SetClock(func() time.Time { return now })

	if err := s.SetOffer("a", Offer{Gold: 1}); !errors.Is(err, ErrState) {
		t.Fatalf("offer before the request is answered: %v", err)
	}

	if err := s.Respond("a", true); !errors.Is(err, ErrState) {
		t.Fatalf("the requester cannot answer: %v", err)
	}

	if err := s.Respond("c", true); !errors.Is(err, ErrNotInTrade) {
		t.Fatalf("a stranger: %v", err)
	}

	if err := s.Respond("b", true); err != nil || s.State() != Open {
		t.Fatalf("respond: %v %v", err, s.State())
	}

	if _, err := s.Accept("a"); !errors.Is(err, ErrEmpty) {
		t.Fatalf("accept of nothing: %v", err)
	}

	_ = s.SetOffer("a", Offer{Gold: 5})

	if _, err := s.Accept("b"); !errors.Is(err, ErrLocked) {
		t.Fatalf("accept right after a change: %v", err)
	}

	now = now.Add(AcceptLock + time.Millisecond)

	if ready, err := s.Accept("a"); err != nil || ready {
		t.Fatalf("first accept: %v %v", ready, err)
	}

	// a change revokes every acceptance
	_ = s.SetOffer("b", Offer{Gold: 2})

	if s.Accepted("a") || s.Accepted("b") {
		t.Fatal("a changed offer must revoke the acceptances")
	}

	now = now.Add(AcceptLock + time.Millisecond)
	_, _ = s.Accept("a")

	if ready, err := s.Accept("b"); err != nil || !ready {
		t.Fatalf("both accepted: %v %v", ready, err)
	}

	s.Finish()

	if s.State() != Done || s.Cancel("a") == nil {
		t.Fatal("a finished trade cannot be cancelled")
	}
}

func TestSessionDeclineAndCancel(t *testing.T) {
	s := NewSession("a", "b")
	_ = s.Respond("b", false)

	if s.State() != Cancelled {
		t.Fatal("declining cancels")
	}

	s = NewSession("a", "b")
	_ = s.Respond("b", true)

	if err := s.SetOffer("a", Offer{Gold: -1}); !errors.Is(err, ErrGold) {
		t.Fatalf("negative gold: %v", err)
	}

	dup := Offer{Items: []d2hero.StoredItem{at("rin", 0, 0), at("rin", 0, 0)}}
	if err := s.SetOffer("a", dup); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate: %v", err)
	}

	if err := s.Cancel("b"); err != nil || s.State() != Cancelled {
		t.Fatalf("cancel: %v", err)
	}
}
