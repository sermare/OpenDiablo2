package d2mapgen

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2ds1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapstamp"
)

// isOutdoorLevel reports the Act 1 wilderness levels the drlgoutdoor port
// covers: Blood Moor .. Tamoe Highland, Burial Grounds and Moo Moo Farm.
func isOutdoorLevel(id int) bool { return (id >= 2 && id <= 7) || id == 0x11 || id == 0x27 }

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

	lay, err := drlgworld.Generate(tb, seed, diff)
	if err != nil {
		return err
	}

	p, err := drlgoutdoor.ParamsFromLayout(tb, lay, levelID, seed)
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
	g.engine.SetWorld(d2mapengine.World{Level: levelID, OriginX: p.Rect.X, OriginY: p.Rect.Y, Rects: worldRects(lay)})

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
		g.Warningf("real outdoor: exact tile build failed (%v); plain rooms use the grid lookup, presets keep the stamped tiles", err)
		return g.placePlainRoomsLookup(lv, rect, region), ", dword lookup"
	}

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

	return plain, fmt.Sprintf(", exact tiles incl. %d preset rooms", presets)
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

	sz := stamp.Size()
	for y := 0; y < sz.Height; y++ {
		for x := 0; x < sz.Width; x++ {
			for _, w := range stamp.Tile(x, y).Walls {
				if !w.Type.Special() || w.Style == startMarkerStyle {
					continue
				}

				dest := 0
				if isCave && hasCave {
					dest = cave
					g.engine.SetWarpDestination(ox+x, oy+y, dest)
				}

				g.Infof("real outdoor: exit tile style=%d at (%d,%d) in %s leads to level %d", w.Style, ox+x, oy+y, path, dest)
			}
		}
	}
}

// startMarkerStyle is the style of the player start special tile (not an exit).
const startMarkerStyle = 30
