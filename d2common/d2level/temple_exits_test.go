package d2level

import "testing"

// The temples have one exit tile, style 0 (the up stairs): it leads back to the Kurast level they hang off.
func TestTempleUpStairsLeadBack(t *testing.T) {
	for temple, back := range map[int]int{94: 80, 95: 80, 96: 81, 97: 81, 98: 82, 99: 82} {
		if to, ok := TileDestination(temple, 0); !ok || to != back {
			t.Errorf("TileDestination(%d, 0) = %d, %v; want %d", temple, to, ok, back)
		}
	}
}
