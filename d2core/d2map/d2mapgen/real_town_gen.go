package d2mapgen

import (
	"fmt"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
)

// HeroDifficulty is the difficulty of the loaded hero; the Act 1 world layout
// (level sizes) depends on it. The app sets it with HeroMapSeed.
var HeroDifficulty d2drlg.Difficulty

// worldRects converts the placement of the Act 1 world search into the level
// rectangles the edge crossing works on.
func worldRects(lay *drlgworld.Layout) map[int]d2level.Rect {
	if lay == nil {
		return nil
	}

	out := make(map[int]d2level.Rect, len(lay.Levels))

	for id, p := range lay.Levels {
		out[id] = d2level.Rect{X: p.Rect.X, Y: p.Rect.Y, W: p.Rect.W, H: p.Rect.H}
	}

	return out
}

// generateRealTown builds the Rogue Encampment for OD2_REALMAPS=1: the town
// preset the world layout chose (TownN1/E1/S1/W1 by the side on which Blood
// Moor lies), alone in a map of the town's level rectangle. Everything around
// the preset is blocked; the way out is the open border towards Blood Moor
// (d2level.EdgeExit), where the original's town preset flows into the Blood
// Moor level.
func (g *MapGenerator) generateRealTown() error {
	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	lay, err := drlgworld.Generate(tb, HeroMapSeed, HeroDifficulty)
	if err != nil {
		return err
	}

	pl, ok := lay.Levels[d2level.RogueEncampment]
	if !ok {
		return fmt.Errorf("the world layout has no town")
	}

	for _, file := range g.asset.Records.LevelPreset(presetB).Files {
		g.engine.AddDS1(file)
	}

	stamp := g.engine.LoadStamp(d2enum.RegionAct1Town, presetB, lay.TownFile)
	if stamp == nil {
		return fmt.Errorf("town preset %d could not be loaded", lay.TownFile)
	}

	size := stamp.Size()
	w, h := pl.Rect.W, pl.Rect.H

	if size.Width > w {
		w = size.Width
	}

	if size.Height > h {
		h = size.Height
	}

	g.engine.ResetMap(d2enum.RegionAct1Town, w, h)
	g.engine.PlaceStamp(stamp, 0, 0)
	g.applyExactTown(d2level.RogueEncampment, HeroMapSeed, HeroDifficulty, d2enum.RegionAct1Town)
	g.engine.BlockEmptyTiles()
	g.engine.UseCollisionPaths(true)
	g.engine.SetWorld(d2mapengine.World{
		Level: d2level.RogueEncampment, OriginX: pl.Rect.X, OriginY: pl.Rect.Y, Rects: worldRects(lay),
	})

	g.Infof("real town: %s (file index %d), preset %dx%d tiles, level rectangle %dx%d at world (%d,%d)",
		stamp.RegionPath(), lay.TownFile, size.Width, size.Height, pl.Rect.W, pl.Rect.H, pl.Rect.X, pl.Rect.Y)

	if os.Getenv("OD2_AUTOMAP_ASCII") != "" {
		x, y := g.engine.GetStartPosition()
		g.logWalkMap(x, y)
	}

	return nil
}

// act23Rects returns the level rectangles of the Act 2 or Act 3 world a level
// belongs to (nil if the world cannot be placed), so that the walk across the
// seamless borders of the desert levels works like in Act 1.
func act23Rects(tb *d2drlg.Tables, levelID int, seed uint32, diff d2drlg.Difficulty) map[int]d2level.Rect {
	var (
		w   *drlgoutdoor.World23
		err error
	)

	if levelID >= 40 && levelID <= 46 {
		w, err = drlgoutdoor.PlaceAct2World(tb, seed, diff)
	} else {
		w, err = drlgoutdoor.PlaceAct3World(tb, seed, diff)
	}

	if err != nil {
		return nil
	}

	out := make(map[int]d2level.Rect, len(w.Rects))
	for id, r := range w.Rects {
		out[id] = d2level.Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
	}

	return out
}
