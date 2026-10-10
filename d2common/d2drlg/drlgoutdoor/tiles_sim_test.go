package drlgoutdoor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// This file measures the engine's old path for a preset DS1 level (the stamp
// path used by GenerateActTown and the Act 1 town before the exact builder:
// MapEngine.PlaceStamp -> PrepareTile, which looks every DS1 cell up by
// (style, sequence, type) in the union of the level type's DT1 files and the
// DT1 files listed in the DS1, and picks among several candidates with
// getRandomTile) against the emulator's real tile records, cell by cell. The
// result is a histogram of how the old path differs. Run it with
// ORACLE_TILES_DIR as TestTileDiffDir.

// simTile is one DT1 tile in the engine's load order.
type simTile struct {
	file string
	idx  int
	t    *d2dt1.Tile
}

// simRandom is a copy of d2mapengine.getRandomTile (the engine seed is never
// set, so it is 0).
func simRandom(tiles []simTile, x, y int) int {
	tileSeed := uint64(0) + uint64(x)
	tileSeed *= uint64(y)
	tileSeed ^= tileSeed << 13
	tileSeed ^= tileSeed >> 17
	tileSeed ^= tileSeed << 5

	weightSum := 0
	for i := range tiles {
		weightSum += int(tiles[i].t.RarityFrameIndex)
	}

	if weightSum == 0 {
		return 0
	}

	random := tileSeed % uint64(weightSum)
	sum := 0

	for i := range tiles {
		sum += int(tiles[i].t.RarityFrameIndex)
		if sum >= int(random) {
			return i
		}
	}

	return 0
}

func simDS1Files(ds1 *d2ds1.DS1) []string {
	var out []string

	for _, f := range ds1.Files {
		f = strings.ToLower(f)
		f = strings.ReplaceAll(f, "c:", "")
		f = strings.ReplaceAll(f, ".tg1", ".dt1")
		f = strings.ReplaceAll(f, "\\d2\\data\\global\\tiles\\", "")
		out = append(out, strings.ReplaceAll(f, "\\", "/"))
	}

	return out
}

type simKey struct{ style, seq, typ int32 }

// simResult is the outcome of the old path for one DS1 cell layer.
type simResult struct {
	reason string // "", "hidden", "prop0", "blank30", "nooptions", "special"
	pick   *simTile
	nopt   int
	cell   d2ds1.Tile
}

// simPlace is one DS1 stamped at an absolute tile position, clipped to w x h
// (what the engine's PlaceStamp / PlaceStampClipped do for it).
type simPlace struct {
	path       string
	x, y, w, h int
}

// simBlank decodes a floor tile like the renderer's tile cache and reports
// whether it has no opaque pixel (such a tile draws nothing: a black void).
func simBlank(d *d2dt1.DT1, idx int) (empty, known bool) {
	if d == nil || idx < 0 || idx >= len(d.Tiles) {
		return false, false
	}

	tl := &d.Tiles[idx]
	minY := int32(0)

	for _, b := range tl.Blocks {
		if int32(b.Y) < minY {
			minY = int32(b.Y)
		}
	}

	h := tl.Height
	if h < 0 {
		h = -h
	}

	if tl.Width <= 0 || h <= 0 {
		return true, true
	}

	px := make([]byte, int(tl.Width)*int(h))
	d2dt1.DecodeTileGfxData(tl.Blocks, &px, -minY, tl.Width)

	for _, v := range px {
		if v != 0 {
			return false, true
		}
	}

	return true, true
}

type simCell struct {
	layer string
	x, y  int
}

// simulate runs the engine's old stamp path for the placements of a level and
// compares it with the emulator's records of the level. Counters go to hist
// (per level and tile kind); the returned map counts cells by outcome.
func simulate(t *testing.T, env *Env, loadDT1 func(string) *d2dt1.DT1, hist map[string]int, level, ltype int, places []simPlace, gold map[simCell][][]interface{}) map[string]int {
	t.Helper()

	cells := map[string]int{}
	sim := map[simCell][]simResult{}

	lt, _ := env.Tables.LvlType(ltype)

	for _, pl := range places {
		raw, err := env.DS1(pl.path)
		if err != nil {
			hist[fmt.Sprintf("level %d | DS1 not loadable %s", level, pl.path)]++
			continue
		}

		ds, err := d2ds1.Unmarshal(raw)
		if err != nil {
			t.Fatal(err)
		}

		var names []string

		for _, s := range lt.Slots {
			if s != "" && s != "0" {
				names = append(names, normDT1(s))
			}
		}

		names = append(names, simDS1Files(ds)...)

		var all []simTile

		seen := map[string]bool{}

		for _, n := range names {
			if seen[n] {
				continue
			}

			seen[n] = true

			d := loadDT1(n)
			if d == nil {
				hist[fmt.Sprintf("level %d | DT1 not loadable %s", level, n)]++
				continue
			}

			for i := range d.Tiles {
				all = append(all, simTile{n, i, &d.Tiles[i]})
			}
		}

		byKey := map[simKey][]simTile{}
		for _, st := range all {
			k := simKey{st.t.Style, st.t.Sequence, st.t.Type}
			byKey[k] = append(byKey[k], st)
		}

		put := func(layer string, x, y int, tl *d2ds1.Tile, typ int32) {
			k := simCell{layer, pl.x + x, pl.y + y}

			switch {
			case tl.Prop1 == 0:
				sim[k] = append(sim[k], simResult{reason: "prop0"})
				return
			case tl.Hidden():
				sim[k] = append(sim[k], simResult{reason: "hidden"})
				return
			case layer == "floors" && tl.Style == 30 && tl.Sequence == 0:
				sim[k] = append(sim[k], simResult{reason: "blank30"})
				return
			}

			opts := byKey[simKey{int32(tl.Style), int32(tl.Sequence), typ}]
			if len(opts) == 0 {
				r := "nooptions"
				if typ >= 10 && layer == "walls" {
					r = "special"
				}

				sim[k] = append(sim[k], simResult{reason: r})

				return
			}

			// the map position seeds the variant pick (MapTile.PrepareTile(x, y))
			i := simRandom(opts, pl.x+x, pl.y+y)
			sim[k] = append(sim[k], simResult{pick: &opts[i], nopt: len(opts), cell: *tl})
		}

		for y := 0; y < pl.h; y++ {
			for x := 0; x < pl.w; x++ {
				for w := range ds.Walls {
					tl := ds.Walls[w].Tile(x, y)
					put("walls", x, y, tl, int32(tl.Type))
				}

				for f := range ds.Floors {
					put("floors", x, y, ds.Floors[f].Tile(x, y), 0)
				}

				for s := range ds.Shadows {
					put("shadows", x, y, ds.Shadows[s].Tile(x, y), 13)
				}
			}
		}
	}

	find := func(file string, idx int) *d2dt1.Tile {
		if d := loadDT1(file); d != nil && idx >= 0 && idx < len(d.Tiles) {
			return &d.Tiles[idx]
		}

		return nil
	}

	extra := func(k simCell, r simResult) {
		cells[k.layer+" extra (old path draws, oracle has none)"]++
		hist[fmt.Sprintf("level %d | %s | EXTRA %s | prop1 %#x unk2 %#x", level, k.layer, r.pick.file, r.cell.Prop1, r.cell.Unknown2)]++
	}

	for k, recs := range gold {
		var got []simResult

		for _, r := range sim[k] {
			if r.pick != nil {
				got = append(got, r)
			}
		}

		for i, r := range recs {
			cells[k.layer+" records"]++

			file, idx, ori := fmt.Sprint(r[4]), int(r[5].(float64)), int(r[2].(float64))
			tt := find(file, idx)

			if k.layer == "floors" {
				if e, ok := simBlank(loadDT1(file), idx); ok && e {
					hist[fmt.Sprintf("level %d | BLANK floor graphic (oracle record) %s #%d", level, file, idx)]++
				}
			}

			var outcome string

			switch {
			case i < len(got) && got[i].pick.file == file && got[i].pick.idx == idx:
				outcome = "same"
			case i < len(got) && tt != nil && got[i].pick.t.Style == tt.Style && got[i].pick.t.Sequence == tt.Sequence && got[i].pick.t.Type == tt.Type:
				outcome = "variant (same key, other tile)"
			case i < len(got):
				outcome = "other tile"
			default:
				var reasons []string

				for _, s := range sim[k] {
					if s.reason != "" {
						reasons = append(reasons, s.reason)
					}
				}

				switch {
				case len(reasons) > 0:
					outcome = "missing (" + strings.Join(reasons, ",") + ")"
				case len(got) > 0:
					outcome = "missing (second record on a cell the old path fills once)"
				default:
					outcome = "missing (no DS1 cell)"
				}
			}

			cells[k.layer+" "+outcome]++

			if outcome != "same" {
				hist[fmt.Sprintf("level %d | %s | %s | ori %d | %s", level, k.layer, file, ori, outcome)]++
			}
		}

		for j := len(recs); j < len(got); j++ {
			extra(k, got[j])
		}
	}

	for k, rs := range sim {
		if _, ok := gold[k]; ok {
			continue
		}

		for _, r := range rs {
			if r.pick != nil {
				extra(k, r)
			}
		}
	}

	return cells
}

// TestTileSimDir measures the old stamp path (see the comment at the top of
// the file) for every level of the dumps in $ORACLE_TILES_DIR: DrlgType 2
// levels (one preset DS1 over the level rectangle) and, where the DRLG port
// can place them, maze levels (one DS1 per room).
func TestTileSimDir(t *testing.T) {
	dir := os.Getenv("ORACLE_TILES_DIR")
	if dir == "" {
		t.Skip("ORACLE_TILES_DIR not set")
	}

	env := testEnv(t)
	files, _ := filepath.Glob(filepath.Join(dir, "o_*.json"))
	sort.Strings(files)

	cache := map[uint32]*drlgworld.Layout{}
	hist := map[string]int{}
	dt1Cache := map[string]*d2dt1.DT1{}

	loadDT1 := func(f string) *d2dt1.DT1 {
		if d, ok := dt1Cache[f]; ok {
			return d
		}

		var d *d2dt1.DT1

		if b, err := env.DS1(f); err == nil {
			d, _ = d2dt1.LoadDT1(b)
		}

		dt1Cache[f] = d

		return d
	}

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
			rec, ok := env.Tables.Level(g.Level)
			if !ok {
				continue
			}

			var places []simPlace

			switch rec.DrlgType {
			case 2:
				places = presetPlaces(t, env, cache, g)
			case 1:
				tb, ok := env.Tables.(*d2drlg.Tables)
				if !ok {
					continue
				}

				base, _ := d2rand.DrlgBaseSeed(g.Seed)

				res, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: g.Level, Difficulty: d2drlg.Normal, BaseSeed: base})
				if err != nil {
					t.Logf("TILESIM level=%d seed=%#x maze not placed: %v", g.Level, g.Seed, err)
					continue
				}

				for _, r := range res.Rooms {
					if r.FileName != "" {
						places = append(places, simPlace{NormalizePrestFile(r.FileName), r.X, r.Y, r.W, r.H})
					}
				}
			default:
				continue
			}

			if len(places) == 0 {
				continue
			}

			gold := map[simCell][][]interface{}{}
			border := 0

			for _, gr := range g.Presets {
				for layer, recs := range gr.Tiles {
					for _, r := range recs {
						rx, ry := int(r[0].(float64)), int(r[1].(float64))
						if rx >= 8 || ry >= 8 { // records of the next room's border cells
							border++
						}

						k := simCell{layer, gr.X + rx, gr.Y + ry}
						dup := false

						for _, o := range gold[k] {
							if recString(o) == recString(r) {
								dup = true
							}
						}

						if !dup {
							gold[k] = append(gold[k], r)
						}
					}
				}
			}

			lvl := fmt.Sprintf("level %d seed %#x", g.Level, g.Seed)
			cells := simulate(t, env, loadDT1, hist, g.Level, rec.LevelType, places, gold)

			keys := make([]string, 0, len(cells))
			for k := range cells {
				keys = append(keys, k)
			}

			sort.Strings(keys)

			t.Logf("TILESIM %s: %d DS1 placements, %d border records merged by cell", lvl, len(places), border)

			for _, k := range keys {
				t.Logf("TILESIM %s | %s = %d", lvl, k, cells[k])
			}
		}
	}

	keys := make([]string, 0, len(hist))
	for k := range hist {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		t.Logf("TILESIM-HIST %s = %d", k, hist[k])
	}
}

// presetPlaces is the DS1 stamp of a DrlgType 2 level over its rectangle.
func presetPlaces(t *testing.T, env *Env, cache map[uint32]*drlgworld.Layout, g goldFullLevel) []simPlace {
	t.Helper()

	p, err := paramsOfLevel(t, env, cache, g.Seed, g.Level)
	if err != nil {
		t.Logf("TILESIM level=%d seed=%#x not simulated: %v", g.Level, g.Seed, err)
		return nil
	}

	var file int

	switch g.Level {
	case 40: // the Act 2 world picks LutW / LutN
		tb, _ := env.Tables.(*d2drlg.Tables)

		w, err := PlaceAct2World(tb, g.Seed, d2drlg.Normal)
		if err != nil {
			return nil
		}

		file = w.TownFile - 1
	default:
		override := -1
		if g.Level == 1 {
			override = cache[g.Seed].TownFile
		}

		pl, err := GeneratePreset(env, p, override)
		if err != nil {
			return nil
		}

		file, p.Rect = pl.File, pl.Rect
	}

	prec, ok := env.Tables.PrestByLevel(g.Level)
	if !ok {
		return nil
	}

	names := prec.File[:]

	if g.Level == 40 { // LvlPrest lists Files=0 and "0" first: the engine uses the non-empty names in order
		names = nil

		for _, f := range prec.File {
			if f != "" && f != "0" {
				names = append(names, f)
			}
		}
	}

	if file < 0 || file >= len(names) {
		return nil
	}

	return []simPlace{{NormalizePrestFile(names[file]), p.Rect.X, p.Rect.Y, p.Rect.W, p.Rect.H}}
}
