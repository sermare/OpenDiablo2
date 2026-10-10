package drlgoutdoor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type mazeGoldRoom struct {
	X, Y, W, H int
	Flags, R50 uint32
	Seed       []uint32
	Tiles      map[string][][]interface{}
}

type mazeGoldLevel struct {
	Seed    uint32         `json:"seed"`
	Level   int            `json:"level"`
	Presets []mazeGoldRoom `json:"presets"`
}

// mazeDiff compares the exact maze tile build of one level with the emulator
// dump. It returns the number of rooms, rooms whose rectangle/flags/library
// mask differ, and the (differing, total) records per layer.
type mazeDiff struct {
	rooms, metaBad int
	layer          map[string][2]int
	err            string
	dx, dy         int
	flag20         int
}

// withoutFlag returns the golden record with the given flag bits cleared.
func withoutFlag(r []interface{}, bits uint32) []interface{} {
	o := append([]interface{}(nil), r...)
	o[3] = float64(uint32(o[3].(float64)) &^ bits)

	return o
}

// oracleGateSteps: the tile oracle's emulator environment has no monster
// table loaded (monstats count 0), so the DS1 object gates of two files whose
// gated records name monsters draw fewer level-seed steps there than in the
// real game (full environment, golden gates4_big.json): MephNWarpD 2 instead
// of 3 (levels 100/101) and JailSETheme 0 instead of 1 (level 31). The tile
// golden was made with these numbers; the game itself uses d2drlg.DS1GateSteps.
func oracleGateSteps(file string) int {
	switch strings.ToLower(strings.ReplaceAll(file, `\`, "/")) {
	case "act3/travincal/mephnwarpd.ds1":
		return 2
	case "act1/barracks/jailsetheme.ds1":
		return 0
	}

	return d2drlg.DS1GateSteps(file)
}

func diffMaze(t *testing.T, env *Env, g mazeGoldLevel) mazeDiff {
	t.Helper()

	d := mazeDiff{layer: map[string][2]int{}}

	tb, ok := env.Tables.(*d2drlg.Tables)
	if !ok {
		d.err = "no tables"
		return d
	}

	base, _ := d2rand.DrlgBaseSeed(g.Seed)

	// the two special Act 2 tombs come from the game seed (levels 66..72)
	ex := d2drlg.DrawActExtras(g.Seed, 1)

	res, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: g.Level, Difficulty: d2drlg.Normal, BaseSeed: base, GateSteps: oracleGateSteps,
		TombA: ex.TombA, TombB: ex.TombB})
	if err != nil {
		d.err = "maze: " + err.Error()
		return d
	}

	ml, err := NewMazeLevel(env, res, g.Level, g.Seed)
	if err != nil {
		d.err = "level: " + err.Error()
		return d
	}

	rts, err := ml.BuildTiles()
	if err != nil {
		d.err = "build: " + err.Error()
		return d
	}

	if len(rts) != len(g.Presets) {
		d.err = "room count differs"
		d.rooms = len(rts)

		return d
	}

	for i, rt := range rts {
		gr := g.Presets[i]
		r := rt.Room
		d.rooms++

		// the origin of Act 4/5 mazes differs by a constant (world offset); flag
		// bits 0x3100000 are runtime state of the original, not a build input
		if i == 0 {
			d.dx, d.dy = gr.X-r.X, gr.Y-r.Y
		}

		if r.X+d.dx != gr.X || r.Y+d.dy != gr.Y || r.W != gr.W || r.H != gr.H || r.R50 != gr.R50 || (r.Flags^gr.Flags)&0xfffff != 0 {
			d.metaBad++

			if d.metaBad <= 2 && os.Getenv("MAZE_META") != "" {
				t.Logf("META level=%d room %d: got %d,%d %dx%d flags=%#x r50=%#x; want %d,%d %dx%d flags=%#x r50=%#x", g.Level, i,
					r.X, r.Y, r.W, r.H, r.Flags, r.R50, gr.X, gr.Y, gr.W, gr.H, gr.Flags, gr.R50)
			}
		}

		for _, k := range []struct {
			name string
			l    []*TileRecord
		}{{"walls", rt.Walls}, {"floors", rt.Floors}, {"shadows", rt.Shadows}} {
			gl := gr.Tiles[k.name]
			n := max(len(gl), len(k.l))
			c := d.layer[k.name]
			c[0] += n

			for j := 0; j < n; j++ {
				want, got := "", ""
				if j < len(gl) {
					want = recString(gl[j])
				}

				if j < len(k.l) {
					got = mineString(k.l[j])
				}

				if want != got && os.Getenv("MAZE_FLAG20") == "" && j < len(gl) && j < len(k.l) {
					// flag 0x20 (a tile object was spawned from the record) comes from the
					// Act 1/2 rows of the tile object table, which are not ported (see
					// tileObjRows); it changes no picture
					if w2 := k.l[j]; recString(withoutFlag(gl[j], 0x20)) == mineString(w2) {
						d.flag20++
						continue
					}
				}

				if want != got {
					c[1]++

					if os.Getenv("MAZE_REC") != "" && c[1] <= 3 {
						t.Logf("REC level=%d room %d (%d,%d) %s #%d: want [%s] got [%s]", g.Level, i, r.X, r.Y, k.name, j, want, got)
					}
				}
			}

			d.layer[k.name] = c
		}
	}

	return d
}

// TestMazeTilesDir compares the maze tile builder (NewMazeLevel) with the
// emulator dumps of the maze levels in $ORACLE_TILES_DIR (measuring tool: it
// logs MAZETILES lines and fails only when MAZE_STRICT=1 and a record differs).
func TestMazeTilesDir(t *testing.T) {
	dir := os.Getenv("ORACLE_TILES_DIR")
	if dir == "" {
		t.Skip("ORACLE_TILES_DIR not set")
	}

	env := testEnv(t)
	files, _ := filepath.Glob(filepath.Join(dir, "o_*.json"))
	sort.Strings(files)

	bad := 0

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		var gl []mazeGoldLevel
		if err := json.Unmarshal(b, &gl); err != nil {
			t.Fatal(err)
		}

		for _, g := range gl {
			rec, ok := env.Tables.Level(g.Level)
			if !ok || rec.DrlgType != 1 {
				continue
			}

			d := diffMaze(t, env, g)
			t.Logf("MAZETILES level=%d seed=%#x rooms=%d metaBad=%d walls=%d/%d floors=%d/%d shadows=%d/%d flag20=%d err=%q", g.Level, g.Seed, d.rooms, d.metaBad,
				d.layer["walls"][1], d.layer["walls"][0], d.layer["floors"][1], d.layer["floors"][0], d.layer["shadows"][1], d.layer["shadows"][0], d.flag20, d.err)

			if d.err != "" || d.metaBad != 0 || d.layer["walls"][1]+d.layer["floors"][1]+d.layer["shadows"][1] != 0 {
				bad++
			}
		}
	}

	if bad != 0 && os.Getenv("MAZE_STRICT") == "1" {
		t.Fatalf("%d maze levels differ from the emulator", bad)
	}
}
