package d2mapgen

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// LevelProvider builds one kind of level into the map engine of a generator.
// The level-change layer asks the providers in order which can load a level, so
// a new generator (for example the full DRLG renderer worked on in
// feat/realmaps-render) is added with RegisterProvider without touching the
// transition code.
type LevelProvider interface {
	// Name is used in logs.
	Name() string
	// CanLoad reports whether the provider can build the level now.
	CanLoad(levelID int) bool
	// Load replaces the contents of the generator's engine with the level.
	Load(g *MapGenerator, levelID int, req LoadRequest) error
}

// LoadRequest carries what a provider needs to build a level.
type LoadRequest struct {
	Seed       uint32
	Difficulty d2drlg.Difficulty
}

// Arrival is where a hero is placed after a level was built, in tiles.
type Arrival struct {
	X, Y float64
}

// RegisterProvider adds a provider; providers registered later are asked first,
// so they can override the built-in ones.
func (g *MapGenerator) RegisterProvider(p LevelProvider) {
	g.providers = append([]LevelProvider{p}, g.providers...)
}

// ProviderFor returns the first provider that can load the level, or nil.
func (g *MapGenerator) ProviderFor(levelID int) LevelProvider {
	for _, p := range g.providers {
		if p.CanLoad(levelID) {
			return p
		}
	}

	return nil
}

// CanLoadLevel reports whether any provider can build the level.
func (g *MapGenerator) CanLoadLevel(levelID int) bool { return g.ProviderFor(levelID) != nil }

// LoadLevel builds a level into the engine and returns the arrival point
// (the engine's start position for it, moved to the nearest walkable
// sub-tile).
func (g *MapGenerator) LoadLevel(levelID int, req LoadRequest) (arrival Arrival, err error) {
	p := g.ProviderFor(levelID)
	if p == nil {
		return Arrival{}, fmt.Errorf("no level provider for level %d", levelID)
	}

	// a provider that trips over bad data must not take the game down
	defer func() {
		if r := recover(); r != nil {
			arrival, err = Arrival{}, fmt.Errorf("%s: level %d: panic: %v", p.Name(), levelID, r)
		}
	}()

	if err := p.Load(g, levelID, req); err != nil {
		return Arrival{}, fmt.Errorf("%s: level %d: %w", p.Name(), levelID, err)
	}

	x, y := g.engine.GetStartPosition()

	if sx, sy, ok := g.engine.NearestOpen(int(x*subtilesPerTile), int(y*subtilesPerTile), arrivalSearchRadius); ok {
		x, y = (float64(sx)+0.5)/subtilesPerTile, (float64(sy)+0.5)/subtilesPerTile
	}

	g.Infof("loaded level %d with provider %s, arrival (%.1f,%.1f)", levelID, p.Name(), x, y)

	return Arrival{X: x, Y: y}, nil
}

const (
	arrivalSearchRadius = 0x32 // the original searches 0x32 around the target
)

// townProvider builds the Act 1 town (Rogue Encampment) with the old overworld
// generator, which is the only preset-based level the engine can build.
type townProvider struct{}

func (townProvider) Name() string { return "act1-town" }

func (townProvider) CanLoad(levelID int) bool { return levelID == d2level.RogueEncampment }

func (townProvider) Load(g *MapGenerator, _ int, _ LoadRequest) error {
	g.GenerateAct1Overworld()
	return nil
}

// maxMazeLevel is the last level id the maze provider tries: Act 1 (caves, crypts, jail, catacombs)
// Act 2 (sewers, palace, tombs, lair, arcane sanctuary; the exe's DRLG_ port: drlgmaze/maze_act23.go)
// and Act 3 (spider caves, flayer dungeons, sewers, temples, Durance of Hate).
const maxMazeLevel = 102

// mazeProvider builds Act 1 maze levels (caves, crypts, jail, catacombs) with
// the DRLG port. It is only active with OD2_REALMAPS=1.
type mazeProvider struct {
	tables *d2drlg.Tables
	ok     map[int]bool
}

func (mazeProvider) Name() string { return "drlg-maze" }

func (p *mazeProvider) CanLoad(levelID int) bool {
	if !RealMapsEnabled() || levelID < 2 || levelID > maxMazeLevel {
		return false
	}

	if v, seen := p.ok[levelID]; seen {
		return v
	}

	return false
}

// probe fills the supported set by trying to generate every Act 1 level once.
func (p *mazeProvider) probe(tb *d2drlg.Tables) {
	p.tables, p.ok = tb, map[int]bool{}

	base, _ := d2rand.DrlgBaseSeed(0)

	for id := 2; id <= maxMazeLevel; id++ {
		_, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: id, BaseSeed: base})
		p.ok[id] = err == nil
	}
}

func (*mazeProvider) Load(g *MapGenerator, levelID int, req LoadRequest) error {
	return g.GenerateRealMaze(levelID, req.Seed, req.Difficulty)
}

// installDefaultProviders registers the built-in providers. The maze provider
// needs the DRLG tables, which come from the archives; if they cannot be read
// it stays inactive.
func (g *MapGenerator) installDefaultProviders() {
	g.providers = []LevelProvider{actTownProvider{}, townProvider{}}

	if !RealMapsEnabled() {
		return
	}

	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		g.Warningf("maze levels unavailable: %v", err)
		return
	}

	mp := &mazeProvider{}
	mp.probe(tb)
	g.providers = append([]LevelProvider{mp}, g.providers...)

	// Act 1 wilderness levels (proven equal to the real game down to the room
	// grids, see drlgoutdoor); the tile records are approximated
	g.providers = append([]LevelProvider{outdoorProvider{}, presetProvider{}}, g.providers...)
}
