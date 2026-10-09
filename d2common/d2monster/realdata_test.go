package d2monster

import (
	"os"
	"path/filepath"
	"testing"
)

// Reads the real 1.14b monstats.txt from $D2_TABLES/monsters/patch_d2 and
// checks the values quoted in monster-ai-2.md. Skipped when unset.
func TestRealMonstats(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", "monstats.txt"))
	if err != nil {
		t.Skip(err)
	}

	tp, err := LoadTxtProfiles(buf)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		id   string
		ai   string
		diff Difficulty
		aip  []int
	}{
		{"skeleton1", "Skeleton", Normal, []int{60, 15, 75, 75}},
		{"skeleton1", "Skeleton", Nightmare, []int{65, 12, 80, 75}},
		{"skeleton1", "Skeleton", Hell, []int{70, 10, 85, 75}},
		{"fallen1", "Fallen", Normal, []int{30, 10, 50, 20}},
		{"sk_archer1", "SkeletonBow", Normal, []int{75, 15, 50, 5, 6}},
		{"cr_archer1", "CorruptArcher", Normal, []int{60, 70, 14, 20, 20, 0, 0, 12}},
	}

	for _, c := range cases {
		class, ok := tp.ByID(c.id)
		if !ok {
			t.Errorf("%s not found", c.id)
			continue
		}

		p, _ := tp.Profile(class, c.diff)
		if p.AI != c.ai {
			t.Errorf("%s AI=%q want %q", c.id, p.AI, c.ai)
		}

		for i, want := range c.aip {
			if got := p.AIP[i+1]; got != want {
				t.Errorf("%s diff %d aip%d=%d want %d", c.id, c.diff, i+1, got, want)
			}
		}

		if _, ok := Lookup(p.AI); !ok {
			t.Errorf("%s: AI %s not implemented", c.id, p.AI)
		}
	}
}
