package d2gamescreen

import "testing"

func TestKindLinesFor(t *testing.T) {
	tests := []struct {
		name              string
		expansion, ladder bool
		wantLines         int
	}{
		{"classic softcore", false, false, 0},
		{"expansion", true, false, 1},
		{"classic ladder", false, true, 1},
		{"expansion ladder", true, true, 2},
	}

	for _, tc := range tests {
		if got := kindLinesFor(tc.expansion, tc.ladder, "EXP", "LAD"); len(got) != tc.wantLines {
			t.Errorf("%s: %d lines %q, want %d", tc.name, len(got), got, tc.wantLines)
		}
	}
}

func TestKindLines(t *testing.T) {
	joined := joinKindLines([]string{"Hardcore", " ", "Expansion Character"})

	a, b := splitKindLines(joined)
	if a == "" || b == "" || a == b {
		t.Fatalf("two lines expected, got %q and %q", a, b)
	}

	one, two := splitKindLines(joinKindLines([]string{"Expansion Character"}))
	if one == "" || two != "" {
		t.Fatalf("one line expected, got %q and %q", one, two)
	}

	if x, y := splitKindLines(joinKindLines(nil)); x != "" || y != "" {
		t.Fatalf("classic softcore has no kind line, got %q %q", x, y)
	}
}
