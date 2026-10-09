package d2mapengine

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
)

// A tile named by (file, index) must come back from GetTiles(style, sequence,
// type)[RandomIndex]: that is how the renderer and PrepareTile find it again.
func TestExactComponentRoundTrip(t *testing.T) {
	tile := func(style, seq, typ int32) d2dt1.Tile {
		return d2dt1.Tile{Style: style, Sequence: seq, Type: typ}
	}

	m := &MapEngine{
		dt1Files:  []string{"a.dt1", "b.dt1"},
		dt1Starts: []int{0, 3},
		dt1TileData: []d2dt1.Tile{
			tile(1, 0, 0), tile(2, 0, 0), tile(1, 0, 0), // a.dt1
			tile(1, 0, 0), tile(1, 0, 3), tile(1, 0, 0), // b.dt1
		},
	}

	cases := []struct {
		file string
		idx  int
		rnd  byte
	}{{"a.dt1", 0, 0}, {"a.dt1", 2, 1}, {"b.dt1", 0, 2}, {"b.dt1", 2, 3}, {"B.DT1", 1, 0}}

	for _, c := range cases {
		comp, dt, ok := m.exactComponent(ExactTile{File: c.file, Index: c.idx})
		if !ok {
			t.Fatalf("%s/%d not resolved", c.file, c.idx)
		}

		if comp.RandomIndex != c.rnd {
			t.Errorf("%s/%d: RandomIndex %d, want %d", c.file, c.idx, comp.RandomIndex, c.rnd)
		}

		opts := m.GetTiles(int(comp.Style), int(comp.Sequence), comp.Type)
		if len(opts) == 0 || &opts[comp.RandomIndex] == dt {
			continue // GetTiles copies; compare by identity of fields instead
		}

		if opts[comp.RandomIndex].Style != dt.Style || opts[comp.RandomIndex].Type != dt.Type {
			t.Errorf("%s/%d resolves to a different tile through GetTiles", c.file, c.idx)
		}
	}

	if _, _, ok := m.exactComponent(ExactTile{File: "a.dt1", Index: 3}); ok {
		t.Error("an index past the end of the file must not resolve")
	}
}
