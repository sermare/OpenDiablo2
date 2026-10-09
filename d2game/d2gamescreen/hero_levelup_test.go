package d2gamescreen

import "testing"

func TestLevelsGained(t *testing.T) {
	// experience needed to leave level 1, 2, 3 (Experience.txt: 500, 1500, 3750)
	bp := func(l int) int { return map[int]int{1: 500, 2: 1500, 3: 3750, 4: 7875}[l] }

	tests := []struct {
		name            string
		exp, level, max int
		want            int
	}{
		{"below the first breakpoint", 330, 1, 99, 0},
		{"exactly the breakpoint", 500, 1, 99, 1},
		{"606 experience at level 1", 606, 1, 99, 1},
		{"a big kill skips levels", 4000, 1, 99, 3},
		{"already there", 1000, 2, 99, 0},
		{"capped at the maximum level", 99999, 1, 3, 2},
		{"no breakpoint known", 99999, 9, 99, 0},
		{"level 1 with no experience", 0, 1, 99, 0},
	}

	for _, tc := range tests {
		if got := levelsGained(tc.exp, tc.level, tc.max, bp); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}
