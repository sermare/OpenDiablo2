package d2player

import (
	"image/color"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// Marker colours: the engine asks for the palette entry nearest to these RGB
// values (FUN_00455d30); the values below are the entries of the Act 1 palette
// that this gives (computed offline with the exe's metric, D2CMP_FindNearestPaletteIndex 0x604300: squared RGB
// distance, first minimum wins; checked against all five act palettes: only the corpse colour (act 2: 252,20,248)
// and the portal colour (act 4: 252,224,52) differ between acts. Which act's palette the exe's table at 0x8166f0
// holds is UNVERIFIED, the Act 1 entries are used).
var (
	automapColorSelf    = color.RGBA{36, 96, 216, 255}   // (0,0,255): the hero's cross
	automapColorOther   = color.RGBA{252, 44, 0, 255}    // (255,0,0): other players, not in the party
	automapColorParty   = color.RGBA{24, 252, 0, 255}    // (0,255,0): party members
	automapColorMinion  = color.RGBA{68, 112, 116, 255}  // (0x44,0x70,0x74): own mercenary and minions
	automapColorNPC     = color.RGBA{244, 244, 244, 255} // (0xf4,0xf4,0xf4) DAT_0079d23e: town folk (monsters with the monstats flag); an exact palette entry (index 32) in every act
	automapColorCorpse  = color.RGBA{164, 32, 252, 255}  // (255,0,255) DAT_0079d23a: a corpse that holds items; Act 1 nearest entry is index 155
	automapColorPortal  = color.RGBA{244, 192, 76, 255}  // (0xf4,0xf4,0): portals
	automapColorTextGld = color.RGBA{199, 179, 119, 255}
)

// automapCross is the closed polyline of the unit marker (table at 0x6d7640 of
// Game.exe, 12 points; every coordinate is doubled when drawn). VERIFIED.
var automapCross = [12][2]int{
	{0, -1}, {2, -2}, {4, -1}, {2, 0}, {4, 1}, {2, 2}, {0, 1}, {-2, 2}, {-4, 1}, {-2, 0}, {-4, -1}, {-2, -2},
}

func (a *Automap) palettePath() string {
	switch a.eng.LevelType().Act {
	case 2:
		return d2resource.PaletteAct2
	case 3:
		return d2resource.PaletteAct3
	case 4:
		return d2resource.PaletteAct4
	case 5:
		return d2resource.PaletteAct5
	default:
		return d2resource.PaletteAct1
	}
}

// sprite returns the MaxiMap (or MaxiMapS) sprite, loaded on first use and
// reloaded when the act's palette changes.
func (a *Automap) sprite(s d2automap.Size) *d2ui.Sprite {
	act := a.eng.LevelType().Act
	if sp := a.sprites[s]; sp != nil && a.spriteAt[s] == act {
		return sp
	}

	path := automapPathFull
	if s == d2automap.SizeMini {
		path = automapPathMini
	}

	sp, err := a.ui.NewSprite(path, a.palettePath())
	if err != nil {
		a.Errorf("AUTOMAP: cannot load %s: %v", path, err)
		a.sprites[s] = nil

		return nil
	}

	a.sprites[s], a.spriteAt[s] = sp, act

	return sp
}

// panelShift tells which side panels are open.
func (a *Automap) panelShift() d2automap.PanelShift {
	switch {
	case a.gc.isLeftPanelOpen():
		return d2automap.PanelLeft
	case a.gc.isRightPanelOpen():
		return d2automap.PanelRight
	}

	return d2automap.PanelNone
}

// Render draws the map, the markers and the level name. It is called before the
// panels, like the original (the automap is drawn before the interface).
func (a *Automap) Render(target d2interface.Surface) {
	if !a.on || a.table == nil || a.eng == nil {
		return
	}

	sp := a.sprite(a.size)
	if sp == nil {
		return
	}

	w, h := target.GetSize()
	hx, hy := a.hero.GetPositionF()
	cx, cy := d2automap.WorldCell(hx, hy)
	shift := a.panelShift()
	left := shift == d2automap.PanelRight // a panel on the right moves the mini map to the left (mode 1)

	if !a.miniSeen || ((left != a.lastLeft || shift != a.lastShift) && a.option("center")) {
		a.miniD210, a.miniD214 = d2automap.MiniBoxOffsets(w, h, left)
		a.miniSeen = true
	}

	a.lastLeft, a.lastShift = left, shift
	lay := d2automap.ComputeLayoutOffsets(a.size, w, h, cx, cy, shift, left, a.miniD210, a.miniD214)

	dest := target
	ox, oy := 0, 0

	if a.size == d2automap.SizeMini {
		// draw into a box-sized surface so that cells are clipped at its border
		bw, bh := lay.Clip.X1-lay.Clip.X0, lay.Clip.Y1-lay.Clip.Y0
		if a.offscr == nil || a.offW != bw || a.offH != bh {
			a.offscr, a.offW, a.offH = a.rend.NewSurface(bw, bh), bw, bh
		}

		a.offscr.Clear(color.Transparent)
		dest, ox, oy = a.offscr, lay.Clip.X0, lay.Clip.Y0
	}

	a.drawCells(dest, sp, lay, ox, oy, w, h, shift)
	a.drawMarkers(dest, lay, ox, oy, cx, cy)

	if a.size == d2automap.SizeMini {
		target.PushTranslation(ox, oy)
		target.Render(a.offscr)
		target.Pop()
	}

	a.drawText(target, w)
}

func (a *Automap) drawCells(dest d2interface.Surface, sp *d2ui.Sprite, lay d2automap.Layout, ox, oy, sw, sh int, shift d2automap.PanelShift) {
	fade := a.option("fade")
	shiftPx := shift.Pixels()

	defer sp.SetColorMod(nil)

	cw, ch := d2automap.CellW, d2automap.CellH
	if a.size == d2automap.SizeMini {
		cw, ch = cw/2, ch/2
	}

	clip := lay.Clip

	for _, c := range a.model().Cells() {
		x, y := lay.ToScreen(c.X, c.Y)
		// the frame is anchored at its bottom left; skip what is entirely off the clip box
		if x+cw <= clip.X0 || x >= clip.X1 || y <= clip.Y0 || y-ch >= clip.Y1 {
			continue
		}

		if err := sp.SetCurrentFrame(c.Cel); err != nil {
			continue
		}

		if fade {
			t := d2automap.CellTransparency(true, a.size, x, y, sw, sh, shiftPx)
			sp.SetColorMod(color.NRGBA{255, 255, 255, uint8(t.Alpha() * 255)})
		}

		sp.SetPosition(x-ox, y-oy)
		sp.Render(dest)
	}
}

// drawCross draws the unit marker polyline at (x,y).
func drawAutomapCross(dest d2interface.Surface, x, y int, c color.Color) {
	n := len(automapCross)
	for i := 0; i < n; i++ {
		p, q := automapCross[i], automapCross[(i+1)%n]
		x1, y1 := x+p[0]*2, y+p[1]*2
		x2, y2 := x+q[0]*2, y+q[1]*2

		dest.PushTranslation(x1, y1)
		dest.DrawLine(x2-x1, y2-y1, c)
		dest.Pop()
	}
}

// playerColor is the marker colour of another player: green for a member of
// the hero's party, red for everybody else (the colours of the original's
// party dots, see the constants).
func (a *Automap) playerColor(p *d2mapentity.Player) color.Color {
	if a.gc.isPartyMember(p) {
		return automapColorParty
	}

	return automapColorOther
}

func (a *Automap) markerColor(c d2automap.ColorID) color.Color {
	switch c {
	case d2automap.ColorSelf:
		return automapColorSelf
	case d2automap.ColorParty:
		return automapColorParty
	case d2automap.ColorMinion:
		return automapColorMinion
	case d2automap.ColorNPC:
		return automapColorNPC
	case d2automap.ColorCorpse:
		return automapColorCorpse
	case d2automap.ColorPortal:
		return automapColorPortal
	case d2automap.ColorBlack:
		return color.RGBA{0, 0, 0, 255}
	}

	return automapColorOther
}

// unitFor describes an entity to the marker logic.
func (a *Automap) unitFor(e d2interface.MapEntity, level int) (d2automap.Unit, bool) {
	switch v := e.(type) {
	case *d2mapentity.Player:
		return d2automap.Unit{Kind: d2automap.UnitPlayer, Self: v == a.hero, Party: v != a.hero && a.gc.isPartyMember(v),
			Dead: v.IsDead(), Level: level}, true
	case *d2mapentity.NPC:
		return d2automap.Unit{Kind: d2automap.UnitMonster, FriendlyNPC: true, Level: level}, true
	case *d2mapentity.Object:
		if rec := v.Record(); rec != nil {
			return d2automap.Unit{Kind: d2automap.UnitObject, ObjectID: rec.Index, Level: level}, true
		}
	}

	return d2automap.Unit{}, false
}

func (a *Automap) drawMarkers(dest d2interface.Surface, lay d2automap.Layout, ox, oy int, hcx, hcy float64) {
	opts := a.markerOptions()
	level := a.currentLevel()
	cdx, cdy := d2automap.CrossOffset(a.size)

	mark := func(tx, ty float64, m d2automap.Marker, name string) {
		cx, cy := d2automap.WorldCell(tx, ty)
		x, y := lay.HeroScreen(cx, cy)
		// the engine draws at (x+8, y-8)
		x, y = x+8, y-8

		if !lay.Clip.Contains(x, y) {
			return
		}

		if m.Cross {
			drawAutomapCross(dest, x-ox+cdx, y-oy+cdy, a.markerColor(m.Color))
		}

		switch m.Label {
		case d2automap.LabelName:
			a.drawMarkerText(dest, name, x-ox, y-oy+d2automap.NameOffsetY, a.markerColor(m.Color))
		case d2automap.LabelStash:
			a.drawMarkerText(dest, "Stash", x-ox, y-oy+d2automap.LabelOffsetY, automapColorTextGld)
		}
	}

	// other units, in a stable order; hostile monsters are not shown by the original either
	ids := make([]string, 0, len(a.eng.Entities()))
	for id := range a.eng.Entities() {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, id := range ids {
		e := a.eng.Entities()[id]

		u, ok := a.unitFor(e, level)
		if !ok || u.Self {
			continue
		}

		ex, ey := e.GetPositionF()

		name := ""

		switch v := e.(type) {
		case *d2mapentity.Player:
			name = v.Name()
		case *d2mapentity.NPC:
			name = v.Label()
		}

		mark(ex, ey, d2automap.Classify(u, opts), name)
	}

	// the hero's own cross is drawn last, on top
	hx, hy := a.hero.GetPositionF()
	mark(hx, hy, d2automap.Classify(d2automap.Unit{Kind: d2automap.UnitPlayer, Self: true, Level: level}, opts), "")
}

// drawMarkerText draws centred text above a marker (AUTOMAP_DrawMarkerNameText).
func (a *Automap) drawMarkerText(dest d2interface.Surface, text string, x, y int, c color.Color) {
	if text == "" {
		return
	}

	if a.nameLabel == nil {
		a.nameLabel = a.ui.NewLabel(d2resource.Font8, d2resource.PaletteStatic)
		if a.nameLabel == nil {
			return
		}

		a.nameLabel.Alignment = d2ui.HorizontalAlignCenter
	}

	a.nameLabel.Color[0] = c
	a.nameLabel.SetText(text)
	a.nameLabel.SetPosition(x, y)
	a.nameLabel.Render(dest)
}

func (a *Automap) drawText(target d2interface.Surface, w int) {
	if a.label == nil {
		a.label = a.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		if a.label == nil {
			return
		}

		a.label.Color[0] = automapColorTextGld
	}

	width := func(s string) int { x, _ := a.label.GetTextMetrics(s); return x }
	_, lh := a.label.GetTextMetrics("A")

	// AUTOMAP_DrawAutomap 0x4564a0: game name, password (join info only, none in single player), level name,
	// version, difficulty (strings by id from the player's own string table), right aligned and 16 apart.
	name := ""
	if a.nameFn != nil {
		name = a.nameFn(a.currentLevel())
	}

	in := d2automap.TextInput{LevelName: name}
	if a.diffFn != nil {
		in.Difficulty = a.diffFn()
	}

	str := func(id int) string { s, _ := a.asset.StringByID(id); return s }

	for _, l := range d2automap.TextStack(d2automap.BuildLines(in, str), w, width) {
		a.label.SetText(l.Text)
		a.label.SetPosition(l.X, l.Y-lh) // the engine's y is the text baseline (UNVERIFIED)
		a.label.Render(target)
	}
}

// textLines are the lines drawn over the automap, in the engine's order: the level name, the version line and, in
// Nightmare and Hell, the difficulty line (AUTOMAP_DrawAutomap 0x4564a0 / 0x4558b0; the string-table texts
// "automapdif" = "Difficulty: %s" and the names behind ids 0x1423 / 0x1422 were read from the real string table).
// The byte AUTOMAP_DrawDifficultyText reads (UI_GetPanelFlagByte 0x449240, DAT_00798694: 1 shows 0x1423, 2 shows 0x1422)
// is the game difficulty: the same byte scales the monsters on creation and the skill description damage (VERIFIED
// by those readers, w44).
func (a *Automap) textLines() []string {
	name := ""
	if a.nameFn != nil {
		name = a.nameFn(a.currentLevel())
	}

	lines := []string{name, d2automap.VersionText}

	if a.diffFn != nil {
		if id := d2automap.DifficultyStringID(a.diffFn()); id != 0 {
			lines = append(lines, d2automap.DifficultyLine(a.asset.TranslateString("automapdif"),
				a.asset.TranslateString(difficultyLabels[id])))
		}
	}

	return lines
}

// difficultyLabels maps the string ids of the difficulty line (0x1423 Nightmare, 0x1422 Hell) to the label numbers
// of the same names in the string table (the same strings the difficulty screen shows).
var difficultyLabels = map[int]int{0x1423: d2enum.NightmareLabel, 0x1422: d2enum.HellLabel}
