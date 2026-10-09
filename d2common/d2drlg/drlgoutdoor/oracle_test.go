package drlgoutdoor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type oraclePM struct {
	Def  int `json:"def_"`
	File int `json:"file"`
	N    int `json:"n"`
}

type oracleOutdoor struct {
	Seed   uint32     `json:"seed"`
	Level  int        `json:"level"`
	Flags  int        `json:"odflags"`
	SeedLo uint32     `json:"seedLo"`
	SeedHi uint32     `json:"seedHi"`
	PMs    []oraclePM `json:"pms"`
}

// TestOracleOutdoor measures how far the (explicitly incomplete) Act 1
// outdoor skeleton is from the real game. It does not fail on differences:
// the port leaves the LvlSub/river/boundary stages unimplemented, so the final
// seed and the full preset list are expected to differ. It logs, per Def the
// port places, how often the placement count equals the real game's.
func TestOracleOutdoor(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rd := func(p string) []byte {
		b, err := os.ReadFile(filepath.Join(root, "drlg", p))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: rd("patch_d2/Levels.txt"), LvlPrest: rd("patch_d2/LvlPrest.txt"),
		LvlPrestBin: rd("bin/patch_d2/lvlprest.bin")})
	if err != nil {
		t.Fatal(err)
	}

	gp := os.Getenv("ORACLE_OUTDOOR")
	if gp == "" {
		gp = filepath.Join("..", "testdata", "outdoor_act1.json")
	}

	b, err := os.ReadFile(gp)
	if err != nil {
		t.Skip(err)
	}

	var gold []oracleOutdoor
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	type key struct{ level, def int }
	eq, ne := map[key]int{}, map[key]int{}
	seedOK, flagsOK, n := 0, 0, 0

	for _, g := range gold {
		w, err := drlgworld.Generate(tb, g.Seed, d2drlg.Normal)
		if err != nil {
			t.Fatal(err)
		}

		pl, ok := w.Levels[g.Level]
		if !ok {
			continue
		}

		base, _ := d2rand.DrlgBaseSeed(g.Seed)
		town := w.Levels[1].Rect
		res, err := GenerateAct1(tb, Params{LevelID: g.Level, BaseSeed: base, Rect: Rect(pl.Rect), OdFlags: pl.Flags, Town: Rect(town)})
		if err != nil {
			t.Logf("seed %#x level %d: %v", g.Seed, g.Level, err)
			continue
		}

		n++

		if pl.Flags == g.Flags {
			flagsOK++
		}

		if res.SeedAfter.Lo == g.SeedLo && res.SeedAfter.Hi == g.SeedHi {
			seedOK++
		}

		want := map[int]int{}
		for _, p := range g.PMs {
			want[p.Def] += p.N
		}

		got := map[int]int{}
		for _, p := range res.Placed {
			got[p.Def]++
		}

		for def, c := range got {
			if want[def] == c {
				eq[key{g.Level, def}]++
			} else {
				ne[key{g.Level, def}]++
			}
		}
	}

	t.Logf("records=%d worldFlagsEqualOracleOdFlags=%d finalSeedEqual=%d", n, flagsOK, seedOK)

	keys := []key{}
	seen := map[key]bool{}

	for k := range eq {
		seen[k] = true
	}

	for k := range ne {
		seen[k] = true
	}

	for k := range seen {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].level != keys[j].level {
			return keys[i].level < keys[j].level
		}

		return keys[i].def < keys[j].def
	})

	for _, k := range keys {
		t.Log(fmt.Sprintf("level %2d def %#04x: placed-count equals oracle in %d, differs in %d", k.level, k.def, eq[k], ne[k]))
	}
}
