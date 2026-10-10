package d2mapgen

import (
	"fmt"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
)

// exactTownTiles builds the tile records of a town the way Game.exe does (the
// preset rooms' DT1 libraries, rarity pick with the room seed, neighbour and
// border reuse): proven equal to the emulator's records for the Act 1 town
// (all four town files, 5 seeds), Lut Gholein (both files), Kurast Docks,
// the Pandemonium Fortress and Harrogath (docs/tile-diagnosis.md,
// TestTileDiffDir). It returns the records and the level rectangle.
func (g *MapGenerator) exactTownTiles(levelID int, seed uint32, diff d2drlg.Difficulty) ([]*drlgoutdoor.RoomTiles, drlgoutdoor.Rect, error) {
	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return nil, drlgoutdoor.Rect{}, err
	}

	env, err := outdoorEnv(g.asset)
	if err != nil {
		return nil, drlgoutdoor.Rect{}, err
	}

	switch levelID {
	case d2level.RogueEncampment:
		lay, err := drlgworld.Generate(tb, seed, diff)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		p, err := drlgoutdoor.ParamsFromLayout(tb, lay, levelID, seed)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		pl, err := drlgoutdoor.GeneratePreset(env, p, lay.TownFile)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		tiles, err := pl.BuildTiles()

		return tiles, pl.Rect, err
	case d2level.LutGholein:
		p, err := drlgoutdoor.ParamsAct23(tb, seed, diff, levelID)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		w, err := drlgoutdoor.PlaceAct2World(tb, seed, diff)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		lv, err := drlgoutdoor.GenerateTown(env, p, w.TownFile)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		tiles, err := lv.BuildTiles()

		return tiles, p.Rect, err
	case d2level.KurastDocks:
		p, err := drlgoutdoor.ParamsAct23(tb, seed, diff, levelID)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		pl, err := drlgoutdoor.GeneratePreset(env, p, -1)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		tiles, err := pl.BuildTiles()

		return tiles, pl.Rect, err
	case d2level.PandemoniumFortress, d2level.Harrogath:
		p, _, err := levelParams(tb, levelID, seed, diff)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		pl, err := drlgoutdoor.GeneratePreset(env, p, -1)
		if err != nil {
			return nil, drlgoutdoor.Rect{}, err
		}

		tiles, err := pl.BuildTiles()

		return tiles, pl.Rect, err
	}

	return nil, drlgoutdoor.Rect{}, fmt.Errorf("level %d is not a town", levelID)
}

// applyExactTown replaces the stamped DS1 tiles of the town (the stamp keeps
// providing the NPCs, objects and marker walls) by the exact records and logs
// the draw statistics (DRAWSTATS). On failure the stamped tiles stay.
func (g *MapGenerator) applyExactTown(levelID int, seed uint32, diff d2drlg.Difficulty, region d2enum.RegionIdType) {
	var (
		tiles []*drlgoutdoor.RoomTiles
		rect  drlgoutdoor.Rect
		err   error
	)

	if os.Getenv("OD2_TOWN_STAMP") == "1" { // debugging switch: the old stamp lookup, for before/after counts
		err = fmt.Errorf("OD2_TOWN_STAMP=1")
	} else {
		tiles, rect, err = g.exactTownTiles(levelID, seed, diff)
	}

	if err != nil {
		g.Infof("town tiles: level %d: exact records unavailable (%v); keeping the stamped DS1 tiles", levelID, err)
	} else {
		_, n := g.applyExactTiles(tiles, rect, region, true)
		g.Infof("town tiles: level %d: exact records for %d rooms (rect %dx%d)", levelID, n, rect.W, rect.H)
	}

	g.Infof("DRAWSTATS level=%d exact=%v %s", levelID, err == nil, g.engine.TileStats())
}
