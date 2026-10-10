package drlgoutdoor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
)

// TestStaleCacheKeys measures how many tile records of one level share their
// renderer image cache key (style, sequence, type) with a record of an earlier
// level that is a different DT1 tile: the renderer's cache is keyed without the
// DT1 file and the palette, so before the cache was cleared on a level change
// each of those records drew the earlier level's picture. The levels are the
// seed-0 dumps of $ORACLE_TILES_DIR (o_0_<level>.json), walked in travel order.
func TestStaleCacheKeys(t *testing.T) {
	dir := os.Getenv("ORACLE_TILES_DIR")
	if dir == "" {
		t.Skip("ORACLE_TILES_DIR not set")
	}

	env := testEnv(t)
	dt1s := map[string]*d2dt1.DT1{}

	key := func(file string, idx int, layer string, ori int) (k [3]int32, ok bool) {
		d, have := dt1s[file]
		if !have {
			if b, err := env.DS1(file); err == nil {
				d, _ = d2dt1.LoadDT1(b)
			}

			dt1s[file] = d
		}

		if d == nil || idx < 0 || idx >= len(d.Tiles) {
			return k, false
		}

		typ := int32(ori)
		if layer == "floors" {
			typ = 0
		} else if layer == "shadows" {
			typ = 13
		}

		return [3]int32{d.Tiles[idx].Style, d.Tiles[idx].Sequence, typ}, true
	}

	type ident struct {
		file string
		idx  int
	}

	seen := map[[3]int32]map[ident]bool{} // the cache after the previous levels

	for _, lvl := range []int{1, 40, 75, 103, 109} {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("o_0_%d.json", lvl)))
		if err != nil {
			t.Skip(err)
		}

		var gl []goldFullLevel
		if err := json.Unmarshal(b, &gl); err != nil {
			t.Fatal(err)
		}

		total, stale := 0, 0
		cur := map[[3]int32]map[ident]bool{}

		for _, gr := range gl[0].Presets {
			for layer, recs := range gr.Tiles {
				for _, r := range recs {
					file, idx, ori := fmt.Sprint(r[4]), int(r[5].(float64)), int(r[2].(float64))

					k, ok := key(file, idx, layer, ori)
					if !ok {
						continue
					}

					total++

					id := ident{file, idx}
					if prev, hit := seen[k]; hit && !prev[id] {
						stale++
					}

					if cur[k] == nil {
						cur[k] = map[ident]bool{}
					}

					cur[k][id] = true
				}
			}
		}

		t.Logf("STALECACHE level %d: %d records, %d with a cache key an earlier level filled with another DT1 tile", lvl, total, stale)

		for k, ids := range cur {
			if seen[k] == nil {
				seen[k] = map[ident]bool{}
			}

			for id := range ids {
				seen[k][id] = true
			}
		}
	}
}
