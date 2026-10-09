package d2monstats

import (
	"os"
	"path/filepath"
	"testing"
)

func synthLvl() MonLvl {
	t := make(MonLvl, 10)
	for i := range t {
		for e := 0; e < 2; e++ {
			for d := 0; d < numDiff; d++ {
				k := (d + 1) * 100
				t[i].AC[e][d] = k + i
				t[i].TH[e][d] = k*2 + i
				t[i].HP[e][d] = k*10 + i*e
				t[i].DM[e][d] = k/10 + i
				t[i].XP[e][d] = k*3 + i
			}
		}
	}

	return t
}

func TestMulDiv(t *testing.T) {
	for _, c := range []struct{ a, b, c, want int }{
		{10, 50, 100, 5}, {3, 50, 100, 1}, {1, 49, 100, 0}, {7, 100, 100, 7}, {0, 5, 100, 0}, {-3, 50, 100, -1},
	} {
		if got := MulDiv(c.a, c.b, 100); got != c.want {
			t.Errorf("MulDiv(%d,%d)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestScale(t *testing.T) {
	ml := synthLvl()
	ratio := &Class{
		MinHP: [3]int{50, 100, 150}, MaxHP: [3]int{100, 100, 200},
		AC: [3]int{100, 50, 200}, Exp: [3]int{100, 100, 100},
		Level: [3]int{1, 2, 3},
	}
	ratio.Attacks[A1] = [3]AttackStats{{100, 50, 100}, {100, 50, 100}, {200, 100, 200}}

	raw := &Class{NoRatio: true, MinHP: [3]int{77, 88, 99}, MaxHP: [3]int{77, 90, 99}, AC: [3]int{5, 6, 7},
		Exp: [3]int{11, 12, 13}, Level: [3]int{3, 4, 5}}
	raw.Attacks[A2] = [3]AttackStats{{9, 1, 2}, {9, 1, 2}, {9, 1, 2}}

	boss := &Class{Boss: true, PrimeEvil: true, MinHP: [3]int{100, 100, 100}, MaxHP: [3]int{100, 100, 100}, Level: [3]int{4, 5, 6}}

	first := func(n int) int { return 0 }
	last := func(n int) int { return n - 1 }

	tests := []struct {
		name      string
		c         *Class
		diff, lvl int
		exp       bool
		roll      func(int) int
		check     func(Stats) string
	}{
		{"nightmare area level", ratio, Nightmare, 5, false, first, func(s Stats) string {
			if s.Level != 5 || s.HP != 2000 || s.HPMax != 2000 {
				return "level/hp"
			}

			return ""
		}},
		{"normal ignores area level", ratio, Normal, 5, false, first, func(s Stats) string {
			if s.Level != 1 || s.HP != 500 {
				return "normal level"
			}

			return ""
		}},
		{"hp min roll", ratio, Normal, 5, false, first, func(s Stats) string {
			if s.HP != MulDiv(1000+0, 50, 100) {
				return "hp min"
			}

			return ""
		}},
		{"hp max roll", ratio, Normal, 5, false, last, func(s Stats) string {
			if s.HP != s.HPMax || s.HPMax != 1000 {
				return "hp max"
			}

			return ""
		}},
		{"ac ratio nightmare", ratio, Nightmare, 7, false, first, func(s Stats) string {
			if s.AC != MulDiv(207, 50, 100) {
				return "ac"
			}

			return ""
		}},
		{"expansion columns", ratio, Normal, 5, true, first, func(s Stats) string {
			if s.HPMax != 1000+1 { // L-HP = 1000 + i*e at normal level 1
				return "expansion hp"
			}

			return ""
		}},
		{"hell damage and TH", ratio, Hell, 2, false, first, func(s Stats) string {
			a := s.Attacks[A1]
			if a.TH != 1204 || a.Min != 32 || a.Max != 64 {
				return "hell attack"
			}

			return ""
		}},
		{"noRatio uses raw values and class level", raw, Nightmare, 9, false, first, func(s Stats) string {
			if s.Level != 4 || s.HP != 88 || s.HPMax != 90 || s.AC != 6 || s.XP != 12 || s.Attacks[A2].Max != 2 {
				return "raw"
			}

			return ""
		}},
		{"boss ignores area level", boss, Hell, 9, false, first, func(s Stats) string {
			if s.Level != 6 {
				return "boss level"
			}

			return ""
		}},
		{"area level zero falls back", ratio, Hell, 0, false, first, func(s Stats) string {
			if s.Level != 3 {
				return "fallback"
			}

			return ""
		}},
		{"level beyond table clamps", ratio, Nightmare, 500, false, nil, func(s Stats) string {
			if s.Level != 500 || s.HP == 0 {
				return "clamp"
			}

			return ""
		}},
		{"difficulty clamps", ratio, 7, 1, false, first, func(s Stats) string {
			if s.Level != 1 {
				return "diff"
			}

			return ""
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := ml.Scale(tc.c, tc.diff, tc.lvl, tc.exp, tc.roll)
			if msg := tc.check(s); msg != "" {
				t.Errorf("%s: %+v", msg, s)
			}
		})
	}
}

func TestRealData(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	read := func(n string) []byte {
		b, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", n))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	ml, err := LoadMonLvl(read("monlvl.txt"))
	if err != nil {
		t.Fatal(err)
	}

	cl, err := LoadClasses(read("monstats.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if len(ml) < 100 || len(cl) < 400 {
		t.Fatalf("tables too small: %d levels, %d classes", len(ml), len(cl))
	}

	and := cl["andariel"]
	if and == nil || !and.Boss || and.Index != 156 {
		t.Fatalf("andariel: %+v", and)
	}

	tests := []struct {
		diff int
		exp  bool
		lvl  int
		hp   int
	}{
		{Normal, false, 12, 1024}, // 40 * 2562% = 1024.8, truncated (VERIFIED 0x0047f2c0)
		{Normal, true, 12, 1024},  // L-HP equals HP at normal
		{Hell, false, 75, 45023},  // 3774 * 1193% = 45023.8 truncated
		{Hell, true, 75, 60031},   // 5032 * 1193% = 60031.76 truncated
	}

	for _, tc := range tests {
		s := ml.Scale(and, tc.diff, 1, tc.exp, func(int) int { return 0 })
		if s.Level != tc.lvl || s.HP != tc.hp {
			t.Errorf("andariel diff %d exp %v: level %d hp %d, want %d / %d", tc.diff, tc.exp, s.Level, s.HP, tc.lvl, tc.hp)
		}
	}

	// A plain monster takes the area level in Nightmare/Hell: fallen1 in a level-30 area.
	f := cl["fallen1"]
	s := ml.Scale(f, Nightmare, 30, false, func(n int) int { return n - 1 })

	if s.Level != 30 || s.HP != MulDiv(ml[30].HP[0][Nightmare], f.MaxHP[Nightmare], 100) {
		t.Errorf("fallen1: %+v", s)
	}

	// Every class scales without panicking in every mode.
	for _, c := range cl {
		for d := 0; d < numDiff; d++ {
			if st := ml.Scale(c, d, 40, true, nil); st.HPMax < st.HPMin || st.HP < 0 {
				t.Errorf("%s diff %d: %+v", c.ID, d, st)
			}
		}
	}
}

func TestHPCapAndTruncation(t *testing.T) {
	ml := synthLvl()
	big := &Class{MinHP: [3]int{1 << 30, 0, 0}, MaxHP: [3]int{1 << 30, 0, 0}}

	if s := ml.Scale(big, Normal, 0, false, nil); s.HP != maxHP {
		t.Errorf("hp cap: %d", s.HP)
	}
}
