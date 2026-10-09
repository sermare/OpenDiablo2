package d2path

import "testing"

// snake is a corridor level whose only way to G winds back and forth: far
// longer than what the original short search (about 18 subtiles) can handle.
const snake = `
S.............................
#############################.
..............................
.#############################
.............................G
`

func TestLongAStar(t *testing.T) {
	g, s, goal := gridFrom(snake)

	t.Run("the original search does not find the way", func(t *testing.T) {
		r, ok := FindPath(g, MaskPlayer, s, goal)
		if ok && !r.Partial {
			t.Fatalf("expected a partial route or none, got %+v", r)
		}
	})

	t.Run("LongAStar reaches the goal along free segments", func(t *testing.T) {
		r, ok := LongAStar(g, MaskPlayer, s, goal, 0)
		if !ok || r.Partial {
			t.Fatalf("ok=%v partial=%v", ok, r.Partial)
		}

		if len(r.Nodes) == 0 || r.Nodes[len(r.Nodes)-1] != goal {
			t.Fatalf("route does not end at the goal: %v", r.Nodes)
		}

		prev := s
		for _, n := range r.Nodes {
			if clear, last := TraceLine(g, MaskPlayer, prev, n); !clear {
				t.Fatalf("segment %v->%v blocked at %v", prev, n, last)
			}

			prev = n
		}
	})

	t.Run("same cell is an empty route", func(t *testing.T) {
		r, ok := LongAStar(g, MaskPlayer, s, s, 0)
		if !ok || len(r.Nodes) != 0 {
			t.Fatalf("ok=%v nodes=%v", ok, r.Nodes)
		}
	})
}

func TestLongAStarUnreachable(t *testing.T) {
	g, s, goal := gridFrom(`
S..#.G
...#..
...#..
`)

	r, ok := LongAStar(g, MaskPlayer, s, goal, 0)
	if !ok || !r.Partial {
		t.Fatalf("want a partial route to the nearest reachable cell, got ok=%v %+v", ok, r)
	}
}

func TestLongAStarNoCornerCutting(t *testing.T) {
	// the only diagonal gap between S and G is a wall corner
	g, s, goal := gridFrom(`
S#.
#..
..G
`)

	r, ok := LongAStar(g, MaskPlayer, s, goal, 0)
	if ok && !r.Partial {
		t.Fatalf("squeezed through a wall corner: %+v", r)
	}
}
