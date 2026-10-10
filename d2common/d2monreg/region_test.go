package d2monreg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// realTables loads the three text tables from D2_TABLES (monsters/patch_d2).
func realTables(t *testing.T) *Tables {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rd := func(name string) []byte {
		b, err := os.ReadFile(filepath.Join(root, "monsters", "patch_d2", name))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	tb, err := ParseTables(rd("monstats.txt"), rd("monstats2.txt"), rd("levels.txt"))
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

type regionGold struct {
	Seed      uint32            `json:"seed"`
	Diff      int               `json:"diff"`
	Exp       int               `json:"exp"`
	SpawnSeed uint32            `json:"spawnSeed"`
	SeedAfter [2]uint32         `json:"seedAfter"`
	Dig       map[string]string `json:"dig"`
}

// canon is the text the golden generator digests per level.
func canon(r *Region) string {
	s := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d", r.Level, r.Act, len(r.Types), r.TotalRarity, r.N2, r.Density, r.UMin, r.UMax,
		r.Wndr, r.Quest, r.MonLvl, r.MonLvl)

	for _, e := range r.Types {
		s += fmt.Sprintf("|c%d,r%d,v%d", e.Class, e.Rarity, len(e.Variants))

		for _, v := range e.Variants {
			s += "," + hex.EncodeToString(v[:])
		}
	}

	return s
}

// TestOracleRegions compares BuildRegions with the real MONREGION_InitForGame
// (unicorn): all levels of 8 game seeds x 3 difficulties x 2 modes.
func TestOracleRegions(t *testing.T) {
	tb := realTables(t)

	gp := os.Getenv("ORACLE_REGIONS")
	if gp == "" {
		gp = filepath.Join("testdata", "regions.json")
	}

	raw, err := os.ReadFile(gp)
	if err != nil {
		t.Skip(err)
	}

	var gold []regionGold
	if err := json.Unmarshal(raw, &gold); err != nil {
		t.Fatal(err)
	}

	bad, levels := 0, 0

	for _, g := range gold {
		seed := d2rand.New(g.Seed)
		regs, spawn := BuildRegions(tb, seed, g.Diff, g.Exp != 0)

		if spawn != g.SpawnSeed || [2]uint32{seed.Lo, seed.Hi} != g.SeedAfter {
			bad++
			t.Errorf("seed %#x diff %d exp %d: region seed %#x / game seed %v, want %#x / %v", g.Seed, g.Diff, g.Exp, spawn, [2]uint32{seed.Lo, seed.Hi}, g.SpawnSeed, g.SeedAfter)
		}

		for lid := 1; lid < len(regs); lid++ {
			want, ok := g.Dig[fmt.Sprint(lid)]
			if regs[lid] == nil {
				if ok {
					bad++
					t.Errorf("seed %#x level %d: no region", g.Seed, lid)
				}

				continue
			}

			levels++
			sum := sha256.Sum256([]byte(canon(regs[lid])))

			if got := hex.EncodeToString(sum[:])[:12]; got != want {
				bad++

				if bad < 20 {
					t.Errorf("seed %#x diff %d exp %d level %d: %s (%s), want digest %s", g.Seed, g.Diff, g.Exp, lid, got, canon(regs[lid]), want)
				}
			}
		}
	}

	t.Logf("%d regions of %d games compared, %d mismatches", levels, len(gold), bad)
}
