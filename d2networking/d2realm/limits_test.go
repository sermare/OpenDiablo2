package d2realm

import (
	"testing"
	"time"
)

func TestGameFull(t *testing.T) {
	tests := []struct {
		members int
		max     byte
		want    bool
	}{
		{0, 8, false}, {7, 8, false}, {8, 8, true},
		{3, 4, false}, {4, 4, true},
		{8, 16, true}, // the exe's cap of 8 beats a larger setting
		{7, 16, false},
		{8, 0, true}, {7, 0, false},
	}

	for _, tc := range tests {
		if got := GameFull(tc.members, tc.max); got != tc.want {
			t.Errorf("GameFull(%d,%d) = %v, want %v", tc.members, tc.max, got, tc.want)
		}
	}
}

func TestJoinDeadline(t *testing.T) {
	now := time.Unix(1000, 0)
	if got := JoinDeadline(now).Sub(now); got != 180*time.Second {
		t.Fatalf("deadline = %v, want 180s", got)
	}
}
