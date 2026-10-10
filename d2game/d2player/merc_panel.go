package d2player

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// MercPanelHost connects the mercenary panel to the game.
type MercPanelHost struct {
	// View is the merc's numbers (false without a living or dead merc of the hero).
	View func() (MercView, bool)
	// ItemCode is the item code worn at a slot ("head", "torso", "weapon", "shield"), "" for none.
	ItemCode func(slot string) string
}

// MercPanel is the hireling screen: the merc's name, level, experience, stats, resistances and its four body
// slots (UI_DrawMercenaryPanel 0x48ee50). It sits on the left like the character panel; the layout is
// Mode800.MercPanelRects (numbers read from the executable). Items are given to and taken from the merc by
// clicking a slot (GameControls.mercPanelClick). The pictures are a stand-in (the character panel's), see
// uilayout_merc.go.
type MercPanel struct {
	asset *d2asset.AssetManager
	ui    *d2ui.UIManager
	host  MercPanelHost

	group  *d2ui.WidgetGroup
	panel  *d2ui.Sprite
	name   *d2ui.Label
	values map[string]*d2ui.Label
	slots  map[string]*d2ui.Label
	onShut func()
	open   bool

	*d2util.Logger
}

// NewMercPanel creates the panel.
func NewMercPanel(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel) *MercPanel {
	p := &MercPanel{asset: asset, ui: ui, values: map[string]*d2ui.Label{}, slots: map[string]*d2ui.Label{}}
	p.Logger = d2util.NewLogger()
	p.Logger.SetLevel(l)
	p.Logger.SetPrefix("Merc Panel")

	return p
}

// SetHost connects the panel to the game.
func (p *MercPanel) SetHost(h MercPanelHost) {
	if p != nil {
		p.host = h
	}
}

// SetOnCloseCb sets the function called when the panel closes.
func (p *MercPanel) SetOnCloseCb(cb func()) {
	if p != nil {
		p.onShut = cb
	}
}

// IsOpen reports whether the panel is shown.
func (p *MercPanel) IsOpen() bool { return p != nil && p.open }

func (p *MercPanel) label(font string, x, y int, centre bool) *d2ui.Label {
	lb := p.ui.NewLabel(font, d2resource.PaletteStatic)
	if centre {
		lb.Alignment = d2ui.HorizontalAlignCenter
	}

	lb.SetPosition(x, y)
	p.group.AddWidget(lb)

	return lb
}

// Load builds the widgets (hidden).
func (p *MercPanel) Load() {
	if p == nil {
		return
	}

	var err error

	m := Mode800

	p.group = p.ui.NewWidgetGroup(d2ui.RenderPriorityHeroStatsPanel)

	frame := p.ui.NewUIFrame(d2ui.FrameLeft)
	p.group.AddWidget(frame)

	if p.panel, err = p.ui.NewSprite(d2resource.InventoryCharacterPanel, d2resource.PaletteSky); err != nil {
		p.Error(err.Error())
	}

	w, h := frame.GetSize()
	p.group.AddWidget(p.ui.NewCustomWidgetCached(p.renderFrames, w, h))

	for _, r := range m.MercPanelRects() {
		switch {
		case r.Name == "close":
			b := p.ui.NewButton(d2ui.ButtonTypeSquareClose, "")
			b.SetVisible(false)
			b.SetPosition(r.X, r.Y)
			b.OnActivated(func() { p.Close() })
			p.group.AddWidget(b)
		case r.Name == "name":
			p.name = p.label(d2resource.Font16, r.X+r.W/2, r.Y, true)
		case len(r.Name) > 6 && r.Name[:6] == "label_":
			// the static captions come from string.tbl by the id of the original's table
			id, _ := MercLabelString(r.Name[6:])
			x := r.X
			if x < 0 {
				x = m.LeftPanelX() + 10 // UNVERIFIED: the x of these captions is a register argument in the original
			}

			lb := p.label(d2resource.Font6, x, r.Y, false)
			lb.SetText(p.asset.TranslateString(id))
		case len(r.Name) > 6 && r.Name[:6] == "value_":
			p.values[r.Name[6:]] = p.label(d2resource.Font6, r.X, r.Y, false)
		case len(r.Name) > 5 && r.Name[:5] == "slot_":
			p.slots[r.Name[5:]] = p.label(d2resource.Font6, r.X+r.W/2, r.Y+r.H/2, true)
		}
	}

	p.group.SetVisible(false)
}

// renderFrames draws the four pictures at the positions of Mode800.PanelPictures.
func (p *MercPanel) renderFrames(target d2interface.Surface) {
	if p.panel == nil {
		return
	}

	order := []struct{ frame, pic int }{{0, 0}, {1, 1}, {3, 3}, {2, 2}}
	pics := Mode800.PanelPictures(Mode800.LeftPanelX())

	for _, o := range order {
		if err := p.panel.SetCurrentFrame(o.frame); err != nil {
			p.Error(err.Error())

			continue
		}

		_, h := p.panel.GetCurrentFrameSize()
		p.panel.SetPosition(pics[o.pic].X, pics[o.pic].Y+h)
		p.panel.Render(target)
	}
}

// Open shows the panel.
func (p *MercPanel) Open() {
	p.open = true
	p.group.SetVisible(true)
	p.refresh()
}

// Close hides the panel.
func (p *MercPanel) Close() {
	if p == nil {
		return
	}

	was := p.open
	p.open = false

	if p.group != nil {
		p.group.SetVisible(false)
	}

	if was && p.onShut != nil {
		p.onShut()
	}
}

// Toggle opens or closes the panel.
func (p *MercPanel) Toggle() {
	if p.open {
		p.Close()
	} else {
		p.Open()
	}
}

// Advance refreshes the texts while the panel is open.
func (p *MercPanel) Advance(float64) {
	if p != nil && p.open {
		p.refresh()
	}
}

func (p *MercPanel) refresh() {
	if p == nil || p.host.View == nil || p.name == nil {
		return
	}

	v, ok := p.host.View()
	if !ok {
		// no merc: the panel closes itself (UI_DrawMercenaryPanel sets the panel state to 0)
		p.Close()

		return
	}

	p.name.SetText(v.Name)

	for n, s := range v.Values() {
		if lb := p.values[n]; lb != nil {
			lb.SetText(s)
		}
	}

	for n, lb := range p.slots {
		code := ""
		if p.host.ItemCode != nil {
			code = p.host.ItemCode(n)
		}

		lb.SetText(code)
	}
}
