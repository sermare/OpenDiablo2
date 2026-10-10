package d2level

import "testing"

// The cave entrances of the Act 3 jungle levels follow the slot rule (Flayer Jungle: Dungeon Fort 86 first, the hole 88 second).
func TestJungleCaveEntranceStyles(t *testing.T) {
	cases := []struct{ level, style, want int }{
		{78, 0, 86}, {78, 1, 88}, {76, 0, 84}, {76, 1, 85},
	}

	for _, c := range cases {
		if to, ok := TileDestination(c.level, c.style); !ok || to != c.want {
			t.Errorf("TileDestination(%d, %d) = %d, %v; want %d", c.level, c.style, to, ok, c.want)
		}
	}
}
