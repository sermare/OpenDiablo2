package d2equip

import "testing"

func TestRollWear(t *testing.T) {
	it := &Item{MaxDurability: 30, Durability: 10}

	for _, tc := range []struct {
		name                 string
		it                   *Item
		percent, chance, amt int
		dur                  int
		broke, lost          bool
	}{
		{"no roll", it, 50, 30, 1, 10, false, false},
		{"loses one", it, 29, 30, 1, 9, false, true},
		{"loses three", it, 0, 30, 3, 7, false, true},
		{"to zero breaks", &Item{MaxDurability: 30, Durability: 1}, 0, 30, 1, 0, true, true},
		{"overshoot breaks", &Item{MaxDurability: 30, Durability: 2}, 0, 30, 5, 0, true, true},
		{"indestructible", &Item{MaxDurability: 30, Durability: 5, Indestructible: true}, 0, 100, 1, 5, false, false},
		{"no durability", &Item{}, 0, 100, 1, 0, false, false},
	} {
		dur, broke, lost := RollWear(tc.it, tc.percent, tc.chance, tc.amt)
		if dur != tc.dur || broke != tc.broke || lost != tc.lost {
			t.Errorf("%s: got %d,%v,%v want %d,%v,%v", tc.name, dur, broke, lost, tc.dur, tc.broke, tc.lost)
		}
	}
}
