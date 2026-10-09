package d2portal

import (
	"errors"
	"testing"
)

func TestPlanOpen(t *testing.T) {
	// cast in the Cold Plains (level 3, act 1): the town end is Rogue Encampment
	p, err := PlanOpen("a", "Ann", 3, 10.5, 20.5, nil)
	if err != nil {
		t.Fatal(err)
	}

	if p.Field.Level != 3 || p.Field.X != 10.5 || p.Town.Level != 1 || !p.Town.Auto || p.Owner != "a" {
		t.Fatalf("pair %+v", p)
	}

	// act 2 field level -> Lut Gholein, act 5 -> Harrogath
	for field, town := range map[int]int{41: 40, 76: 75, 104: 103, 112: 109} {
		q, err := PlanOpen("a", "Ann", field, 1, 1, nil)
		if err != nil || q.Town.Level != town {
			t.Errorf("field %d: town %d (%v), want %d", field, q.Town.Level, err, town)
		}
	}

	// cast in town without a pair is refused; with one the town end moves
	if _, err := PlanOpen("a", "Ann", 1, 5, 5, nil); !errors.Is(err, ErrNoDestination) {
		t.Fatalf("town cast without pair: %v", err)
	}

	moved, err := PlanOpen("a", "Ann", 1, 5, 6, &p)
	if err != nil || moved.Field != p.Field || moved.Town.X != 5 || moved.Town.Auto {
		t.Fatalf("moved %+v %v", moved, err)
	}

	if _, err := PlanOpen("a", "Ann", 0, 0, 0, nil); !errors.Is(err, ErrBadLevel) {
		t.Fatalf("bad level: %v", err)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	a, _ := PlanOpen("a", "Ann", 3, 1, 1, nil)
	b, _ := PlanOpen("b", "Bob", 4, 2, 2, nil)

	sa, _, had := r.Open(a)
	if had || sa.ID != 1 {
		t.Fatalf("first open: %+v %v", sa, had)
	}

	sb, _, _ := r.Open(b)
	if sb.ID != 2 {
		t.Fatalf("second id %d", sb.ID)
	}

	// both levels see the pair: the field end and the town end
	if got := r.In(3); len(got) != 1 || got[0].Owner != "a" || got[0].InTown || got[0].Dest.Level != 1 {
		t.Fatalf("level 3: %+v", got)
	}

	if got := r.In(1); len(got) != 2 || !got[0].InTown || got[0].Dest.Level != 3 {
		t.Fatalf("town: %+v", got)
	}

	// recasting in the field replaces the owner's pair (new id, old field end gone)
	a2, _ := PlanOpen("a", "Ann", 5, 9, 9, nil)

	sa2, old, had := r.Open(a2)
	if !had || old.ID != 1 || sa2.ID != 3 || len(r.In(3)) != 0 || len(r.In(5)) != 1 {
		t.Fatalf("replace: new %+v old %+v had %v", sa2, old, had)
	}

	// casting in town keeps the id and the field end
	mv, _ := PlanOpen("a", "Ann", 1, 7, 7, &sa2)
	mv.ID = sa2.ID

	if got, _, _ := r.Open(mv); got.ID != sa2.ID || len(r.Pairs()) != 2 {
		t.Fatalf("move town end: %+v pairs %d", got, len(r.Pairs()))
	}

	if _, ok := r.Close("a"); !ok || len(r.Pairs()) != 1 {
		t.Fatal("close")
	}

	r2 := NewRegistry()
	r2.Replace(r.Pairs())

	if len(r2.Pairs()) != 1 || r2.next != 3 {
		t.Fatalf("replace content: %+v next %d", r2.Pairs(), r2.next)
	}
}

func TestCanUse(t *testing.T) {
	p := Portal{Owner: "a"}
	tests := []struct {
		user              string
		party, restricted bool
		wantErr           bool
	}{
		{"a", false, true, false},
		{"b", true, true, false},
		{"b", false, true, true},
		{"b", false, false, false},
	}

	for _, tc := range tests {
		if err := CanUse(p, tc.user, tc.party, tc.restricted); (err != nil) != tc.wantErr {
			t.Errorf("%+v: %v", tc, err)
		}
	}
}
