package drlgoutdoor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
)

// goldFullRoom is one room of a full emulator dump (gen_tiles2.py): the
// records of the room as [x, y, orientation, flags, dt1 file, tile index].
type goldFullRoom struct {
	X, Y  int
	Tiles map[string][][]interface{}
}

type goldFullLevel struct {
	Seed    uint32         `json:"seed"`
	Level   int            `json:"level"`
	Rooms   []goldFullRoom `json:"rooms"`
	Presets []goldFullRoom `json:"presets"`
}

func recString(r []interface{}) string {
	parts := make([]string, len(r))

	for i, v := range r {
		if f, ok := v.(float64); ok {
			parts[i] = fmt.Sprintf("%.0f", f)
		} else {
			parts[i] = fmt.Sprint(v)
		}
	}

	return strings.Join(parts, " ")
}

func mineString(r *TileRecord) string {
	return fmt.Sprint(r.X, " ", r.Y, " ", r.Ori, " ", r.Flags, " ", r.Tile.File(), " ", r.Tile.Idx)
}

// TestTileDiffDir diffs the exact tile builder against full emulator dumps
// (every *.json of $ORACLE_TILES_DIR, written by gen_tiles2.py with one level
// per file) record by record and prints the number of differing records per
// level and layer plus a histogram of the differing records by DT1 file.
// It never fails on a mismatch; it is a measuring tool.
func TestTileDiffDir(t *testing.T) {
	dir := os.Getenv("ORACLE_TILES_DIR")
	if dir == "" {
		t.Skip("ORACLE_TILES_DIR not set")
	}

	env := testEnv(t)
	files, _ := filepath.Glob(filepath.Join(dir, "o_*.json"))
	sort.Strings(files)

	cache := map[uint32]*drlgworld.Layout{}
	byFile := map[string]int{}

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		var gl []goldFullLevel
		if err := json.Unmarshal(b, &gl); err != nil {
			t.Fatal(err)
		}

		for _, g := range gl {
			res, err := tilesOfLevel(t, env, cache, g.Seed, g.Level)
			if err != nil {
				t.Logf("TILEDIFF level=%d seed=%#x error=%v", g.Level, g.Seed, err)
				continue
			}

			var mine []*RoomTiles

			for _, rt := range res {
				if rt != nil && rt.Room.Type == 2 {
					mine = append(mine, rt)
				}
			}

			if len(mine) != len(g.Presets) {
				t.Logf("TILEDIFF level=%d seed=%#x rooms=%d golden_rooms=%d (room count differs)", g.Level, g.Seed, len(mine), len(g.Presets))
				continue
			}

			total := map[string][2]int{} // layer -> {records, differing}

			for i, rt := range mine {
				gr := g.Presets[i]

				for _, k := range []struct {
					name string
					l    []*TileRecord
				}{{"walls", rt.Walls}, {"floors", rt.Floors}, {"shadows", rt.Shadows}} {
					gl := gr.Tiles[k.name]
					n := len(gl)

					if len(k.l) > n {
						n = len(k.l)
					}

					c := total[k.name]
					c[0] += n

					for j := 0; j < n; j++ {
						want, got := "", ""
						if j < len(gl) {
							want = recString(gl[j])
						}

						if j < len(k.l) {
							got = mineString(k.l[j])
						}

						if want != got {
							c[1]++

							if j < len(gl) {
								byFile[fmt.Sprintf("%s %s", k.name, gl[j][4])]++
							}
						}
					}

					total[k.name] = c
				}
			}

			t.Logf("TILEDIFF level=%d seed=%#x rooms=%d walls=%d/%d floors=%d/%d shadows=%d/%d (differing/records)", g.Level, g.Seed, len(mine),
				total["walls"][1], total["walls"][0], total["floors"][1], total["floors"][0], total["shadows"][1], total["shadows"][0])
		}
	}

	keys := make([]string, 0, len(byFile))
	for k := range byFile {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		t.Logf("TILEDIFF-FILE %s %d", k, byFile[k])
	}
}
