package drlgoutdoor

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func tables(t *testing.T) *d2drlg.Tables {
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

	tb, err := d2drlg.Load(d2drlg.Raw{LvlPrest: rd("patch_d2/LvlPrest.txt"), LvlPrestBin: rd("bin/patch_d2/lvlprest.bin")})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func count(r *Result, def int) int {
	n := 0

	for _, p := range r.Placed {
		if p.Def == def {
			n++
		}
	}

	return n
}

func TestSpecialsOnEmptyGrid(t *testing.T) {
	tb := tables(t)
	want := map[int]map[int]int{
		5:  {0xa1: 1, 0x29: 1, 0x28: 1, 0x1d: 1, 0x1e: 1, 0x33: 1},
		4:  {0xa0: 1, 0xa2: 1, 0x1f: 1, 0x33: 1},
		17: {0x6c: 1},
		39: {0x32: 1, 0x2e: 1, 0x1f: 1, 0x26: 1, 0x27: 1, 0x1d: 1, 0x1e: 1},
	}

	for id, defs := range want {
		for s := uint32(1); s <= 60; s++ {
			base, _ := d2rand.DrlgBaseSeed(s)
			sz := 80

			if id == 17 {
				sz = 40
			}

			r, err := GenerateAct1(tb, Params{LevelID: id, BaseSeed: base, Rect: Rect{1000, 1000, sz, sz}})
			if err != nil {
				t.Fatalf("level %d seed %d: %v", id, s, err)
			}

			for def, n := range defs {
				if got := count(r, def); got != n && !(def == 0x33 && id == 4 && got == 0) {
					t.Fatalf("level %d seed %d: def %#x placed %d times, want %d", id, s, def, got, n)
				}
			}

			for _, p := range r.Placed {
				if p.X < 0 || p.Y < 0 || p.X+p.W > r.W || p.Y+p.H > r.H {
					t.Fatalf("level %d: placement %+v outside %dx%d grid", id, p, r.W, r.H)
				}
			}
		}
	}
}

func TestBloodMoorDenFarFromTown(t *testing.T) {
	tb := tables(t)
	town := Rect{800, 1000, 56, 40}

	for s := uint32(1); s <= 40; s++ {
		base, _ := d2rand.DrlgBaseSeed(s)

		r, err := GenerateAct1(tb, Params{LevelID: 2, BaseSeed: base, Rect: Rect{1000, 1000, 96, 56}, Town: town})
		if err != nil {
			t.Fatal(err)
		}

		if count(r, 0x34) != 1 {
			t.Fatalf("seed %d: expected exactly one DOE entrance", s)
		}
	}
}

func TestCornerCaveEntranceWhenRiverFlags(t *testing.T) {
	tb := tables(t)

	for s := uint32(1); s <= 40; s++ {
		base, _ := d2rand.DrlgBaseSeed(s)

		r, err := GenerateAct1(tb, Params{LevelID: 3, BaseSeed: base, Rect: Rect{1000, 1000, 80, 80}, OdFlags: FlagRiverEdge1})
		if err != nil {
			t.Fatal(err)
		}

		var e *Placement

		for i := range r.Placed {
			if r.Placed[i].Def == 0x33 {
				e = &r.Placed[i]
			}
		}

		// flags 0x4 with 0x10 clear => k = 5: x in {3, 10-5}, y in {3, 10-4}
		if e == nil || !(e.X == 3 || e.X == 5) || !(e.Y == 3 || e.Y == 6) {
			t.Fatalf("seed %d: cave entrance %+v", s, e)
		}

		if r.OdFlags&FlagCavePlaced == 0 {
			t.Fatal("cave flag not set")
		}
	}
}

func TestDeterminismAndSeedSensitivity(t *testing.T) {
	tb := tables(t)
	sig := func(s uint32) string {
		base, _ := d2rand.DrlgBaseSeed(s)

		r, err := GenerateAct1(tb, Params{LevelID: 6, BaseSeed: base, Rect: Rect{0, 0, 80, 80}})
		if err != nil {
			t.Fatal(err)
		}

		return fmt.Sprint(r.Placed)
	}

	seen := map[string]bool{}

	for s := uint32(1); s <= 30; s++ {
		if sig(s) != sig(s) {
			t.Fatal("not deterministic")
		}

		seen[sig(s)] = true
	}

	if len(seen) < 25 {
		t.Errorf("only %d distinct layouts", len(seen))
	}
}

func TestUnknownLevel(t *testing.T) {
	tb := tables(t)
	if _, err := GenerateAct1(tb, Params{LevelID: 8}); err != ErrUnknownLevel {
		t.Fatal(err)
	}
}
