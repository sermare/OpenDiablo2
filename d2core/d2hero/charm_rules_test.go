package d2hero

import "testing"

func TestCharmActive(t *testing.T) {
	tests := []struct {
		name       string
		identified bool
		req, level int
		want       bool
	}{
		{"identified, no requirement", true, 0, 1, true},
		{"unidentified", false, 0, 90, false},
		{"level reached", true, 70, 70, true},
		{"level short", true, 70, 69, false},
	}

	for _, tc := range tests {
		if got := CharmActive(tc.identified, tc.req, tc.level); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
