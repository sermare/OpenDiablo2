package d2hero

import "testing"

func TestFreshHero(t *testing.T) {
	stats := func(level, exp int) *HeroStatsState { return &HeroStatsState{Level: level, Experience: exp} }

	tests := []struct {
		name  string
		state HeroState
		want  bool
	}{
		{"new level 1 hero, no containers", HeroState{Stats: stats(1, 0)}, true},
		{"empty containers", HeroState{Stats: stats(1, 0), Containers: &HeroContainers{}}, true},
		{"has played: experience", HeroState{Stats: stats(1, 12), Containers: &HeroContainers{}}, false},
		{"has played: level", HeroState{Stats: stats(2, 0)}, false},
		{"has an item", HeroState{Stats: stats(1, 0), Containers: &HeroContainers{Items: []StoredItem{{Code: "hp1"}}}}, false},
		{"wears an item", HeroState{Stats: stats(1, 0), Containers: &HeroContainers{Equipped: []StoredItem{{Code: "sst"}}}}, false},
		{"has a belt", HeroState{Stats: stats(1, 0), Containers: &HeroContainers{BeltCode: "lbl"}}, false},
		{"no stats", HeroState{}, false},
	}

	for _, tc := range tests {
		if got := freshHero(&tc.state); got != tc.want {
			t.Errorf("%s: freshHero = %v, want %v", tc.name, got, tc.want)
		}
	}
}
