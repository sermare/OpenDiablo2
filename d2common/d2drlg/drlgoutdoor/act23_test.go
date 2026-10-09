package drlgoutdoor

import (
	"crypto/sha1" //nolint:gosec // matches the digest of the golden generator, not a security use
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
)

// The Act 2 / Act 3 goldens (testdata/outdoor_act2.json, outdoor_act3.json) are
// numbers and digests produced by diffing the Python reference port against the
// real Game.exe DRLG running in an emulator (drlg-act23-outdoor.md): 25 seeds x
// 3 difficulties x every outdoor level. The tests need the extracted tables
// (D2_TABLES); Act 2 additionally needs the DS1 files of the border matcher
// (D2_DS1_ROOT) and skips its level tests without them.

type g23Nb struct {
	Lvl, Dir, F8 int
	Rect         [4]int
}

func (n *g23Nb) UnmarshalJSON(b []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil || len(raw) != 4 {
		return fmt.Errorf("bad neighbour %s", b)
	}

	for i, dst := range []interface{}{&n.Lvl, &n.Dir, &n.F8, &n.Rect} {
		if err := json.Unmarshal(raw[i], dst); err != nil {
			return err
		}
	}

	return nil
}

type g23Grids struct {
	Flag [][]uint32 `json:"flag"`
	Def  [][]uint32 `json:"defg"`
	B    [][]uint32 `json:"B"`
}

type g23 struct {
	Act, Diff   int
	Seed        uint32
	Level       int
	Rect        [4]int
	Base        uint32
	Flip        int
	OdFlagsInit int
	Vis, Warp   []int
	Nb          []g23Nb
	Stages      [][]interface{}
	SeedFinal   [2]uint32
	DigF        string
	DigDef      string
	DigB        string
	Nrooms      int
	Nplain      int
	DigRooms    string
	DigABC      string
	Jungle      *struct {
		Arr   [12]int
		Count int
	}
	Grids *g23Grids
	Rooms [][]interface{}
}

func loadG23(t *testing.T, name string) []g23 {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Skip(err)
	}

	var g []g23
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}

	return g
}

func tables23(t *testing.T) *d2drlg.Tables {
	t.Helper()

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
		LvlPrestBin: rd("bin/patch_d2/lvlprest.bin"), LvlSub: rd("patch_d2/LvlSub.txt")})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

// dg is the golden generator's digest: first 16 hex digits of the sha1 of the
// compact JSON of the nested lists.
func dg(x interface{}) string {
	b, _ := json.Marshal(x)
	s := sha1.Sum(b) //nolint:gosec

	return hex.EncodeToString(s[:])[:16]
}

type world23Key struct {
	act, diff int
	seed      uint32
}

func worldFor(t *testing.T, tb *d2drlg.Tables, cache map[world23Key]*World23, g g23) *World23 {
	t.Helper()

	k := world23Key{g.Act, g.Diff, g.Seed}
	if w, ok := cache[k]; ok {
		return w
	}

	var (
		w   *World23
		err error
	)

	if g.Act == 1 {
		w, err = PlaceAct2World(tb, g.Seed, d2drlg.Difficulty(g.Diff))
	} else {
		w, err = PlaceAct3World(tb, g.Seed, d2drlg.Difficulty(g.Diff))
	}

	if err != nil {
		t.Fatalf("act %d seed %#x diff %d: %v", g.Act, g.Seed, g.Diff, err)
	}

	cache[k] = w

	return w
}

func checkParams23(p Params, g g23, bad func(string, ...interface{})) {
	if p.Rect != (Rect{g.Rect[0], g.Rect[1], g.Rect[2], g.Rect[3]}) {
		bad("rect", p.Rect, g.Rect)
	}

	if p.BaseSeed != g.Base {
		bad("base seed", p.BaseSeed, g.Base)
	}

	if g.Act == 2 && p.Flip != g.Flip {
		bad("flip", p.Flip, g.Flip)
	}

	if p.OdFlags != g.OdFlagsInit {
		bad("od.flags", p.OdFlags, g.OdFlagsInit)
	}

	if !reflect.DeepEqual(p.Vis[:], g.Vis) || !reflect.DeepEqual(p.Warp[:], g.Warp) {
		bad("vis/warp", p.Vis, p.Warp, g.Vis, g.Warp)
	}

	var nb []g23Nb

	for _, n := range p.Neighbors {
		nb = append(nb, g23Nb{n.Level, n.Dir, b2i(n.F8), [4]int{n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H}})
	}

	if !reflect.DeepEqual(nb, g.Nb) && !(len(nb) == 0 && len(g.Nb) == 0) {
		bad("neighbours", nb, g.Nb)
	}

	if g.Jungle != nil && (p.Jungle.Arr != g.Jungle.Arr || p.Jungle.Count != g.Jungle.Count) {
		bad("jungle array", p.Jungle, *g.Jungle)
	}
}

// TestAct23World diffs the world placers (rectangles, jungle arrays, vis links,
// neighbour lists, act extras) against the emulator.
func TestAct23World(t *testing.T) {
	tb := tables23(t)
	cache := map[world23Key]*World23{}
	n := 0

	for _, name := range []string{"outdoor_act2.json", "outdoor_act3.json"} {
		for _, g := range loadG23(t, name) {
			w := worldFor(t, tb, cache, g)

			p, err := w.Params(tb, g.Level)
			if err != nil {
				t.Fatal(err)
			}

			fails := 0
			checkParams23(p, g, func(what string, a ...interface{}) {
				if fails++; fails <= 2 {
					t.Errorf("act %d diff %d seed %#x level %d: %s %v", g.Act, g.Diff, g.Seed, g.Level, what, a)
				}
			})

			n++
		}
	}

	t.Logf("%d level parameter sets compared", n)
}

func ds1Loader23() DS1Loader {
	if root := os.Getenv("D2_DS1_ROOT"); root != "" {
		return DirLoader(root)
	}

	return nil
}

func roomKey(r *Room) []interface{} {
	if r.Type == 1 {
		return []interface{}{1, r.X, r.Y, r.W, r.H, r.S4, r.Flags, r.R50, r.Info54, r.Info58, r.SubType, r.SubTheme, r.Mask,
			[]uint32{r.Seed.Lo, r.Seed.Hi}}
	}

	return []interface{}{2, r.X, r.Y, r.W, r.H, r.S4, r.R50, r.PrestDef, r.File}
}

func compareLevel23(t *testing.T, lv *Level, g g23, bad func(string, ...interface{})) {
	t.Helper()

	// stage by stage: level seed and od.flags at the entry of every stage
	if len(lv.Trace) != len(g.Stages) {
		bad("stage count", len(lv.Trace), len(g.Stages))
	} else {
		for i, s := range lv.Trace {
			w := g.Stages[i]
			if s.Name != w[0].(string) || float64(s.Lo) != w[1].(float64) || float64(s.Hi) != w[2].(float64) || float64(s.OdFlags) != w[3].(float64) {
				bad("stage", i, s, w)
				break
			}
		}
	}

	if got := dg(lv.Flag.Rows()); got != g.DigF {
		bad("flag grid", got, g.DigF)
	}

	if got := dg(lv.Def.Rows()); got != g.DigDef {
		bad("def grid", got, g.DigDef)
	}

	if got := dg(lv.GridB.Rows()); got != g.DigB {
		bad("B grid", got, g.DigB)
	}

	if len(lv.Rooms) != g.Nrooms {
		bad("room count", len(lv.Rooms), g.Nrooms)
	}

	keys := make([][]interface{}, 0, len(lv.Rooms))
	for _, r := range lv.Rooms {
		keys = append(keys, roomKey(r))
	}

	if got := dg(keys); got != g.DigRooms {
		bad("room list", got, g.DigRooms)
	}

	if [2]uint32{lv.Seed.Lo, lv.Seed.Hi} != g.SeedFinal {
		bad("final level seed", lv.Seed, g.SeedFinal)
	}

	if g.Grids != nil {
		for name, pair := range map[string][2][][]uint32{"flag": {lv.Flag.Rows(), g.Grids.Flag}, "def": {lv.Def.Rows(), g.Grids.Def}, "B": {lv.GridB.Rows(), g.Grids.B}} {
			if !reflect.DeepEqual(pair[0], pair[1]) {
				bad("full grid "+name, name)
			}
		}

		for i, wr := range g.Rooms {
			if !reflect.DeepEqual(roundTrip(keys[i]), wr) {
				bad("full room", i, keys[i], wr)
				break
			}
		}
	}

	// plain-room tile grids A/B/C (needs the DS1 files for sub-themes)
	abc := []map[string][][]uint32{}

	nPlain := 0

	for _, r := range lv.Rooms {
		if r.Type != 1 {
			continue
		}

		nPlain++

		gr, err := lv.BuildRoomGrids(r, nil)
		if err != nil {
			nPlain = -1

			break
		}

		abc = append(abc, map[string][][]uint32{"A": gr.A.Rows(), "B": gr.B.Rows(), "C": gr.C.Rows()})
	}

	switch {
	case nPlain < 0:
		// cannot build without the DS1 library
	case nPlain != g.Nplain:
		bad("plain room count", nPlain, g.Nplain)
	case dg(abc) != g.DigABC:
		bad("room A/B/C grids", dg(abc), g.DigABC)
	}
}

func roundTrip(x interface{}) interface{} {
	b, _ := json.Marshal(x)

	var out interface{}

	_ = json.Unmarshal(b, &out)

	return out
}

func testLevels23(t *testing.T, file string, needDS1 bool) {
	tb := tables23(t)
	ds1 := ds1Loader23()

	if needDS1 && ds1 == nil {
		t.Skip("D2_DS1_ROOT not set (the Act 2 border matcher loads DS1 files)")
	}

	env := NewEnv(tb, ds1)
	cache := map[world23Key]*World23{}
	counts := map[string]int{}
	n := 0

	for _, g := range loadG23(t, file) {
		w := worldFor(t, tb, cache, g)

		p, err := w.Params(tb, g.Level)
		if err != nil {
			t.Fatal(err)
		}

		lv, err := Generate(env, p)
		if err != nil {
			t.Errorf("act %d diff %d seed %#x level %d: %v", g.Act, g.Diff, g.Seed, g.Level, err)
			continue
		}

		n++

		compareLevel23(t, lv, g, func(what string, a ...interface{}) {
			if counts[what]++; counts[what] <= 3 {
				t.Errorf("act %d diff %d seed %#x level %d: %s %v", g.Act, g.Diff, g.Seed, g.Level, what, a)
			}
		})
	}

	t.Logf("%d levels compared; mismatch counts %v", n, counts)
}

func TestOracleAct2Levels(t *testing.T) { testLevels23(t, "outdoor_act2.json", true) }
func TestOracleAct3Levels(t *testing.T) { testLevels23(t, "outdoor_act3.json", false) }

func TestRectsAdjacent(t *testing.T) {
	for _, c := range []struct {
		a, b Rect
		want bool
	}{
		{Rect{0, 0, 80, 80}, Rect{80, 0, 80, 80}, true},   // flush east
		{Rect{0, 0, 80, 80}, Rect{80, 40, 80, 80}, true},  // staggered
		{Rect{0, 0, 80, 80}, Rect{80, 80, 80, 80}, false}, // corner contact only
		{Rect{0, 0, 80, 80}, Rect{81, 0, 80, 80}, false},  // gap
		{Rect{0, 80, 80, 80}, Rect{20, 0, 40, 80}, true},  // north, narrower
	} {
		if got := rectsAdjacent(c.a, c.b); got != c.want {
			t.Errorf("rectsAdjacent(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestJungleChildRect(t *testing.T) {
	par := &jungleNode{r: Rect{1000, 808, 64, 192}}

	for d, want := range [5]Rect{{1000, 616, 64, 192}, {936, 744, 64, 192}, {1064, 744, 64, 192}, {936, 680, 64, 192}, {1064, 680, 64, 192}} {
		if got := jungleChild(par, d, 64, 192).r; got != want {
			t.Errorf("dir %d: %v, want %v", d, got, want)
		}
	}
}

func TestDS1GateNames(t *testing.T) {
	for in, want := range map[string]int{
		`data\global\tiles\Act3\Kurast\Burbs08x08_1.ds1`: 2,
		`Act3\Kurast\Slums16x16_0.ds1`:                   1,
		`Act3\Kurast\Slums08x08_0.ds1`:                   0,
		`Act2\Outdoors\TombEnt1.ds1`:                     0,
		``:                                               0,
	} {
		if got := ds1GateSteps(in); got != want {
			t.Errorf("ds1GateSteps(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestTowns(t *testing.T) {
	tb := tables23(t)
	env := NewEnv(tb, nil)

	for _, c := range []struct {
		id, rooms int
		world     func() (*World23, error)
	}{
		{40, 49, func() (*World23, error) { return PlaceAct2World(tb, 270358346, d2drlg.Normal) }},
		{75, 48, func() (*World23, error) { return PlaceAct3World(tb, 270358346, d2drlg.Normal) }},
	} {
		w, err := c.world()
		if err != nil {
			t.Fatal(err)
		}

		p, err := w.Params(tb, c.id)
		if err != nil {
			t.Fatal(err)
		}

		lv, err := GenerateTown(env, p, w.TownFile)
		if err != nil {
			t.Fatal(err)
		}

		if len(lv.Rooms) != c.rooms {
			t.Errorf("town %d: %d rooms, want %d", c.id, len(lv.Rooms), c.rooms)
		}

		if !strings.HasPrefix(fmt.Sprint(lv.Rooms[0].X), "1000") {
			t.Errorf("town %d: first room at %d, want the level origin", c.id, lv.Rooms[0].X)
		}
	}
}
