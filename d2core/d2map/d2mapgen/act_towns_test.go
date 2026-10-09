package d2mapgen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

func TestChooseTownStart(t *testing.T) {
	tests := []struct {
		name         string
		cands        []startCand
		wantX, wantY int
		ok           bool
	}{
		{"none", nil, 0, 0, false},
		{"style 30", []startCand{{5, 5, 8, 0}, {7, 9, 30, 0}}, 7, 9, true},
		{"prefers sequence 0 (LutN has two)", []startCand{{32, 13, 30, 11}, {30, 40, 30, 0}}, 30, 40, true},
		{"falls back to style 33 (Fortress)", []startCand{{3, 1, 8, 2}, {9, 8, 33, 0}}, 9, 8, true},
		{"style 30 beats 33", []startCand{{9, 8, 33, 0}, {1, 2, 30, 0}}, 1, 2, true},
		{"nothing usable", []startCand{{3, 1, 8, 2}}, 0, 0, false},
	}

	for _, tc := range tests {
		x, y, _, ok := chooseTownStart(tc.cands)
		if ok != tc.ok || x != tc.wantX || y != tc.wantY {
			t.Errorf("%s: got (%d,%d,%v), want (%d,%d,%v)", tc.name, x, y, ok, tc.wantX, tc.wantY, tc.ok)
		}
	}
}

func TestTownFileIndex(t *testing.T) {
	if townFileIndex(1, 123, 75) != 0 || townFileIndex(0, 123, 75) != 0 {
		t.Fatal("a single file is always index 0")
	}

	seen := map[int]bool{}

	for base := uint32(0); base < 64; base++ {
		i := townFileIndex(2, base, d2level.LutGholein)
		if i < 0 || i > 1 {
			t.Fatalf("index %d out of range", i)
		}

		if townFileIndex(2, base, d2level.LutGholein) != i {
			t.Fatal("not deterministic")
		}

		seen[i] = true
	}

	if len(seen) != 2 {
		t.Errorf("both Lut Gholein variants should occur over 64 seeds: %v", seen)
	}
}

func TestActTownTable(t *testing.T) {
	for _, act := range []int{2, 3, 4, 5} {
		if !IsActTown(d2level.ActStartLevel(act)) {
			t.Errorf("act %d town is not covered", act)
		}
	}

	if IsActTown(d2level.RogueEncampment) {
		t.Error("the Rogue Encampment has its own provider")
	}

	p := actTownProvider{}
	if !p.CanLoad(d2level.Harrogath) || p.CanLoad(3) {
		t.Error("CanLoad")
	}
}

// TestRealTownDS1 reads the real town files from $D2_DS1 (a folder holding
// LutW.ds1, LutN.ds1, DockTown3.ds1, Fortress.ds1, townWest.ds1) and checks
// the start markers and the NPC counts the provider relies on.
func TestRealTownDS1(t *testing.T) {
	dir := os.Getenv("D2_DS1")
	if dir == "" {
		t.Skip("D2_DS1 not set")
	}

	tests := []struct {
		file         string
		act          int32
		wantX, wantY int
		npcs         int // type 1 (monster) objects in the DS1
	}{
		{"LutW.ds1", 2, 30, 40, 32},
		{"LutN.ds1", 2, 30, 40, 34},
		{"DockTown3.ds1", 3, 23, 33, 22},
		{"Fortress.ds1", 4, 9, 8, 4},
		{"townWest.ds1", 5, 19, 4, 13},
	}

	for _, tc := range tests {
		b, err := os.ReadFile(filepath.Join(dir, tc.file))
		if err != nil {
			t.Fatal(err)
		}

		d, err := d2ds1.Unmarshal(b)
		if err != nil {
			t.Fatalf("%s: %v", tc.file, err)
		}

		if d.Act != tc.act {
			t.Errorf("%s: act %d, want %d", tc.file, d.Act, tc.act)
		}

		w, h := d.Size()

		var cands []startCand

		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				for i := range d.Walls {
					if wt := d.Walls[i].Tile(x, y); wt.Type.Special() {
						cands = append(cands, startCand{x, y, int(wt.Style), int(wt.Sequence)})
					}
				}
			}
		}

		x, y, _, ok := chooseTownStart(cands)
		if !ok || x != tc.wantX || y != tc.wantY {
			t.Errorf("%s: start (%d,%d,%v), want (%d,%d)", tc.file, x, y, ok, tc.wantX, tc.wantY)
		}

		n := 0

		for _, o := range d.Objects {
			if o.Type == 1 {
				n++
			}
		}

		if n < tc.npcs-2 { // monster objects, some of them place markers
			t.Errorf("%s: %d monster objects, want about %d", tc.file, n, tc.npcs)
		}
	}
}
