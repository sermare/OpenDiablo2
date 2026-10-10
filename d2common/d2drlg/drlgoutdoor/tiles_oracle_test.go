package drlgoutdoor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
)

// The tile golden is produced by running the real Game.exe room tile build in
// the oracle (~/git/drlg-oracle/gen_tiles.py + testdata/gen_tiles_compact.py):
// per level, for every plain room in creation order, the final room seed, the
// record counts and a digest over all tile records (x, y, orientation, flags,
// DT1 file, tile index).

// goldTileRoom is [x, y, seedLo, seedHi, digest, nWalls, nFloors, nShadows].
type goldTileRoom struct {
	X, Y  int
	Seed  [2]uint32
	Dig   string
	N     [3]int
	Tiles map[string][][]interface{} // only in the "full" part
}

func (g *goldTileRoom) UnmarshalJSON(b []byte) error {
	var a []json.RawMessage
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}

	if len(a) != 8 {
		return fmt.Errorf("tile golden room has %d fields", len(a))
	}

	var n [8]float64

	for i, f := range a {
		if i == 4 {
			if err := json.Unmarshal(f, &g.Dig); err != nil {
				return err
			}

			continue
		}

		if err := json.Unmarshal(f, &n[i]); err != nil {
			return err
		}
	}

	g.X, g.Y = int(n[0]), int(n[1])
	g.Seed = [2]uint32{uint32(n[2]), uint32(n[3])}
	g.N = [3]int{int(n[5]), int(n[6]), int(n[7])}

	return nil
}

type goldTileLevel struct {
	Seed    uint32         `json:"seed"`
	Level   int            `json:"level"`
	Rooms   []goldTileRoom `json:"rooms"`
	Presets []goldTileRoom `json:"presets"`
	Full    *struct {
		Rooms   []map[string][][]interface{} `json:"rooms"`
		Presets []map[string][][]interface{} `json:"presets"`
	} `json:"full"`
}

func canonRoom(rt *RoomTiles) string {
	var sb strings.Builder

	for _, k := range []struct {
		name string
		l    []*TileRecord
	}{{"walls", rt.Walls}, {"floors", rt.Floors}, {"shadows", rt.Shadows}} {
		sb.WriteString(k.name + ":")

		for i, r := range k.l {
			if i > 0 {
				sb.WriteString(";")
			}

			fmt.Fprintf(&sb, "%d,%d,%d,%x,%s,%d", r.X, r.Y, r.Ori, r.Flags, r.Tile.File(), r.Tile.Idx)
		}

		sb.WriteString("|")
	}

	fmt.Fprintf(&sb, "%08x,%08x", rt.Seed.Lo, rt.Seed.Hi)

	return sb.String()
}

func digestRoom(rt *RoomTiles) string {
	h := sha256.Sum256([]byte(canonRoom(rt)))
	return hex.EncodeToString(h[:])[:12]
}

func TestOracleTiles(t *testing.T) {
	env := testEnv(t)

	files := []string{"tiles_act1.json", "tiles_act23.json", "tiles_act45.json", "tiles_towns.json", "tiles_presets.json", "tiles_presets7.json"}
	if gp := os.Getenv("ORACLE_TILES"); gp != "" {
		files = strings.Split(gp, ",")
	}

	var gold []goldTileLevel

	for _, f := range files {
		gp := f
		if !strings.Contains(f, "/") {
			gp = filepath.Join("..", "testdata", f)
		}

		b, err := os.ReadFile(gp)
		if err != nil {
			t.Skip(err)
		}

		var g []goldTileLevel
		if err := json.Unmarshal(b, &g); err != nil {
			t.Fatal(err)
		}

		gold = append(gold, g...)
	}

	cache := map[uint32]*drlgworld.Layout{}
	rooms, bad := 0, 0
	badType := [2]int{}

	for _, gl := range gold {
		name := fmt.Sprintf("seed %#x level %d", gl.Seed, gl.Level)

		res, err := tilesOfLevel(t, env, cache, gl.Seed, gl.Level)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			bad++

			continue
		}

		if gl.Full != nil {
			for i := range gl.Rooms {
				gl.Rooms[i].Tiles = gl.Full.Rooms[i]
			}

			for i := range gl.Presets {
				gl.Presets[i].Tiles = gl.Full.Presets[i]
			}
		}

		for typ, list := range [][]goldTileRoom{gl.Rooms, gl.Presets} {
			var mine []*RoomTiles

			for _, rt := range res {
				if rt != nil && rt.Room.Type == typ+1 {
					mine = append(mine, rt)
				}
			}

			if gl.Presets == nil && typ == 1 {
				break
			}

			if len(mine) != len(list) {
				t.Errorf("%s: %d rooms of type %d, golden has %d", name, len(mine), typ+1, len(list))
				bad++

				continue
			}

			for i, rt := range mine {
				g := list[i]
				rooms++

				if rt.Room.X != g.X || rt.Room.Y != g.Y {
					t.Errorf("%s room %d: at %d,%d, golden %d,%d", name, i, rt.Room.X, rt.Room.Y, g.X, g.Y)
					bad++

					continue
				}

				seedOK := rt.Seed.Lo == g.Seed[0] && rt.Seed.Hi == g.Seed[1]
				n := [3]int{len(rt.Walls), len(rt.Floors), len(rt.Shadows)}

				if digestRoom(rt) == g.Dig && seedOK && n == g.N {
					continue
				}

				bad++
				badType[typ]++

				if badType[typ] <= 4 {
					t.Errorf("%s type %d room %d (%d,%d): mismatch seedOK=%v counts %v want %v", name, typ+1, i, g.X, g.Y, seedOK, n, g.N)
					explain(t, rt, g)
				}
			}
		}
	}

	t.Logf("%d levels, %d rooms (plain + preset) compared, %d mismatches", len(gold), rooms, bad)
}

func explain(t *testing.T, rt *RoomTiles, g goldTileRoom) {
	if g.Tiles == nil {
		return
	}

	for _, k := range []struct {
		name string
		l    []*TileRecord
	}{{"walls", rt.Walls}, {"floors", rt.Floors}, {"shadows", rt.Shadows}} {
		gl := g.Tiles[k.name]

		for i, r := range k.l {
			if i >= len(gl) {
				break
			}

			parts := make([]string, len(gl[i]))
			for j, v := range gl[i] {
				if f, ok := v.(float64); ok {
					parts[j] = fmt.Sprintf("%.0f", f)
				} else {
					parts[j] = fmt.Sprint(v)
				}
			}

			want := strings.Join(parts, " ")
			got := fmt.Sprint(r.X, " ", r.Y, " ", r.Ori, " ", r.Flags, " ", r.Tile.File(), " ", r.Tile.Idx)

			if want != got {
				t.Logf("  %s[%d]: got %s want %s", k.name, i, got, want)
				break
			}
		}
	}
}

// paramsOfLevel derives the generator inputs of a level of any act (Normal
// difficulty) the way the golden was made.
func paramsOfLevel(t *testing.T, env *Env, cache map[uint32]*drlgworld.Layout, seed uint32, id int) (Params, error) {
	t.Helper()

	var (
		p   Params
		err error
	)

	switch {
	case id < 40:
		lay := cache[seed]
		if lay == nil {
			if lay, err = drlgworld.Generate(env.Tables, seed, d2drlg.Normal); err != nil {
				return p, err
			}

			cache[seed] = lay
		}

		p, err = ParamsFromLayout(env.Tables, lay, id, seed)
	case id < 103:
		p, err = ParamsAct23(env.Tables, seed, d2drlg.Normal, id)
	default:
		act := 3
		gen := drlgworld.GenerateAct4

		if id >= 109 {
			act, gen = 4, drlgworld.GenerateAct5
		}

		lay, e := gen(env.Tables, seed, d2drlg.Normal)
		if e != nil {
			return p, e
		}

		p, err = ParamsFromLayout45(env.Tables, lay, act, id, seed, d2drlg.Normal)
	}

	if err != nil {
		// DrlgType 2 levels outside every world layout (Act 1 treasure caves,
		// the maze-like presets): rectangle, vis and warp from the level record
		if rec, ok := env.Tables.Level(id); ok && rec.DrlgType == 2 {
			return ParamsPreset(env.Tables, id, seed, d2drlg.Normal)
		}
	}

	return p, err
}

// tilesOfLevel generates a level of any act (Normal difficulty) the way the
// golden was made and builds its tiles.
func tilesOfLevel(t *testing.T, env *Env, cache map[uint32]*drlgworld.Layout, seed uint32, id int) ([]*RoomTiles, error) {
	t.Helper()

	p, err := paramsOfLevel(t, env, cache, seed, id)
	if err != nil {
		return nil, err
	}

	if tb, ok := env.Tables.(*d2drlg.Tables); ok && id == 40 {
		// Lut Gholein: the file slot comes from the Act 2 world (LutW / LutN)
		w, err := PlaceAct2World(tb, seed, d2drlg.Normal)
		if err != nil {
			return nil, err
		}

		lv, err := GenerateTown(env, p, w.TownFile)
		if err != nil {
			return nil, err
		}

		return lv.BuildTiles()
	}

	if rec, ok := env.Tables.Level(id); ok && rec.DrlgType == 2 {
		override := -1

		if id == 1 || id == 27 { // the Act 1 world forces the town file (TownN1/E1/S1/W1) and Courtyard 1's file
			lay := cache[seed]
			if lay == nil {
				return nil, fmt.Errorf("no world layout for seed %#x", seed)
			}

			override = PresetFileOverride(lay, id)
		}

		pl, err := GeneratePreset(env, p, override)
		if err != nil {
			return nil, err
		}

		return pl.BuildTiles()
	}

	lv, err := Generate(env, p)
	if err != nil {
		return nil, err
	}

	return lv.BuildTiles()
}
