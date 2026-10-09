package d2herostats

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRealExpCapAtLevel99 pins the experience cap with the real Experience.txt:
// every class caps at row MaxLvl-1 (3520485254 for 99, VERIFIED 0x0057c510), a
// clvl 99 hero stays 99 however large a kill is, and a kill never lowers it.
func TestRealExpCapAtLevel99(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "Experience.txt"))
	if err != nil {
		t.Skip(err)
	}

	tabs, err := ParseExperience(data)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range ClassNames {
		tb := tabs[c]
		if tb.MaxLevel != 99 || tb.ExpCap() != 3520485254 {
			t.Errorf("%s: max %d cap %d", c, tb.MaxLevel, tb.ExpCap())
		}

		for _, tc := range []struct {
			level int
			exp   int64
			want  int64
		}{
			{99, tb.ExpCap(), tb.ExpCap()},
			{99, tb.ExpCap() + 1_000_000, tb.ExpCap()},
			{98, tb.ExpCap() + 5_000_000_000, tb.ExpCap()},
			{1, 10, 10},
		} {
			got, _ := tb.ApplyExperience(tc.level, tc.exp)
			if got != tc.want {
				t.Errorf("%s clvl %d exp %d: got %d want %d", c, tc.level, tc.exp, got, tc.want)
			}
		}

		p := Progress{Level: 99, Experience: tb.ExpCap()}
		if n := tb.AddExperience(&p, 1<<40); n != 0 || p.Level != 99 || p.Experience != tb.ExpCap() {
			t.Errorf("%s: clvl 99 after huge gain: %+v levels %d", c, p, n)
		}
	}
}
