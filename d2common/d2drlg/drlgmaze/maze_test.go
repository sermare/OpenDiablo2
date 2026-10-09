package drlgmaze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// syntheticTables builds minimal tables (Act 1 maze levels, a LvlPrest row for
// every Def up to 400 with 2 files) so the tests run without game data.
func syntheticTables(t *testing.T) *d2drlg.Tables {
	t.Helper()

	var lv, mz, pr strings.Builder

	lv.WriteString("Name\tId\tAct\tSizeX\tSizeY\tSizeX(N)\tSizeY(N)\tSizeX(H)\tSizeY(H)\tOffsetX\tOffsetY\tDepend\tDrlgType\tLevelType\n")
	mz.WriteString("Name\tLevel\tRooms\tRooms(N)\tRooms(H)\tSizeX\tSizeY\tMerge\n")

	type m struct{ id, typ, n, nm, nh, sz int }

	for _, x := range []m{
		{8, 3, 1, 1, 1, 24}, {9, 3, 4, 4, 4, 24}, {10, 3, 6, 6, 6, 24}, {11, 3, 4, 4, 4, 24}, {12, 3, 4, 4, 4, 24},
		{18, 4, 12, 24, 36, 8}, {19, 4, 12, 24, 36, 8}, {21, 4, 6, 12, 24, 8}, {22, 4, 6, 12, 20, 8}, {23, 4, 6, 12, 16, 8}, {24, 4, 6, 12, 12, 8},
		{29, 8, 12, 12, 12, 12}, {30, 8, 14, 14, 14, 12}, {31, 8, 16, 16, 16, 12},
		{34, 10, 12, 12, 12, 12}, {35, 10, 14, 14, 14, 12}, {36, 10, 16, 16, 16, 12},
	} {
		fmt.Fprintf(&lv, "L%d\t%d\t0\t200\t200\t200\t200\t200\t200\t%d\t1000\t0\t1\t%d\n", x.id, x.id, 1000+x.id*10, x.typ)
		fmt.Fprintf(&mz, "M%d\t%d\t%d\t%d\t%d\t%d\t%d\t500\n", x.id, x.id, x.n, x.nm, x.nh, x.sz, x.sz)
	}

	pr.WriteString("Name\tDef\tLevelId\tSizeX\tSizeY\tFiles\tFile1\tFile2\tFile3\tFile4\tFile5\tFile6\tDt1Mask\n")

	for d := 0; d < 400; d++ {
		fmt.Fprintf(&pr, "P%d\t%d\t0\t24\t24\t2\tdef%d_a.ds1\tdef%d_b.ds1\t0\t0\t0\t0\t0\n", d, d, d, d)
	}

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: []byte(lv.String()), LvlMaze: []byte(mz.String()), LvlPrest: []byte(pr.String())})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

// realTables loads the extracted 1.14b tables when D2_TABLES is set.
func realTables(t *testing.T) *d2drlg.Tables {
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

	tb, err := d2drlg.Load(d2drlg.Raw{Levels: rd("patch_d2/Levels.txt"), LvlMaze: rd("patch_d2/LvlMaze.txt"),
		LvlPrest: rd("patch_d2/LvlPrest.txt"), LvlPrestBin: rd("bin/patch_d2/lvlprest.bin")})
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

var mazeLevels = []int{8, 9, 10, 11, 12, 18, 19, 21, 22, 23, 24, 29, 30, 31, 34, 35, 36}

// expectedStamps is how many special rooms each finisher converts or adds.
var expectedStamps = map[int]int{8: 2, 9: 3, 10: 3, 11: 2, 12: 2, 18: 2, 19: 2, 21: 2, 22: 2, 23: 2, 24: 2, 29: 3, 30: 3, 31: 2, 34: 1, 35: 2, 36: 1}

func checkInvariants(t *testing.T, tb Tables, res *Result, id int, diff d2drlg.Difficulty) {
	t.Helper()

	mz, _ := tb.Maze(id)
	target := mz.Rooms[diff]
	extra := len(res.Rooms) - target

	// Room count equals LvlMaze Rooms plus at most one added room per finisher
	// stamp (a stamp adds a room only when no dead end of that kind exists)
	// and, for catacombs, the start rooms are part of the target.
	if extra < 0 || extra > expectedStamps[id] {
		t.Errorf("level %d: %d rooms, LvlMaze says %d (+ up to %d stamps)", id, len(res.Rooms), target, expectedStamps[id])
	}

	base := typeBase[res.LevelType]

	for i, a := range res.Rooms {
		for j, b := range res.Rooms {
			if i < j && a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H {
				t.Errorf("level %d: rooms %d and %d overlap", id, i, j)
			}
		}

		doors := 0

		for _, k := range a.Links {
			b := res.Rooms[k]

			back := false
			for _, kk := range b.Links {
				if kk == i {
					back = true
				}
			}

			if !back {
				t.Errorf("level %d: link %d->%d is not mutual", id, i, k)
			}

			switch {
			case b.X+b.W == a.X && b.Y == a.Y:
				doors |= dirMask[West]
			case b.X == a.X+a.W && b.Y == a.Y:
				doors |= dirMask[East]
			case b.Y+b.H == a.Y && b.X == a.X:
				doors |= dirMask[North]
			case b.Y == a.Y+a.H && b.X == a.X:
				doors |= dirMask[South]
			default:
				t.Errorf("level %d: linked rooms %d,%d are not edge adjacent", id, i, k)
			}
		}

		if doors != a.Doors {
			t.Errorf("level %d room %d: door mask %d != geometry %d", id, i, a.Doors, doors)
		}

		// Plain rooms (not special, not theme upgraded) have Def = base + door mask.
		if !a.Locked && a.Def != base+a.Doors {
			t.Errorf("level %d room %d: Def %d != base %d + doors %d", id, i, a.Def, base, a.Doors)
		}
	}

	// connected
	seen := map[int]bool{0: true}
	queue := []int{0}

	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]

		for _, k := range res.Rooms[c].Links {
			if !seen[k] {
				seen[k] = true

				queue = append(queue, k)
			}
		}
	}

	if len(seen) != len(res.Rooms) {
		t.Errorf("level %d: only %d of %d rooms connected", id, len(seen), len(res.Rooms))
	}

	if res.MinX != 0 && res.MinX != func() int { l, _ := tb.Level(id); return l.OffsetX }() {
		t.Errorf("level %d: bbox min x %d != level offset", id, res.MinX)
	}
}

func countDefs(res *Result, defs ...int) int {
	n := 0

	for _, r := range res.Rooms {
		for _, d := range defs {
			if r.Def == d {
				n++
			}
		}
	}

	return n
}

func runAll(t *testing.T, tb *d2drlg.Tables) {
	for _, diff := range []d2drlg.Difficulty{d2drlg.Normal, d2drlg.Nightmare, d2drlg.Hell} {
		for _, id := range mazeLevels {
			for s := uint32(1); s <= 40; s++ {
				seed := s * 0x9E3779B1
				base, _ := d2rand.DrlgBaseSeed(seed)

				res, err := Generate(tb, Params{LevelID: id, Difficulty: diff, BaseSeed: base})
				if err != nil {
					t.Fatalf("level %d seed %#x: %v", id, seed, err)
				}

				checkInvariants(t, tb, res, id, diff)

				// finishers: each special Def exactly once (verified tables)
				switch id {
				case 8:
					if countDefs(res, 83, 84, 85, 86) != 1 || countDefs(res, 95, 96, 97, 98) != 1 {
						t.Errorf("DoE: prev/special counts wrong")
					}
				case 9:
					if countDefs(res, 83, 84, 85, 86) != 1 || countDefs(res, 91, 92, 93, 94) != 1 || countDefs(res, 99, 100, 101, 102) != 1 {
						t.Errorf("level 9 specials wrong")
					}
				case 29:
					if countDefs(res, 251, 249, 250, 248) != 1 || countDefs(res, 243, 241, 242, 240) != 1 {
						t.Errorf("jail 1 specials wrong")
					}
				case 31:
					if countDefs(res, 247, 245, 246, 244) != 1 || countDefs(res, 243, 241, 242, 240) != 0 {
						t.Errorf("jail 3 specials wrong")
					}
				}
			}
		}
	}
}

func TestInvariantsSynthetic(t *testing.T) { runAll(t, syntheticTables(t)) }

func TestInvariantsRealTables(t *testing.T) { runAll(t, realTables(t)) }

func TestDeterministicAndSeedSensitive(t *testing.T) {
	tb := syntheticTables(t)
	sig := func(seed uint32, id int) string {
		base, _ := d2rand.DrlgBaseSeed(seed)

		res, err := Generate(tb, Params{LevelID: id, BaseSeed: base})
		if err != nil {
			t.Fatal(err)
		}

		return fmt.Sprint(res.Rooms, res.Chunks)
	}

	distinct := map[string]bool{}

	for s := uint32(1); s <= 50; s++ {
		a, b := sig(s, 18), sig(s, 18)
		if a != b {
			t.Fatalf("seed %d not deterministic", s)
		}

		distinct[a] = true
	}

	if len(distinct) < 40 {
		t.Errorf("only %d distinct layouts from 50 seeds", len(distinct))
	}
}

// The Den of Evil has LvlMaze Rooms=1, so everything beyond the first room
// comes from the finisher: Prev + DOE special room => exactly 3 rooms, and the
// middle room is the one the two others are attached to (independently
// derivable from the tables, not from this code).
func TestDenOfEvilShape(t *testing.T) {
	tb := syntheticTables(t)

	for s := uint32(1); s < 30; s++ {
		base, _ := d2rand.DrlgBaseSeed(s)

		res, err := Generate(tb, Params{LevelID: 8, BaseSeed: base})
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Rooms) != 3 {
			t.Fatalf("DoE has %d rooms", len(res.Rooms))
		}

		if len(res.Chunks) != 27 { // 3 rooms of 24x24 tiled in 9 chunks of 8x8
			t.Fatalf("DoE has %d chunks", len(res.Chunks))
		}
	}
}

func TestUnsupportedType(t *testing.T) {
	tb := syntheticTables(t)
	if _, err := Generate(tb, Params{LevelID: 99}); err == nil {
		t.Fatal("expected error for unknown level")
	}
}
