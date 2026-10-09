package d2skilldesc

import "testing"

func TestFormatLine(t *testing.T) {
	tests := []struct {
		name         string
		kind         int
		a, b         string
		value        int
		want         string
		wantModelled bool
	}{
		{"jab damage", KindSignedValue, "Damage: ", " percent", 120, "Damage: +120 percent", true},
		{"negative bonus keeps its sign", KindSignedValue, "Enemy Defense: ", "", -35, "Enemy Defense: -35", true},
		{"slow missiles", KindValue, "Ranged attacks slowed to ", " percent", 50, "Ranged attacks slowed to 50 percent", true},
		{"multiple shot", KindCount, " arrows", "", 7, "7 arrows", true},
		{"zero value has no line", KindValue, "x", "y", 0, "", false},
		{"unmodelled kind", 40, "x", "y", 1, "", false},
		{"kind 0 is empty", 0, "x", "y", 1, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := FormatLine(tc.kind, tc.a, tc.b, tc.value)
			if ok != tc.wantModelled || got != tc.want {
				t.Fatalf("got (%q,%v) want (%q,%v)", got, ok, tc.want, tc.wantModelled)
			}
		})
	}
}

func TestManaCost(t *testing.T) {
	tests := []struct {
		name  string
		label string
		cost  int
		want  string
		ok    bool
	}{
		{"Fire Bolt shows 2.5", "Mana Cost: ", 2*256 + 128, "Mana Cost: 2.5", true},
		{"Magic Arrow level 2 truncates the tenth", "Mana Cost: ", 352, "Mana Cost: 1.3", true},
		{"whole", "Mana Cost: ", 9 * 256, "Mana Cost: 9", true},
		{"above 25 mana is whole", "Mana Cost: ", 26*256 + 128, "Mana Cost: 26", true},
		{"free skill has no line", "Mana Cost: ", 0, "", false},
		{"under a tenth has no line", "Mana Cost: ", 5, "", false},
		{"no label", "", 512, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ManaCost(tc.label, tc.cost)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("got (%q,%v) want (%q,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}
