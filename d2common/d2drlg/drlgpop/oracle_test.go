package drlgpop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

type goldRoom struct {
	X     int      `json:"x"`
	Y     int      `json:"y"`
	W     int      `json:"w"`
	H     int      `json:"h"`
	PDef  int      `json:"pdef"`
	PFile int      `json:"pfile"`
	PM    int      `json:"pm"`
	PMX   int      `json:"pmx"`
	PMY   int      `json:"pmy"`
	Nodes [][6]int `json:"nodes"`
	Err   string   `json:"err"`
}

// goldCall is one DRLG_FilterPresetObjects call: the preset map, whether it
// ran while the level was generated (level seed) or lazily for a room (room
// seed), the seed before and after and the number of nodes it added.
type goldCall struct {
	PM    int       `json:"pm"`
	Stage string    `json:"stage"`
	Seed0 [2]uint32 `json:"seed0"`
	Seed1 [2]uint32 `json:"seed1"`
	N0    int       `json:"n0"`
	N1    int       `json:"n1"`
}

type goldLevel struct {
	Act   int        `json:"act"`
	Seed  uint32     `json:"seed"`
	Diff  int        `json:"diff"`
	Level int        `json:"level"`
	Rooms []goldRoom `json:"rooms"`
	Calls []goldCall `json:"calls"`
}

// TestOraclePresetUnits replays, for every preset room of the emulated game,
// what happens to the DS1 objects of its preset map: parse (monpreset / object
// id mapping), DRLG_FilterPresetObjects with the room seed of the first room
// that touches the map (RNG consumption compared), and the hand-over of the
// nodes inside the room rectangle (order and relative coordinates compared).
// Golden: numbers only, produced by gen_pop1.py from the real game code.
func TestOraclePresetUnits(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	ds1 := os.Getenv("D2_DS1_ROOT")

	if root == "" || ds1 == "" {
		t.Skip("D2_TABLES / D2_DS1_ROOT not set")
	}

	gold := loadPopGold(t)

	nm := realNames(t)

	rd := func(p string) []byte {
		b, err := os.ReadFile(filepath.Join(root, "drlg", p))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: rd("patch_d2/Levels.txt"), LvlPrest: rd("patch_d2/LvlPrest.txt"),
		LvlPrestBin: rd("bin/patch_d2/lvlprest.bin"), LvlSub: rd("patch_d2/LvlSub.txt"), LvlTypes: rd("patch_d2/LvlTypes.txt")})
	if err != nil {
		t.Fatal(err)
	}

	load := dirLoader(ds1)
	bad, nodes, calls := 0, 0, 0
	extras := map[[2]int]int{}

	for _, lv := range gold {
		pms := map[int][]Node{}
		dbg := map[int]string{}
		allNodes := map[int][]Node{}

		// every preset map is filtered once, either while the level is
		// generated or when its first room is built
		src := map[int]goldRoom{}
		for _, r := range lv.Rooms {
			if _, ok := src[r.PM]; !ok {
				src[r.PM] = r
			}
		}

		for _, c := range lv.Calls {
			r, ok := src[c.PM]
			if !ok {
				continue
			}

			name := fmt.Sprintf("act%d seed %#x diff %d level %d pm %d (%s)", lv.Act, lv.Seed, lv.Diff, lv.Level, c.PM, c.Stage)

			p, ok := tb.PrestByDef(r.PDef)
			if !ok || r.PFile < 0 || r.PFile >= len(p.File) {
				t.Fatalf("%s: no preset file for def %d file %d", name, r.PDef, r.PFile)
			}

			data, err := load(drlgoutdoor.NormalizePrestFile(p.File[r.PFile]))
			if err != nil {
				t.Skipf("%s: %v", name, err)
			}

			pat, err := drlgoutdoor.ParsePattern(data)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			raws := make([]Raw, len(pat.Objects))
			for i, o := range pat.Objects {
				raws[i] = Raw{o.Type, o.ID, o.X, o.Y, o.Flags}
			}

			all := ResolveDS1(pat.Version, pat.Act, raws, nm, d2records.DS1ObjectClass)
			dbg[c.PM] = fmt.Sprintf("%s ver %d act %d raws %v", p.File[r.PFile], pat.Version, pat.Act, raws)
			for i := range all {
				all[i].X += r.PMX * Subtile
				all[i].Y += r.PMY * Subtile
			}

			allNodes[c.PM] = all
			seed := &d2rand.Seed{Lo: c.Seed0[0], Hi: c.Seed0[1]}
			pms[c.PM] = Filter(all, nm, seed)
			calls++

			if got := [2]uint32{seed.Lo, seed.Hi}; got != c.Seed1 {
				bad++
				t.Errorf("%s: %s filter seed %x, want %x", name, p.File[r.PFile], got, c.Seed1)
			}

			if len(pms[c.PM]) != c.N1-c.N0 {
				bad++
				t.Errorf("%s: %s kept %d nodes, want %d", name, p.File[r.PFile], len(pms[c.PM]), c.N1-c.N0)
			}
		}

		for ri, r := range lv.Rooms {
			name := fmt.Sprintf("act%d seed %#x diff %d level %d room %d (%d,%d)", lv.Act, lv.Seed, lv.Diff, lv.Level, ri, r.X, r.Y)

			if r.Err != "" {
				continue // the emulated build of that room failed (not a Go problem)
			}

			room, rest := TakeForRoom(pms[r.PM], Rect{r.X, r.Y, r.W, r.H})
			pms[r.PM] = rest

			// The game's room list also holds units made by the tile build (level
			// warps, door objects, "navi" markers): the DS1 nodes must be an
			// in-order subsequence of it.
			got := make([][6]int, len(room))
			for i, n := range room {
				got[i] = [6]int{n.Class0, n.ID, n.X, n.Y, n.Kind, n.Flags}
			}

			j := 0

			for _, w := range r.Nodes {
				if j < len(got) && got[j] == w {
					j++
					nodes++

					continue
				}

				extras[[2]int{w[4], w[1]}]++

				// a unit the DS1 itself holds at that spot would be a lost node
				for _, a := range allNodes[r.PM] {
					if a.Kind == w[4] && a.ID == w[1] && a.X-r.X*Subtile == w[2] && a.Y-r.Y*Subtile == w[3] {
						bad++
						t.Errorf("%s: DS1 node %v missing from the Go list", name, w)
					}
				}
			}

			if j != len(got) {
				bad++
				t.Errorf("%s: DS1 nodes %v are not a subsequence of %v\n%s", name, got, r.Nodes, dbg[r.PM])
			}

			if bad > 30 {
				t.Fatal("too many mismatches")
			}
		}
	}

	t.Logf("%d levels, %d filter calls, %d DS1 nodes compared, %d mismatches; units of the tile build in the golden (kind,id)->count: %v", len(gold), calls, nodes, bad, extras)
}

// dirLoader resolves DS1 names under <root>/{patch_d2,d2exp,d2data}/data/global/tiles.
func dirLoader(root string) func(string) ([]byte, error) {
	return func(file string) ([]byte, error) {
		p := strings.ToLower(strings.ReplaceAll(file, "\\", "/"))

		for _, m := range []string{"patch_d2", "d2exp", "d2data"} {
			if b, err := os.ReadFile(filepath.Join(root, m, "data/global/tiles", p)); err == nil {
				return b, nil
			}
		}

		return nil, os.ErrNotExist
	}
}
