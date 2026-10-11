package d2monsters

import "testing"

func TestRestoresMonster(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		questSet bool
		class    int
		want     bool
	}{
		{"level 108 quest set, Diablo", 108, true, 0xf3, true},
		{"level 108 quest set, other", 108, true, 0x10, false},
		{"level 108 quest clear, other", 108, false, 0x10, true},
		{"level 107 quest set, other", 107, true, 0x10, true},
		{"level 1 quest set, Diablo", 1, true, 0xf3, true},
	}

	for _, tc := range tests {
		if got := RestoresMonster(tc.level, tc.questSet, tc.class); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
