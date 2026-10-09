package drlgoutdoor

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
)

// The goldens are numbers produced by running the real Game.exe DRLG in an
// emulator (see ~/git/d2-re-notes/drlg3.md). The test needs the extracted
// tables (D2_TABLES) and the DS1 pattern files (D2_DS1_ROOT, a folder holding
// patch_d2/d2exp/d2data with data/global/tiles inside); it skips without them.

type goldNeighbor struct {
	Lvl  int    `json:"lvl"`
	Dir  int    `json:"dir"`
	F8   int    `json:"f8"`
	Fc   int    `json:"fc"`
	Rect [4]int `json:"rect"`
}

type goldStage []interface{}

type goldRiver struct {
	N     int        `json:"n"`
	Ends  [][3]int   `json:"ends"`
	Start [][2]int   `json:"start"`
	End   [][2]int   `json:"end"`
	Junc  [][3]int   `json:"junc"`
	Paths [][][2]int `json:"paths"`
}

type goldRoom struct {
	Type     int                   `json:"type"`
	X        int                   `json:"x"`
	Y        int                   `json:"y"`
	W        int                   `json:"w"`
	H        int                   `json:"h"`
	S4       uint32                `json:"s4"`
	Flags    uint32                `json:"flags"`
	R50      uint32                `json:"r50"`
	Info54   uint32                `json:"info54"`
	Info58   uint32                `json:"info58"`
	SubType  int                   `json:"subtype"`
	SubTheme int                   `json:"subtheme"`
	Mask     uint32                `json:"mask"`
	SeedMask [2]uint32             `json:"seedAfterMask"`
	PrestDef int                   `json:"prestDef"`
	File     int                   `json:"file"`
	ABC      map[string][][]uint32 `json:"gridsABC"`
	SeedBld  [2]uint32             `json:"seedAfterBuild"`
}

type goldFull struct {
	Grids map[string][][]uint32 `json:"grids"`
	River goldRiver             `json:"river"`
	Rooms []goldRoom            `json:"rooms"`
}

type goldLevel struct {
	Rect         [4]int            `json:"rect"`
	LevelType    int               `json:"levelType"`
	OdFlagsInit  int               `json:"odFlagsInit"`
	Vis          [][]int           `json:"vis"`
	Neighbors    []goldNeighbor    `json:"neighbors"`
	Polygon      [][4]int          `json:"polygonAtAct"`
	SeedStages   [2]uint32         `json:"seedAfterStages"`
	OdFlagsAfter int               `json:"odFlagsAfterStages"`
	SeedRooms    [2]uint32         `json:"seedAfterRooms"`
	Counters     map[string][2]int `json:"counters"`
	Stages       []goldStage       `json:"stages"`
	GridDigest   map[string]string `json:"gridDigest"`
	RiverDigest  string            `json:"riverDigest"`
	RiverN       int               `json:"riverN"`
	NRooms       int               `json:"nRooms"`
	RoomsDigest  string            `json:"roomsDigest"`
	NPlain       int               `json:"nPlainRooms"`
	RoomGridsDig string            `json:"roomGridsDigest"`
	Full         *goldFull         `json:"full"`
}

type goldSeed struct {
	Seed   uint32               `json:"seed"`
	Base   uint32               `json:"base"`
	Levels map[string]goldLevel `json:"levels"`
}

func testEnv(t *testing.T) *Env {
	t.Helper()

	root := os.Getenv("D2_TABLES")
	ds1 := os.Getenv("D2_DS1_ROOT")

	if root == "" || ds1 == "" {
		t.Skip("D2_TABLES / D2_DS1_ROOT not set")
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

	return NewEnv(tb, DirLoader(ds1))
}

// DirLoader resolves DS1 names under <root>/{patch_d2,d2exp,d2data}/data/global/tiles.
func DirLoader(root string) DS1Loader {
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

func loadGolden(t *testing.T) []goldSeed {
	t.Helper()

	gp := os.Getenv("ORACLE_OUTDOOR")
	if gp == "" {
		gp = filepath.Join("..", "testdata", "outdoor_act1.json")
	}

	b, err := os.ReadFile(gp)
	if err != nil {
		t.Skip(err)
	}

	var g []goldSeed
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}

	return g
}

// digests: sha256 (first 8 bytes, hex) over a little-endian uint32 stream.
type digester struct{ buf []byte }

func (d *digester) u32(vs ...uint32) {
	for _, v := range vs {
		d.buf = binary.LittleEndian.AppendUint32(d.buf, v)
	}
}

func (d *digester) i(vs ...int) {
	for _, v := range vs {
		d.u32(uint32(v))
	}
}

func (d *digester) grid(rows [][]uint32) {
	h := len(rows)
	w := 0

	if h > 0 {
		w = len(rows[0])
	}

	d.i(w, h)

	for _, r := range rows {
		d.u32(r...)
	}
}

func (d *digester) sum() string {
	s := sha256.Sum256(d.buf)
	return hex.EncodeToString(s[:8])
}

func gridDigest(g *Grid) string {
	var d digester
	d.grid(g.Rows())

	return d.sum()
}

func riverDigest(rv River) string {
	var d digester
	d.i(rv.N)

	for _, a := range rv.Ends {
		d.i(a[:]...)
	}

	for _, a := range rv.Start {
		d.i(a[:]...)
	}

	for _, a := range rv.End {
		d.i(a[:]...)
	}

	for _, a := range rv.Junc {
		d.i(a[:]...)
	}

	for _, p := range rv.Paths {
		d.i(len(p))

		for _, q := range p {
			d.i(q[:]...)
		}
	}

	return d.sum()
}

func roomWords(d *digester, r *Room) {
	d.i(r.Type, r.X, r.Y, r.W, r.H)
	d.u32(r.S4, r.Flags, r.R50)

	if r.Type == 1 {
		d.u32(r.Info54, r.Info58)
		d.i(r.SubType, r.SubTheme)
		d.u32(r.Mask, r.Seed.Lo, r.Seed.Hi)
	} else {
		d.i(r.PrestDef, r.File)
	}
}

func goldenParams(t *testing.T, env *Env, gs goldSeed, id int) Params {
	t.Helper()

	lay, err := drlgworld.Generate(env.Tables, gs.Seed, d2drlg.Normal)
	if err != nil {
		t.Fatal(err)
	}

	p, err := ParamsFromLayout(env.Tables, lay, id, gs.Seed)
	if err != nil {
		t.Fatal(err)
	}

	return p
}

func TestOracleOutdoor(t *testing.T) {
	env := testEnv(t)
	gold := loadGolden(t)

	counts := map[string]int{}
	levels := 0

	for _, gs := range gold {
		for key, gl := range gs.Levels {
			var id int
			fmt.Sscan(key, &id)

			p := goldenParams(t, env, gs, id)
			name := fmt.Sprintf("seed %#x level %d", gs.Seed, id)

			bad := func(what string, a ...interface{}) {
				counts[what]++
				if counts[what] <= 3 {
					t.Errorf("%s: %s %v", name, what, a)
				}
			}

			if p.BaseSeed != gs.Base {
				bad("base seed", p.BaseSeed, gs.Base)
			}

			if p.Rect != (Rect{gl.Rect[0], gl.Rect[1], gl.Rect[2], gl.Rect[3]}) {
				bad("rect", p.Rect, gl.Rect)
			}

			if p.OdFlags != gl.OdFlagsInit {
				bad("initial od.flags", p.OdFlags, gl.OdFlagsInit)
			}

			checkInputs(p, gl, bad)

			lv, err := Generate(env, p)
			if err != nil {
				t.Errorf("%s: %v", name, err)
				counts["generate error"]++

				continue
			}

			levels++
			compareLevel(t, lv, gl, bad)
		}
	}

	t.Logf("%d levels of %d seeds compared; mismatch counts: %v", levels, len(gold), counts)
}

func checkInputs(p Params, gl goldLevel, bad func(string, ...interface{})) {
	if gl.Vis != nil {
		for k := 0; k < 8; k++ {
			if p.Vis[k] != gl.Vis[0][k] || p.Warp[k] != gl.Vis[1][k] {
				bad("vis/warp", p.Vis, p.Warp, gl.Vis)
				break
			}
		}
	}

	var nb []goldNeighbor

	for _, n := range p.Neighbors {
		f8 := 0
		if n.F8 {
			f8 = 1
		}

		nb = append(nb, goldNeighbor{Lvl: n.Level, Dir: n.Dir, F8: f8, Fc: n.Flag, Rect: [4]int{n.Rect.X, n.Rect.Y, n.Rect.W, n.Rect.H}})
	}

	if !reflect.DeepEqual(nb, gl.Neighbors) && !(len(nb) == 0 && len(gl.Neighbors) == 0) {
		bad("neighbours", nb, gl.Neighbors)
	}
}

func compareLevel(t *testing.T, lv *Level, gl goldLevel, bad func(string, ...interface{})) {
	t.Helper()

	var poly [][4]int
	for _, v := range lv.PolygonAtAct {
		poly = append(poly, [4]int{v.X, v.Y, v.B, v.F})
	}

	if !reflect.DeepEqual(poly, gl.Polygon) {
		bad("polygon", poly, gl.Polygon)
	}

	// stage-by-stage level seed and od.flags at the entry of every stage
	if len(lv.Trace) != len(gl.Stages) {
		bad("stage count", len(lv.Trace), len(gl.Stages))
	} else {
		for i, s := range lv.Trace {
			g := gl.Stages[i]
			if s.Name != g[0].(string) || float64(s.Lo) != g[1].(float64) || float64(s.Hi) != g[2].(float64) || float64(s.OdFlags) != g[3].(float64) {
				bad("stage seed", s, g)
				break
			}
		}
	}

	end := lv.Trace[len(lv.Trace)-1]
	if [2]uint32{end.Lo, end.Hi} != gl.SeedStages || end.OdFlags != gl.OdFlagsAfter {
		bad("final seed after stages", end, gl.SeedStages, gl.OdFlagsAfter)
	}

	for name, g := range map[string]*Grid{"def": lv.Def, "B": lv.GridB, "flag": lv.Flag, "D": lv.GridD} {
		if got := gridDigest(g); got != gl.GridDigest[name] {
			bad("grid "+name, got, gl.GridDigest[name])
		}
	}

	if got := riverDigest(lv.River); got != gl.RiverDigest || lv.River.N != gl.RiverN {
		bad("road network", got, gl.RiverDigest)
	}

	for k, v := range gl.Counters {
		var d int
		fmt.Sscan(k, &d)

		if lv.Counters[d] != v {
			bad("preset file counter", d, lv.Counters[d], v)
			break
		}
	}

	if len(lv.Counters) != len(gl.Counters) {
		bad("preset counter count", len(lv.Counters), len(gl.Counters))
	}

	if len(lv.Rooms) != gl.NRooms {
		bad("room count", len(lv.Rooms), gl.NRooms)
	}

	var rd digester

	for _, r := range lv.Rooms {
		roomWords(&rd, r)
	}

	if got := rd.sum(); got != gl.RoomsDigest {
		bad("room list", got, gl.RoomsDigest)
	}

	if [2]uint32{lv.Seed.Lo, lv.Seed.Hi} != gl.SeedRooms {
		bad("final level seed", lv.Seed, gl.SeedRooms)
	}

	// plain-room tile grids A/B/C and final room seeds
	var gd digester

	var tail digester

	nPlain := 0

	for _, r := range lv.Rooms {
		if r.Type != 1 {
			continue
		}

		nPlain++

		g, err := lv.BuildRoomGrids(r, nil)
		if err != nil {
			bad("room build error", err)
			return
		}

		gd.grid(g.A.Rows())
		gd.grid(g.B.Rows())
		gd.grid(g.C.Rows())
		tail.u32(g.Seed.Lo, g.Seed.Hi)
	}

	gd.buf = append(gd.buf, tail.buf...)

	if nPlain != gl.NPlain || gd.sum() != gl.RoomGridsDig {
		bad("room tile grids / room seeds", nPlain, gl.NPlain)
	}

	if gl.Full != nil {
		compareFull(lv, gl.Full, bad)
	}
}

// compareFull diffs field by field against the full golden for a better report.
func compareFull(lv *Level, f *goldFull, bad func(string, ...interface{})) {
	for name, g := range map[string]*Grid{"def": lv.Def, "B": lv.GridB, "flag": lv.Flag, "D": lv.GridD} {
		if !reflect.DeepEqual(g.Rows(), f.Grids[name]) {
			bad("full grid "+name, g.Rows(), f.Grids[name])
		}
	}

	if len(lv.River.Paths) != len(f.River.Paths) {
		bad("full river paths", len(lv.River.Paths), len(f.River.Paths))
	} else {
		for i := range f.River.Paths {
			if !reflect.DeepEqual(lv.River.Paths[i], f.River.Paths[i]) && !(len(lv.River.Paths[i]) == 0 && len(f.River.Paths[i]) == 0) {
				bad("full river path", i, lv.River.Paths[i], f.River.Paths[i])
			}
		}
	}

	if len(lv.Rooms) != len(f.Rooms) {
		bad("full room count", len(lv.Rooms), len(f.Rooms))
		return
	}

	for i, gr := range f.Rooms {
		r := lv.Rooms[i]
		if r.Type != gr.Type || r.X != gr.X || r.Y != gr.Y || r.W != gr.W || r.H != gr.H || r.S4 != gr.S4 || r.Flags != gr.Flags || r.R50 != gr.R50 {
			bad("full room", i, *r, gr)
			return
		}

		if r.Type == 1 {
			if r.Info54 != gr.Info54 || r.Info58 != gr.Info58 || r.Mask != gr.Mask || [2]uint32{r.Seed.Lo, r.Seed.Hi} != gr.SeedMask {
				bad("full plain room", i, *r, gr)
				return
			}

			g, err := lv.BuildRoomGrids(r, nil)
			if err != nil {
				bad("full room build", err)
				return
			}

			if !reflect.DeepEqual(g.A.Rows(), gr.ABC["A"]) || !reflect.DeepEqual(g.B.Rows(), gr.ABC["B"]) ||
				!reflect.DeepEqual(g.C.Rows(), gr.ABC["C"]) || [2]uint32{g.Seed.Lo, g.Seed.Hi} != gr.SeedBld {
				bad("full room grids", i)
				return
			}
		} else if r.PrestDef != gr.PrestDef || r.File != gr.File {
			bad("full preset room", i, *r, gr)
			return
		}
	}
}
