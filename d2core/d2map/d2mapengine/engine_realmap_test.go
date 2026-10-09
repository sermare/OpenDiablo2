package d2mapengine

import (
	"bytes"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

// testEngine builds a w x h engine without any game data. Every tile gets a
// floor unless ground is false for it.
func testEngine(w, h int, ground func(x, y int) bool) (*MapEngine, *bytes.Buffer) {
	logs := &bytes.Buffer{}
	l := d2util.NewLogger()
	l.Writer = logs

	m := &MapEngine{Logger: l}
	m.size.Width, m.size.Height = w, h
	m.tiles = make([]MapTile, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ground(x, y) {
				f := d2ds1.Tile{}
				f.Prop1 = 1
				m.tiles[x+y*w].Components = d2mapstamp.Tile{Floors: []d2ds1.Tile{f}}
			}
		}
	}

	return m, logs
}

func TestBlockEmptyTilesAndOutsideIsWall(t *testing.T) {
	m, _ := testEngine(4, 4, func(x, y int) bool { return x < 2 })
	m.BlockEmptyTiles()

	tests := []struct {
		name        string
		subX, subY  int
		wantBlocked bool
	}{
		{"ground tile", 3, 3, false},
		{"void tile", 12, 3, true},
		{"negative is a wall", -1, 0, true},
		{"beyond the map is a wall", 4 * 5, 0, true},
		{"beyond the map (y)", 0, 400, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := m.SubTileAt(tt.subX, tt.subY).BlockWalk; got != tt.wantBlocked {
				t.Fatalf("BlockWalk=%v want %v", got, tt.wantBlocked)
			}
		})
	}
}

func TestPrepareTileSkipsEmptyAndBlankFloors(t *testing.T) {
	m, logs := testEngine(1, 1, func(x, y int) bool { return false })

	empty := d2ds1.Tile{}
	blank := d2ds1.Tile{}
	blank.Prop1, blank.Style = 1, blankFloorStyle // no DT1 has a graphic for 30/0

	tile := &m.tiles[0]
	tile.Components = d2mapstamp.Tile{
		Floors:  []d2ds1.Tile{empty, blank},
		Walls:   []d2ds1.Tile{empty},
		Shadows: []d2ds1.Tile{empty},
	}

	tile.PrepareTile(0, 0, m)

	if strings.Contains(logs.String(), "Unknown tile") {
		t.Fatalf("empty / blank cells must not be looked up:\n%s", logs)
	}

	if !tile.Components.Floors[1].Hidden() {
		t.Fatal("blank floor 30/0 should be hidden")
	}

	// a real floor that is missing from the DT1 files is still reported
	missing := d2ds1.Tile{}
	missing.Prop1, missing.Style, missing.Sequence = 1, 5, 7
	tile.Components.Floors = []d2ds1.Tile{missing}
	tile.PrepareTile(0, 0, m)

	if !strings.Contains(logs.String(), "Unknown tile ID [5 7 0]") {
		t.Fatalf("a missing floor tile should still warn, log:\n%s", logs)
	}
}

func TestPathFindAroundWalls(t *testing.T) {
	// a wall across the middle with a gap at the top; the hero is below it
	m, _ := testEngine(6, 6, func(x, y int) bool { return true })

	for x := 0; x < 6; x++ {
		if x == 5 {
			continue
		}

		for s := range m.tiles[x+3*6].SubTiles {
			m.tiles[x+3*6].SubTiles[s] = d2dt1.SubTileFlags{BlockWalk: true}
		}
	}

	m.UseCollisionPaths(true)

	start := d2vector.NewPosition(2, 20) // sub-tile coordinates, below the wall
	dest := d2vector.NewPosition(2, 5)   // above it

	path := m.PathFind(start, dest)
	if len(path) < 2 {
		t.Fatalf("want a detour with corners through the gap, got %v", path)
	}

	last := path[len(path)-1]
	if last.X() != dest.X() || last.Y() != dest.Y() {
		t.Fatalf("path ends at (%v,%v), want (%v,%v)", last.X(), last.Y(), dest.X(), dest.Y())
	}

	// without collision paths the hero just stops at the wall
	m.UseCollisionPaths(false)

	if p := m.PathFind(start, dest); len(p) != 1 || p[0].Y() <= 15 {
		t.Fatalf("straight-line mode should stop at the wall, got %v", p)
	}
}

func TestStartPositionOverride(t *testing.T) {
	m, _ := testEngine(2, 2, func(x, y int) bool { return true })

	if x, y := m.GetStartPosition(); x != 1 || y != 1 {
		t.Fatalf("default start is the map centre, got (%v,%v)", x, y)
	}

	m.SetStartPosition(1.05, 0.05)

	if x, y := m.GetStartPosition(); x != 1.05 || y != 0.05 {
		t.Fatalf("override ignored: (%v,%v)", x, y)
	}
}
