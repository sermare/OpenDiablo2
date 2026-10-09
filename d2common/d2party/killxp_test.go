package d2party

import "testing"

func TestSplitKillXP(t *testing.T) {
	for _, tc := range []struct {
		name   string
		xp     int
		levels []int
		want   []int
	}{
		{"solo", 1000, []int{30}, []int{1000}},
		{"two equal", 1000, []int{30, 30}, []int{673, 673}},
		{"two unequal", 1000, []int{10, 30}, []int{336, 1010}},
		{"empty", 1000, nil, []int{}},
	} {
		got := SplitKillXP(tc.xp, tc.levels)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: len %d", tc.name, len(got))
		}

		for i := range got {
			if d := got[i] - tc.want[i]; d < -1 || d > 1 { // float rounding at the unit
				t.Errorf("%s[%d]: got %d want %d", tc.name, i, got[i], tc.want[i])
			}
		}
	}
}
