package d2ground

import (
	"reflect"
	"testing"
)

func TestDropCells(t *testing.T) {
	none := func(int, int) bool { return false }

	tests := []struct {
		name    string
		n, rad  int
		blocked func(x, y int) bool
		want    []Cell
	}{
		{"centre first", 1, 3, none, []Cell{{5, 5}}},
		{"ring order", 5, 3, none, []Cell{{5, 5}, {5, 4}, {4, 5}, {6, 5}, {5, 6}}},
		{"centre blocked", 2, 3, func(x, y int) bool { return x == 5 && y == 5 }, []Cell{{5, 4}, {4, 5}}},
		{"out of radius", 5, 0, none, []Cell{{5, 5}}},
		{"all blocked", 3, 4, func(int, int) bool { return true }, nil},
	}

	for _, tc := range tests {
		got := DropCells(5, 5, tc.n, tc.rad, tc.blocked)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestChestTreasureClass(t *testing.T) {
	levels := map[string]int{
		"Act 1 Chest A": 0, "Act 1 Chest B": 5, "Act 1 Chest C": 9,
		"Act 1 (N) Chest A": 38, "Act 2 Chest A": 12,
	}
	lookup := func(n string) (int, bool) { l, ok := levels[n]; return l, ok }

	tests := []struct {
		act   int
		diff  Difficulty
		level int
		want  string
	}{
		{1, Normal, 1, "Act 1 Chest A"},
		{1, Normal, 5, "Act 1 Chest B"},
		{1, Normal, 8, "Act 1 Chest B"},
		{1, Normal, 30, "Act 1 Chest C"},
		{1, Nightmare, 40, "Act 1 (N) Chest A"},
		{2, Normal, 1, "Act 2 Chest A"},
		{3, Normal, 1, ""},
		{0, Normal, 1, "Act 1 Chest A"},
	}

	for _, tc := range tests {
		if got := ChestTreasureClass(tc.act, tc.diff, tc.level, lookup); got != tc.want {
			t.Errorf("act %d diff %d lvl %d: got %q want %q", tc.act, tc.diff, tc.level, got, tc.want)
		}
	}
}

func TestGoldAmount(t *testing.T) {
	tests := []struct{ base, mul, gf, want int }{
		{100, 0, 0, 100},
		{100, 255, 0, 100},
		{100, 1280, 0, 500},
		{100, 0, 50, 150},
		{0, 0, 0, 1},
		{1, 10, 0, 1},
	}

	for _, tc := range tests {
		if got := GoldAmount(tc.base, tc.mul, tc.gf); got != tc.want {
			t.Errorf("GoldAmount(%d,%d,%d)=%d want %d", tc.base, tc.mul, tc.gf, got, tc.want)
		}
	}
}
