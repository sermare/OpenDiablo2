package d2skilldesc

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// realRow reads the descline/dsc2line/dsc3line column `line` (for example
// "descline2") of the skilldesc.txt row named `name` from D2_TABLES.
func realRow(t *testing.T, name, line string) Row {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	b, err := os.ReadFile(filepath.Join(dir, "skills", "patch_d2", "skilldesc.txt"))
	if err != nil {
		t.Skip("skilldesc.txt not readable: ", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n")
	head := strings.Split(lines[0], "\t")
	col := map[string]int{}

	for i, h := range head {
		col[strings.ToLower(h)] = i
	}

	for _, l := range lines[1:] {
		f := strings.Split(l, "\t")
		if len(f) < len(head) || f[0] != name {
			continue
		}

		get := func(c string) string { return f[col[strings.Replace(line, "line", c, 1)]] }
		kind, _ := strconv.Atoi(f[col[line]])

		return Row{Kind: kind, TextA: get("texta"), TextB: get("textb"), CalcA: get("calca"), CalcB: get("calcb")}
	}

	t.Fatalf("row %q not found", name)

	return Row{}
}

// evalCalc is a tiny integer evaluator: numbers, names (vars, lvl), + - * /
// and parentheses.
func evalCalc(src string, vars map[string]int) int {
	p := &calcParser{s: strings.ReplaceAll(src, " ", ""), vars: vars}

	return p.sum()
}

type calcParser struct {
	s    string
	i    int
	vars map[string]int
}

func (p *calcParser) sum() int {
	v := p.prod()

	for p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
		op := p.s[p.i]
		p.i++

		if r := p.prod(); op == '+' {
			v += r
		} else {
			v -= r
		}
	}

	return v
}

func (p *calcParser) prod() int {
	v := p.atom()

	for p.i < len(p.s) && (p.s[p.i] == '*' || p.s[p.i] == '/') {
		op := p.s[p.i]
		p.i++

		if r := p.atom(); op == '*' {
			v *= r
		} else if r != 0 {
			v /= r
		}
	}

	return v
}

func (p *calcParser) atom() int {
	if p.i >= len(p.s) {
		return 0
	}

	if p.s[p.i] == '(' {
		p.i++
		v := p.sum()
		p.i++

		return v
	}

	j := p.i
	for j < len(p.s) && (unicode.IsLetter(rune(p.s[j])) || unicode.IsDigit(rune(p.s[j]))) {
		j++
	}

	tok := p.s[p.i:j]
	p.i = j

	if n, err := strconv.Atoi(tok); err == nil {
		return n
	}

	return p.vars[tok]
}

// realTr is the English text of the keys the rows below use (string.tbl,
// patchstring.tbl and expansionstring.tbl of 1.14b).
func realTr(s string) string {
	return map[string]string{
		"StrSkill89": "Walk/Run Speed: ", "StrSkill23": " percent", "StrSkill106": "Attack Speed: ",
		"EskillLifeSteal": "Life Steal: ", "eskillincasemasteryX": "%d Percent Chance of Critical Strike",
		"Convphy2elemalt": "Converts %d%% Physical Damage to Magic Damage", "WeapDamsk": "Weapon Damage",
		"Eskillpowerup3": "Charge 3 - ", "Eskillphoenix3": "chaos ice bolt damage: ",
		"StrSkill113": "Hydra Fire Damage: ", "StrSkill20": "Duration: ", "StrSkill82": "Fire Duration: ",
		"StrSkill63Patch": "over ", "StrSkill68": "Average ", "StrSkill34": " per second",
		"StrSkill15": " second", "StrSkill16": " seconds", "StrSkill8": "Poison Damage: ",
		"StrSkill5": "Fire Damage: ", "Eskillfistsoffire1": "fire damage: ", "Eskillpowerup1": "Charge 1 - ",
	}[s]
}

func TestMoreKindsRealRows(t *testing.T) {
	tests := []struct {
		name, skill, line string
		kind              int
		vars              map[string]int
		ctx               *Ctx
		want              string
		ok                bool
	}{
		// 51: macr is the mastery critical-strike percent, here 12 and 54.
		{"sword mastery lvl1", "sword mastery", "descline1", 51, map[string]int{"macr": 12}, nil,
			"12 Percent Chance of Critical Strike", true},
		{"sword mastery lvl20", "sword mastery", "descline1", 51, map[string]int{"macr": 54}, nil,
			"54 Percent Chance of Critical Strike", true},
		// 66: 1+(lvl-1) percent.
		{"magic arrow conversion lvl1", "magic arrow", "descline6", 66, map[string]int{"lvl": 1}, nil,
			"Converts 1% Physical Damage to Magic Damage", true},
		{"magic arrow conversion lvl20", "magic arrow", "descline6", 66, map[string]int{"lvl": 20}, nil,
			"Converts 20% Physical Damage to Magic Damage", true},
		// 52: feral rage life steal 4 to par2*lvl+8 with par2=3.
		{"feral rage steal lvl1", "feral rage", "descline3", 52, map[string]int{"lvl": 1, "par2": 3}, nil,
			"Life Steal: +4-11 percent", true},
		{"feral rage steal lvl20", "feral rage", "descline3", 52, map[string]int{"lvl": 20, "par2": 3}, nil,
			"Life Steal: +4-68 percent", true},
		// 62: royal strike charge 3 range.
		{"royal strike lvl1", "royal strike", "descline3", 62, map[string]int{"m3en": 5, "m3ex": 9}, nil,
			"Charge 3 - chaos ice bolt damage: 5-9", true},
		{"royal strike equal", "royal strike", "descline3", 62, map[string]int{"m3en": 7, "m3ex": 7}, nil,
			"Charge 3 - 7chaos ice bolt damage: ", true},
		// 72 and 73: weapon damage fraction.
		{"blade fury", "blade fury", "descline2", 72, nil, nil, "+3/4 Weapon Damage", true},
		{"strafe", "strafe", "dsc2line1", 73, nil, nil, "3/4 Weapon Damage", true},
		// 23: blizzard duration, 3*level + 100 frames.
		{"blizzard lvl1", "blizzard", "descline2", 23, nil,
			&Ctx{MissileRange: func() int { return 103 }}, "Duration: 4.1 seconds", true},
		{"blizzard lvl20", "blizzard", "descline2", 23, nil,
			&Ctx{MissileRange: func() int { return 160 }}, "Duration: 6.4 seconds", true},
		// 24: hydra explosion range uses texta as label.
		{"hydra lvl1", "hydra", "descline2", 24, nil,
			&Ctx{Elem: func() (int, int, int) { return 10, 14, 1 }}, "Hydra Fire Damage: 10-14", true},
		{"hydra lvl20", "hydra", "descline2", 24, nil,
			&Ctx{Elem: func() (int, int, int) { return 200, 260, 1 }}, "Hydra Fire Damage: 200-260", true},
		{"hydra zero", "hydra", "descline2", 24, nil,
			&Ctx{Elem: func() (int, int, int) { return 0, 0, 1 }}, "", false},
		// 14: poison javelin, 50 frames = 2 seconds.
		{"poison javelin lvl1", "poison javelin", "descline2", 14, nil,
			&Ctx{ElemOverTime: func() (int, int, int, int) { return 12, 12, 50, 5 }},
			"Poison Damage: 12\nover 2 seconds", true},
		{"poison javelin lvl20", "poison javelin", "descline2", 14, nil,
			&Ctx{ElemOverTime: func() (int, int, int, int) { return 400, 450, 130, 5 }},
			"Poison Damage: 400-450\nover 5.2 seconds", true},
		// 22: meteor burn per second.
		{"meteor", "meteor", "descline2", 22, nil,
			&Ctx{MissileDamage: func() (int, int, int) { return 8, 12, 1 }},
			"Average Fire Damage: 8-12 per second", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			row := realRow(t, tc.skill, tc.line)
			if row.Kind != tc.kind {
				t.Fatalf("table row kind %d, want %d", row.Kind, tc.kind)
			}

			eval := func(s string) int { return evalCalc(s, tc.vars) }
			got, ok := RowLineCtx(row, realTr, eval, tc.ctx)

			if ok != tc.ok || got != tc.want {
				t.Fatalf("got %q,%v want %q,%v", got, ok, tc.want, tc.ok)
			}
		})
	}
}
