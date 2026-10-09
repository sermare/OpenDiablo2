package d2path

import (
	"math/rand"
	"reflect"
	"testing"
)

func randomGrid(rnd *rand.Rand, w, h int, density float64) *CellGrid {
	g := NewCellGrid(-3, 2, w, h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if rnd.Float64() < density {
				g.Set(g.X0+x, g.Y0+y, FlagWalk)
			}
		}
	}

	return g
}

// TestLongAStarMatchesReference compares the array based search with the original map based one on
// random grids: routes, partial flags and failures must be identical.
func TestLongAStarMatchesReference(t *testing.T) {
	rnd := rand.New(rand.NewSource(42))

	for i := 0; i < 400; i++ {
		w, h := 5+rnd.Intn(90), 5+rnd.Intn(90)
		g := randomGrid(rnd, w, h, []float64{0, 0.15, 0.3, 0.42}[i%4])

		from := Point{g.X0 + rnd.Intn(w), g.Y0 + rnd.Intn(h)}
		to := Point{g.X0 + rnd.Intn(w), g.Y0 + rnd.Intn(h)}

		if i%7 == 0 { // goals outside the grid and negative coordinates
			to = Point{g.X0 - 1 - rnd.Intn(4), g.Y0 + rnd.Intn(h)}
		}

		budget := []int{0, 0, 50, 700}[i%4]

		wantR, wantOK := longAStarRef(g, MaskPlayer, from, to, budget)
		gotR, gotOK := LongAStar(g, MaskPlayer, from, to, budget)

		if wantOK != gotOK || wantR.Partial != gotR.Partial || !reflect.DeepEqual(wantR.Nodes, gotR.Nodes) {
			t.Fatalf("case %d (%dx%d %v->%v budget %d): got ok=%v %+v, want ok=%v %+v",
				i, w, h, from, to, budget, gotOK, gotR, wantOK, wantR)
		}
	}
}

func benchGrid() (*CellGrid, Point, Point) {
	rnd := rand.New(rand.NewSource(3))
	g := randomGrid(rnd, 400, 400, 0.25)
	// the goal sits in a pocket the search cannot enter: the whole reachable area is explored
	to := Point{g.X0 + 200, g.Y0 + 200}

	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx != 0 || dy != 0 {
				g.Set(to.X+dx, to.Y+dy, FlagWalk)
			}
		}
	}

	return g, Point{g.X0 + 2, g.Y0 + 2}, to
}

func BenchmarkLongAStar(b *testing.B) {
	g, from, to := benchGrid()

	b.Run("array", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			LongAStar(g, MaskPlayer, from, to, 0)
		}
	})
	b.Run("reference", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			longAStarRef(g, MaskPlayer, from, to, 0)
		}
	})
}
