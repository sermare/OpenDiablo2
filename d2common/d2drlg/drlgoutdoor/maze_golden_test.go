package drlgoutdoor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// The numbers-only golden of the maze tile records (testdata/tiles_maze.json):
// per level and room the record counts and a digest of the emulator's records
// (flag 0x20, set by the unported Act 1/2 tile object rows, is left out), and
// how many rooms of the level the Go build matched when the golden was written
// (a floor against regressions). No game data is in the file.
type mazeGoldenRoom struct {
	N      [3]int `json:"n"`
	Digest string `json:"d"`
}

type mazeGoldenLevel struct {
	Level    int              `json:"level"`
	Seed     uint32           `json:"seed"`
	Rooms    []mazeGoldenRoom `json:"rooms"`
	MinMatch int              `json:"min_match"`
}

func mazeDigest(layers ...[]string) string {
	h := sha256.New()

	for _, l := range layers {
		for _, r := range l {
			h.Write([]byte(r))
			h.Write([]byte{'\n'})
		}

		h.Write([]byte{'#'})
	}

	return hex.EncodeToString(h.Sum(nil))[:12]
}

func goldDigest(gr mazeGoldRoom) ([3]int, string) {
	var (
		n  [3]int
		ls [][]string
	)

	for i, k := range []string{"walls", "floors", "shadows"} {
		var l []string

		for _, r := range gr.Tiles[k] {
			l = append(l, recString(withoutFlag(r, 0x20)))
		}

		n[i] = len(l)
		ls = append(ls, l)
	}

	return n, mazeDigest(ls...)
}

func mineDigest(rt *RoomTiles) ([3]int, string) {
	var (
		n  [3]int
		ls [][]string
	)

	for i, list := range [][]*TileRecord{rt.Walls, rt.Floors, rt.Shadows} {
		var l []string

		for _, r := range list {
			c := *r
			c.Flags &^= 0x20
			l = append(l, mineString(&c))
		}

		n[i] = len(l)
		ls = append(ls, l)
	}

	return n, mazeDigest(ls...)
}

func buildMazeRooms(env *Env, level int, seed uint32) ([]*RoomTiles, error) {
	tb, ok := env.Tables.(*d2drlg.Tables)
	if !ok {
		return nil, fmt.Errorf("no tables")
	}

	base, _ := d2rand.DrlgBaseSeed(seed)

	ex := d2drlg.DrawActExtras(seed, 1)

	res, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: level, Difficulty: d2drlg.Normal, BaseSeed: base,
		GateSteps: oracleGateSteps, TombA: ex.TombA, TombB: ex.TombB})
	if err != nil {
		return nil, err
	}

	ml, err := NewMazeLevel(env, res, level, seed)
	if err != nil {
		return nil, err
	}

	return ml.BuildTiles()
}

// TestMazeTilesGolden checks the maze tile build against the committed numbers
// (D2_TABLES and D2_DS1_ROOT needed). MAZE_WRITE_GOLDEN=<file> together with
// ORACLE_TILES_DIR rewrites the golden from the emulator dumps (o_*.json; the
// levels found there replace the same levels of the committed file, the others
// are kept).
func TestMazeTilesGolden(t *testing.T) {
	env := testEnv(t)
	path := filepath.Join("..", "testdata", "tiles_maze.json")

	if out := os.Getenv("MAZE_WRITE_GOLDEN"); out != "" {
		writeMazeGolden(t, env, out)
		return
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var gold []mazeGoldenLevel
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	total, matched := 0, 0

	for _, lv := range gold {
		total += len(lv.Rooms)

		rts, err := buildMazeRooms(env, lv.Level, lv.Seed)
		if err != nil || len(rts) != len(lv.Rooms) {
			if lv.MinMatch != 0 {
				t.Errorf("level %d: the maze tile build failed (%v) but matched %d rooms before", lv.Level, err, lv.MinMatch)
			}

			continue
		}

		m := 0

		for i, rt := range rts {
			n, d := mineDigest(rt)
			if d == lv.Rooms[i].Digest && n == lv.Rooms[i].N {
				m++
			}
		}

		matched += m

		if m < lv.MinMatch {
			t.Errorf("level %d: %d of %d rooms equal the emulator, was %d", lv.Level, m, len(rts), lv.MinMatch)
		}
	}

	t.Logf("MAZEGOLDEN levels=%d rooms=%d equal=%d", len(gold), total, matched)
}

func writeMazeGolden(t *testing.T, env *Env, out string) {
	t.Helper()

	files, _ := filepath.Glob(filepath.Join(os.Getenv("ORACLE_TILES_DIR"), "o_*.json"))
	sort.Strings(files)

	var all []mazeGoldenLevel

	have := map[int]bool{}

	for _, f := range files {
		b, _ := os.ReadFile(f)

		var gl []mazeGoldLevel
		if err := json.Unmarshal(b, &gl); err != nil {
			t.Fatal(err)
		}

		for _, g := range gl {
			rec, ok := env.Tables.Level(g.Level)
			if !ok || rec.DrlgType != 1 {
				continue
			}

			lv := mazeGoldenLevel{Level: g.Level, Seed: g.Seed}

			for _, gr := range g.Presets {
				n, d := goldDigest(gr)
				lv.Rooms = append(lv.Rooms, mazeGoldenRoom{N: n, Digest: d})
			}

			if rts, err := buildMazeRooms(env, g.Level, g.Seed); err == nil && len(rts) == len(lv.Rooms) {
				for i, rt := range rts {
					if n, d := mineDigest(rt); d == lv.Rooms[i].Digest && n == lv.Rooms[i].N {
						lv.MinMatch++
					}
				}
			}

			all = append(all, lv)
			have[g.Level] = true
		}
	}

	if old, err := os.ReadFile(filepath.Join("..", "testdata", "tiles_maze.json")); err == nil {
		var prev []mazeGoldenLevel
		if json.Unmarshal(old, &prev) == nil {
			for _, p := range prev {
				if !have[p.Level] {
					all = append(all, p)
				}
			}
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].Level < all[j].Level })

	b, _ := json.Marshal(all)
	if err := os.WriteFile(out, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestMazeTilesSeeds builds every maze level of the committed golden for more
// game seeds: it only checks that the build does
// not stop with an error (no emulator numbers for these seeds).
func TestMazeTilesSeeds(t *testing.T) {
	env := testEnv(t)

	b, err := os.ReadFile(filepath.Join("..", "testdata", "tiles_maze.json"))
	if err != nil {
		t.Fatal(err)
	}

	var gold []mazeGoldenLevel
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	for _, seed := range []uint32{1, 2, 0xdeadbeef, 0x12345678, 0x101d574a, 777, 31337, 99999} {
		for _, lv := range gold {
			if _, err := buildMazeRooms(env, lv.Level, seed); err != nil {
				t.Errorf("seed %#x level %d: %v", seed, lv.Level, err)
			}
		}
	}
}
