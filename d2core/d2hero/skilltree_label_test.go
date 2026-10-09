package d2hero

import "testing"

func TestTreeLabel(t *testing.T) {
	tests := []struct {
		base, eff int
		text, col string
		shown     bool
	}{
		{0, 0, "", "", false},
		{5, 5, "5", "", true},
		{20, 23, "23", "[blue]", true},
		{0, 3, "3", "[blue]", true},
		{4, 2, "2", "[red]", true},
	}

	for _, tt := range tests {
		text, col, shown := TreeLabel(tt.base, tt.eff)
		if text != tt.text || col != tt.col || shown != tt.shown {
			t.Errorf("TreeLabel(%d,%d) = %q,%q,%v", tt.base, tt.eff, text, col, shown)
		}
	}
}
