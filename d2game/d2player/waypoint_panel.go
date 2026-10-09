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

const (
	wpRowHeight = 24
	wpWidth     = 260
	wpPadding   = 10
	wpTitleH    = 34
	screenW     = 800
	screenH     = 600
)

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
	h := wpTitleH + len(p.rows)*wpRowHeight + wpPadding
	left, top = (screenW-wpWidth)/2, (screenH-h)/2

	return left, top, left + wpWidth, top + h
}

func (p *WaypointPanel) rowAt(mx, my int) int {
	left, top, right, bottom := p.bounds()
	if mx < left || mx >= right || my < top+wpTitleH || my >= bottom-wpPadding {
		return -1
	}

	return (my - top - wpTitleH) / wpRowHeight
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

	left, top, _, _ := p.bounds()

	target.PushTranslation(left, top)
	target.DrawRect(wpWidth, wpTitleH+len(p.rows)*wpRowHeight+wpPadding, wpBackground)
	target.Pop()

	tw, _ := p.title.GetTextMetrics(p.title.GetText())
	p.title.SetPosition(left+(wpWidth-tw)/2, top+wpTitleH-12)
	p.title.Render(target)

	for i, l := range p.labels {
		rowTop := top + wpTitleH + i*wpRowHeight

		if i == p.hover && p.rows[i].Enabled() {
			target.PushTranslation(left, rowTop)
			target.DrawRect(wpWidth, wpRowHeight, wpHighlight)
			target.Pop()
		}

		l.SetPosition(left+wpPadding, rowTop+wpRowHeight-6)
		l.Render(target)
	}
}
