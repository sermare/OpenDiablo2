package d2mapengine

import (
	"math/rand"
	"reflect"
	"testing"
)

// The dense search must give exactly the route of the map based one, for any world.
func TestFindPathDenseMatchesSparse(t *testing.T) {
	rng := rand.New(rand.NewSource(1))

	for n := 0; n < 300; n++ {
		size := 20 + rng.Intn(100)
		density := rng.Float64() * 0.4
		cells := make(map[pt]bool)

		for i := 0; i < int(float64(size*size)*density); i++ {
			cells[pt{rng.Intn(size), rng.Intn(size)}] = true
		}

		blocked := func(x, y int) bool { return x < 0 || y < 0 || x >= size || y >= size || cells[pt{x, y}] }
		start, goal := pt{rng.Intn(size), rng.Intn(size)}, pt{rng.Intn(size), rng.Intn(size)}
		delete(cells, start)

		wantPath, wantReached := findPathSparse(blocked, start, goal)
		gotPath, gotReached := findPathDense(blocked, start, goal)

		if gotReached != wantReached || !reflect.DeepEqual(gotPath, wantPath) {
			t.Fatalf("world %d size=%d %v->%v: dense=%v/%v sparse=%v/%v", n, size, start, goal, gotPath, gotReached, wantPath, wantReached)
		}
	}
}

func TestFindPathDenseMatchesSparseCamp(t *testing.T) {
	for _, c := range [][2]pt{{{20, 20}, {150, 150}}, {{150, 100}, {150, 200}}, {{5, 5}, {290, 290}}, {{150, 150}, {10, 10}}} {
		wp, wr := findPathSparse(campWorld, c[0], c[1])
		gp, gr := findPathDense(campWorld, c[0], c[1])

		if gr != wr || !reflect.DeepEqual(gp, wp) {
			t.Errorf("%v: dense=%v/%v sparse=%v/%v", c, gp, gr, wp, wr)
		}
	}
}
