package d2gamescreen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2herostats"
)

func TestLevelsGained(t *testing.T) {
	// experience needed to leave level 1, 2, 3 (Experience.txt: 500, 1500, 3750)
	bp := func(l int) int { return map[int]int{1: 500, 2: 1500, 3: 3750, 4: 7875}[l] }

	tests := []struct {
		name            string
		exp, level, max int
		want            int
	}{
		{"below the first breakpoint", 330, 1, 99, 0},
		{"exactly the breakpoint", 500, 1, 99, 1},
		{"606 experience at level 1", 606, 1, 99, 1},
		{"a big kill skips levels", 4000, 1, 99, 3},
		{"already there", 1000, 2, 99, 0},
		{"capped at the maximum level", 99999, 1, 3, 2},
		{"no breakpoint known", 99999, 9, 99, 0},
		{"level 1 with no experience", 0, 1, 99, 0},
	}

	for _, tc := range tests {
		if got := levelsGained(tc.exp, tc.level, tc.max, bp); got != tc.want {
			t.Errorf("%s: got %d, want %d", tc.name, got, tc.want)
		}
	}
}

// The d2herostats table (built the way advanceHeroLevel builds it) must agree
// with levelsGained, and cap experience at row MaxLvl-1.
func TestHeroExpTableMatchesLevelsGained(t *testing.T) {
	rows := []int64{0, 500, 1500, 3750, 7875, 14175}
	tbl := d2herostats.NewExpTable(5, func(l int) int64 { return rows[l] })
	bp := func(l int) int { return int(rows[l]) }

	for exp := int64(0); exp < 7875; exp += 37 {
		for lvl := 1; lvl <= 5; lvl++ {
			if lvl > tbl.LevelFor(exp) {
				continue
			}

			_, n := tbl.ApplyExperience(lvl, exp)
			if want := levelsGained(int(exp), lvl, 5, bp); n != want {
				t.Fatalf("exp %d lvl %d: table %d vs levelsGained %d", exp, lvl, n, want)
			}
		}
	}

	if got, _ := tbl.ApplyExperience(1, 1<<40); got != 7875 {
		t.Errorf("cap = %d, want row MaxLvl-1 = 7875", got)
	}
}

// Level 94 save oracle: experience 2411280845 is level 94 in Experience.txt.
func TestLevel94Oracle(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(dir, "Experience.txt"))
	if err != nil {
		t.Skip(err)
	}

	tabs, err := d2herostats.ParseExperience(data)
	if err != nil {
		t.Fatal(err)
	}

	tbl := tabs["Sorceress"]
	if lv := tbl.LevelFor(2411280845); lv != 94 {
		t.Errorf("level = %d, want 94", lv)
	}

	if _, n := tbl.ApplyExperience(93, 2411280845); n != 1 {
		t.Errorf("93 -> %d levels, want 1", n)
	}

	if got, _ := tbl.ApplyExperience(1, 1<<40); got != 3520485254 {
		t.Errorf("cap = %d, want 3520485254", got)
	}
}
