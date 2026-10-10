package d2s

import (
	"fmt"
	"testing"
)

// Fuzz targets seeded only with synthetic saves (built here from nothing; no game data), so they
// run on CI. The targets in robust_test.go additionally use real saves when present locally.

// synthTables is miniTables plus the character-section widths of the sixteen core stats, so that
// complete synthetic saves parse.
func synthTables(t testing.TB) *ItemTables {
	t.Helper()

	stat := miniStatCost

	for id := 1; id < 16; id++ {
		st, _ := DefaultStatStorage(id)
		stat += fmt.Sprintf("charstat%d\t%d\t0\t0\t1\t0\t\t%d\t\n", id, id, st.Bits)
	}

	tb, err := NewItemTables([]byte(stat), []byte(miniArmor), []byte(miniWeapons),
		[]byte(miniMisc), []byte(miniItemTypes))
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func synthSeeds(t testing.TB) ([][]byte, *ItemTables) {
	tb := synthTables(t)

	var skills [numSkills]byte

	skills[3] = 5

	base := buildBody(t, map[int]uint64{StatStrength: 30, StatLevel: 5, StatGold: 100}, skills)

	// buildBody ends after the (empty) item list: add the empty corpse list and the mercenary tag
	base = append(base, 'J', 'M', 0, 0)
	base = append(base, mercTag...)
	base = fixup(base)

	seeds := [][]byte{base, base[:HeaderSize], base[:len(base)/2]}

	c, err := Parse(base, tb)
	if err != nil {
		t.Fatalf("base seed: %v", err)
	}

	it, err := NewItem(Item{Code: "key", ID: 7, Level: 5}, tb)
	if err != nil {
		t.Fatalf("item: %v", err)
	}

	c.Items = append(c.Items, it)

	out, err := Write(c, tb)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	seeds = append(seeds, out)

	return seeds, tb
}

func FuzzParseSynthetic(f *testing.F) {
	seeds, tb := synthSeeds(f)
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, d := range [][]byte{data, fixup(append([]byte(nil), data...))} {
			c, err := Parse(d, tb)
			if err != nil {
				continue
			}

			// whatever parses must survive being written and parsed again
			if out, err := Write(c, tb); err == nil {
				_, _ = Parse(out, tb)
			}
		}
	})
}

func FuzzParseItemListSynthetic(f *testing.F) {
	seeds, tb := synthSeeds(f)
	for _, s := range seeds {
		if c, err := Parse(s, tb); err == nil && c.Body != nil && c.Body.ItemsOffset <= len(s) {
			f.Add(s[c.Body.ItemsOffset:])
		}
	}

	f.Add([]byte("JM\x00\x00"))
	f.Add([]byte("JM\xff\xff"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseItemList(data, tb)
	})
}

// TestSynthSeedsParse makes sure the seeds reach the parsers (a seed that fails the header check
// would leave the fuzz targets exploring only error paths).
func TestSynthSeedsParse(t *testing.T) {
	seeds, tb := synthSeeds(t)
	if len(seeds) < 4 {
		t.Fatalf("only %d seeds: the item seed was not built", len(seeds))
	}

	for _, s := range []int{0, 3} {
		c, err := Parse(seeds[s], tb)
		if err != nil {
			t.Fatalf("seed %d: %v", s, err)
		}

		if s == 3 && len(c.Items) != 1 {
			t.Fatalf("item seed has %d items", len(c.Items))
		}
	}
}

func FuzzParseHeaderBody(f *testing.F) {
	seeds, _ := synthSeeds(f)
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseHeader(data)
		_, _ = ParseBody(data, nil)
	})
}
