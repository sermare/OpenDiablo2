package drlgworld

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
)

func levels(t *testing.T) *d2drlg.Tables {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	b, err := os.ReadFile(filepath.Join(root, "drlg", "patch_d2", "Levels.txt"))
	if err != nil {
		t.Skip(err)
	}

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: b})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func touches(a, b Rect) bool {
	xo := a.X < b.X+b.W && b.X < a.X+a.W
	yo := a.Y < b.Y+b.H && b.Y < a.Y+a.H

	return (xo && (a.Y+a.H == b.Y || b.Y+b.H == a.Y)) || (yo && (a.X+a.W == b.X || b.X+b.W == a.X))
}

func TestLayoutInvariants(t *testing.T) {
	lv := levels(t)
	townFiles := map[int]int{}

	for s := uint32(1); s <= 3000; s++ {
		seed := s * 2654435761

		l, err := Generate(lv, seed, d2drlg.Normal)
		if err != nil {
			t.Fatalf("seed %#x: %v", seed, err)
		}

		townFiles[l.TownFile]++

		// Anchors come straight from Levels.txt offsets (independent facts).
		for id, want := range map[int][2]int{4: {1000, 1000}, 0x27: {5000, 1148}, 0x1a: {3000, 1000}} {
			p := l.Levels[id]
			if p.Rect.X != want[0] || p.Rect.Y != want[1] {
				t.Fatalf("seed %#x: level %d at %v", seed, id, p.Rect)
			}
		}

		// Blood Moor is 56x96 or 96x56 depending on the parity of its direction.
		bm := l.Levels[2]
		if (bm.Dir&1 == 1) != (bm.Rect.W == 96 && bm.Rect.H == 56) || (bm.Dir&1 == 0) != (bm.Rect.W == 56 && bm.Rect.H == 96) {
			t.Fatalf("seed %#x: blood moor dir %d size %v", seed, bm.Dir, bm.Rect)
		}

		// Tamoe Highland sits exactly below the Monastery.
		mon, tam := l.Levels[0x1a].Rect, l.Levels[7].Rect
		if tam.X != mon.X || tam.Y != mon.Y+mon.H {
			t.Fatalf("seed %#x: tamoe %v monastery %v", seed, tam, mon)
		}

		// Pinwheel neighbours touch their reference (cold plains / stony field).
		if cp := l.Levels[3].Rect; !touches(cp, l.Levels[4].Rect) && !overlapsEdgeSlide(cp, l.Levels[4].Rect) {
			t.Fatalf("seed %#x: cold plains %v not next to stony field %v", seed, cp, l.Levels[4].Rect)
		}

		// No two levels of cluster 1 overlap except the permitted reference pairs.
		c1 := []int{4, 3, 2, 1, 0x11}
		for i, a := range c1 {
			for _, b := range c1[:i] {
				if a == 1 && b == 2 { // town vs its ref Blood Moor: allowed to overlap exits
					continue
				}

				if (a == 3 && b == 4) || (a == 2 && b == 3) || (a == 0x11 && b == 3) {
					continue
				}

				if overlap(l.Levels[a].Rect, l.Levels[b].Rect) {
					t.Fatalf("seed %#x: levels %d and %d overlap", seed, a, b)
				}
			}
		}

		// Town direction satisfies the compat table.
		town := l.Levels[1]
		if compat[town.Dir+4*(town.Flip+2*(bm.Dir+4*bm.Flip))] == 0 {
			t.Fatalf("seed %#x: town placement violates compat table", seed)
		}

		// Determinism.
		l2, _ := Generate(lv, seed, d2drlg.Normal)
		for id, p := range l.Levels {
			if *l2.Levels[id] != *p {
				t.Fatalf("seed %#x not deterministic", seed)
			}
		}
	}

	t.Logf("town file index distribution over 3000 seeds: %v", townFiles)

	if len(townFiles) < 2 {
		t.Errorf("town variant never varies: %v", townFiles)
	}
}

// overlapsEdgeSlide: pinwheel placements slide 16 tiles into the neighbour.
func overlapsEdgeSlide(a, b Rect) bool {
	return overlap(Rect{a.X, a.Y, a.W + 16, a.H + 16}, b) || overlap(Rect{a.X - 16, a.Y - 16, a.W + 32, a.H + 32}, b)
}

func TestSeedSensitivity(t *testing.T) {
	lv := levels(t)
	seen := map[Rect]bool{}

	for s := uint32(1); s <= 200; s++ {
		l, err := Generate(lv, s, d2drlg.Hell)
		if err != nil {
			t.Fatal(err)
		}

		seen[l.Levels[1].Rect] = true
	}

	if len(seen) < 8 {
		t.Errorf("only %d distinct town rects over 200 seeds", len(seen))
	}
}
