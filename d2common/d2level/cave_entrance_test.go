package d2level

import "testing"

func TestCaveEntranceDestination(t *testing.T) {
	tests := []struct {
		level, want int
		ok          bool
	}{
		{2, 8, true},    // Blood Moor -> Den of Evil
		{3, 9, true},    // Cold Plains -> Cave Level 1
		{4, 10, true},   // Stony Field -> Underground Passage 1
		{5, 10, true},   // Dark Wood -> Underground Passage 1
		{6, 11, true},   // Black Marsh -> Hole 1
		{7, 12, true},   // Tamoe Highland -> Pit 1
		{1, 0, false},   // the town has no cave
		{8, 0, false},   // a cave has no wilderness cave entrance
		{17, 0, false},  // Burial Grounds: crypt and mausoleum use their own warp ids
		{999, 0, false}, // unknown
	}

	for _, tc := range tests {
		got, ok := CaveEntranceDestination(tc.level)
		if ok != tc.ok || got != tc.want {
			t.Errorf("CaveEntranceDestination(%d) = %d %v, want %d %v", tc.level, got, ok, tc.want, tc.ok)
		}
	}
}
