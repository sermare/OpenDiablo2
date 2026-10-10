package d2combat

import "testing"

func TestReduceComponentFromResistInput(t *testing.T) {
	player := ResistInput{HasMaxResist: true}
	tests := []struct {
		name         string
		dmg, flat    int
		in           ResistInput
		unresistable bool
		want         int
	}{
		{"zero damage", 0, 5, ResistInput{Resist: 50}, false, 0},
		{"plain", 100, 0, ResistInput{Resist: 25, HasMaxResist: true}, false, 75},
		{"flat before percent", 100, 20, ResistInput{Resist: 50, HasMaxResist: true}, false, 40},
		{"flat larger than damage stays negative", 10, 15, player, false, -5},
		{"cap 75", 100, 0, ResistInput{Resist: 200, HasMaxResist: true}, false, 25},
		{"physical cap 50", 100, 0, ResistInput{Resist: 90, IsPhysical: true, NoDifficultyPenalty: true}, false, 50},
		{"penalty then cap", 100, 0, ResistInput{Resist: 100, HasMaxResist: true, DifficultyPenalty: -40}, false, 40},
		{"negative resist raises damage", 100, 0, ResistInput{Resist: -50}, false, 150},
		// VERIFIED 0x579c90 (hit.go): the ignore flag skips the flat reduction and a positive resist, it does not zero the damage
		{"unresistable: positive resist is skipped", 100, 10, ResistInput{Resist: 30, HasMaxResist: true}, true, 100},
		{"unresistable: no flat, negative resist stays", 100, 10, ResistInput{Resist: -50}, true, 150},
		{"unresistable with no resist keeps the damage", 100, 10, player, true, 100},
		{"ignore context: immunity above the cap is full", 100, 0, ResistInput{Resist: 100, Ignore: true}, false, 0},
		{"ignore context: 100 resist is full immunity", 100, 0, ResistInput{Resist: 100, Ignore: true}, true, 100},
	}

	for _, tc := range tests {
		got, _ := ReduceComponent(tc.dmg, tc.flat, EffectiveResist(tc.in), tc.unresistable, false, 0, 0)
		if got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}
