package drlgpop

import (
	"os"
	"path/filepath"
	"testing"
)

// realNames loads the monster name tables from D2_TABLES (monsters/patch_d2
// and itemgen/patch_d2); the test is skipped without them.
func realNames(t *testing.T) *Names {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rd := func(p ...string) []byte {
		b, err := os.ReadFile(filepath.Join(append([]string{root}, p...)...))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	n, err := ParseNames(rd("monsters", "patch_d2", "monstats.txt"), rd("itemgen", "patch_d2", "SuperUniques.txt"),
		rd("monsters", "patch_d2", "monplace.txt"), rd("monsters", "patch_d2", "monpreset.txt"), 0)
	if err != nil {
		t.Fatal(err)
	}

	return n
}

// The counts below are what the emulated game reads from the compiled
// tables: 734 monstats rows, 66 superuniques, 37 monplace codes and 47/59/39/
// 28/56 monpreset rows per act (the loader at 0x65bc50 builds the per-act
// index at 0x964730).
func TestNamesCounts(t *testing.T) {
	n := realNames(t)

	if n.MonstatsCount != 734 || n.SuperCount != 66 || len(n.Place) != 37 {
		t.Fatalf("counts monstats=%d super=%d place=%d", n.MonstatsCount, n.SuperCount, len(n.Place))
	}

	for act, want := range []int{47, 59, 39, 28, 56} {
		if got := len(n.Presets[act]); got != want {
			t.Errorf("act %d: %d monpreset rows, want %d", act, got, want)
		}
	}

	// gheed = monstats 147, Bishibosh = superunique 0, place_fallen = place 17
	if e := n.Presets[0][0]; e != (PresetEntry{CatMonster, 147}) {
		t.Errorf("act 1 row 0 = %+v", e)
	}

	if e := n.Presets[0][34]; e != (PresetEntry{CatSuper, 0}) {
		t.Errorf("act 1 row 34 (Bishibosh) = %+v", e)
	}

	if e := n.Presets[0][10]; e != (PresetEntry{CatPlace, 17}) {
		t.Errorf("act 1 row 10 (place_fallen) = %+v", e)
	}
}
