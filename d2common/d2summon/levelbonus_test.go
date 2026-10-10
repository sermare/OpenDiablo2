package d2summon

import (
	"os"
	"path/filepath"
	"testing"
)

const monlvlFixture = "Level\tAC\tAC(N)\tAC(H)\tL-AC\tL-AC(N)\tL-AC(H)\tTH\tTH(N)\tTH(H)\tL-TH\tL-TH(N)\tL-TH(H)\n" +
	"0\t1\t1\t1\t1\t1\t1\t1\t1\t1\t1\t1\t1\n" +
	"1\t6\t92\t147\t6\t108\t173\t8\t108\t216\t8\t108\t216\n" +
	"2\t12\t99\t158\t12\t118\t189\t16\t110\t218\t16\t112\t222\n"

func TestLevelBonus(t *testing.T) {
	tp := loadFixture(t)

	if _, _, ok := tp.LevelBonus(1, Normal); ok {
		t.Fatal("bonus without monlvl")
	}

	if err := tp.LoadMonLvl([]byte(monlvlFixture)); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		level  int
		diff   Difficulty
		ac, th int
	}{
		{1, Normal, 6, 8},
		{1, Nightmare, 108, 108},
		{2, Hell, 189, 222},  // the expansion (L-) columns
		{99, Normal, 12, 16}, // above the table: the last row
		{-3, Hell, 1, 1},
	}

	for _, tc := range tests {
		ac, th, ok := tp.LevelBonus(tc.level, tc.diff)
		if !ok || ac != tc.ac || th != tc.th {
			t.Errorf("level %d diff %d: ac=%d th=%d ok=%v, want %d/%d", tc.level, tc.diff, ac, th, ok, tc.ac, tc.th)
		}
	}
}

func TestComputeAddsLevelBonus(t *testing.T) {
	tp := loadFixture(t)
	sk := must(t, tp, "necroskeleton")

	base := Compute(sk, Normal, Mods{}, nil)
	got := Compute(sk, Normal, Mods{LevelAC: 6, LevelAR: 8}, nil)

	if got.Defense != base.Defense+6 || got.AR != base.AR+8 {
		t.Errorf("ac %d->%d ar %d->%d", base.Defense, got.Defense, base.AR, got.AR)
	}
}

// TestRealMonLvl checks the loader against the real patch_d2 monlvl.txt.
func TestRealMonLvl(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	data, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", "monlvl.txt"))
	if err != nil {
		t.Skip(err)
	}

	tp := loadFixture(t)
	if err := tp.LoadMonLvl(data); err != nil {
		t.Fatal(err)
	}

	ac, th, ok := tp.LevelBonus(1, Nightmare)
	if !ok || ac != 108 || th != 108 {
		t.Errorf("level 1 nightmare: %d/%d", ac, th)
	}
}
