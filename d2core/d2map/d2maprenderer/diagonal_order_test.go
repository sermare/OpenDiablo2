package d2maprenderer

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
)

func TestDiagonalOrder(t *testing.T) {
	// rows 2..4 with ragged ranges
	rows := &TileRows{StartY: 2, From: []int{3, 0, 1}, To: []int{6, 2, 1}}

	got := rows.DiagonalOrder(2, 5, nil)

	want := 0

	for y := 2; y < 5; y++ {
		from, to := rows.Row(y)
		want += to - from
	}

	if len(got) != want {
		t.Fatalf("got %d tiles, want %d", len(got), want)
	}

	seen := map[TileRef]bool{}

	for i, r := range got {
		if seen[r] {
			t.Errorf("tile %v listed twice", r)
		}

		seen[r] = true

		if i > 0 {
			p := got[i-1]
			dp, dr := p.X+p.Y, r.X+r.Y

			if dp > dr || (dp == dr && p.X >= r.X) {
				t.Errorf("order broken between %v and %v", p, r)
			}
		}
	}

	again := rows.DiagonalOrder(2, 5, got)
	if len(again) != len(got) || &again[0] != &got[0] {
		t.Errorf("buffer not reused")
	}

	if n := len((&TileRows{}).DiagonalOrder(0, 0, nil)); n != 0 {
		t.Errorf("empty rows gave %d tiles", n)
	}
}

func TestIsBlankFloor(t *testing.T) {
	tests := []struct {
		name string
		tile func() d2ds1.Tile
		want bool
	}{
		{"filler", blankTile(1, 30, 0, 0), true},
		{"other sequence", blankTile(1, 30, 1, 0), false},
		{"other style", blankTile(1, 6, 0, 0), false},
		{"empty cell", blankTile(0, 30, 0, 0), false},
		{"hidden", blankTile(1, 30, 0, 1), false},
	}

	for _, tc := range tests {
		tile := tc.tile()
		if got := isBlankFloor(&tile); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

// blankTile builds a floor cell (promoted fields cannot be set in a composite literal).
func blankTile(prop1, style, seq, hidden int) func() d2ds1.Tile {
	return func() d2ds1.Tile {
		var t d2ds1.Tile
		t.Prop1, t.Style, t.Sequence, t.HiddenBytes = byte(prop1), byte(style), byte(seq), byte(hidden)

		return t
	}
}
