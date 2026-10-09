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

	gp := os.Getenv("ORACLE_TILES")
	if gp == "" {
		gp = filepath.Join("..", "testdata", "tiles_act1.json")
	}

	b, err := os.ReadFile(gp)
	if err != nil {
		t.Skip(err)
	}

	var gold []goldTileLevel
	if err := json.Unmarshal(b, &gold); err != nil {
		t.Fatal(err)
	}

	cache := map[uint32]*drlgworld.Layout{}
	rooms, bad := 0, 0
	badType := [2]int{}

	for _, gl := range gold {
		name := fmt.Sprintf("seed %#x level %d", gl.Seed, gl.Level)

		lay := cache[gl.Seed]
		if lay == nil {
			lay, err = drlgworld.Generate(env.Tables, gl.Seed, d2drlg.Normal)
			if err != nil {
				t.Fatal(err)
			}

			cache[gl.Seed] = lay
		}

		p, err := ParamsFromLayout(env.Tables, lay, gl.Level, gl.Seed)
		if err != nil {
			t.Fatal(err)
		}

		lv, err := Generate(env, p)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		res, err := lv.BuildTiles()
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
