package drlgoutdoor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// goldLogic is the numbers-only golden of the logic region lists (testdata/logic_regions.json): per level and room
// {x, y, w, h, type, whole-room flag, [[x0, y0, x1, y1, skip, id]...]}, the clipped rectangles of the real
// Game.exe's region list (Room1+0x64 -> +0x30), newest node first, without the zero sentinel node.
type goldLogic struct {
	Seed  uint32          `json:"seed"`
	Level int             `json:"level"`
	Diff  int             `json:"diff"`
	Rooms [][]interface{} `json:"rooms"`
}

func intsOf(v []interface{}) []int {
	out := make([]int, len(v))

	for i, x := range v {
		out[i] = int(x.(float64))
	}

	return out
}

// logicRooms builds the tiles of a level of any kind the way the golden was made.
func logicRooms(t *testing.T, env *Env, cache map[uint32]*drlgworld.Layout, seed uint32, id int) ([]*RoomTiles, error) {
	t.Helper()

	if rec, ok := env.Tables.Level(id); ok && rec.DrlgType == 1 {
		tb := env.Tables.(*d2drlg.Tables)
		base, _ := d2rand.DrlgBaseSeed(seed)
		params := drlgmaze.Params{LevelID: id, Difficulty: d2drlg.Normal, BaseSeed: base}

		if id >= 40 && id <= 74 {
			ex := d2drlg.DrawActExtras(seed, 1)
			params.TombA, params.TombB = ex.TombA, ex.TombB
		}

		res, err := drlgmaze.Generate(tb, params)
		if err != nil {
			return nil, err
		}

		ml, err := NewMazeLevel(env, res, id, seed)
		if err != nil {
			return nil, err
		}

		return ml.BuildTiles()
	}

	return tilesOfLevel(t, env, cache, seed, id)
}

// TestOracleLogicRegions compares the logic region lists of every room with the real game's (needs D2_TABLES and
// D2_DS1_ROOT; LOGIC_GOLDEN=<file> replaces the committed golden).
func TestOracleLogicRegions(t *testing.T) {
	env := testEnv(t)

	path := os.Getenv("LOGIC_GOLDEN")
	if path == "" {
		path = filepath.Join("..", "testdata", "logic_regions.json")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}

	var gold []goldLogic
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	cache := map[uint32]*drlgworld.Layout{}
	levels, rooms, bad, badID := 0, 0, 0, 0
	badLevels := map[int]int{}

	for _, g := range gold {
		rts, err := logicRooms(t, env, cache, g.Seed, g.Level)
		if err != nil {
			t.Errorf("seed %#x level %d: %v", g.Seed, g.Level, err)
			continue
		}

		if len(rts) != len(g.Rooms) {
			t.Errorf("seed %#x level %d: %d rooms, golden %d", g.Seed, g.Level, len(rts), len(g.Rooms))
			continue
		}

		levels++

		for i, gr := range g.Rooms {
			rooms++

			rt := rts[i]
			want := gr[6].([]interface{})

			var diff string

			// the room flag 0x800000 after the build (LvlPrest Populate = 0, or set by the build itself): the room
			// gets no natural monsters
			if len(gr) > 7 && (rt.Room.Flags&0x800000) != uint32(gr[7].(float64)) {
				diff = fmt.Sprintf("no-populate flag %#x, golden %#x", rt.Room.Flags&0x800000, uint32(gr[7].(float64)))
			}

			// the maze generator works in its own coordinates: compare relative to the room
			dx, dy := int(gr[0].(float64))-rt.Room.X, int(gr[1].(float64))-rt.Room.Y

			if len(rt.Logic) != len(want) {
				diff = fmt.Sprintf("%d regions, golden %d", len(rt.Logic), len(want))
			} else {
				for k, w := range want {
					wv, n := intsOf(w.([]interface{})), rt.Logic[k]
					skip := 0

					if n.Skip {
						skip = 1
					}

					ox, oy := dx, dy
					if n.X0 == 0 && n.X1 == 0 && n.Y0 == 0 && n.Y1 == 0 {
						ox, oy = 0, 0 // clipped away: all zero
					}

					if n.X0+ox != wv[0] || n.Y0+oy != wv[1] || n.X1+ox != wv[2] || n.Y1+oy != wv[3] || skip != wv[4] {
						diff = fmt.Sprintf("region %d: %v skip %v, golden %v", k, [4]int{n.X0, n.Y0, n.X1, n.Y1}, n.Skip, wv)
						break
					}

					if (n.ID == 0) != (wv[5] == 0) {
						badID++
					}
				}
			}

			if diff != "" {
				bad++
				badLevels[g.Level]++

				if bad <= 5 {
					t.Errorf("seed %#x level %d room %d (%d,%d %dx%d Def %d): %s", g.Seed, g.Level, i, rt.Room.X, rt.Room.Y, rt.Room.W, rt.Room.H, rt.Room.PrestDef, diff)
				}
			}
		}
	}

	t.Logf("%d levels, %d rooms, %d differ (by level %v), %d region ids zero/non-zero differ", levels, rooms, bad, badLevels, badID)

	if bad != 0 || badID != 0 {
		t.Fail()
	}
}
