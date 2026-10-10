package d2level

import "testing"

// Durance of Hate (tile styles observed in the generated levels): the "WarpU"/"Warp" rooms (styles 0 and 1)
// lead up, the "WarpD" room (style 3) leads down. Level 100 used to resolve only style 4 as "down", so the
// 9g-act3-durance scenario failed with "level 101 was never built".
func TestDuranceTileDestinations(t *testing.T) {
	for _, c := range []struct {
		level, style, want int
		ok                 bool
	}{
		{100, 3, 101, true}, {100, 4, 101, true}, {100, 1, 83, true}, {100, 0, 83, true},
		{101, 3, 102, true}, {101, 0, 100, true}, {101, 1, 100, true},
		{102, 0, 101, true}, {102, 1, 101, true}, {102, 3, 0, false},
	} {
		got, ok := TileDestination(c.level, c.style)
		if ok != c.ok || got != c.want {
			t.Errorf("TileDestination(%d, %d) = %d, %v; want %d, %v", c.level, c.style, got, ok, c.want, c.ok)
		}
	}
}
