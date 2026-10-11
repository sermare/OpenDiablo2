package d2missile

import (
	"math"
	"testing"
)

func TestSpiralNodes(t *testing.T) {
	n := SpiralNodes(50.5, 50.5)
	if len(n) != 77 {
		t.Fatalf("%d nodes, want 77", len(n))
	}

	// every node is in a different subtile than the one before it, and the radius only grows
	prevR := 0.0

	for i, p := range n {
		if i > 0 && math.Floor(p[0]) == math.Floor(n[i-1][0]) && math.Floor(p[1]) == math.Floor(n[i-1][1]) {
			t.Errorf("node %d repeats the subtile", i)
		}

		r := math.Hypot(p[0]-50.5, p[1]-50.5)
		if r < prevR-1e-9 {
			t.Errorf("node %d radius %v shrank from %v", i, r, prevR)
		}

		prevR = r
	}

	// the spiral starts towards +x
	if n[0][0] <= 50.5 {
		t.Errorf("first node %v does not lead towards +x", n[0])
	}
}

func TestSpiralMissileCurves(t *testing.T) {
	w := newWorld()
	s := NewSim(w, nil)
	sp := fireBolt()
	sp.Range = 200

	m, err := s.Create(CreateParams{Spec: sp, Level: 1, X: 50, Y: 50, DestX: 80, DestY: 50, Owner: Owner{ID: "hero", IsPlayer: true}})
	if err != nil {
		t.Fatal(err)
	}

	m.Spiral = true
	run(s, w, 60)

	// a straight bolt would be 56 subtiles away on y == 50; the spiral stays within its radius and leaves the line
	if math.Hypot(m.X-50, m.Y-50) > 17 || m.Y == 50 {
		t.Errorf("spiral position (%v,%v)", m.X, m.Y)
	}
}
