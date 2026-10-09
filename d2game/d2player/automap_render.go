package d2player

import (
	"image/color"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2automap"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// Marker colours: the engine asks for the palette entry nearest to these RGB
// values (FUN_00455d30); the values below are the entries of the Act 1 palette
// that this gives (computed offline, which palette the engine searches is
// UNVERIFIED but the colours only differ slightly between acts).
var (
	automapColorSelf    = color.RGBA{36, 96, 216, 255}  // (0,0,255): the hero's cross
	automapColorOther   = color.RGBA{252, 44, 0, 255}   // (255,0,0): other players, not in the party
	automapColorParty   = color.RGBA{24, 252, 0, 255}   // (0,255,0): party members
	automapColorMinion  = color.RGBA{68, 112, 116, 255} // (0x44,0x70,0x74): own mercenary and minions
	automapColorNPC     = color.RGBA{72, 160, 52, 255}  // (0x48,0xa0,0x34): friendly town folk
	automapColorPortal  = color.RGBA{244, 192, 76, 255} // (0xf4,0xf4,0): portals
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
	lay := d2automap.ComputeLayout(a.size, w, h, cx, cy, shift, a.miniLeft)

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

	a.drawCells(dest, sp, lay, ox, oy, w, h)
	a.drawMarkers(dest, lay, ox, oy, cx, cy)

	if a.size == d2automap.SizeMini {
		target.PushTranslation(ox, oy)
		target.Render(a.offscr)
		target.Pop()
	}

	a.drawText(target, w)
}

func (a *Automap) drawCells(dest d2interface.Surface, sp *d2ui.Sprite, lay d2automap.Layout, ox, oy, sw, sh int) {
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

func (a *Automap) drawMarkers(dest d2interface.Surface, lay d2automap.Layout, ox, oy int, hcx, hcy float64) {
	mark := func(tx, ty float64, c color.Color) {
		cx, cy := d2automap.WorldCell(tx, ty)
		x, y := lay.HeroScreen(cx, cy)
		// the engine draws at (x+8, y-8)
		x, y = x+8, y-8

		if !lay.Clip.Contains(x, y) {
			return
		}

		drawAutomapCross(dest, x-ox, y-oy, c)
	}

	// other units: players, friendly NPCs and portals; hostile monsters are not
	// shown by the original either
	ids := make([]string, 0, len(a.eng.Entities()))
	for id := range a.eng.Entities() {
		ids = append(ids, id)
	}

	sort.Strings(ids)

	for _, id := range ids {
		e := a.eng.Entities()[id]
		ex, ey := e.GetPositionF()

		switch v := e.(type) {
		case *d2mapentity.Player:
			if v != a.hero {
				mark(ex, ey, automapColorOther)
			}
		case *d2mapentity.NPC:
			mark(ex, ey, automapColorNPC)
		case *d2mapentity.Object:
			if rec := v.Record(); rec != nil && (rec.Index == 59 || rec.Index == 60) {
				mark(ex, ey, automapColorPortal)
			}
		}
	}

	hx, hy := a.hero.GetPositionF()
	mark(hx, hy, automapColorSelf)
}

func (a *Automap) drawText(target d2interface.Surface, w int) {
	if a.label == nil {
		a.label = a.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		if a.label == nil {
			return
		}

		a.label.Alignment = d2ui.HorizontalAlignRight
		a.label.Color[0] = automapColorTextGld
	}

	name := ""
	if a.nameFn != nil {
		name = a.nameFn(a.currentLevel())
	}

	if name == "" {
		return
	}

	a.label.SetText(name)
	a.label.SetPosition(w-16, 40)
	a.label.Render(target)
}
