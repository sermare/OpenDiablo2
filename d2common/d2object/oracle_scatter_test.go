package d2object

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// oracleField is the shared collision predicate of the emulator oracle (~/git/drlg-oracle/gen_objscatter.py).
type oracleField struct{}

func (oracleField) Blocked(x, y, w, h, mask int) bool {
	return (x*131+y*31+w*7+h*3+mask+(x>>2)*17)%7 == 0
}

type oracleRoom struct {
	X, Y, W, H int
	Seed       uint32
	Made       [][5]int
	Calls      [][2]int
	RoomSeed   [2]uint32 `json:"room_seed"`
	GameSeed   [2]uint32 `json:"game_seed"`
	Seen       int
	Shrine     int
	Large      int
	Small      int
	LargeXY    [][2]int `json:"large_xy"`
	SmallXY    [][2]int `json:"small_xy"`
}

type oracleLevel struct {
	Level     int
	Groups    [8]int
	Probs     [8]int
	Themes    int
	MonLvl1   int
	GameSeed0 [2]uint32 `json:"game_seed0"`
	Total     int
	Rooms     []oracleRoom
}

// TestOracleScatter replays OBJRGN_SpawnRoomObjects (run in the emulator on the real objects / objgroup / levels tables,
// golden numbers only) room by room for every level that has object groups and compares the objects made (id, x, y), the
// room and game seed after each room, and the region counters. Needs D2_TABLES for objects.txt and objgroup.txt.
func TestOracleScatter(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	gp := os.Getenv("ORACLE_OBJSCATTER")
	if gp == "" {
		gp = filepath.Join("testdata", "objscatter_oracle.json.gz")
	}

	f, err := os.Open(gp)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	var gold struct{ Levels []oracleLevel }
	if err := json.NewDecoder(zr).Decode(&gold); err != nil {
		t.Fatal(err)
	}

	od, err := os.ReadFile(filepath.Join(dir, "objects.txt"))
	if err != nil {
		t.Skip(err)
	}

	gd, err := os.ReadFile(filepath.Join(dir, "objgroup.txt"))
	if err != nil {
		t.Skip(err)
	}

	defs, err := ParseObjects(od)
	if err != nil {
		t.Fatal(err)
	}

	grps, err := ParseGroups(gd)
	if err != nil {
		t.Fatal(err)
	}

	groups := map[int]Group{}
	for _, g := range grps {
		groups[g.Offset] = g
	}

	bad, rooms, objs := 0, 0, 0
	byKey := map[string]*[2]int{}

	for _, lv := range gold.Levels {
		s := &Scatter{Defs: defs, Groups: groups, Game: &d2rand.Seed{Lo: lv.GameSeed0[0], Hi: lv.GameSeed0[1]},
			Region: &Region{Total: lv.Total}, Col: oracleField{}}
		row := LevelRow{ID: lv.Level, Groups: lv.Groups, Probs: lv.Probs, Themes: lv.Themes, MonLvl1: lv.MonLvl1}

		prev := oracleRoom{GameSeed: lv.GameSeed0}

		for k, r := range lv.Rooms {
			// every room starts from the game's own state before it (independent rooms: one wrong handler does not
			// shift the verdict of the following ones); the large / small lists are the game's own
			s.Game = &d2rand.Seed{Lo: prev.GameSeed[0], Hi: prev.GameSeed[1]}
			s.Region = &Region{Total: lv.Total, Seen: prev.Seen, Shrine: prev.Shrine}

			s.Region.Large = append(s.Region.Large, prev.LargeXY...)
			s.Region.Small = append(s.Region.Small, prev.SmallXY...)

			prev = r

			var fired, fns []int
			var goCalls [][2]int
			s.Trace = func(fn, id int) {
				fired = append(fired, fn*1000+id)
				fns = append(fns, fn)
				goCalls = append(goCalls, [2]int{fn, id})
			}
			rc := &RoomCtx{Rect: Rect{r.X, r.Y, r.W, r.H}, Seed: d2rand.New(r.Seed)}
			s.Made = nil
			var seeds [][2]int
			s.Create = func(Placed) { seeds = append(seeds, [2]int{int(s.Game.Lo), int(rc.Seed.Lo)}) }

			if _, err := s.Room(row, rc); err != nil {
				t.Fatalf("level %d room %d: %v", lv.Level, k, err)
			}

			rooms++
			objs += len(s.Made)

			var why string

			switch {
			case fmt.Sprint(goCalls) != fmt.Sprint(r.Calls) && !(len(goCalls) == 0 && len(r.Calls) == 0):
				why = fmt.Sprintf("handler calls go %v game %v", goCalls, r.Calls)
			case len(s.Made) != len(r.Made):
				why = "object count"
			case rc.Seed.Lo != r.RoomSeed[0] || rc.Seed.Hi != r.RoomSeed[1]:
				why = "room seed"
			case s.Game.Lo != r.GameSeed[0] || s.Game.Hi != r.GameSeed[1]:
				why = "game seed"
			case s.Region.Seen != r.Seen || s.Region.Shrine != r.Shrine || len(s.Region.Large) != r.Large || len(s.Region.Small) != r.Small:
				why = "region counters"
			default:
				for i, p := range s.Made {
					if p.ID != r.Made[i][0] || p.X != r.Made[i][1] || p.Y != r.Made[i][2] {
						why = "object position"

						break
					}
				}
			}

			key := fmt.Sprint(uniq(fns))
			if byKey[key] == nil {
				byKey[key] = &[2]int{}
			}

			if why != "" {
				byKey[key][1]++
			} else {
				byKey[key][0]++
			}

			if why != "" {
				bad++

				if dump := os.Getenv("ORACLE_DUMP"); dump == fmt.Sprintf("%d:%d", lv.Level, k) {
					t.Logf("go   %v\ngame %v\nrect %v", s.Made, r.Made, Rect{r.X, r.Y, r.W, r.H})
					for i := 0; i < len(s.Made) && i < len(r.Made); i++ {
						if s.Made[i].ID != r.Made[i][0] || s.Made[i].X != r.Made[i][1] || s.Made[i].Y != r.Made[i][2] || seeds[i] != [2]int{r.Made[i][3], r.Made[i][4]} {
							t.Logf("first difference at object %d: go %v seeds %v, game %v", i, s.Made[i], seeds[i], r.Made[i])

							break
						}
					}
				}

				if want := os.Getenv("ORACLE_KEY"); (want == "" && bad <= 12) || (want != "" && want == key) {
					t.Errorf("level %d room %d: %s differs (go %d objects, game %d; seen go %d game %d) handlers(fn*1000+id)=%v", lv.Level, k, why,
						len(s.Made), len(r.Made), s.Region.Seen, r.Seen, fired)
				}

				continue
			}
		}
	}

	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		t.Logf("PopulateFn set %-12s ok %3d  differ %3d", k, byKey[k][0], byKey[k][1])
	}

	if bad > 0 {
		t.Errorf("%d room(s) differ from the game", bad)
	}

	t.Logf("%d levels, %d rooms, %d objects compared, %d level(s) differ", len(gold.Levels), rooms, objs, bad)
}

func uniq(a []int) []int {
	sort.Ints(a)

	out := a[:0:0]
	for i, v := range a {
		if i == 0 || v != a[i-1] {
			out = append(out, v)
		}
	}

	return out
}
