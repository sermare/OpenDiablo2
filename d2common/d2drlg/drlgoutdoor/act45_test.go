package drlgoutdoor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
)

// goldAct45 is one record of testdata/outdoor_act45.json: numbers produced by
// the real Game.exe DRLG running in an emulator (12 seeds x 3 difficulties x
// 16 levels of Acts 4 and 5, drlg-act45-outdoor.md). Grids, room lists and
// the final level seed are compared through digests (first 16 hex chars of the
// sha256 of the compact JSON of the Python structures).
type goldAct45 struct {
	Seed    uint32            `json:"seed"`
	Diff    int               `json:"diff"`
	Level   int               `json:"level"`
	Base    uint32            `json:"base"`
	Rect    [4]int            `json:"rect"`
	Vis     [8]int            `json:"vis"`
	Flags0  *int              `json:"flags0"`
	Nbrs    [][3]int          `json:"nbrs"`
	File    *int              `json:"file"`
	SeedLo  uint32            `json:"seedLo"`
	SeedHi  uint32            `json:"seedHi"`
	NRooms  int               `json:"nrooms"`
	Rooms   string            `json:"rooms"`
	GridsIn map[string]string `json:"grids"`
}

func jsonDigest(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	s := sha256.Sum256(b)

	return hex.EncodeToString(s[:])[:16]
}

func rowsOf(g *Grid) [][]uint32 { return g.Rows() }

// roomTuple is the emulator's room tuple: (x, y, w, h, type, s4, seed lo, mask,
// def, file, flags, Dt1Mask).
func roomTuple(r *Room) []interface{} {
	t := []interface{}{r.X, r.Y, r.W, r.H, r.Type, r.S4, r.Seed.Lo, nil, nil, nil, r.Flags, r.R50}
	if r.Type == 1 {
		t[7] = r.Mask
	} else {
		t[8], t[9] = r.PrestDef, r.File
	}

	return t
}

func roomsDigest(rs []*Room) string {
	out := make([][]interface{}, len(rs))
	for i, r := range rs {
		out[i] = roomTuple(r)
	}

	return jsonDigest(out)
}

func testEnv45(t *testing.T) *Env {
	return testEnv(t)
}

func loadGold45(t *testing.T) []goldAct45 {
	t.Helper()

	p := os.Getenv("ORACLE_OUTDOOR45")
	if p == "" {
		p = filepath.Join("..", "testdata", "outdoor_act45.json")
	}

	b, err := os.ReadFile(p)
	if err != nil {
		t.Skip(err)
	}

	var g []goldAct45
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}

	return g
}

func TestOracleAct45(t *testing.T) {
	env := testEnv45(t)
	gold := loadGold45(t)

	type key struct {
		seed uint32
		diff int
		act  int
	}

	layouts := map[key]*drlgworld.Layout{}
	counts := map[string]int{}
	checked := map[int]int{}

	for _, g := range gold {
		act := 3
		if g.Level >= 109 {
			act = 4
		}

		k := key{g.Seed, g.Diff, act}

		lay := layouts[k]
		if lay == nil {
			var err error
			if act == 3 {
				lay, err = drlgworld.GenerateAct4(env.Tables, g.Seed, d2drlg.Difficulty(g.Diff))
			} else {
				lay, err = drlgworld.GenerateAct5(env.Tables, g.Seed, d2drlg.Difficulty(g.Diff))
			}

			if err != nil {
				t.Fatal(err)
			}

			layouts[k] = lay
		}

		name := fmt.Sprintf("seed %#x diff %d level %d", g.Seed, g.Diff, g.Level)
		bad := func(what string, a ...interface{}) {
			counts[what]++
			if counts[what] <= 3 {
				t.Errorf("%s: %s %v", name, what, a)
			}
		}

		p, err := ParamsFromLayout45(env.Tables, lay, act, g.Level, g.Seed, d2drlg.Difficulty(g.Diff))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if p.BaseSeed != g.Base {
			bad("base", p.BaseSeed, g.Base)
		}

		if p.Rect != (Rect{g.Rect[0], g.Rect[1], g.Rect[2], g.Rect[3]}) {
			bad("rect", p.Rect, g.Rect)
		}

		if p.Vis != g.Vis {
			bad("vis", p.Vis, g.Vis)
		}

		var seedLo, seedHi uint32

		var rooms []*Room

		if g.Flags0 != nil { // outdoor level
			if p.OdFlags != *g.Flags0 {
				bad("od.flags", p.OdFlags, *g.Flags0)
			}

			var nb [][3]int

			for _, n := range p.Neighbors {
				f8 := 0
				if n.F8 {
					f8 = 1
				}

				nb = append(nb, [3]int{n.Level, n.Dir, f8})
			}

			if fmt.Sprint(nb) != fmt.Sprint(g.Nbrs) {
				bad("neighbours", nb, g.Nbrs)
			}

			lv, err := Generate(env, p)
			if err != nil {
				bad("generate error", err)
				continue
			}

			for gk, grid := range map[string]*Grid{"def": lv.Def, "B": lv.GridB, "flag": lv.Flag, "D": lv.GridD} {
				if d := jsonDigest(rowsOf(grid)); d != g.GridsIn[gk] {
					bad("grid "+gk, d, g.GridsIn[gk])
				}
			}

			seedLo, seedHi, rooms = lv.Seed.Lo, lv.Seed.Hi, lv.Rooms
		} else {
			pl, err := GeneratePreset(env, p, -1)
			if err != nil {
				bad("generate error", err)
				continue
			}

			if g.File != nil && pl.File != *g.File {
				bad("file index", pl.File, *g.File)
			}

			seedLo, seedHi, rooms = pl.Seed.Lo, pl.Seed.Hi, pl.Rooms
		}

		checked[g.Level]++

		if dir := os.Getenv("ORACLE45_DUMP"); dir != "" && g.Seed == 0x101D574A && g.Diff == 0 {
			out := make([][]interface{}, len(rooms))
			for i, r := range rooms {
				out[i] = roomTuple(r)
			}

			b, _ := json.Marshal(out)
			_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("go_%d.json", g.Level)), b, 0o644)
		}

		if len(rooms) != g.NRooms {
			bad("room count", len(rooms), g.NRooms)
		}

		if d := roomsDigest(rooms); d != g.Rooms {
			bad("rooms", d, g.Rooms)
			counts[fmt.Sprint("rooms level ", g.Level)]++
		}

		if seedLo != g.SeedLo || seedHi != g.SeedHi {
			bad("final seed", seedLo, seedHi, g.SeedLo, g.SeedHi)
		}
	}

	t.Logf("levels compared %v; mismatches %v", checked, counts)
}
