package drlgmaze

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

type oracleRoom struct {
	X    int `json:"x"`
	Y    int `json:"y"`
	W    int `json:"w"`
	H    int `json:"h"`
	Def  int `json:"def_"`
	File int `json:"file"`
}

type oracleMaze struct {
	Seed   uint32       `json:"seed"`
	Diff   int          `json:"diff"`
	Level  int          `json:"level"`
	SeedLo uint32       `json:"seedLo"`
	SeedHi uint32       `json:"seedHi"`
	Rooms  []oracleRoom `json:"rooms"`

	TombA int `json:"tombA"`
	TombB int `json:"tombB"`
	X27   int `json:"x27"`
	Y27   int `json:"y27"`
	W27   int `json:"w27"`
	H27   int `json:"h27"`
	Side  int `json:"side27"`
	LvX   int `json:"lvx"`
	LvY   int `json:"lvy"`
	LvW   int `json:"lvw"`
	LvH   int `json:"lvh"`
	X108  int `json:"x108"`
	Y108  int `json:"y108"`
	W108  int `json:"w108"`
	H108  int `json:"h108"`

	// Compact goldens carry the chunk count and an FNV-1a 64 hash of the
	// sorted room keys instead of the room list.
	N int    `json:"n"`
	H uint64 `json:"h"`
}

// roomHash is the hash stored in compact goldens: FNV-1a 64 over the sorted
// room keys joined by ';'.
func roomHash(keys []string) uint64 {
	sort.Strings(keys)

	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.Join(keys, ";")))

	return h.Sum64()
}

func key(r oracleRoom) string {
	return fmt.Sprintf("%d,%d,%d,%d,def%d,f%d", r.X, r.Y, r.W, r.H, r.Def, r.File)
}

// TestOracleMaze compares Generate with golden data produced by emulating the
// real Game.exe (DRLG_GenerateLevel) with unicorn: per-level room list (the
// DrlgRooms after commit: rect, LvlPrest Def, file index) and the final level
// seed. Golden files contain only derived numbers.
func TestOracleMaze(t *testing.T) {
	tb := realTables(t)

	if gp := os.Getenv("ORACLE_MAZE"); gp != "" {
		runOracleMaze(t, tb, gp)

		return
	}

	for _, f := range []string{"maze_act1.json", "maze_act23.json", "maze_act45.json"} {
		t.Run(f, func(t *testing.T) { runOracleMaze(t, tb, filepath.Join("..", "testdata", f)) })
	}
}

func runOracleMaze(t *testing.T, tb *d2drlg.Tables, gp string) {
	t.Helper()

	b, err := os.ReadFile(gp)
	if err != nil {
		t.Skip(err)
	}

	var gold []oracleMaze
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	type stat struct{ n, seedOK, countOK, setOK, defOK, rectOK int }
	stats := map[int]*stat{}
	var firstBad []string

	for _, g := range gold {
		base, _ := d2rand.DrlgBaseSeed(g.Seed)
		res, err := Generate(tb, Params{LevelID: g.Level, Difficulty: d2drlg.Difficulty(g.Diff), BaseSeed: base,
			TombA: g.TombA, TombB: g.TombB, L27: Level27{g.X27, g.Y27, g.W27, g.H27, g.Side}, L108: Level27{X: g.X108, Y: g.Y108, W: g.W108, H: g.H108}})
		st := stats[g.Level]
		if st == nil {
			st = &stat{}
			stats[g.Level] = st
		}
		st.n++

		if err != nil {
			t.Errorf("level %d: %v", g.Level, err)
			continue
		}

		if res.Seed.Lo == g.SeedLo && res.Seed.Hi == g.SeedHi {
			st.seedOK++
		}

		wantN := len(g.Rooms)
		if g.H != 0 {
			wantN = g.N
		}

		if len(res.Chunks) == wantN {
			st.countOK++
		}

		if (g.Level == 28 || g.Level == 107) && g.LvW > 0 && (res.RectX != g.LvX || res.RectY != g.LvY || res.RectW != g.LvW || res.RectH != g.LvH) {
			t.Errorf("rect seed %#x diff %d: rect %d,%d %dx%d want %d,%d %dx%d", g.Seed, g.Diff,
				res.RectX, res.RectY, res.RectW, res.RectH, g.LvX, g.LvY, g.LvW, g.LvH)
		}

		if g.H != 0 {
			var keys []string
			for _, c := range res.Chunks {
				rm := res.Rooms[c.Room]
				keys = append(keys, key(oracleRoom{c.X, c.Y, c.W, c.H, rm.Def, rm.File}))
			}

			if roomHash(keys) == g.H {
				st.setOK++
				st.rectOK++
			} else if len(firstBad) < 6 {
				firstBad = append(firstBad, fmt.Sprintf("seed %#x diff %d level %d: room hash differs", g.Seed, g.Diff, g.Level))
			}

			continue
		}

		want := map[string]int{}
		wantRect := map[string]int{}
		for _, r := range g.Rooms {
			want[key(r)]++
			wantRect[fmt.Sprintf("%d,%d,%d,%d", r.X, r.Y, r.W, r.H)]++
		}

		got := map[string]int{}
		gotRect := map[string]int{}
		for _, c := range res.Chunks {
			rm := res.Rooms[c.Room]
			or := oracleRoom{c.X, c.Y, c.W, c.H, rm.Def, rm.File}
			got[key(or)]++
			gotRect[fmt.Sprintf("%d,%d,%d,%d", c.X, c.Y, c.W, c.H)]++
		}

		if fmt.Sprint(sortedKV(got)) == fmt.Sprint(sortedKV(want)) {
			st.setOK++
		} else if len(firstBad) < 6 {
			firstBad = append(firstBad, fmt.Sprintf("seed %#x diff %d level %d: chunks go %d oracle %d", g.Seed, g.Diff, g.Level, len(res.Chunks), len(g.Rooms)))
		}

		if fmt.Sprint(sortedKV(gotRect)) == fmt.Sprint(sortedKV(wantRect)) {
			st.rectOK++
		}
	}

	ids := []int{}
	for id := range stats {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	for _, id := range ids {
		s := stats[id]
		t.Logf("level %2d: n=%d finalSeedOK=%d chunkCountOK=%d rectSetOK=%d rect+def+fileSetOK=%d", id, s.n, s.seedOK, s.countOK, s.rectOK, s.setOK)
	}

	for _, m := range firstBad {
		t.Log(m)
	}

	for _, id := range ids {
		if s := stats[id]; s.seedOK != s.n || s.setOK != s.n {
			t.Errorf("level %d: %d/%d final seeds and %d/%d room sets match the oracle", id, s.seedOK, s.n, s.setOK, s.n)
		}
	}
}

func sortedKV(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, fmt.Sprintf("%s=%d", k, v))
	}

	sort.Strings(out)

	return out
}
