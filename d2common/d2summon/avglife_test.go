package d2summon

import "testing"

func TestAvgLife(t *testing.T) {
	tp := loadFixture(t)

	tests := []struct {
		id   string
		diff Difficulty
		want int
		ok   bool
	}{
		{"necroskeleton", Normal, 21, true},
		{"NecroSkeleton", Hell, 42, true},
		{"druidhawk", Normal, 26, true}, // (20+32)/2
		{"claygolem", Nightmare, 175, true},
		{"nosuchmonster", Normal, 0, false},
		{"claygolem", Difficulty(7), 0, false},
	}

	for _, tc := range tests {
		got, ok := tp.AvgLife(tc.id, tc.diff)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s/%d: got %d,%v want %d,%v", tc.id, tc.diff, got, ok, tc.want, tc.ok)
		}
	}
}
