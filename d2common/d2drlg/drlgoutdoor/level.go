package drlgoutdoor

import (
	"errors"
	"fmt"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Flag grid F cell bits (drlg3.md 1.3).
const (
	cellOccupied  = 0x1
	cellCliffSide = 0x2
	cellRoad      = 0x80
	cellNoRoom    = 0x100
	cellPreset    = 0x200
	cellExit      = 0x400
	cellWaypoint  = 0x800
	cellShrine    = 0x1000
	cellFileMask  = 0xf0000
	cellBlocked   = 0x1b81
)

// od.flags bits (drlg3.md 1.4).
const (
	FlagRiverEdge1  = 0x4
	FlagRiverEdge2  = 0x8
	FlagRiverEdge3  = 0x10
	FlagCliffCorner = 0x20
	FlagCavePlaced  = 0x40
	FlagTownTransS0 = 0x80
	FlagTownTransS1 = 0x100
	FlagTownTransE0 = 0x200
	FlagTownTransE1 = 0x400
)

// Rect is a rectangle in tiles.
type Rect struct{ X, Y, W, H int }

// Neighbor is one entry of the outdoor neighbour list (od+0x264).
type Neighbor struct {
	Level int
	Dir   int // side of THIS level: 0 W, 1 N, 2 E, 3 S
	F8    bool
	Flag  int
	Rect  Rect
}

// Params are the inputs of one level: everything the world search and the
// link registration decide before the level generator runs.
type Params struct {
	ID        int
	Rect      Rect
	Neighbors []Neighbor
	// Adjacent holds the rectangles of the outdoor levels linked through a vis slot without a warp, when the
	// neighbour list does not have them (Act 4/5, see ParamsFromLayout45); only read by markTownAdjacency.
	Adjacent  map[int]Rect
	Vis, Warp [8]int
	OdFlags   int
	BaseSeed  uint32
	Town      Rect // rectangle of level 1, used by the Blood Moor farthest-cave search

	// Flip is the Act 3 Kurast layout flip bit (drlg+0x474) and Jungle the
	// piece array of levels 76..78 (level+0x1bc / 0x1b8), both computed by the
	// Act 3 world placer (see PlaceAct3World).
	Flip   int
	Jungle JungleInfo
}

// JungleInfo is what the jungle placer stores in a jungle level: the 2x6 piece
// Defs (row-major) and the number of special ("clearing") pieces among them.
type JungleInfo struct {
	Arr   [12]int
	Count int
}

// DS1Loader returns the bytes of a DS1 file named like the tables name them
// (for example "Act1/Outdoors/BorderCliffs.ds1").
type DS1Loader func(file string) ([]byte, error)

// Env bundles the shared, read-only inputs of the generator.
type Env struct {
	Tables d2drlg.Source
	DS1    DS1Loader

	mu    sync.Mutex
	cache map[string]*Pattern
	dt1   map[string]*DT1
}

// NewEnv builds an Env.
func NewEnv(t d2drlg.Source, load DS1Loader) *Env {
	return &Env{Tables: t, DS1: load, cache: map[string]*Pattern{}, dt1: map[string]*DT1{}}
}

// Pattern loads and caches a DS1 pattern file.
func (e *Env) Pattern(file string) (*Pattern, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if p, ok := e.cache[file]; ok {
		return p, nil
	}

	if e.DS1 == nil {
		return nil, errors.New("drlgoutdoor: no DS1 loader")
	}

	b, err := e.DS1(file)
	if err != nil {
		return nil, fmt.Errorf("drlgoutdoor: %s: %w", file, err)
	}

	p, err := ParsePattern(b)
	if err != nil {
		return nil, fmt.Errorf("drlgoutdoor: %s: %w", file, err)
	}

	e.cache[file] = p

	return p, nil
}

// GameError is a fatal error of the original generator (its error code).
type GameError struct{ Code int }

func (e GameError) Error() string { return fmt.Sprintf("drlgoutdoor: game fatal error %#x", e.Code) }

// Vertex is one polygon vertex (drlg3.md 1.2, +0x64 ring).
type Vertex struct {
	X, Y int
	B    int // "cliff side" byte
	F    int // bit 0: starts an exit segment, bit 1: exit leads to a preset level
	Next *Vertex
}

// Trace is the level seed and od.flags at the entry of a generator stage.
type Trace struct {
	Name    string
	Lo, Hi  uint32
	OdFlags int
}

// River is the road network data of a level (od+0x80.. arrays and paths).
type River struct {
	N     int
	Ends  [][3]int // raw endpoints (x, y, type)
	Start [][2]int // snapped start points
	End   [][2]int // snapped end points
	Junc  [][3]int // raw junctions
	Paths [][][2]int
}

// Room is one room of the cell-to-room pass.
type Room struct {
	Type       int // 1 plain outdoor room, 2 preset room
	X, Y, W, H int
	S4         uint32
	Seed       d2rand.Seed // room seed after the sub-theme mask roll
	Flags      uint32
	R50        uint32 // tile library Dt1Mask
	Info54     uint32
	Info58     uint32
	SubType    int
	SubTheme   int
	Mask       uint32
	PrestDef   int
	File       int
	// PrestX, PrestY, PrestW and PrestH are the rectangle of the preset map the
	// room is a chunk of (the DS1 origin and size).
	PrestX, PrestY, PrestW, PrestH int
	// GateSeed is the level seed when the preset's DS1 was loaded and
	// GateSteps the number of gated draws taken from it (preset rooms).
	GateSeed  d2rand.Seed
	GateSteps int
}

type counter struct{ n, ctr int }

// Level is the generated Act 1 outdoor level.
type Level struct {
	Params Params
	W, H   int // size in cells
	LType  int

	Def, GridB, Flag, GridD *Grid

	OdFlags int
	Seed    *d2rand.Seed // level seed (after rooms once CreateRooms ran)
	Poly    *Vertex
	River   River
	Rooms   []*Room

	// PolygonAtAct is the boundary polygon right after scaling to cells.
	PolygonAtAct []Vertex
	// Trace has the seed at the entry of every stage; the last entry, "end",
	// is the state after the specials, before the room pass.
	Trace []Trace
	// Counters are the per-Def preset file counters (n files, current).
	Counters map[int][2]int

	env  *Env
	ctr  map[int]*counter
	town Rect
	err  error
	sub  *subCallbacks // custom LvlSub cell rules (Act 5 barricades)

	// presetSize overrides the Def size in placePresetRooms (town levels).
	presetSize [2]int

	// logicCtr is the level's logic region id counter (level +0x1dc).
	logicCtr int
}

// Level accessors for callers.
func (l *Level) ID() int { return l.Params.ID }

func (l *Level) trace(name string) {
	l.Trace = append(l.Trace, Trace{name, l.Seed.Lo, l.Seed.Hi, l.OdFlags})
}

func cdiv(a, b int) int { return a / b } // Go's integer division truncates like C

func sgn(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}

	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

// sar3 is (v + (v<0 ? 7 : 0)) >> 3 with an arithmetic shift.
func sar3(v int) int {
	if v < 0 {
		v += 7
	}

	return v >> 3
}

// min and max shadow the Go 1.21 builtins, which the module's language level
// (go 1.16) does not allow.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
