package d2hero

import "testing"

func TestDeathGoldPenalty(t *testing.T) {
	tests := []struct {
		name                  string
		level, carried, stash int
		pk                    bool
		want                  GoldPenalty
	}{
		{"monster level 10", 10, 1000, 0, false, GoldPenalty{Lost: 100, Dropped: 900}},
		{"level cap 20", 90, 1000, 0, false, GoldPenalty{Lost: 200, Dropped: 800}},
		{"stash covers shortfall", 20, 100, 900, false, GoldPenalty{Lost: 200, NewStash: 800}},
		{"no gold", 30, 0, 0, false, GoldPenalty{}},
		{"pk keeps rest", 10, 1000, 0, true, GoldPenalty{Lost: 100, NewCarried: 900, Dropped: 100}},
		{"stash counts in the loss", 10, 1000, 1000, false, GoldPenalty{Lost: 200, NewStash: 1000, Dropped: 800}},
		{"pk shortfall from stash", 20, 100, 900, true, GoldPenalty{Lost: 200, NewStash: 800, Dropped: 200}},
		{"pk stash counts", 10, 1000, 1000, true, GoldPenalty{Lost: 200, NewCarried: 800, NewStash: 1000, Dropped: 200}},
		{"negative inputs", 10, -5, -5, false, GoldPenalty{}},
	}

	for _, tc := range tests {
		if got := DeathGoldPenalty(tc.level, tc.carried, tc.stash, tc.pk); got != tc.want {
			t.Errorf("%s: got %+v want %+v", tc.name, got, tc.want)
		}
	}
}
