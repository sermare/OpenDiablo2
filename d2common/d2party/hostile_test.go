package d2party

import (
	"testing"
	"time"
)

func TestHostileCooldown(t *testing.T) {
	r := afield()
	clock := time.Unix(1000, 0)
	r.Now = func() time.Time { return clock }

	if _, err := r.DeclareHostile("a", "b"); err != nil {
		t.Fatal(err)
	}

	_ = r.SetHostile("a", "b", false)

	tests := []struct {
		advance time.Duration
		want    error
	}{
		{10 * time.Second, ErrHostileWait},
		{49 * time.Second, ErrHostileWait}, // 59 s in total
		{1 * time.Second, nil},             // 60 s: allowed again
	}

	for _, tc := range tests {
		clock = clock.Add(tc.advance)

		if _, err := r.DeclareHostile("a", "b"); err != tc.want {
			t.Fatalf("after +%v: %v want %v", tc.advance, err, tc.want)
		}
	}

	// the cooldown belongs to the declarer
	// (a's declaration already made b hostile to a; b withdraws its own flag first)
	_ = r.SetHostile("b", "a", false)

	if d, err := r.DeclareHostile("b", "a"); err != nil || !d.Changed {
		t.Fatalf("%+v %v", d, err)
	}
}

func TestPeaceIsNotThrottled(t *testing.T) {
	r := afield()
	r.Now = func() time.Time { return time.Unix(1000, 0) }

	if _, err := r.DeclareHostile("a", "b"); err != nil {
		t.Fatal(err)
	}

	if err := r.SetHostile("a", "b", false); err != nil || r.Hostile("a", "b") {
		t.Fatalf("peace: %v", err)
	}
}

func TestDeclareAlreadyHostileIsNoop(t *testing.T) {
	r := afield()
	if _, err := r.DeclareHostile("a", "b"); err != nil {
		t.Fatal(err)
	}

	d, err := r.DeclareHostile("a", "b")
	if err != nil || d.Changed || d.PortalsToClose() {
		t.Fatalf("%+v %v", d, err)
	}
}

func TestDeclareAcrossPartiesKeepsParty(t *testing.T) {
	r := afield()
	_ = r.Invite("a", "d")
	_, _ = r.Accept("d")

	d, err := r.DeclareHostile("a", "b")
	if err != nil || d.LeftParty || !r.SameParty("a", "d") {
		t.Fatalf("%+v %v", d, err)
	}
}

// afield is game() with a, b and c in a town (hostility is declared and withdrawn in town, w45).
func afield() *Roster { return game() }

func TestHostileOnlyInTown(t *testing.T) {
	r := game() // a and b stand in area 1, the Rogue Encampment

	r.SetArea("a", 2)

	if _, err := r.DeclareHostile("a", "b"); err != ErrHostileTown {
		t.Fatalf("afield: %v want ErrHostileTown", err)
	}

	if r.Hostile("a", "b") {
		t.Fatal("declared outside town")
	}

	r.SetArea("a", 1)

	if _, err := r.DeclareHostile("a", "b"); err != nil {
		t.Fatalf("in town: %v", err)
	}

	// peace uses the same gate: refused afield, the flag stays
	r.SetArea("a", 2)

	if err := r.SetHostile("a", "b", false); err != ErrHostileTown || !r.Hostile("a", "b") {
		t.Fatalf("peace afield: %v hostile=%v", err, r.Hostile("a", "b"))
	}

	r.SetArea("a", 40) // Lut Gholein is a town too

	if err := r.SetHostile("a", "b", false); err != nil || r.Hostile("a", "b") {
		t.Fatalf("peace in town: %v", err)
	}
}
