package d2monsters

import (
	"path/filepath"
	"testing"
)

// every treasure class a monster (or super unique) names for Normal,
// Nightmare or Hell exists in treasureclassex.txt, so a drop in a higher
// difficulty never silently falls back to nothing. Skipped without D2_TABLES.
func TestRealTreasureClassesExist(t *testing.T) {
	root, _ := loadReal(t)
	dir := filepath.Join(root, "monsters", "patch_d2")
	tc := readRaw(t, filepath.Join(dir, "treasureclassex.txt"))
	ms := readRaw(t, filepath.Join(dir, "monstats.txt"))
	su := readRaw(t, filepath.Join(dir, "superuniques.txt"))

	have := map[string]bool{}
	for _, r := range tc.rows {
		have[tc.s(r, "Treasure Class")] = true
	}

	for _, r := range ms.rows {
		if ms.s(r, "Id") == "Expansion" {
			continue
		}

		for _, c := range []string{"TreasureClass1", "TreasureClass2", "TreasureClass3", "TreasureClass4"} {
			for _, s := range []string{"", "(N)", "(H)"} {
				if n := ms.s(r, c+s); n != "" && !have[n] {
					t.Errorf("%s %s%s: treasure class %q is not in treasureclassex.txt", ms.s(r, "Id"), c, s, n)
				}
			}
		}
	}

	for _, r := range su.rows {
		for _, c := range []string{"TC", "TC(N)", "TC(H)"} {
			if n := su.s(r, c); n != "" && !have[n] {
				t.Errorf("super unique %s %s: treasure class %q is not in treasureclassex.txt", su.s(r, "Superunique"), c, n)
			}
		}
	}
}
