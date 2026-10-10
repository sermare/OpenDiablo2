package d2mapgen

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

// isOutdoorLevel reports the outdoor levels the drlgoutdoor port covers: the
// Act 1 wilderness (Blood Moor .. Tamoe Highland, Burial Grounds, Moo Moo
// Farm), Act 4 (Outer Steppes, Plains of Despair, City of the Damned, Chaos
// Sanctuary) and Act 5 (Bloody Foothills, Frigid Highlands, Arreat Plateau,
// Frozen Tundra), the Act 2 desert (41..46) and the Act 3 jungle and Kurast
// (76..83). Level 134 (Forgotten Sands) is not covered.
func isOutdoorLevel(id int) bool {
	switch {
	case isAct23Outdoor(id):
		return true
	case id >= 2 && id <= 7, id == 0x11, id == 0x27:
		return true
	case id >= 104 && id <= 106, id == 108, id >= 110 && id <= 112, id == 117:
		return true
	}

	return false
}

// isPresetLevel reports the Act 4/5 DrlgType 2 levels drlgoutdoor.GeneratePreset
// covers: Pandemonium Fortress, Harrogath, Arreat Summit, Nihlathak's Temple,
// Halls of Vaught, Throne of Destruction, Worldstone Chamber, Pandemonium Finale.
func isPresetLevel(id int) bool {
	switch id {
	case 103, 109, 120, 121, 124, 131, 132, 136:
		return true
	}

	return isAct1Preset(id) || isAct3Preset(id)
}

// isAct3Preset reports the Act 3 dungeon levels that are a single preset DS1 (Levels.txt DrlgType 2): the
// treasure rooms of the Flayer Dungeons (90, 91), Sewers 2 (93), the six temples (94..99) and the third level
// of the Durance of Hate (102).
func isAct3Preset(id int) bool {
	switch {
	case id == 90, id == 91, id == 93, id >= 94 && id <= 99, id == 102:
		return true
	}

	return false
}

// isAct1Preset reports the Act 1 DrlgType 2 levels outside the world layout: the
// small cave levels Cave Level 2 .. Underground Passage Level 2 (Hole/Pit 2,
// ids 13..16) and Catacombs Level 4 (37, Andariel). GeneratePreset is proven
// equal to the real game for them (TestOraclePresetAct1).
func isAct1Preset(id int) bool { return (id >= 13 && id <= 16) || id == 37 }

// levelParams runs the world placement of the level's act and derives the
// generator inputs (rectangle, od.flags, vis/warp, neighbour list).
func levelParams(tb *d2drlg.Tables, levelID int, seed uint32, diff d2drlg.Difficulty) (drlgoutdoor.Params, *drlgworld.Layout, error) {
	rec, _ := tb.Level(levelID)

	if isAct3Preset(levelID) { // a dungeon of its own: no world, the rectangle is the level's size
		p := drlgoutdoor.Params{ID: levelID, Vis: rec.Vis, Warp: rec.Warp,
			Rect: drlgoutdoor.Rect{W: rec.SizeX[diff], H: rec.SizeY[diff]}}
		p.BaseSeed, _ = d2rand.DrlgBaseSeed(seed)

		return p, nil, nil
	}

	if isAct1Preset(levelID) {
		p, err := drlgoutdoor.ParamsPreset(tb, levelID, seed, diff)

		return p, nil, err
	}

	if isAct23Outdoor(levelID) { // no drlgworld.Layout: the Act 2/3 placers have their own world
		p, err := drlgoutdoor.ParamsAct23(tb, seed, diff, levelID)

		return p, nil, err
	}

	switch rec.Act {
	case 3, 4:
		var (
			lay *drlgworld.Layout
			err error
		)

		if rec.Act == 3 {
			lay, err = drlgworld.GenerateAct4(tb, seed, diff)
		} else {
			lay, err = drlgworld.GenerateAct5(tb, seed, diff)
		}

		if err != nil {
			return drlgoutdoor.Params{}, nil, err
		}

		p, err := drlgoutdoor.ParamsFromLayout45(tb, lay, rec.Act, levelID, seed, diff)

		return p, lay, err
	}

	lay, err := drlgworld.Generate(tb, seed, diff)
	if err != nil {
		return drlgoutdoor.Params{}, nil, err
	}

	p, err := drlgoutdoor.ParamsFromLayout(tb, lay, levelID, seed)

	return p, lay, err
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

// presetProvider builds the Act 4/5 DrlgType 2 levels. Only active with
// OD2_REALMAPS=1.
type presetProvider struct{}

func (presetProvider) Name() string { return "drlg-preset" }

func (presetProvider) CanLoad(levelID int) bool { return RealMapsEnabled() && isPresetLevel(levelID) }

func (presetProvider) Load(g *MapGenerator, levelID int, req LoadRequest) error {
	return g.GenerateRealPreset(levelID, req.Seed, req.Difficulty)
}

// GenerateRealOutdoor replaces the map with the DRLG outdoor level for the
// hero's seed: the world layout decides the level rectangle and its exits,
// drlgoutdoor generates the cell grids and the room list (proven equal to the
// real game's), every preset room's DS1 is stamped, and every plain 8x8 room
// is built from its A/B/C grids.
//
// The tiles are the exact records of the original (drlgoutdoor Level.BuildTiles,
// proven equal to Game.exe's 0x680720 and 0x6696c0 for every tested level): each
// record names a DT1 file and tile index of the room's own library, and the tree
// markers of sub-theme patterns are created. Preset rooms are still stamped from
// their DS1 first, for the entities and the marker tiles. The hero's entry is
// the first road end (the exit towards the town or the first exit marker), not
// the original warp rule.
func (g *MapGenerator) GenerateRealOutdoor(levelID int, seed uint32, diff d2drlg.Difficulty) error {
	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	p, lay, err := levelParams(tb, levelID, seed, diff)
	if err != nil {
		return err
	}

	env, err := outdoorEnv(g.asset)
	if err != nil {
		return err
	}

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
	rects := worldRects(lay)
	if lay == nil && isAct23Outdoor(levelID) {
		rects = act23Rects(tb, levelID, seed, diff) // the Act 2/3 placers have their own world
	}

	g.engine.SetWorld(d2mapengine.World{Level: levelID, OriginX: p.Rect.X, OriginY: p.Rect.Y, Rects: rects})

	var (
		mon     monsterStats
		presets int
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

			g.markWarpTiles(stamp, path, ox, oy, levelID)

			roomSeed := d2rand.New(levelSeed.Lo + uint32(def)*0x9E3779B1 + uint32(xc*131+yc))
			g.placeMonsters(stamp, levelID, diff, ox, oy, pr.SizeX, pr.SizeY, roomSeed, &mon)
		}
	}

	// the real tile records of every room (DT1 library of the room, rarity pick with
	// the room seed), grouped per map cell
	plain, exact := g.placeExactTiles(lv, p.Rect, region)

	g.engine.BlockEmptyTiles()
	g.engine.UseCollisionPaths(true)

	sx, sy, how := g.outdoorEntry(lv, p.Rect)
	g.engine.SetStartPosition(sx, sy)

	g.Infof("real outdoor: level %d seed %#x: %d rooms (%d plain%s, %d presets stamped), map %dx%d tiles",
		levelID, seed, len(lv.Rooms), plain, exact, presets, p.Rect.W, p.Rect.H)
	g.Infof("real outdoor: hero entry at tile (%.1f,%.1f) %s", sx, sy, how)

	if os.Getenv("OD2_AUTOMAP_ASCII") != "" {
		g.logWalkMap(sx, sy)
	}

	g.Infof("real outdoor: DS1 monsters: %d direct, %d place_* markers resolved, %d super uniques, %d groups, %d skipped",
		mon.direct, mon.place, mon.super, mon.groups, mon.skipped)

	return nil
}

// placeExactTiles sets the map cells of all rooms from the exact tile records
// (plain rooms and presets alike; the preset DS1 stamps keep providing the
// entities and the marker tiles). When the exact build fails (a code path of
// the original that is not ported, or a DT1 that cannot be loaded) the plain
// rooms fall back to the old dword lookup and the presets keep their stamped
// tiles. The first result is the number of plain rooms, the second a note for
// the log.
func (g *MapGenerator) placeExactTiles(lv *drlgoutdoor.Level, rect drlgoutdoor.Rect, region d2enum.RegionIdType) (int, string) {
	tiles, err := lv.BuildTiles()
	if err != nil {
		g.Infof("real outdoor: exact tiles unavailable for this level type (%v); plain rooms use the grid lookup, presets keep the stamped tiles", err)
		return g.placePlainRoomsLookup(lv, rect, region), ", dword lookup"
	}

	plain, presets := g.applyExactTiles(tiles, rect, region)

	return plain, fmt.Sprintf(", exact tiles incl. %d preset rooms", presets)
}

// applyExactTiles puts the records of every built room on the map cells and
// returns the number of plain and preset rooms.
func (g *MapGenerator) applyExactTiles(tiles []*drlgoutdoor.RoomTiles, rect drlgoutdoor.Rect, region d2enum.RegionIdType) (int, int) {
	type cell struct{ floors, walls, shadows []d2mapengine.ExactTile }

	cells := map[[2]int]*cell{}
	get := func(x, y int) *cell {
		k := [2]int{x, y}
		if cells[k] == nil {
			cells[k] = &cell{}
		}

		return cells[k]
	}

	exact := func(r *drlgoutdoor.TileRecord) d2mapengine.ExactTile {
		return d2mapengine.ExactTile{File: r.Tile.File(), Index: r.Tile.Idx}
	}

	plain, presets := 0, 0

	for _, rt := range tiles {
		if rt == nil {
			continue
		}

		if rt.Room.Type == 1 {
			plain++
		} else {
			presets++
		}

		for _, r := range rt.Floors {
			c := get(rt.Room.X+r.X, rt.Room.Y+r.Y)
			c.floors = append(c.floors, exact(r))
		}

		for _, r := range rt.Walls {
			c := get(rt.Room.X+r.X, rt.Room.Y+r.Y)
			c.walls = append(c.walls, exact(r))
		}

		for _, r := range rt.Shadows {
			c := get(rt.Room.X+r.X, rt.Room.Y+r.Y)
			c.shadows = append(c.shadows, exact(r))
		}
	}

	for k, c := range cells {
		g.engine.SetExactTiles(k[0]-rect.X, k[1]-rect.Y, region, true, c.floors, c.walls, c.shadows)
	}

	return plain, presets
}

// placePlainRoomsLookup is the pre-exact approximation: the floor/wall dwords of
// the room grids resolved through the engine's tile lookup.
func (g *MapGenerator) placePlainRoomsLookup(lv *drlgoutdoor.Level, rect drlgoutdoor.Rect, region d2enum.RegionIdType) int {
	plain := 0

	for _, r := range lv.Rooms {
		if r.Type != 1 {
			continue
		}

		gr, err := lv.BuildRoomGrids(r, nil)
		if err != nil {
			g.Warningf("real outdoor: room grids: %v", err)
			continue
		}

		for ty := 0; ty < 8; ty++ {
			for tx := 0; tx < 8; tx++ {
				g.engine.SetTile(r.X-rect.X+tx, r.Y-rect.Y+ty, region,
					roomTile(gr.A.Get(tx, ty), gr.B.Get(tx, ty), gr.C.Get(tx, ty)))
			}
		}

		plain++
	}

	return plain
}

// GenerateRealPreset replaces the map with an Act 4/5 DrlgType 2 level: the
// room list and file index come from drlgoutdoor.GeneratePreset (proven equal
// to the real game's), the level's DS1 is stamped over the whole rectangle.
func (g *MapGenerator) GenerateRealPreset(levelID int, seed uint32, diff d2drlg.Difficulty) error {
	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	p, lay, err := levelParams(tb, levelID, seed, diff)
	if err != nil {
		return err
	}

	env, err := outdoorEnv(g.asset)
	if err != nil {
		return err
	}

	pl, err := drlgoutdoor.GeneratePreset(env, p, -1)
	if err != nil {
		return err
	}

	rec, _ := tb.Level(levelID)
	pr, _ := tb.PrestByDef(pl.Def)
	region := d2enum.RegionIdType(rec.LevelType)

	if g.asset.Records.Level.Types[region] == nil {
		return fmt.Errorf("level %d: no LvlTypes row %d", levelID, rec.LevelType)
	}

	if pl.File >= len(pr.File) || pr.File[pl.File] == "" {
		return fmt.Errorf("level %d: Def %d has no file %d", levelID, pl.Def, pl.File)
	}

	g.engine.ResetMap(region, pl.Rect.W, pl.Rect.H)
	if lay != nil {
		g.engine.SetWorld(d2mapengine.World{Level: levelID, OriginX: pl.Rect.X, OriginY: pl.Rect.Y, Rects: worldRects(lay)})
	}

	path := drlgoutdoor.NormalizePrestFile(pr.File[pl.File])
	g.engine.AddDS1(path)

	stamp := g.engine.LoadStampPath(region, pl.Def, path)
	if stamp == nil {
		return fmt.Errorf("level %d: cannot load %s", levelID, path)
	}

	g.engine.PlaceStampClipped(stamp, 0, 0, pl.Rect.W, pl.Rect.H)

	// the exact records of the preset rooms (checked against Game.exe for the
	// Act 4/5 preset levels); on failure the stamped DS1 tiles stay
	exact := ""

	if tiles, err := pl.BuildTiles(); err != nil {
		g.Infof("real preset: exact tiles unavailable (%v); keeping the stamped tiles", err)
	} else {
		_, n := g.applyExactTiles(tiles, pl.Rect, region)
		exact = fmt.Sprintf(", exact tiles for %d rooms", n)
	}

	var mon monsterStats

	levelSeed := d2rand.LevelSeed(p.BaseSeed, uint32(levelID))
	g.placeMonsters(stamp, levelID, diff, 0, 0, pl.Rect.W, pl.Rect.H, d2rand.New(levelSeed.Lo+uint32(pl.Def)), &mon)

	g.engine.BlockEmptyTiles()
	g.engine.UseCollisionPaths(true)

	var (
		sx, sy float64
		how    string
	)

	if isAct1Preset(levelID) {
		sx, sy, how = g.findEntry(&drlgmaze.Result{}, []roomRect{{0, 0, pl.Rect.W, pl.Rect.H, path}}, 0)
	} else {
		sx, sy, how = g.outdoorEntry(&drlgoutdoor.Level{}, pl.Rect)
	}

	g.engine.SetStartPosition(sx, sy)

	g.Infof("real preset: level %d seed %#x: Def %d file %d (%s), %d rooms%s, map %dx%d tiles",
		levelID, seed, pl.Def, pl.File, path, len(pl.Rooms), exact, pl.Rect.W, pl.Rect.H)
	g.Infof("real preset: hero entry at tile (%.1f,%.1f) %s", sx, sy, how)

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

// markWarpTiles resolves the special (exit) wall tiles of a stamped preset: the
// cave entrance presets (Act1/Caves/...) lead to the level's cave. Other
// special tiles keep the style based lookup of d2level.TileDestination.
func (g *MapGenerator) markWarpTiles(stamp *d2mapstamp.Stamp, path string, ox, oy, levelID int) {
	cave, hasCave := d2level.CaveEntranceDestination(levelID)
	isCave := strings.Contains(strings.ToLower(path), "/caves/")

	// Act 2 desert levels (41..45) have exactly one tomb, lair or temple behind them; the
	// entrance presets (Act2/Outdoors/TombEnt*.ds1 ...) carry styles (2 seen) that are not the
	// LvlWarp ids of the Levels.txt slots (33..36), so the preset stands for that one exit
	if !hasCave {
		cave, hasCave = d2level.SingleTileDestination(levelID)
		isCave = hasCave && d2level.ActOfLevel(levelID) >= 2 // also City of the Damned -> River of Flame, Arreat Plateau -> Crystalline Passage
	}

	sz := stamp.Size()
	for y := 0; y < sz.Height; y++ {
		for x := 0; x < sz.Width; x++ {
			for _, w := range stamp.Tile(x, y).Walls {
				if !w.Type.Special() || w.Style == startMarkerStyle {
					continue
				}

				dest := 0
				if to, ok := d2level.Act3SlotDestination(levelID, int(w.Style)); ok {
					// Act 3: the style of the entrance tile is the Vis slot of the dungeon
					dest = to
					g.engine.SetWarpDestination(ox+x, oy+y, dest)
				} else if isCave && hasCave {
					dest = cave
				}

				if d, ok := d2level.OutdoorExitByPreset(levelID, path); ok {
					dest = d
				}

				if dest != 0 {
					g.engine.SetWarpDestination(ox+x, oy+y, dest)
				}

				g.Infof("real outdoor: exit tile style=%d at (%d,%d) in %s leads to level %d", w.Style, ox+x, oy+y, path, dest)
			}
		}
	}
}

// startMarkerStyle is the style of the player start special tile (not an exit).
const startMarkerStyle = 30
