package d2skill

import "testing"

func TestEffectiveLevel(t *testing.T) {
	fb := &Skill{ID: 36, Name: "Fire Bolt", CharClass: "sor"}
	b := ItemSkillBonus{All: 1, Class: 2, Tab: map[int]int{1: 3}, Single: map[int]int{36: 4}}

	for _, tc := range []struct {
		name  string
		base  int
		b     ItemSkillBonus
		sk    *Skill
		class string
		page  int
		want  int
	}{
		{"no bonus", 5, ItemSkillBonus{}, fb, "sor", 1, 5},
		{"all stacking, own tab", 5, b, fb, "sor", 1, 5 + 1 + 2 + 3 + 4},
		{"other tab", 5, b, fb, "sor", 2, 5 + 1 + 2 + 4},
		{"other class gets only single", 0, b, fb, "pal", 1, 4},
		{"unlearned skill gets level from items", 0, b, fb, "sor", 1, 10},
		{"never negative", 0, ItemSkillBonus{All: -3}, fb, "sor", 1, 0},
		{"nil skill", 3, b, nil, "sor", 1, 3},
	} {
		if got := EffectiveLevel(tc.base, tc.b, tc.sk, tc.class, tc.page); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}
