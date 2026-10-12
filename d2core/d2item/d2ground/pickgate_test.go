package d2ground

import "testing"

func TestCanPickUp(t *testing.T) {
	tests := []struct {
		name string
		held []Held
		c    Candidate
		want bool
	}{
		{"plain item", []Held{{Code: "hp1"}}, Candidate{Code: "hp1"}, true},
		{"single copy unique held", []Held{{UniqueCode: "Annihilus"}}, Candidate{UniqueCode: "Annihilus", SingleCopy: true}, false},
		{"single copy unique worn", []Held{{UniqueCode: "Annihilus", Worn: true}}, Candidate{UniqueCode: "Annihilus", SingleCopy: true}, true},
		{"unique without carry1", []Held{{UniqueCode: "x"}}, Candidate{UniqueCode: "x"}, true},
		{"other unique", []Held{{UniqueCode: "a"}}, Candidate{UniqueCode: "b", SingleCopy: true}, true},
		{"same quest item", []Held{{Code: "bks", Quest: 3}}, Candidate{Code: "bks", Quest: 3}, false},
		{"paired quest code", []Held{{Code: "bkd", Quest: 3}}, Candidate{Code: "bks", Quest: 3}, false},
		{"pair, other quest byte", []Held{{Code: "bkd", Quest: 4}}, Candidate{Code: "bks", Quest: 3}, true},
		{"qf2 vs qhr", []Held{{Code: "qhr", Quest: 9}}, Candidate{Code: "qf2", Quest: 9}, false},
		{"nothing held", nil, Candidate{Code: "hst", Quest: 2}, true},
	}
	for _, tc := range tests {
		if got := CanPickUp(tc.held, tc.c); got != tc.want {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
