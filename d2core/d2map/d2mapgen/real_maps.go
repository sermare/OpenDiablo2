package d2mapgen

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgmaze"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg/drlgworld"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
)

// HeroMapSeed is the game seed of the loaded hero (a .d2s stores it at 0xAB).
// The app sets it before the local server is started.
var HeroMapSeed uint32

// RealMapsEnabled reports whether OD2_REALMAPS=1 asks for the DRLG port.
// Only complete parts are wired: Act 1 maze levels (caves, crypts, jail,
// catacombs). Everything else keeps using the old generator.
func RealMapsEnabled() bool { return os.Getenv("OD2_REALMAPS") == "1" }

// AutomapLevel returns the level id of OD2_AUTOMAP, or 0.
func AutomapLevel() int {
	v, _ := strconv.Atoi(os.Getenv("OD2_AUTOMAP"))
	return v
}

func automapDifficulty() d2drlg.Difficulty {
	v, _ := strconv.Atoi(os.Getenv("OD2_AUTOMAP_DIFF"))
	if v < 0 || v > 2 {
		v = 0
	}

	return d2drlg.Difficulty(v)
}

// LoadDRLGTables builds the DRLG tables from the game archives. lvlprest.bin
// is used when present (it is authoritative where the txt is stripped).
func LoadDRLGTables(a *d2asset.AssetManager) (*d2drlg.Tables, error) {
	get := func(p string) []byte {
		b, err := a.LoadFile(p)
		if err != nil {
			return nil
		}

		return b
	}

	raw := d2drlg.Raw{
		Levels:      get("/data/global/excel/Levels.txt"),
		LvlMaze:     get("/data/global/excel/LvlMaze.txt"),
		LvlPrest:    get("/data/global/excel/LvlPrest.txt"),
		LvlPrestBin: get("/data/global/excel/lvlprest.bin"),
		LvlTypes:    get("/data/global/excel/LvlTypes.txt"),
		LvlSub:      get("/data/global/excel/LvlSub.txt"),
	}
	if raw.Levels == nil || raw.LvlMaze == nil || (raw.LvlPrest == nil && raw.LvlPrestBin == nil) {
		return nil, fmt.Errorf("DRLG tables missing from the archives")
	}

	t, err := d2drlg.Load(raw)
	if err != nil && raw.LvlPrestBin != nil { // unknown bin layout: fall back to the txt
		raw.LvlPrestBin = nil
		t, err = d2drlg.Load(raw)
	}

	return t, err
}

// LogDRLGSummary generates the level for the seed and logs a summary through
// logf (one line per room plus an ASCII room grid for maze levels). It backs
// OD2_AUTOMAP=<levelId>, so a map can be checked without clicking.
func LogDRLGSummary(a *d2asset.AssetManager, seed uint32, levelID int, diff d2drlg.Difficulty, logf func(string, ...interface{})) {
	tb, err := LoadDRLGTables(a)
	if err != nil {
		logf("AUTOMAP error: %v", err)
		return
	}

	base, _ := d2rand.DrlgBaseSeed(seed)
	logf("AUTOMAP seed=%#x base=%#x level=%d difficulty=%d", seed, base, levelID, diff)

	if world, err := drlgworld.Generate(tb, seed, diff); err == nil {
		logf("AUTOMAP world town=%s(file %d) bloodmoor=%v stony=%v coldplains=%v barracksExitSide=%d",
			townName(tb, world.TownFile), world.TownFile, world.Levels[2].Rect, world.Levels[4].Rect, world.Levels[3].Rect, world.BarracksExitSide)
	} else {
		logf("AUTOMAP world error: %v", err)
	}

	res, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: levelID, Difficulty: diff, BaseSeed: base})
	if err != nil {
		logf("AUTOMAP level %d: %v", levelID, err)
		return
	}

	logf("AUTOMAP maze level=%d type=%d rooms=%d chunks=%d bbox=(%d,%d)-(%d,%d)",
		levelID, res.LevelType, len(res.Rooms), len(res.Chunks), res.MinX, res.MinY, res.MaxX, res.MaxY)

	for i, r := range res.Rooms {
		logf("AUTOMAP room %d at (%d,%d) %dx%d def=%d doors=%04b file=%d %s special=%v", i, r.X, r.Y, r.W, r.H, r.Def, r.Doors, r.File, r.FileName, r.Locked)
	}

	for _, line := range roomGrid(res) {
		logf("AUTOMAP | %s", line)
	}

	logf("AUTOMAP exactness: not proven against the real game (no oracle); see d2drlg package docs")
}

func townName(tb *d2drlg.Tables, file int) string {
	if p, ok := tb.PrestByDef(1); ok && file >= 0 && file < len(p.File) {
		return p.File[file]
	}

	return "?"
}

// roomGrid draws rooms as cells: the index of each room in base 36, '#' for
// special rooms, with doors drawn as '-' / '|' between connected rooms.
func roomGrid(res *drlgmaze.Result) []string {
	rw, rh := res.Rooms[0].W, res.Rooms[0].H
	cols := (res.MaxX-res.MinX)/rw + 1
	rows := (res.MaxY-res.MinY)/rh + 1
	grid := make([][]byte, rows*2)

	for i := range grid {
		grid[i] = []byte(strings.Repeat(" ", cols*2))
	}

	cell := map[int][2]int{}

	for i, r := range res.Rooms {
		cx, cy := (r.X-res.MinX)/rw, (r.Y-res.MinY)/rh
		cell[i] = [2]int{cx, cy}
		ch := strconv.FormatInt(int64(i%36), 36)[0]

		if r.Locked {
			ch = '#'
		}

		grid[cy*2][cx*2] = ch
	}

	for i, r := range res.Rooms {
		for _, k := range r.Links {
			a, b := cell[i], cell[k]
			if a[1] == b[1] {
				grid[a[1]*2][a[0]+b[0]+0] = '-'
			} else {
				grid[a[1]+b[1]][a[0]*2] = '|'
			}
		}
	}

	out := make([]string, len(grid))
	for i := range grid {
		out[i] = string(grid[i])
	}

	return out
}

// GenerateRealMaze replaces the map with the DRLG maze level for the hero's
// seed: every room's chosen preset DS1 is stamped at its room position.
// Sub-theme/object details beyond the DS1 contents are not generated.
func (g *MapGenerator) GenerateRealMaze(levelID int, seed uint32, diff d2drlg.Difficulty) error {
	tb, err := LoadDRLGTables(g.asset)
	if err != nil {
		return err
	}

	base, _ := d2rand.DrlgBaseSeed(seed)

	res, err := drlgmaze.Generate(tb, drlgmaze.Params{LevelID: levelID, Difficulty: diff, BaseSeed: base})
	if err != nil {
		return err
	}

	lvl, _ := tb.Level(levelID)
	region := d2enum.RegionIdType(lvl.LevelType)
	w, h := res.MaxX-res.MinX+2, res.MaxY-res.MinY+2

	g.engine.ResetMap(region, w, h)

	placed := 0

	for _, r := range res.Rooms {
		rec, ok := g.asset.Records.Level.Presets[r.Def]
		if !ok || rec.Files[0] == "" || rec.Files[0] == "0" || r.File < 0 {
			g.Warningf("real maze: Def %d has no usable preset files, room skipped", r.Def)
			continue
		}

		for _, f := range rec.Files {
			g.engine.AddDS1(f)
		}

		stamp := g.engine.LoadStamp(region, r.Def, r.File)
		if stamp == nil {
			continue
		}

		sz := stamp.Size()
		ox, oy := r.X-res.MinX, r.Y-res.MinY

		if ox+sz.Width > w || oy+sz.Height > h {
			g.Warningf("real maze: stamp %s (%dx%d) does not fit, skipped", r.FileName, sz.Width, sz.Height)
			continue
		}

		g.engine.PlaceStamp(stamp, ox, oy)

		placed++
	}

	g.Infof("real maze: level %d seed %#x: %d rooms, %d stamps placed", levelID, seed, len(res.Rooms), placed)

	return nil
}
