package drlgoutdoor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
)

// TestOraclePresetAct1 compares GeneratePreset for the Act 1 DrlgType 2 levels
// 13..16 (treasure caves) and 37 (Catacombs Level 4) with numbers produced by
// the real Game.exe DRLG in an emulator: 12 seeds x 3 difficulties x 5 levels.
// Needs D2_TABLES and D2_DS1_ROOT; skips without.
func TestOraclePresetAct1(t *testing.T) {
	env := testEnv(t)

	b, err := os.ReadFile(filepath.Join("..", "testdata", "preset_act1.json"))
	if err != nil {
		t.Skip(err)
	}

	var gold []goldAct45
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	n := 0

	for _, g := range gold {
		p, err := ParamsPreset(env.Tables, g.Level, g.Seed, d2drlg.Difficulty(g.Diff))
		if err != nil {
			t.Fatal(err)
		}

		if p.BaseSeed != g.Base || p.Vis != g.Vis || p.Rect != (Rect{g.Rect[0], g.Rect[1], g.Rect[2], g.Rect[3]}) {
			t.Errorf("seed %#x diff %d level %d: params %v / %v vs gold base %d rect %v vis %v", g.Seed, g.Diff, g.Level, p.BaseSeed, p.Rect, g.Base, g.Rect, g.Vis)
			continue
		}

		pl, err := GeneratePreset(env, p, -1)
		if err != nil {
			t.Errorf("seed %#x level %d: %v", g.Seed, g.Level, err)
			continue
		}

		if g.File != nil && pl.File != *g.File {
			t.Errorf("seed %#x level %d: file %d want %d", g.Seed, g.Level, pl.File, *g.File)
		}

		if pl.Seed.Lo != g.SeedLo || pl.Seed.Hi != g.SeedHi {
			t.Errorf("seed %#x diff %d level %d: final seed %d:%d want %d:%d", g.Seed, g.Diff, g.Level, pl.Seed.Lo, pl.Seed.Hi, g.SeedLo, g.SeedHi)
		}

		if len(pl.Rooms) != g.NRooms || roomsDigest(pl.Rooms) != g.Rooms {
			t.Errorf("seed %#x diff %d level %d: rooms differ (%d vs %d)", g.Seed, g.Diff, g.Level, len(pl.Rooms), g.NRooms)
		}

		n++
	}

	t.Logf("%d/%d records equal", n, len(gold))
}
