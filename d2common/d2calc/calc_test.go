package d2calc

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testEnv is a skill context: level, Param1..8 and the base levels of other
// skills.
type testEnv struct {
	ZeroEnv
	lvl    int
	par    [9]int // 1-based
	blvl   map[string]int
	fields map[string]int
	stats  map[string]int
	rnd    int
}

func (e *testEnv) Field(code string) int {
	if v, ok := e.fields[code]; ok {
		return v
	}

	switch code {
	case "lvl":
		return e.lvl
	case "ln12":
		return LN(e.par[1], e.par[2], e.lvl)
	case "ln34":
		return LN(e.par[3], e.par[4], e.lvl)
	case "ln56":
		return LN(e.par[5], e.par[6], e.lvl)
	case "ln78":
		return LN(e.par[7], e.par[8], e.lvl)
	case "dm12":
		return DM(e.par[1], e.par[2], e.lvl)
	case "dm34":
		return DM(e.par[3], e.par[4], e.lvl)
	}

	if len(code) == 4 && code[:3] == "par" {
		return e.par[code[3]-'0']
	}

	return 0
}

func (e *testEnv) Skill(name, field string) int {
	if field == "blvl" {
		return e.blvl[name]
	}

	return 0
}

func (e *testEnv) Stat(name, mode string) int { return e.stats[name+"."+mode] }
func (e *testEnv) Rand(a, b int) int          { return a + e.rnd%(b-a) }

func TestEvalRealSkillRows(t *testing.T) {
	tests := []struct {
		name string
		src  string
		env  testEnv
		want int
	}{
		// Charged Bolt calc1: bolts per cast = min(24, 3+(lvl-1))
		{"chargedbolt L1", "min(24,ln12)", testEnv{lvl: 1, par: [9]int{0, 3, 1}}, 3},
		{"chargedbolt L5", "min(24,ln12)", testEnv{lvl: 5, par: [9]int{0, 3, 1}}, 7},
		{"chargedbolt L22", "min(24,ln12)", testEnv{lvl: 22, par: [9]int{0, 3, 1}}, 24},
		{"chargedbolt L30 capped", "min(24,ln12)", testEnv{lvl: 30, par: [9]int{0, 3, 1}}, 24},
		// Charged Bolt synergy: Lightning blvl * par8 (6)
		{"chargedbolt synergy", "(skill('Lightning'.blvl))*par8",
			testEnv{lvl: 10, par: [9]int{8: 6}, blvl: map[string]int{"Lightning": 7}}, 42},
		// Fire Bolt synergy: (Fire Ball + Meteor) * 16
		{"firebolt synergy", "(skill('Fire Ball'.blvl)+skill('Meteor'.blvl))*par8",
			testEnv{lvl: 10, par: [9]int{8: 16}, blvl: map[string]int{"Fire Ball": 5, "Meteor": 3}}, 128},
		{"firebolt synergy none", "(skill('Fire Ball'.blvl)+skill('Meteor'.blvl))*par8",
			testEnv{lvl: 10, par: [9]int{8: 16}}, 0},
		// Ice Bolt synergy with five terms, par8 15
		{"icebolt synergy",
			"(skill('Frost Nova'.blvl)+skill('Ice Blast'.blvl)+skill('Glacial Spike'.blvl)+skill('Blizzard'.blvl)+skill('Frozen Orb'.blvl))*par8",
			testEnv{lvl: 1, par: [9]int{8: 15}, blvl: map[string]int{"Frost Nova": 1, "Ice Blast": 2, "Glacial Spike": 3, "Blizzard": 4, "Frozen Orb": 5}}, 225},
		// Frost Nova synergy (Blizzard + Frozen Orb) * 10
		{"frostnova synergy", "(skill('Blizzard'.blvl)+skill('Frozen Orb'.blvl))*par8",
			testEnv{par: [9]int{8: 10}, blvl: map[string]int{"Blizzard": 6, "Frozen Orb": 4}}, 100},
		// Frozen Armor duration: ln34 + (Shiver+Chilling)*par7 = 3000+300*9 + 6*250
		{"frozen armor len", "ln34+(skill('Shiver Armor'.blvl)+skill('Chilling Armor'.blvl))*par7",
			testEnv{lvl: 10, par: [9]int{3: 3000, 4: 300, 7: 250}, blvl: map[string]int{"Shiver Armor": 2, "Chilling Armor": 4}}, 7200},
		// Frozen Armor chill length, integer division after the product
		{"frozen armor calc1", "ln56*(100+((skill('Shiver Armor'.blvl)+skill('Chilling Armor'.blvl))*par8))/100",
			testEnv{lvl: 10, par: [9]int{5: 30, 6: 3, 8: 5}, blvl: map[string]int{"Shiver Armor": 2, "Chilling Armor": 4}}, 74},
		// Howl velocity bonus
		{"howl calc1", "par1 * (lvl-1)", testEnv{lvl: 5, par: [9]int{0, 2}}, 8},
		// Bash damage percent and to-hit
		{"bash calc1", "ln12+skill('Stun'.blvl)*par8",
			testEnv{lvl: 4, par: [9]int{0, 50, 5, 0, 0, 0, 0, 0, 5}, blvl: map[string]int{"Stun": 3}}, 80},
		{"bash tohit", "15+lvl*5+skill('Concentrate'.blvl)*par7",
			testEnv{lvl: 6, par: [9]int{7: 5}, blvl: map[string]int{"Concentrate": 2}}, 55},
		// Jab damage percent -15+3(l-1)
		{"jab calc1 L1", "ln34", testEnv{lvl: 1, par: [9]int{3: -15, 4: 3}}, -15},
		{"jab calc1 L20", "ln34", testEnv{lvl: 20, par: [9]int{3: -15, 4: 3}}, 42},
		// Inner Sight armor class: negative of the elemental min
		{"inner sight", "-edmn", testEnv{fields: map[string]int{"edmn": 70}}, -70},
		// Fire Arrow synergy
		{"firearrow synergy", "(skill('Exploding Arrow'.blvl)) * par8",
			testEnv{par: [9]int{8: 12}, blvl: map[string]int{"Exploding Arrow": 4}}, 48},
		// Find Item diminishing returns
		{"dm12 L1", "dm12", testEnv{lvl: 1, par: [9]int{0, 5, 60}}, 13},
		{"dm12 L20", "dm12", testEnv{lvl: 20, par: [9]int{0, 5, 60}}, 51},
		{"stat accr", "stat('passive_fire_mastery'.accr)*2",
			testEnv{stats: map[string]int{"passive_fire_mastery.accr": 21}}, 42},
		{"rand", "rand(par3,par4)", testEnv{par: [9]int{3: 5, 4: 10}, rnd: 7}, 7},
		{"rand empty range", "rand(par4,par3)", testEnv{par: [9]int{3: 5, 4: 10}, rnd: 7}, 10},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := Compile(tc.src, KindSkill).Eval(&tc.env); got != tc.want {
				t.Fatalf("%q = %d, want %d", tc.src, got, tc.want)
			}
		})
	}
}

func TestLanguageQuirks(t *testing.T) {
	tests := []struct {
		src  string
		want int
	}{
		{"1+2*3", 7},
		{"10-2-3", 5},
		{"2^3^2", 64}, // left associative (the game), not 512
		{"2^0", 1},
		{"2^-1", 1},
		{"7/0", 0},
		{"-7/2", -3},
		{"7/2", 3},
		{"1<2", 1},
		{"2<=1", 0},
		{"3==3", 1},
		{"3!=3", 0},
		{"1+1<3", 1},  // comparison is looser than +
		{"1<2==1", 1}, // equal rank, left to right: (1<2)==1
		{"(lvl>3)?(10):(20)", 20},
		{"1 ? 2 : 3 + 10", 12}, // '?' binds tighter than +: (1?2:3)+10
		{"0 ? 2 : 3 + 10", 13},
		{"5 + 1 ? 2 : 3", 7}, // '?' binds tighter than the '+' before it: 5+(1?2:3)
		{"-3+5", 2},
		{"--3", 3},
		{"+4", 4},
		{"4 = 4", 4}, // a lone '=' is ignored: two operands, the top of the stack wins
		{"nosuchcode+3", 3},
		{"(2+3", 5}, // unclosed parenthesis tolerated
		{"max(1,min(9,4))", 4},
		{"min(2*3,4+1)", 5},
		{"", 0},
		{"2147483647+1", -2147483648}, // int32 wraparound
	}

	for _, tc := range tests {
		if got := Compile(tc.src, KindSkill).Eval(&testEnv{}); got != tc.want {
			t.Errorf("%q = %d, want %d", tc.src, got, tc.want)
		}
	}
}

func TestFieldCodesAreFirstFourChars(t *testing.T) {
	e := &testEnv{lvl: 4, par: [9]int{0, 3, 2}}
	// codes are case insensitive; ln12 = 3 + 3*2
	if got := Compile("PAR1+Ln12", KindSkill).Eval(e); got != 3+(3+3*2) {
		t.Fatalf("got %d", got)
	}
}

func TestMissileCodes(t *testing.T) {
	if !IsFieldCode("dl12", KindMissile) || IsFieldCode("dl12", KindSkill) {
		t.Fatal("dl12 must be a missile-only code")
	}

	e := &testEnv{fields: map[string]int{"dl12": 9}}
	if got := Compile("dl12*2", KindMissile).Eval(e); got != 18 {
		t.Fatalf("got %d", got)
	}
	// a skill-only code in the missile dialect is the literal 0
	if got := Compile("clc1+1", KindMissile).Eval(&testEnv{fields: map[string]int{"clc1": 5}}); got != 1 {
		t.Fatalf("clc1 is unknown in misscalc, got %d", got)
	}
}

func TestNilAndEmpty(t *testing.T) {
	var p *Program
	if p.Eval(ZeroEnv{}) != 0 || !p.Empty() || p.Source() != "" {
		t.Fatal("nil program")
	}
}

func TestLNDM(t *testing.T) {
	if LN(3, 1, 0) != 0 || LN(30, 12, 1) != 30 || LN(30, 12, 11) != 150 {
		t.Fatal("LN")
	}
	// Warmth: 30 + 12(l-1) mana recovery percent
	if LN(30, 12, 20) != 258 {
		t.Fatal("warmth L20")
	}
}

// TestRealTablesCompile compiles every calc column of the real skills.txt and
// missiles.txt. It needs D2_TABLES (a folder with skills/patch_d2/*.txt).
func TestRealTablesCompile(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	for _, tc := range []struct {
		file string
		kind Kind
	}{{"skills.txt", KindSkill}, {"missiles.txt", KindMissile}} {
		f, err := os.Open(filepath.Join(root, "skills", "patch_d2", tc.file))
		if err != nil {
			t.Skip(err)
		}

		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)

		var header []string

		n := 0

		for sc.Scan() {
			cols := strings.Split(sc.Text(), "\t")
			if header == nil {
				header = cols
				continue
			}

			for i, c := range cols {
				if i >= len(header) || c == "" || !strings.Contains(strings.ToLower(header[i]), "calc") {
					continue
				}

				Compile(c, tc.kind).Eval(ZeroEnv{}) // must not panic
				n++
			}
		}

		f.Close()

		if n == 0 {
			t.Fatalf("%s: no calc cells found", tc.file)
		}
	}
}
