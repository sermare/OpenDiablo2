package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapengine"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The automap (Tab). The algorithm follows UI\automap.cpp of Game.exe 1.14b, see
// package d2automap for what is verified. Summary:
//
//   - while the game runs, whenever the hero has moved about a tile (FUN_004546b0)
//     the tiles around the hero (the ones the original has drawn on screen) are
//     looked up in AutoMap.txt/automap.bin by (level type, orientation, style,
//     sequence) and the resulting MaxiMap.dc6 frames are stored per level in
//     the "revealed" model, which also survives leaving the level;
//   - the object cells (shrines, wells, stairs...) come from objects.txt AutoMap;
//   - Tab shows or hides the map, "full" draws it over the whole screen centred on
//     the hero, "mini" in a box at the top right with the half-size cells.
//
// Options (Options -> Automap): size, fade (translucent cells near the hero, see
// d2automap.CellTransparency), centre (see d2automap.MiniBoxOffsets), show party
// and show names (markers, d2automap.Classify). The revealed cells are kept per
// level for the session; they are not saved with the character (the original
// does not keep them in the .d2s either). UNVERIFIED: the percentages of the fade.

const (
	automapPathFull = "/data/global/ui/automap/maximap.dc6"
	automapPathMini = "/data/global/ui/automap/maximaps.dc6"
	automapBinPath  = "/data/global/excel/automap.bin"
	automapSeedMul  = 2654435761
)

// Automap is the automap overlay of the game controls.
type Automap struct {
	asset *d2asset.AssetManager
	ui    *d2ui.UIManager
	rend  d2interface.Renderer
	hero  *d2mapentity.Player
	eng   *d2mapengine.MapEngine
	gc    *GameControls

	table    *d2automap.Table
	tableErr error
	objCel   map[int]int // objects.txt row -> AutoMap cel

	on   bool
	size d2automap.Size

	store     *d2automap.Store // revealed cells per level, kept for the session
	override  map[string]bool  // console overrides of the options (verification aid)
	miniD210  int              // DAT_0079d210/214: mini map offsets, see d2automap.MiniBoxOffsets
	miniD214  int
	miniSeen  bool // the offsets have been computed
	lastLeft  bool
	lastShift d2automap.PanelShift
	level     int
	levelFn   func() int
	nameFn    func(id int) string
	lastCX    float64
	lastCY    float64
	haveLast  bool
	lastAdded int

	sprites   map[d2automap.Size]*d2ui.Sprite
	spriteAt  map[d2automap.Size]int // act the palette was loaded for
	label     *d2ui.Label
	nameLabel *d2ui.Label
	offscr    d2interface.Surface
	offW      int
	offH      int

	*d2util.Logger
}

// newAutomap creates the automap and binds the "automap" console command.
func newAutomap(gc *GameControls, term d2interface.Terminal) *Automap {
	a := &Automap{
		asset:    gc.asset,
		ui:       gc.ui,
		rend:     gc.renderer,
		hero:     gc.hero,
		eng:      gc.mapEngine,
		gc:       gc,
		store:    d2automap.NewStore(),
		override: map[string]bool{},
		sprites:  map[d2automap.Size]*d2ui.Sprite{},
		spriteAt: map[d2automap.Size]int{},
		size:     d2automap.SizeFull,
		Logger:   d2util.NewLogger(),
	}
	a.Logger.SetPrefix("Automap")

	if term != nil {
		_ = term.Bind("automap", "automap on|off|toggle|full|mini|fade|nofade|names|nonames|party|noparty|center|nocenter|stats", []string{"mode"}, a.command)
	}

	return a
}

// Automap returns the automap overlay.
func (g *GameControls) Automap() *Automap { return g.automap }

// SetLevelSource tells the automap how to find the current level id and its name.
func (a *Automap) SetLevelSource(level func() int, name func(id int) string) {
	a.levelFn, a.nameFn = level, name
}

// On reports whether the map is shown.
func (a *Automap) On() bool { return a.on }

// Size reports the current size.
func (a *Automap) Size() d2automap.Size { return a.size }

// Toggle is the Tab key.
func (a *Automap) Toggle() {
	if !a.on {
		a.size = automapSizeOption() // Options -> Automap Options -> Automap Size
	}

	a.SetOn(!a.on)
}

// SetOn shows or hides the map.
func (a *Automap) SetOn(on bool) {
	a.on, a.miniSeen = on, false
	a.logState()
}

// SetSize selects the full-screen or the mini map and shows it.
func (a *Automap) SetSize(s d2automap.Size) {
	a.size, a.on, a.miniSeen = s, true, false
	a.logState()
}

func (a *Automap) command(args []string) error {
	mode := "toggle"
	if len(args) > 0 {
		mode = strings.ToLower(args[0])
	}

	switch mode {
	case "on":
		a.SetOn(true)
	case "off":
		a.SetOn(false)
	case "toggle":
		a.Toggle()
	case "full":
		a.SetSize(d2automap.SizeFull)
	case "mini":
		a.SetSize(d2automap.SizeMini)
	case "fade", "nofade", "names", "nonames", "party", "noparty", "center", "nocenter":
		key := map[string]string{"fade": "fade", "names": "names", "party": "party", "center": "center"}[strings.TrimPrefix(mode, "no")]
		a.override[key] = !strings.HasPrefix(mode, "no")
		a.Infof("AUTOMAP option %s=%v", key, a.override[key])
	case "stats":
		a.logState()
		a.logMarkers()
	default:
		return fmt.Errorf("automap: unknown mode %q (on|off|toggle|full|mini|fade|names|party|center|stats)", mode)
	}

	return nil
}

func (a *Automap) logState() {
	sz := "full"
	if a.size == d2automap.SizeMini {
		sz = "mini"
	}

	m := a.model()
	a.Infof("AUTOMAP state on=%v size=%s level=%d cells=%d floor=%d wall=%d object=%d",
		a.on, sz, a.currentLevel(), m.Count(), m.CountLayer(d2automap.LayerFloor),
		m.CountLayer(d2automap.LayerWall), m.CountLayer(d2automap.LayerObject))
}

// logMarkers logs the units that get a marker (verification aid).
func (a *Automap) logMarkers() {
	hx, hy := a.hero.GetPositionF()
	a.Infof("AUTOMAP marker self id=%s pos=(%.1f,%.1f)", a.hero.ID(), hx, hy)

	for id, e := range a.eng.Entities() {
		x, y := e.GetPositionF()

		switch v := e.(type) {
		case *d2mapentity.Player:
			if v != a.hero {
				a.Infof("AUTOMAP marker other-player id=%s name=%q party=%v pos=(%.1f,%.1f)", id, v.Name(),
					a.gc.isPartyMember(v), x, y)
			}
		case *d2mapentity.NPC:
			a.Infof("AUTOMAP marker npc %q pos=(%.1f,%.1f)", v.Label(), x, y)
		}
	}
}

func (a *Automap) currentLevel() int {
	if a.levelFn != nil {
		return a.levelFn()
	}

	return 0
}

func (a *Automap) model() *d2automap.Model {
	id := a.currentLevel()

	return a.store.Level(id)
}

// option reads an automap option: a console override, else Options -> Automap.
func (a *Automap) option(key string) bool {
	if v, ok := a.override[key]; ok {
		return v
	}

	return automapOptionOn(key)
}

func (a *Automap) markerOptions() d2automap.Options {
	return d2automap.Options{ShowParty: a.option("party"), ShowNames: a.option("names")}
}

// loadTable reads automap.bin (authoritative; the txt may be stripped) or else AutoMap.txt.
func (a *Automap) loadTable() {
	if a.table != nil || a.tableErr != nil {
		return
	}

	if data, err := a.asset.LoadFile(automapBinPath); err == nil && len(data) > 0 {
		t, perr := d2automap.ParseBin(data)
		if perr == nil {
			a.table = t
			a.Infof("AUTOMAP table: automap.bin, %d rows", len(t.Rows()))
			a.loadObjectCels()

			return
		}

		a.Warningf("automap.bin: %v; trying AutoMap.txt", perr)
	}

	data, err := a.asset.LoadFile(d2resource.AutoMap)
	if err != nil {
		a.tableErr = err
		a.Errorf("AUTOMAP: no automap.bin and no AutoMap.txt: %v", err)

		return
	}

	t, err := d2automap.ParseTxt(strings.NewReader(string(data)))
	if err != nil {
		a.tableErr = err
		a.Errorf("AUTOMAP: AutoMap.txt: %v", err)

		return
	}

	a.table = t
	a.Infof("AUTOMAP table: AutoMap.txt, %d rows", len(t.Rows()))
	a.loadObjectCels()
}

func (a *Automap) loadObjectCels() {
	a.objCel = map[int]int{}

	data, err := a.asset.LoadFile(d2resource.ObjectDetails)
	if err != nil {
		return
	}

	d := d2txt.LoadDataDictionary(data)

	for idx := 0; d.Next(); idx++ {
		if v := d.Number("AutoMap"); v > 0 {
			a.objCel[idx] = v
		}
	}
}

// Advance runs the reveal (also while the map is hidden, like the original).
func (a *Automap) Advance(elapsed float64) {
	a.loadTable()

	if a.table == nil || a.eng == nil || a.hero == nil {
		return
	}

	lvl := a.currentLevel()
	if lvl != a.level {
		a.level, a.haveLast = lvl, false
	}

	hx, hy := a.hero.GetPositionF()
	cx, cy := d2automap.WorldCell(hx, hy)

	if a.haveLast && !d2automap.MovedFar(a.lastCX, a.lastCY, cx, cy) {
		return
	}

	a.lastCX, a.lastCY, a.haveLast = cx, cy, true

	m := a.model()
	before := m.Count()
	seed := uint32(lvl+1) * automapSeedMul

	w, h := a.screenSize()
	m.RevealTiles(a.table, tileSource{a.eng}, a.eng.LevelType().ID, seed, hx, hy, d2automap.ScreenArea(w, h))
	a.revealObjects(m, hx, hy, d2automap.ScreenArea(w, h))
	a.lastAdded = m.Count() - before

	if a.lastAdded > 0 {
		a.Debugf("AUTOMAP reveal level=%d type=%d hero=(%.1f,%.1f) added=%d total=%d floor=%d wall=%d object=%d",
			lvl, a.eng.LevelType().ID, hx, hy, a.lastAdded, m.Count(),
			m.CountLayer(d2automap.LayerFloor), m.CountLayer(d2automap.LayerWall), m.CountLayer(d2automap.LayerObject))
	}
}

func (a *Automap) screenSize() (w, h int) {
	return 800, 600
}

func (a *Automap) revealObjects(m *d2automap.Model, hx, hy float64, area d2automap.Area) int {
	added := 0
	hcx, hcy := d2automap.WorldCell(hx, hy)

	for _, e := range a.eng.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok {
			continue
		}

		rec := ob.Record()
		if rec == nil {
			continue
		}

		cel := a.objCel[rec.Index]
		if cel <= 0 {
			continue
		}

		ox, oy := ob.GetPositionF()
		cx, cy := d2automap.WorldCell(ox, oy)

		if cx-hcx > float64(area.HalfW) || hcx-cx > float64(area.HalfW) || cy-hcy > float64(area.HalfH) || hcy-cy > float64(area.HalfH) {
			continue
		}

		if m.AddObject(cel, ox, oy) {
			added++
		}
	}

	return added
}

// tileSource adapts the map engine's tiles to d2automap.TileSource.
type tileSource struct{ eng *d2mapengine.MapEngine }

func (s tileSource) Size() (w, h int) {
	sz := s.eng.Size()
	return sz.Width, sz.Height
}

func (s tileSource) Tiles(tx, ty int) []d2automap.TileRef {
	t := s.eng.TileAt(tx, ty)
	if t == nil {
		return nil
	}

	var refs []d2automap.TileRef

	for i := range t.Components.Floors {
		f := &t.Components.Floors[i]
		if f.Hidden() || f.Prop1 == 0 {
			continue
		}

		refs = append(refs, d2automap.TileRef{Layer: d2automap.LayerFloor, Type: int(d2enum.TileFloor), Style: int(f.Style), Sequence: int(f.Sequence)})
	}

	for i := range t.Components.Walls {
		w := &t.Components.Walls[i]
		if w.Hidden() || w.Prop1 == 0 {
			continue
		}

		refs = append(refs, d2automap.TileRef{Layer: d2automap.LayerWall, Type: int(w.Type), Style: int(w.Style), Sequence: int(w.Sequence)})
	}

	return refs
}
