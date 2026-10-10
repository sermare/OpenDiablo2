package d2mapengine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// TileStats counts what the renderer will find on the map: how many cells have
// a floor, and how many tile references (floor, wall, shadow) have no DT1 tile
// (the renderer logs "Could not locate tile" for those and draws a 10x10
// placeholder). NoFloor are the cells that draw no ground at all: they show as
// black voids (BlockEmptyTiles also makes them unwalkable).
type TileStats struct {
	Cells, NoFloor                           int
	Floors, Walls, Shadows                   int
	FloorMissing, WallMissing, ShadowMissing int
	WallSpecial                              int
	MissingKeys                              map[string]int
}

// Fallbacks is the number of tile references the renderer cannot resolve.
func (s TileStats) Fallbacks() int { return s.FloorMissing + s.WallMissing + s.ShadowMissing }

// String is the DRAWSTATS log line body.
func (s TileStats) String() string {
	keys := make([]string, 0, len(s.MissingKeys))
	for k := range s.MissingKeys {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var sb strings.Builder

	for i, k := range keys {
		if i == 20 {
			sb.WriteString(" ...")
			break
		}

		fmt.Fprintf(&sb, " %s=%d", k, s.MissingKeys[k])
	}

	return fmt.Sprintf("cells=%d nofloor=%d floors=%d walls=%d shadows=%d special=%d fallbacks=%d (floor %d wall %d shadow %d)%s",
		s.Cells, s.NoFloor, s.Floors, s.Walls, s.Shadows, s.WallSpecial, s.Fallbacks(), s.FloorMissing, s.WallMissing, s.ShadowMissing, sb.String())
}

func (m *MapEngine) hasTile(style, seq, typ int32) bool {
	for i := range m.dt1TileData {
		t := &m.dt1TileData[i]
		if t.Style == style && t.Sequence == seq && t.Type == typ {
			return true
		}
	}

	return false
}

// TileStats walks the map cells like the renderer's tile cache does and counts
// the drawable floors, walls and shadows and the references without a DT1 tile.
func (m *MapEngine) TileStats() TileStats {
	s := TileStats{MissingKeys: map[string]int{}}
	known := map[[3]int32]bool{}

	has := func(style, seq, typ int32) bool {
		k := [3]int32{style, seq, typ}
		if v, ok := known[k]; ok {
			return v
		}

		v := m.hasTile(style, seq, typ)
		known[k] = v

		return v
	}

	for i := range m.tiles {
		c := &m.tiles[i].Components
		s.Cells++
		floors := 0

		for j := range c.Floors {
			f := &c.Floors[j]
			if f.Hidden() || f.Prop1 == 0 {
				continue
			}

			s.Floors++

			if !has(int32(f.Style), int32(f.Sequence), 0) {
				s.FloorMissing++
				s.MissingKeys[fmt.Sprintf("floor:%d/%d", f.Style, f.Sequence)]++

				continue
			}

			floors++
		}

		if floors == 0 {
			s.NoFloor++
		}

		for j := range c.Walls {
			w := &c.Walls[j]
			if w.Hidden() || w.Prop1 == 0 {
				continue
			}

			if w.Type.Special() {
				s.WallSpecial++
				continue
			}

			s.Walls++

			if !has(int32(w.Style), int32(w.Sequence), int32(w.Type)) {
				s.WallMissing++
				s.MissingKeys[fmt.Sprintf("wall:%d/%d/%d", w.Style, w.Sequence, w.Type)]++
			}
		}

		for j := range c.Shadows {
			sh := &c.Shadows[j]
			if sh.Hidden() || sh.Prop1 == 0 {
				continue
			}

			s.Shadows++

			if !has(int32(sh.Style), int32(sh.Sequence), int32(d2enum.TileShadow)) {
				s.ShadowMissing++
				s.MissingKeys[fmt.Sprintf("shadow:%d/%d", sh.Style, sh.Sequence)]++
			}
		}
	}

	return s
}
