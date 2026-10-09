package d2missile

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

type scanTarget struct {
	fakeTarget
	serial int
}

func (s *scanTarget) Serial() int { return s.serial }

func cand(id string, serial, x, y int, alive bool) Candidate {
	return Candidate{Target: &scanTarget{fakeTarget{id: id, alive: alive}, serial}, X: x, Y: y}
}

func ids(ts []Target) (out []string) {
	for _, t := range ts {
		out = append(out, t.ID())
	}

	return out
}

// Verified 0x569510 / 0x569100: euclidean radius in subtiles (squared
// compare, edge inclusive), living only, line of sight, lowest unit id first.
func TestScanFiltersAndOrder(t *testing.T) {
	g := d2path.NewCellGrid(-5, -20, 205, 40)
	owner := d2path.Point{X: 0, Y: 0}
	cs := []Candidate{
		cand("far", 1, 20, 0, true),  // outside the radius
		cand("edge", 9, 10, 0, true), // exactly on the radius
		cand("diag", 5, 8, 7, true),  // 64+49 = 113 > 100: out
		cand("dead", 2, 3, 0, false), // dead
		cand("near", 7, 1, 1, true),  // inside
		cand("low", 3, 5, 5, true),   // inside, lowest id of the valid ones
		cand("tie", 3, 6, 5, true),   // same id: stable order keeps listing order
	}

	got := ids(Scan(g, owner, 0.9, 0.2, 10, cs))
	want := []string{"low", "tie", "near", "edge"}

	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}

	if best := lowestSerial(Scan(g, owner, 0.9, 0.2, 10, cs)); best.ID() != "tie" {
		t.Errorf("lowestSerial picked %s, want the later of the tie (tie)", best.ID())
	}
}

func TestScanLineOfSight(t *testing.T) {
	g := d2path.NewCellGrid(-5, -20, 205, 40)
	g.Set(5, 0, d2path.FlagWall)

	owner := d2path.Point{X: 0, Y: 0}
	cs := []Candidate{cand("behind", 1, 8, 0, true), cand("beside", 2, 8, 3, true)}

	got := ids(Scan(g, owner, 8, 0, 6, cs))
	if len(got) != 1 || got[0] != "beside" {
		t.Errorf("wall in the line: got %v, want [beside]", got)
	}

	// a walk-only bit does not block sight (mask 4 only)
	g2 := d2path.NewCellGrid(-5, -20, 205, 40)
	g2.Set(5, 0, d2path.FlagWalk)

	if got := ids(Scan(g2, owner, 8, 0, 6, cs)); len(got) != 2 {
		t.Errorf("walk bit blocked sight: %v", got)
	}
}

// lowest unit id wins, whatever the listing order.
func TestLowestSerialOrder(t *testing.T) {
	a := &scanTarget{fakeTarget{id: "a", alive: true}, 9}
	b := &scanTarget{fakeTarget{id: "b", alive: true}, 4}

	if got := lowestSerial([]Target{a, b}); got.ID() != "b" {
		t.Errorf("picked %s, want b (lowest id)", got.ID())
	}
}
