package d2skilldesc

import (
	"reflect"
	"testing"
)

func testTr(s string) string {
	return map[string]string{"A": "Dur: ", "B": " sec", "Syn": "Fire Arrow", "Pl": "Fire Damage"}[s]
}

func testEval(s string) int {
	return map[string]int{"ln": 7, "neg": -4, "hi": 20, "par8": 12}[s]
}

func TestRowLine(t *testing.T) {
	tests := []struct {
		name string
		row  Row
		want string
		ok   bool
	}{
		{"plain", Row{Kind: 12, TextA: "A", CalcA: "ln"}, "Dur: 7", true},
		{"plain with tail", Row{Kind: 19, TextA: "A", TextB: "B", CalcA: "ln"}, "Dur: 7 sec", true},
		{"signed neg", Row{Kind: KindSignedNeg, TextA: "A", CalcA: "neg"}, "Dur: -4", true},
		{"range", Row{Kind: 52, TextA: "A", TextB: "B", CalcA: "ln", CalcB: "hi"}, "Dur: 7-20 sec", true},
		{"synergy", Row{Kind: 63, TextA: "Syn", TextB: "Pl", CalcA: "par8"}, "Fire Arrow: +12% Fire Damage", true},
		{"synergy header", Row{Kind: 40, TextA: "A", TextB: "Syn"}, "Dur: Fire Arrow", true},
		{"empty header", Row{Kind: 40}, "", false},
		{"unmodelled", Row{Kind: 66}, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RowLine(tc.row, testTr, testEval)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("got (%q,%v) want (%q,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestDamage(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		min, max int
		want     string
		ok       bool
	}{
		{"range", "Damage: ", 3, 9, "Damage: 3 to 9", true},
		{"fixed", "Damage: ", 5, 5, "Damage: 5", true},
		{"none", "Damage: ", 0, 0, "", false},
		{"no label", "", 1, 2, "", false},
	}

	for _, tc := range tests {
		got, ok := Damage(tc.label, tc.min, tc.max)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: got (%q,%v)", tc.name, got, ok)
		}
	}
}

func TestNextLevel(t *testing.T) {
	d := Desc{
		Lines:  []Row{{Kind: 12, TextA: "A", CalcA: "ln"}},
		Lines2: []Row{{Kind: 66}, {Kind: 4, TextA: "A", CalcA: "hi"}},
	}

	tests := []struct {
		name         string
		level, maxLv int
		dmg          string
		want         []string
	}{
		{"block", 3, 20, "Damage: 1 to 2", []string{"Next:", "Damage: 1 to 2", "Dur: 7", "Dur: 20"}},
		{"capped", 20, 20, "", nil},
		{"uncapped when max 0", 99, 0, "", []string{"Next:", "Dur: 7", "Dur: 20"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextLevel("Next:", d, tc.level, tc.maxLv, tc.dmg, testTr, testEval)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}

	if got := NextLevel("Next:", Desc{}, 1, 0, "", testTr, testEval); got != nil {
		t.Fatalf("empty desc should give no block, got %v", got)
	}
}
