package d2skilldesc

import "testing"

func dmgTr(s string) string {
	return map[string]string{
		"StrSkill4": "Damage: ", "StrSkill5": "Fire Damage: ", "StrSkill6": "Cold Damage: ",
		"StrSkill7": "Lightning Damage: ", "StrSkill8": "Poison Damage: ", "StrSkill39": "Magic Damage: ",
		"StrSkill10": "To Attack Rating: ", "StrSkill23": " percent", "StrSkill13": "Cold Length: ",
		"StrSkill14": "Poison Length: ", "StrSkill20": "Duration: ", "StrSkill42": "Life: ",
		"StrSkill15": " second", "StrSkill16": " seconds",
		"T": "Take ", "U": "x", "Fmt": "Charges %d", "Syn": "Shock Field",
	}[s]
}

func dmgEval(s string) int {
	return map[string]int{"pct": 50, "flat": 3, "neg": -4, "f60": 60, "f130": 130, "f25": 25, "f10": 10, "f75": 75, "par": 40}[s]
}

func TestDamageRows(t *testing.T) {
	ctx := &Ctx{
		ToHit: func() int { return 20 },
		Phys:  func() (int, int) { return 10, 20 },
		Elem:  func() (int, int, int) { return 5, 9, 4 },
		// 100 frames = 4 seconds
		ElemLen:  func() int { return 100 },
		Life:     func() (int, bool) { return 200, true },
		CurseDiv: 2,
	}

	tests := []struct {
		name string
		row  Row
		ctx  *Ctx
		want string
		ok   bool
	}{
		{"kind 8 to-hit", Row{Kind: 8}, ctx, "To Attack Rating: +20 percent", true},
		{"kind 8 zero", Row{Kind: 8}, &Ctx{ToHit: func() int { return 0 }}, "", false},
		{"kind 8 no hook", Row{Kind: 8}, nil, "", false},
		{"kind 9 range", Row{Kind: 9}, ctx, "Damage: 10-20", true},
		{"kind 9 percent and flat", Row{Kind: 9, CalcA: "pct", CalcB: "flat"}, ctx, "Damage: 18-33", true},
		{"kind 9 equal ends use plus", Row{Kind: 9}, &Ctx{Phys: func() (int, int) { return 7, 7 }}, "Damage: +7", true},
		{"kind 9 texta textb first", Row{Kind: 9, TextA: "T", TextB: "U"}, ctx, "Take xDamage: 10-20", true},
		{"kind 9 nothing", Row{Kind: 9}, &Ctx{Phys: func() (int, int) { return 0, 0 }}, "", false},
		{"kind 10 cold", Row{Kind: 10}, ctx, "Cold Damage: 5-9", true},
		{"kind 10 equal has no plus", Row{Kind: 10}, &Ctx{Elem: func() (int, int, int) { return 6, 6, 1 }}, "Fire Damage: 6", true},
		{"kind 10 magic", Row{Kind: 10}, &Ctx{Elem: func() (int, int, int) { return 1, 2, 3 }}, "Magic Damage: 1-2", true},
		{"kind 10 zero", Row{Kind: 10}, &Ctx{Elem: func() (int, int, int) { return 0, 0, 1 }}, "", false},
		{"kind 10 no element", Row{Kind: 10}, &Ctx{Elem: func() (int, int, int) { return 1, 2, 0 }}, "", false},
		{"kind 11 cold length", Row{Kind: 11}, ctx, "Cold Length: 4 seconds", true},
		{"kind 11 poison length", Row{Kind: 11}, &Ctx{
			Elem:    func() (int, int, int) { return 1, 2, 5 },
			ElemLen: func() int { return 25 },
		}, "Poison Length: 1 second", true},
		{"kind 11 other element has none", Row{Kind: 11}, &Ctx{
			Elem:    func() (int, int, int) { return 1, 2, 1 },
			ElemLen: func() int { return 25 },
		}, "", false},
		{"kind 13 life", Row{Kind: 13, CalcA: "pct", CalcB: "flat"}, ctx, "Life: 303", true},
		{"kind 13 no hook", Row{Kind: 13}, &Ctx{}, "", false},
		{"kind 13 zero", Row{Kind: 13}, &Ctx{Life: func() (int, bool) { return 0, true }}, "", false},
		{"kind 15 label", Row{Kind: 15, TextA: "T", TextB: "U"}, nil, "Take : x", true},
		{"kind 16 whole range", Row{Kind: 16, CalcA: "f25", CalcB: "f75"}, nil, "Duration: 1-3 seconds", true},
		{"kind 16 tenths", Row{Kind: 16, CalcA: "f60", CalcB: "f130"}, nil, "Duration: 2.4-5.2 seconds", true},
		{"kind 16 zero", Row{Kind: 16}, nil, "", false},
		{"kind 18 text", Row{Kind: 18, TextA: "T"}, nil, "Take ", true},
		{"kind 18 empty", Row{Kind: 18}, nil, "", false},
		{"kind 25 two texts", Row{Kind: 25, TextA: "T", TextB: "U"}, nil, "Take x", true},
		{"kind 31 divided by the difficulty divisor", Row{Kind: 31, TextA: "T", CalcA: "f130"}, ctx, "Take 2.6 seconds", true},
		{"kind 31 without divisor", Row{Kind: 31, TextA: "T", CalcA: "f130"}, nil, "Take 5.2 seconds", true},
		{"kind 67 signed no percent", Row{Kind: 67, TextA: "Syn", TextB: "U", CalcA: "par"}, nil, "Shock Field: +40 x", true},
		{"kind 67 negative", Row{Kind: 67, TextA: "Syn", TextB: "U", CalcA: "neg"}, nil, "Shock Field: -4 x", true},
		{"kind 67 no texta", Row{Kind: 67, TextB: "U", CalcA: "par"}, nil, "+40 x", true},
		{"kind 71 formatted", Row{Kind: 71, TextA: "Syn", TextB: "Fmt", CalcA: "par"}, nil, "Shock Field: Charges 40", true},
		{"kind 71 needs both texts", Row{Kind: 71, TextA: "Syn", CalcA: "par"}, nil, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := RowLineCtx(tc.row, dmgTr, dmgEval, tc.ctx)
			if got != tc.want || ok != tc.ok {
				t.Fatalf("got (%q,%v) want (%q,%v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestBlockCtxOrder(t *testing.T) {
	// Magic Arrow's rows (1, 11, 10, 9, 8): the damage lines sit where the
	// rows are, not in front of the block.
	rows := []Row{{Kind: 1}, {Kind: 11}, {Kind: 10}, {Kind: 9}, {Kind: 8}}
	ctx := &Ctx{
		Mana:    func() (string, bool) { return "Mana Cost: 2", true },
		ToHit:   func() int { return 5 },
		Phys:    func() (int, int) { return 3, 4 },
		Elem:    func() (int, int, int) { return 1, 2, 4 },
		ElemLen: func() int { return 50 },
	}

	got := BlockCtx(rows, dmgTr, dmgEval, ctx)
	want := []string{"Mana Cost: 2", "Cold Length: 2 seconds", "Cold Damage: 1-2", "Damage: 3-4", "To Attack Rating: +5 percent"}

	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q want %q", i, got[i], want[i])
		}
	}
}
