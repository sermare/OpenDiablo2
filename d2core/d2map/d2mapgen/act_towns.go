package d2mapgen

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgoutdoor"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// The towns of Acts 2-5 are single preset DS1 files (Levels.txt DrlgType 2).
// LvlPrest Def of each (verified in the extracted tables, LevelId column):
// Lut Gholein 301 (LutW.ds1 / LutN.ds1), Kurast Docks 529 (DockTown3.ds1),
// Pandemonium Fortress 797 (Fortress.ds1), Harrogath 863 (townWest.ds1). The
// LvlTypes row (palette and DT1 set) is the Levels.txt LevelType: 12, 20, 26,
// 29, which are also the RegionIdType values of the town regions.
var actTownPrest = map[int]int{
	d2level.LutGholein:          301,
	d2level.KurastDocks:         529,
	d2level.PandemoniumFortress: 797,
	d2level.Harrogath:           863,
}

// IsActTown reports whether the level is the town of act 2-5.
func IsActTown(levelID int) bool { _, ok := actTownPrest[levelID]; return ok }

// townStartStyle is the DS1 "special" wall tile style 30 the engine already
// uses as the hero's start tile in Act 1 (MapEngine.GetStartPosition).
const (
	townStartStyle    = 30
	townAltStartStyle = 33 // UNVERIFIED fallback for the Fortress, whose DS1 has no style 30
)

type startCand struct {
	X, Y, Style, Seq int
}

// chooseTownStart picks the hero's start tile among the special wall tiles of
// a town DS1: a style 30 tile (prefer sequence 0, which is the only one in
// every file but LutN, which has a second one in its north), else a style 33
// tile. UNVERIFIED: the real game's arrival rule is the start type 5 position
// finder (0x61ad90); style 30 is only known to be the Act 1 start tile.
func chooseTownStart(c []startCand) (x, y int, how string, ok bool) {
	for _, style := range []int{townStartStyle, townAltStartStyle} {
		var best *startCand

		for i := range c {
			if c[i].Style != style {
				continue
			}

			if best == nil || (c[i].Seq == 0 && best.Seq != 0) {
				best = &c[i]
			}
		}

		if best != nil {
			return best.X, best.Y, fmt.Sprintf("special tile style %d", style), true
		}
	}

	return 0, 0, "", false
}

// townFileIndex picks the preset file of a town with several files (only Lut
// Gholein has two). The real game rolls the level seed
// (DRLG_AllocPresetMap, Roll(Files)) but LvlPrest says Files=0 for Lut Gholein
// while two file names are listed, so which roll applies is UNVERIFIED: this
// rolls the level seed once over the listed files.
func townFileIndex(n int, base uint32, level int) int {
	if n <= 1 {
		return 0
	}

	return int(d2rand.LevelSeed(base, uint32(level)).Roll(int32(n)))
}

// actTownProvider builds the towns of Acts 2-5 from their preset DS1.
type actTownProvider struct{}

func (actTownProvider) Name() string { return "act-town" }

func (actTownProvider) CanLoad(levelID int) bool { return IsActTown(levelID) }

func (actTownProvider) Load(g *MapGenerator, levelID int, req LoadRequest) error {
	return g.GenerateActTown(levelID, req.Seed, req.Difficulty)
}

// GenerateActTown replaces the map with the town of act 2-5: its preset DS1
// is stamped at the origin with the LvlTypes row of the level (palette region
// and DT1 library), the NPCs come from the DS1's monster objects (names via
// monpreset and monstats, done by the stamp), and the hero starts on the DS1's
// start marker.
func (g *MapGenerator) GenerateActTown(levelID int, seed uint32, diff d2drlg.Difficulty) error {
	def, ok := actTownPrest[levelID]
	if !ok {
		return fmt.Errorf("level %d is not a town of act 2-5", levelID)
	}

	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	rec, ok := tb.Level(levelID)
	if !ok {
		return fmt.Errorf("level %d: no Levels.txt row", levelID)
	}

	pr, ok := tb.PrestByDef(def)
	if !ok || pr.LevelID != levelID {
		return fmt.Errorf("level %d: LvlPrest def %d does not belong to it", levelID, def)
	}

	var files []string

	for _, f := range pr.File {
		if f != "" && f != "0" {
			files = append(files, f)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("level %d: LvlPrest def %d lists no files", levelID, def)
	}

	base, _ := d2rand.DrlgBaseSeed(seed)
	idx := townFileIndex(len(files), base, levelID)

	// OD2_REALMAPS=1: Lut Gholein is the preset the Act 2 world layout chose
	// (LutW when Rocky Waste lies to its west, LutN when it lies to its north)
	// inside its level rectangle, so the walk out of the gate reaches the border
	// of Rocky Waste
	var rects map[int]d2level.Rect

	if RealMapsEnabled() && levelID == d2level.LutGholein {
		if w, err := drlgoutdoor.PlaceAct2World(tb, seed, diff); err == nil && w.TownFile >= 1 && w.TownFile <= len(files) {
			idx = w.TownFile - 1
			rects = act23Rects(tb, levelID, seed, diff)
		}
	}
	path := drlgoutdoor.NormalizePrestFile(files[idx])
	region := d2enum.RegionIdType(rec.LevelType)

	if g.asset.Records.Level.Types[region] == nil {
		return fmt.Errorf("level %d: no LvlTypes row %d", levelID, rec.LevelType)
	}

	stamp := g.engine.LoadStampPath(region, def, path)
	if stamp == nil {
		return fmt.Errorf("level %d: could not load %s", levelID, path)
	}

	size := stamp.Size()
	w, h := size.Width, size.Height

	if r, ok := rects[levelID]; ok {
		w, h = maxInt(w, r.W), maxInt(h, r.H)
	}

	g.engine.ResetMap(region, w, h)
	g.engine.AddDS1(path) // ResetMap dropped the DT1 list
	g.engine.PlaceStamp(stamp, 0, 0)

	// the real game's tile records instead of the stamp's (style, sequence, type)
	// lookup; Lut Gholein only when the world chose its file (OD2_REALMAPS=1)
	if levelID != d2level.LutGholein || RealMapsEnabled() {
		g.applyExactTown(levelID, seed, diff, region)
	} else {
		g.Infof("DRAWSTATS level=%d exact=false %s", levelID, g.engine.TileStats())
	}

	g.engine.BlockEmptyTiles()

	if r, ok := rects[levelID]; ok {
		g.engine.UseCollisionPaths(true)
		g.engine.SetWorld(d2mapengine.World{Level: levelID, OriginX: r.X, OriginY: r.Y, Rects: rects})
	}

	var cands []startCand

	for y := 0; y < size.Height; y++ {
		for x := 0; x < size.Width; x++ {
			t := g.engine.TileAt(x, y)
			if t == nil {
				continue
			}

			for i := range t.Components.Walls {
				if w := &t.Components.Walls[i]; w.Type.Special() {
					cands = append(cands, startCand{x, y, int(w.Style), int(w.Sequence)})
				}
			}
		}
	}

	sx, sy, how, found := chooseTownStart(cands)
	if !found {
		sx, sy, how = size.Width/2, size.Height/2, "map centre (no start marker)"
	}

	g.engine.SetStartPosition(float64(sx)+0.5, float64(sy)+0.5)

	g.Infof("act town: level %d (%s) region %d palette act %d file %d/%d %s map %dx%d start (%d,%d) %s",
		levelID, rec.Name, int(region), rec.Act+1, idx+1, len(files), path, size.Width, size.Height, sx, sy, how)

	g.placeTownExtras(levelID, sx, sy)
	g.logTownNPCs(levelID)

	return nil
}

// townExtras are town NPCs that the DS1 monster lists do not contain (checked
// on the real files: Act 3 has no Hratli, Act 5 no Larzuk; the original
// spawns them from its quest/town scripts). They are placed near the start
// marker at a GUESSED offset (tiles), then moved to the nearest walkable
// tile; their real positions are UNVERIFIED. Drehya/Anya is not placed: she
// is a prisoner of the Frigid Highlands until the Prison of Ice quest.
var townExtras = map[int][]struct {
	Key    string
	DX, DY int
}{
	d2level.KurastDocks: {{"hratli", 5, -4}},
	d2level.Harrogath:   {{"larzuk", 4, 3}},
}

func (g *MapGenerator) placeTownExtras(levelID, sx, sy int) {
	for _, ex := range townExtras[levelID] {
		stats := g.asset.Records.Monster.Stats[ex.Key]
		if stats == nil {
			continue
		}

		tx, ty, ok := g.nearestWalkable(sx+ex.DX, sy+ex.DY, entrySearchTiles)
		if !ok {
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					g.Warningf("act town: extra NPC %s: %v", ex.Key, r)
				}
			}()

			npc, err := g.engine.NewNPC(tx*subtilesPerTile+middleOfTile, ty*subtilesPerTile+middleOfTile, stats, 0)
			if err != nil {
				g.Warningf("act town: extra NPC %s: %v", ex.Key, err)
				return
			}

			g.engine.AddEntity(npc)
			g.Infof("act town: added %s (not in the DS1) near (%d,%d)", ex.Key, tx, ty)
		}()
	}
}

// logTownNPCs lists the NPCs the DS1 produced (TOWN NPC lines are read by the
// act travel autotest).
func (g *MapGenerator) logTownNPCs(levelID int) {
	n := 0

	for _, e := range g.engine.Entities() {
		npc, ok := e.(interface{ MonstatID() int })
		if !ok {
			continue
		}

		x, y := e.GetPositionF()
		g.Infof("TOWN NPC level=%d name=%q class=%d at (%.1f,%.1f)", levelID, e.Label(), npc.MonstatID(), x, y)

		n++
	}

	g.Infof("TOWN level=%d npcs=%d", levelID, n)
}
