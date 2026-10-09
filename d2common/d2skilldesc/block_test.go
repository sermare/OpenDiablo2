package d2skilldesc

import (
	"reflect"
	"testing"
)

func testTr(s string) string {
	return map[string]string{
		"A": "Dur: ", "B": " sec", "Syn": "Fire Arrow", "Pl": "Fire Damage",
		"Head": "%s Receives Bonuses From:", "StrSkill15": " second", "StrSkill16": " seconds",
		"StrSkill36": " yard", "StrSkill26": " yards",
	}[s]
}

func testEval(s string) int {
	return map[string]int{"ln": 7, "neg": -4, "hi": 20, "par8": 12, "zero": 0, "one": 1, "sec": 100, "f25": 25, "frac": 38, "r": 9}[s]
}

func TestRowLine(t *testing.T) {
	tests := []struct {
		name string
		row  Row
		want string
		ok   bool
	}{
		{"kind 2 signed", Row{Kind: 2, TextA: "A", TextB: "B", CalcA: "ln"}, "Dur: +7 sec", true},
		{"kind 2 negative", Row{Kind: 2, TextA: "A", TextB: "B", CalcA: "neg"}, "Dur: -4 sec", true},
		{"kind 2 zero has no line", Row{Kind: 2, TextA: "A", CalcA: "zero"}, "", false},
		{"kind 3 plain", Row{Kind: 3, TextA: "A", TextB: "B", CalcA: "ln"}, "Dur: 7 sec", true},
		{"kind 4 text and signed value", Row{Kind: 4, TextA: "A", TextB: "B", CalcA: "ln"}, "Dur: +7", true},
		{"kind 5 text and value", Row{Kind: 5, TextA: "A", TextB: "B", CalcA: "neg"}, "Dur: -4", true},
		{"kind 6 signed value then text", Row{Kind: 6, TextA: "B", CalcA: "ln"}, "+7 sec", true},
		{"kind 7 value then text", Row{Kind: 7, TextA: "B", CalcA: "ln"}, "7 sec", true},
		{"kind 12 whole seconds", Row{Kind: 12, TextA: "A", CalcA: "sec"}, "Dur: 4 seconds", true},
		{"kind 12 one second", Row{Kind: 12, TextA: "A", CalcA: "f25"}, "Dur: 1 second", true},
		{"kind 12 tenths", Row{Kind: 12, TextA: "A", CalcA: "frac"}, "Dur: 1.5 seconds", true},
		{"kind 12 zero", Row{Kind: 12, TextA: "A", CalcA: "zero"}, "", false},
		{"kind 19 yards", Row{Kind: 19, TextA: "A", CalcA: "r"}, "Dur: 6 yards", true},
		{"kind 19 one yard", Row{Kind: 19, TextA: "A", CalcA: "one"}, "Dur: 1.6 yard", true}, // the original picks " yard" when the whole part is 1 (0x4e54b4)
		{"kind 37 thirds", Row{Kind: 37, TextA: "A", CalcA: "r"}, "Dur: 3 yards", true},
		{"kind 38 range", Row{Kind: 38, TextA: "A", TextB: "B", CalcA: "ln", CalcB: "hi"}, "Dur: 7-20 sec", true},
		{"kind 38 equal ends", Row{Kind: 38, TextA: "A", TextB: "B", CalcA: "ln", CalcB: "ln"}, "Dur: 7 sec", true},
		{"synergy 63", Row{Kind: 63, TextA: "Syn", TextB: "Pl", CalcA: "par8"}, "Fire Arrow: +12% Fire Damage", true},
		{"synergy 63 zero", Row{Kind: 63, TextA: "Syn", TextB: "Pl", CalcA: "zero"}, "", false},
		{"synergy header", Row{Kind: 40, TextA: "Head", TextB: "Syn"}, "Fire Arrow Receives Bonuses From:", true},
		{"empty header", Row{Kind: 40}, "", false},
		{"mana row needs the mana hook", Row{Kind: KindMana}, "", false},
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

func TestBlockMana(t *testing.T) {
	rows := []Row{{Kind: KindMana}, {Kind: 3, TextA: "A", CalcA: "ln"}}
	mana := func() (string, bool) { return "Mana Cost: 2.5", true }

	if got, want := BlockMana(rows, testTr, testEval, mana), []string{"Mana Cost: 2.5", "Dur: 7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}

	if got, want := Block(rows, testTr, testEval), []string{"Dur: 7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("without a mana hook: got %v want %v", got, want)
	}
}

func TestCurrentLevel(t *testing.T) {
	if got := CurrentLevel("Current Skill Level: ", 7); got != "Current Skill Level: 7" {
		t.Fatalf("got %q", got)
	}
}

func TestNextLevel(t *testing.T) {
	d := Desc{
		Lines:  []Row{{Kind: 12, TextA: "A", CalcA: "sec"}, {Kind: KindMana}},
		Lines2: []Row{{Kind: 66}, {Kind: 5, TextA: "A", CalcA: "hi"}},
	}
	mana := func() (string, bool) { return "Mana Cost: 3", true }

	tests := []struct {
		name         string
		level, maxLv int
		want         []string
	}{
		{"block without the dsc2 rows", 3, 20, []string{"Next:", "Dur: 4 seconds", "Mana Cost: 3"}},
		{"capped", 20, 20, nil},
		{"uncapped when max 0", 99, 0, []string{"Next:", "Dur: 4 seconds", "Mana Cost: 3"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextLevel("Next:", d, tc.level, tc.maxLv, testTr, testEval, &Ctx{Mana: mana})
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}

	if got := NextLevel("Next:", Desc{}, 1, 0, testTr, testEval, nil); got != nil {
		t.Fatalf("empty desc should give no block, got %v", got)
	}
}
