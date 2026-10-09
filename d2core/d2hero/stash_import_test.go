package d2hero

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestMergeStash(t *testing.T) {
	known := func(code string) bool { return code != "zzz" }
	size := func(code string) (int, int) {
		if code == "big" {
			return 2, 4
		}

		return 1, 1
	}

	item := func(code string, x, y uint8) d2s.Item {
		return d2s.Item{Code: code + " ", X: x, Y: y, Location: d2s.LocationStored, Page: 5}
	}

	c := &HeroContainers{Items: []StoredItem{{Code: "aaa", Page: PageStash, X: 0, Y: 0}}}

	added, skipped := MergeStash(c, []d2s.Item{
		item("bbb", 0, 0), // taken: moves to the first free cell (1,0)
		item("bbb", 5, 7), // free: stays
		item("zzz", 2, 2), // unknown record
		item("big", 5, 0), // does not fit at x=5: relocated
		{Code: "ccc ", Ear: true},
	}, known, size)

	if added != 3 || len(skipped) != 2 {
		t.Fatalf("added=%d skipped=%v", added, skipped)
	}

	got := c.Page(PageStash)
	if len(got) != 4 || got[1].X != 1 || got[1].Y != 0 || got[2].X != 5 || got[2].Y != 7 {
		t.Fatalf("placement %+v", got)
	}

	// no two items overlap
	var used [stashRows][stashCols]int

	for _, s := range got {
		w, h := size(s.Code)
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				used[s.Y+dy][s.X+dx]++
				if used[s.Y+dy][s.X+dx] > 1 {
					t.Fatalf("overlap at %d,%d", s.X+dx, s.Y+dy)
				}
			}
		}
	}

	// a full stash skips instead of failing
	full := &HeroContainers{}
	many := make([]d2s.Item, stashCols*stashRows+1)

	for i := range many {
		many[i] = item("bbb", 0, 0)
	}

	if added, skipped := MergeStash(full, many, known, size); added != stashCols*stashRows || len(skipped) != 1 {
		t.Fatalf("full: added=%d skipped=%v", added, skipped)
	}
}
