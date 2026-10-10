package d2player

import (
	"fmt"
	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The waypoint panel of the original (UI_DrawWaygatePanel 0x499190, table 0x721860; numbers verified unless
// said otherwise): a left panel (320 x 432 at (80, 60)); the header text stands on y 0x30 of the panel; the rows
// have their icon bottom at 89 + 36*i and their text bottom at 84 + 35*i (panel space, up to 9 rows); the act tab
// stands at x 5 + 62*(act-1) on y 94 (expansion); the close button is 32x32 standing at (0x111, 0x1a1).
const (
	wpPanelX, wpPanelY   = 80, 60
	wpPanelW, wpPanelH   = 320, 432
	wpRowX, wpRowTop     = 17, 60 // top-left of row 0 inside the panel: UNVERIFIED (inferred from the icon bottom)
	wpRowPitch           = 36
	wpRowH               = 29  // UNVERIFIED
	wpRowW               = 283 // UNVERIFIED
	wpTextBottom0        = 84
	wpTextPitch          = 35
	wpTextX              = 60 // UNVERIFIED: the text column
	wpHeaderBottom       = 0x30
	wpTabX, wpTabPitch   = 5, 62 // wpTabPitch is only the fallback; the table wpTabXs is the original's
	wpTabBottom          = 94
	wpCloseX, wpCloseTop = 0x111, 0x1a1 - 32
	screenW              = 800
	screenH              = 600
)

// wpTabXs are the act tab x positions inside the panel (the table in UI_DrawWaygatePanel, expansion, verified).
//
//nolint:gochecknoglobals // table
var wpTabXs = [5]int{5, 0x43, 0x81, 0xbf, 0xfd}

//nolint:gochecknoglobals // colours
var (
	wpBackground = color.RGBA{R: 0, G: 0, B: 0, A: 215}
	wpHighlight  = color.RGBA{R: 90, G: 20, B: 20, A: 220}
	wpWhite      = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	wpGrey       = color.RGBA{R: 110, G: 110, B: 110, A: 255}
	wpGold       = color.RGBA{R: 255, G: 215, B: 0, A: 255}
)

// WaypointRow is one line of the waypoint panel.
type WaypointRow struct {
	d2level.WaypointEntry
	Name string
}

// WaypointPanel lists the waypoints of an act. Rows the hero has activated and
// the engine can load are selectable; the others are drawn greyed out.
type WaypointPanel struct {
	asset    *d2asset.AssetManager
	ui       *d2ui.UIManager
	title    *d2ui.Label
	labels   []*d2ui.Label
	rows     []WaypointRow
	act      int
	current  int
	open     bool
	hover    int
	onChoose func(level int)
	tabs     *d2ui.Sprite // expwaygatetabs.dc6: frame 2*act = selected, 2*act+1 = other
	closeBtn *d2ui.Sprite // buysellbtn.dc6: frame 10 = close, 11 = pressed
}

// NewWaypointPanel creates a closed panel.
func NewWaypointPanel(asset *d2asset.AssetManager, ui *d2ui.UIManager) *WaypointPanel {
	return &WaypointPanel{asset: asset, ui: ui, hover: -1}
}

// Open shows the panel for an act. current is the level the hero stands in
// (marked); onChoose is called with the level of a chosen, enabled row.
func (p *WaypointPanel) Open(act, current int, rows []WaypointRow, onChoose func(level int)) {
	p.act, p.current, p.rows, p.onChoose = act, current, rows, onChoose
	p.hover, p.open = -1, true

	p.loadArt()

	p.title = p.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	p.title.SetText(fmt.Sprintf("Waypoints - Act %d", act))
	p.title.Color[0] = wpGold

	p.labels = p.labels[:0]

	for _, r := range rows {
		l := p.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		text := r.Name

		if r.Level == current {
			text += " *"
		}

		l.SetText(text)

		if r.Enabled() {
			l.Color[0] = wpWhite
		} else {
			l.Color[0] = wpGrey
		}

		p.labels = append(p.labels, l)
	}
}

// loadArt loads the act tab and close button pictures once.
func (p *WaypointPanel) loadArt() {
	var err error

	if p.tabs == nil {
		if p.tabs, err = p.ui.NewSprite(d2resource.WPTabs, d2resource.PaletteSky); err != nil {
			p.tabs = nil
		}
	}

	if p.closeBtn == nil {
		if p.closeBtn, err = p.ui.NewSprite(d2resource.BuySellButton, d2resource.PaletteSky); err != nil {
			p.closeBtn = nil
		}
	}
}

// tabBox is the box of the act tab i (0..4): UI_DrawWaygatePanel 0x499190 draws frame 2*i of expwaygatetabs.dc6 at
// x = panel + wpTabXs[i], bottom y 94 (expansion; verified).
func (p *WaypointPanel) tabBox(i int) (x, top, w, h int) {
	x = wpPanelX + wpTabXs[i]
	w, h = 62, 33

	if p.tabs != nil {
		if fw, fh, err := p.tabs.GetFrameSize(2 * i); err == nil {
			w, h = fw, fh
		}
	}

	return x, wpTabBottom - h + 0, w, h
}

// closeBox is the 32x32 close button (frame 10 of buysellbtn.dc6): left panel x + 0x111, bottom 0x1a1 + 60 (verified).
func closeBox() (x, y, w, h int) {
	return wpPanelX + wpCloseX, wpPanelY + wpCloseTop, 32, 32
}

// Close hides the panel.
func (p *WaypointPanel) Close() { p.open = false }

// IsOpen reports whether the panel is visible.
func (p *WaypointPanel) IsOpen() bool { return p.open }

// Act returns the act the panel lists.
func (p *WaypointPanel) Act() int { return p.act }

// Rows returns the rows of the open panel.
func (p *WaypointPanel) Rows() []WaypointRow { return p.rows }

// Choose picks the row of a level, as a click would. It returns an error if
// the panel is closed, the level is not listed or the row is not enabled.
func (p *WaypointPanel) Choose(level int) error {
	if !p.open {
		return fmt.Errorf("waypoint panel is not open")
	}

	for _, r := range p.rows {
		if r.Level != level {
			continue
		}

		if !r.Enabled() {
			return fmt.Errorf("waypoint %d is not available (active=%v loadable=%v)", level, r.Active, r.Loadable)
		}

		p.Close()

		if p.onChoose != nil {
			p.onChoose(level)
		}

		return nil
	}

	return fmt.Errorf("level %d is not in the waypoint list of act %d", level, p.act)
}

func (p *WaypointPanel) bounds() (left, top, right, bottom int) {
	return wpPanelX, wpPanelY, wpPanelX + wpPanelW, wpPanelY + wpPanelH
}

// rowBox is the hit rectangle of row i on the screen.
func rowBox(i int) (x, y, w, h int) {
	return wpPanelX + wpRowX, wpPanelY + wpRowTop + i*wpRowPitch, wpRowW, wpRowH
}

func (p *WaypointPanel) rowAt(mx, my int) int {
	for i := range p.rows {
		if x, y, w, h := rowBox(i); mx >= x && mx < x+w && my >= y && my < y+h {
			return i
		}
	}

	return -1
}

// Contains reports whether the point is on the panel.
func (p *WaypointPanel) Contains(mx, my int) bool {
	if !p.open {
		return false
	}

	left, top, right, bottom := p.bounds()

	return mx >= left && mx < right && my >= top && my < bottom
}

// OnMouseMove updates the highlighted row.
func (p *WaypointPanel) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !p.open {
		return false
	}

	p.hover = p.rowAt(event.X(), event.Y())

	return p.Contains(event.X(), event.Y())
}

// OnMouseButtonDown handles clicks; it returns true when the panel took it.
func (p *WaypointPanel) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !p.open || !p.Contains(event.X(), event.Y()) {
		return false
	}

	if event.Button() == d2enum.MouseButtonLeft {
		if x, y, w, h := closeBox(); event.X() >= x && event.X() < x+w && event.Y() >= y && event.Y() < y+h {
			p.Close()

			return true
		}

		if i := p.rowAt(event.X(), event.Y()); i >= 0 && i < len(p.rows) && p.rows[i].Enabled() {
			_ = p.Choose(p.rows[i].Level)
		}
	}

	return true
}

// Render draws the panel.
func (p *WaypointPanel) Render(target d2interface.Surface) {
	if !p.open {
		return
	}

	p.place()

	left, top, _, _ := p.bounds()

	target.PushTranslation(left, top)
	target.DrawRect(wpPanelW, wpPanelH, wpBackground)
	target.Pop()

	p.renderArt(target)
	p.title.Render(target)

	for i, l := range p.labels {
		if i == p.hover && p.rows[i].Enabled() {
			x, y, w, h := rowBox(i)
			target.PushTranslation(x, y)
			target.DrawRect(w, h, wpHighlight)
			target.Pop()
		}

		l.Render(target)
	}
}

// renderArt draws the act tabs and the close button. The original draws a tab for every act whose quest flag is set
// (flags 7, 0xf, 0x17, 0x1a for acts 2..5); the engine does not carry those flags here, so the tabs of the acts up to
// the panel's own are drawn (UNVERIFIED approximation); the panel's act is the selected one (frame 2*i, else 2*i+1).
func (p *WaypointPanel) renderArt(target d2interface.Surface) {
	if p.tabs != nil {
		for i := 0; i < p.act && i < len(wpTabXs); i++ {
			frame := 2*i + 1
			if i == p.act-1 {
				frame = 2 * i
			}

			x, top, _, h := p.tabBox(i)

			if p.tabs.SetCurrentFrame(frame) == nil {
				p.tabs.SetPosition(x, top+h)
				p.tabs.Render(target)
			}
		}
	}

	if p.closeBtn != nil {
		x, y, _, h := closeBox()

		if p.closeBtn.SetCurrentFrame(10) == nil {
			p.closeBtn.SetPosition(x, y+h)
			p.closeBtn.Render(target)
		}
	}
}

// place puts the header and the row texts where the original draws them.
func (p *WaypointPanel) place() {
	tw, th := p.title.GetTextMetrics(p.title.GetText())
	p.title.SetPosition(wpPanelX+(wpPanelW-tw)/2, wpPanelY+wpHeaderBottom-th)

	for i, l := range p.labels {
		_, h := l.GetTextMetrics(l.GetText())
		l.SetPosition(wpPanelX+wpTextX, wpPanelY+wpTextBottom0+i*wpTextPitch-h)
	}
}

// layoutRects returns the panel box, the header and row text anchors, the row hit boxes
// and the act tab and close button pictures, for the layout log.
func (p *WaypointPanel) layoutRects() []UIRect {
	p.place()

	l, t, r, b := p.bounds()
	rs := []UIRect{
		{"waypoint", "panel", l, t, r - l, b - t},
		textAnchor("waypoint", "text.header", p.title),
	}

	for i := 0; i < p.act && i < len(wpTabXs); i++ {
		x, top, w, h := p.tabBox(i)
		rs = append(rs, UIRect{"waypoint", fmt.Sprintf("tab%d", i+1), x, top, w, h})
	}

	cx, cy, cw, ch := closeBox()
	rs = append(rs, UIRect{"waypoint", "close", cx, cy, cw, ch})

	for i, lab := range p.labels {
		x, y, w, h := rowBox(i)
		rs = append(rs, UIRect{"waypoint", fmt.Sprintf("row%d", i), x, y, w, h}, textAnchor("waypoint", fmt.Sprintf("text.row%d", i), lab))
	}

	return rs
}
