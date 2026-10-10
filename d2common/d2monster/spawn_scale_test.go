package d2monster

import "testing"

type scaleFake struct {
	base, next, level, steps []int
}

func (f scaleFake) Known(c int) bool      { return c >= 0 && c < len(f.base) }
func (f scaleFake) BaseID(c int) int      { return f.base[c] }
func (f scaleFake) NextInClass(c int) int { return f.next[c] }
func (f scaleFake) Level(c int) int       { return f.level[c] }
func (f scaleFake) ChainSteps(c int) int  { return f.steps[c] }

// chain 0->1->2->3 (levels 1, 10, 20, 30), class 4 stands alone
var scaleTab = scaleFake{
	base:  []int{0, 0, 0, 0, 4},
	next:  []int{1, 2, 3, -1, -1},
	level: []int{1, 10, 20, 30, 5},
	steps: []int{3, 3, 3, 3, 0},
}

func TestScaleSpawnClass(t *testing.T) {
	tests := []struct {
		name   string
		class  int
		mons   []int
		monLvl int
		want   int
	}{
		{"no level monsters", 0, nil, 99, 0},
		{"level lists the chain", 0, []int{4, 2}, 1, 2},
		{"first matching entry wins", 0, []int{3, 1}, 1, 3},
		{"walk stops below the level", 0, []int{4}, 10, 1},
		{"walk goes on while level fits", 0, []int{4}, 19, 2},
		{"walk goes to the end", 0, []int{4}, 40, 3},
		{"first step already too high", 0, []int{4}, 0, 0},
		{"class without chain", 4, []int{1}, 50, 4},
		{"invalid base", 77, []int{1}, 50, 77},
	}

	for _, tc := range tests {
		if got := ScaleSpawnClass(scaleTab, tc.class, tc.mons, tc.monLvl); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}
