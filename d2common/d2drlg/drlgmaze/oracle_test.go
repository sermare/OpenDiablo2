package drlgmaze

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	gp := os.Getenv("ORACLE_MAZE")
	if gp == "" {
		gp = filepath.Join("..", "testdata", "maze_act1.json")
	}

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
		res, err := Generate(tb, Params{LevelID: g.Level, Difficulty: d2drlg.Difficulty(g.Diff), BaseSeed: base})
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

		if len(res.Chunks) == len(g.Rooms) {
			st.countOK++
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
