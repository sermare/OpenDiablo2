package d2trade

import "testing"

func TestTomeMatches(t *testing.T) {
	tp := AffixesOf(nil, []string{"Town Portal"}, "")
	id := AffixesOf(nil, []string{"Identify"}, "")

	tests := []struct {
		name         string
		scroll, tome AffixSet
		want         bool
	}{
		{"same suffix", tp, AffixesOf(nil, []string{"Town Portal"}, ""), true},
		{"other book", tp, id, false},
		{"no affixes on both", AffixSet{}, AffixSet{}, true},
		{"prefix differs", tp, AffixesOf([]string{"Cruel"}, []string{"Town Portal"}, ""), false},
		{"second suffix differs", tp, AffixesOf(nil, []string{"Town Portal", "of X"}, ""), false},
		{"auto prefix differs", tp, AffixesOf(nil, []string{"Town Portal"}, "auto"), false},
		{"extra beyond three ignored", AffixesOf(nil, []string{"a", "b", "c", "d"}, ""), AffixesOf(nil, []string{"a", "b", "c", "e"}, ""), true},
	}

	for _, tt := range tests {
		if got := TomeMatches(tt.scroll, tt.tome); got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestTomeDropMatches(t *testing.T) {
	a := AffixesOf([]string{"x"}, []string{"Town Portal", "y"}, "")
	b := AffixesOf(nil, []string{"Town Portal"}, "")
	c := AffixesOf(nil, []string{"Identify"}, "")

	if !TomeDropMatches(a, b) || TomeDropMatches(a, c) {
		t.Error("the cursor drop compares the first suffix only")
	}
}

func TestTomeRoom(t *testing.T) {
	tests := []struct{ qty, max, want int }{{0, 20, 20}, {19, 20, 1}, {20, 20, 0}, {25, 20, 0}, {-3, 20, 20}}

	for _, tt := range tests {
		if got := TomeRoom(tt.qty, tt.max); got != tt.want {
			t.Errorf("TomeRoom(%d,%d) = %d, want %d", tt.qty, tt.max, got, tt.want)
		}
	}
}
