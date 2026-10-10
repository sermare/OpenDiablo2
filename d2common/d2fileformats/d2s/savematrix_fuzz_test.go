package d2s

import (
	"bytes"
	"testing"
)

// matrixSeeds returns full saves with large item sets (corpse, mercenary, golem) for the fuzz targets.
func matrixSeeds(t testing.TB) ([][]byte, *ItemTables) {
	tb := synthTables(t)

	var seeds [][]byte

	for _, class := range []Class{Sorceress, Necromancer} {
		c := matrixBase(t, tb)
		c.Header.Class = class
		c.Header.Status = StatusExpansion | StatusHardcore
		c.Header.Mercenary = Mercenary{ID: 5, NameID: 1, Type: 2, Experience: 3}

		main, corpse, merc, golem := matrixItems(t, tb)
		// the fuzzing engine is very slow on multi-kilobyte inputs: keep the seeds small, the large
		// layouts are covered by TestMatrixItemsAllClasses
		c.Items, c.Corpse, c.HasCorpse, c.MercItems = main[:15], corpse[:1], true, merc[:1]

		if class == Necromancer {
			c.Golem = &golem
		}

		out, err := Write(c, tb)
		if err != nil {
			t.Fatal(err)
		}

		seeds = append(seeds, out)
	}

	return seeds, tb
}

// TestMatrixFuzzSeeds makes sure the matrix seeds reach the deep parsers.
func TestMatrixFuzzSeeds(t *testing.T) {
	seeds, tb := matrixSeeds(t)
	for i, s := range seeds {
		c, err := Parse(s, tb)
		if err != nil || len(c.Items) < 10 || len(c.MercItems) == 0 {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
}

// FuzzMatrixWriteStable: whatever parses and can be written must be stable: Write(Parse(x)) parses
// and rewrites to the same bytes.
func FuzzMatrixWriteStable(f *testing.F) {
	seeds, tb := matrixSeeds(f)
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, d := range [][]byte{data, fixup(append([]byte(nil), data...))} {
			c, err := Parse(d, tb)
			if err != nil {
				continue
			}

			out, err := Write(c, tb)
			if err != nil {
				continue // a parsed value the writer refuses is acceptable
			}

			back, err := Parse(out, tb)
			if err != nil {
				t.Fatalf("written file does not parse: %v", err)
			}

			again, err := Write(back, tb)
			if err != nil || !bytes.Equal(out, again) {
				t.Fatalf("write is not stable: %v", err)
			}
		}
	})
}
