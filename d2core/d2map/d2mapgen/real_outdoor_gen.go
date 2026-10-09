package d2mapgen

import (
	"fmt"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

// isOutdoorLevel reports the outdoor levels the drlgoutdoor port covers: the
// Act 1 wilderness (Blood Moor .. Tamoe Highland, Burial Grounds, Moo Moo
// Farm), the Act 2 desert (41..46) and the Act 3 jungle and Kurast (76..83).
func isOutdoorLevel(id int) bool {
	return (id >= 2 && id <= 7) || id == 0x11 || id == 0x27 || isAct23Outdoor(id)
}

func isAct23Outdoor(id int) bool { return (id >= 41 && id <= 46) || (id >= 76 && id <= 83) }

// outdoorProvider builds Act 1 wilderness levels with the DRLG port. Like the
// maze provider it is only active with OD2_REALMAPS=1.
type outdoorProvider struct{}

func (outdoorProvider) Name() string { return "drlg-outdoor" }

func (outdoorProvider) CanLoad(levelID int) bool { return RealMapsEnabled() && isOutdoorLevel(levelID) }

func (outdoorProvider) Load(g *MapGenerator, levelID int, req LoadRequest) error {
	return g.GenerateRealOutdoor(levelID, req.Seed, req.Difficulty)
}

// GenerateRealOutdoor replaces the map with the DRLG outdoor level for the
// hero's seed: the world layout decides the level rectangle and its exits,
// drlgoutdoor generates the cell grids and the room list (proven equal to the
// real game's), every preset room's DS1 is stamped, and every plain 8x8 room
// is built from its A/B/C grids.
//
// Approximations (not proven): the real game creates the tile records of a
// plain room per cell with the DT1 library of the room's Dt1Mask (0x680720,
// only mapped structurally). Here the floor/wall dwords of the grids are
// resolved through the engine's tile lookup, which picks among every DT1 of the
// level type, and the random tree markers of sub-theme patterns are not
// created. The hero's entry is the first road end (the exit towards the town
// or the first exit marker), not the original warp rule.
func (g *MapGenerator) GenerateRealOutdoor(levelID int, seed uint32, diff d2drlg.Difficulty) error {
	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	var p drlgoutdoor.Params

	if isAct23Outdoor(levelID) {
		p, err = drlgoutdoor.ParamsAct23(tb, seed, diff, levelID)
	} else {
		var lay *drlgworld.Layout

		if lay, err = drlgworld.Generate(tb, seed, diff); err == nil {
			p, err = drlgoutdoor.ParamsFromLayout(tb, lay, levelID, seed)
		}
	}

	if err != nil {
		return err
	}

	env := drlgoutdoor.NewEnv(tb, func(file string) ([]byte, error) {
		return g.asset.LoadFile("/data/global/tiles/" + file)
	})

	lv, err := drlgoutdoor.Generate(env, p)
	if err != nil {
		return err
	}

	rec, _ := tb.Level(levelID)
	region := d2enum.RegionIdType(rec.LevelType)

	if g.asset.Records.Level.Types[region] == nil {
		return fmt.Errorf("level %d: no LvlTypes row %d", levelID, rec.LevelType)
	}

	g.engine.ResetMap(region, p.Rect.W, p.Rect.H)

	var (
		mon     monsterStats
		presets int
		plain   int
	)

	levelSeed := d2rand.LevelSeed(p.BaseSeed, uint32(levelID))

	// preset rooms: one stamp per preset origin cell
	for yc := 0; yc < lv.H; yc++ {
		for xc := 0; xc < lv.W; xc++ {
			fl := lv.Flag.Get(xc, yc)
			def := int(lv.Def.Get(xc, yc))

			if fl&0x200 == 0 || def == 0 {
				continue
			}

			pr, ok := tb.PrestByDef(def)
			if !ok || file(fl) >= len(pr.File) || pr.File[file(fl)] == "" {
				g.Warningf("real outdoor: Def %d file %d unknown, preset skipped", def, file(fl))
				continue
			}

			path := drlgoutdoor.NormalizePrestFile(pr.File[file(fl)])

			g.engine.AddDS1(path)

			stamp := g.engine.LoadStampPath(region, def, path)
			if stamp == nil {
				continue
			}

			ox, oy := xc*8, yc*8
			g.engine.PlaceStampClipped(stamp, ox, oy, pr.SizeX, pr.SizeY)

			presets++

			roomSeed := d2rand.New(levelSeed.Lo + uint32(def)*0x9E3779B1 + uint32(xc*131+yc))
			g.placeMonsters(stamp, levelID, diff, ox, oy, pr.SizeX, pr.SizeY, roomSeed, &mon)
		}
	}

	// plain rooms
	for _, r := range lv.Rooms {
		if r.Type != 1 {
			continue
		}

		gr, err := lv.BuildRoomGrids(r, nil)
		if err != nil {
			return err
		}

		for ty := 0; ty < 8; ty++ {
			for tx := 0; tx < 8; tx++ {
				g.engine.SetTile(r.X-p.Rect.X+tx, r.Y-p.Rect.Y+ty, region,
					roomTile(gr.A.Get(tx, ty), gr.B.Get(tx, ty), gr.C.Get(tx, ty)))
			}
		}

		plain++
	}

	g.engine.BlockEmptyTiles()
	g.engine.UseCollisionPaths(true)

	sx, sy, how := g.outdoorEntry(lv, p.Rect)
	g.engine.SetStartPosition(sx, sy)

	g.Infof("real outdoor: level %d seed %#x: %d rooms (%d plain, %d presets stamped), map %dx%d tiles",
		levelID, seed, len(lv.Rooms), plain, presets, p.Rect.W, p.Rect.H)
	g.Infof("real outdoor: hero entry at tile (%.1f,%.1f) %s", sx, sy, how)

	if os.Getenv("OD2_AUTOMAP_ASCII") != "" {
		g.logWalkMap(sx, sy)
	}

	g.Infof("real outdoor: DS1 monsters: %d direct, %d place_* markers resolved, %d super uniques, %d groups, %d skipped",
		mon.direct, mon.place, mon.super, mon.groups, mon.skipped)

	return nil
}

// roomTile turns the A (orientation), B (wall) and C (floor) dwords of one
// room cell into an engine tile. The wall dword has the DS1 layout with the
// 8-bit sequence of the exe's tile pick.
func file(fl uint32) int { return int(fl>>16) & 0xf }

func roomTile(a, b, c uint32) d2mapstamp.Tile {
	var t d2mapstamp.Tile

	if c&2 != 0 {
		var f d2ds1.Tile
		f.Prop1, f.Sequence, f.Style = byte(c), byte(c>>8), byte(c>>20&0x3f)
		t.Floors = append(t.Floors, f)
	}

	if b&1 != 0 {
		var w d2ds1.Tile
		w.Prop1, w.Sequence, w.Style = byte(b), byte(b>>8), byte(b>>20&0x3f)
		w.Type = d2enum.TileType(a)

		if b&0x80000000 != 0 {
			w.HiddenBytes = 1
		}

		t.Walls = append(t.Walls, w)
	}

	return t
}

// outdoorEntry picks the hero's start: next to the first road end (exit
// towards the town or a neighbour), else the middle of the level.
func (g *MapGenerator) outdoorEntry(lv *drlgoutdoor.Level, rect drlgoutdoor.Rect) (x, y float64, how string) {
	cx, cy := rect.W/2, rect.H/2
	how = "(level centre)"

	if len(lv.River.Ends) > 0 {
		cx, cy = lv.River.Ends[0][0]-rect.X, lv.River.Ends[0][1]-rect.Y
		how = "near the first exit"
	}

	if tx, ty, ok := g.nearestWalkable(cx, cy, 2*entrySearchTiles); ok {
		return float64(tx) + 0.05, float64(ty) + 0.05, how
	}

	return float64(rect.W) / 2, float64(rect.H) / 2, "(fallback: map centre, nothing walkable found)"
}
