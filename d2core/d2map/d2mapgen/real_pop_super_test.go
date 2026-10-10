package d2mapgen

import "testing"

func TestSuperFallbackAllowed(t *testing.T) {
	tests := []struct {
		name               string
		diff               int
		made, stacks, want bool
	}{
		{"normal, not made", 0, false, false, true},
		{"hell, not made", 2, false, false, true},
		{"above hell", 3, false, false, false},
		{"made already", 1, true, false, false},
		{"made already but stacks", 1, true, true, true},
	}

	for _, tt := range tests {
		if got := superFallbackAllowed(tt.diff, tt.made, tt.stacks); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}
