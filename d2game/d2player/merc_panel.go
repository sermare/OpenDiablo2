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

func (p *MercPanel) newLabel(font string, centre bool) *d2ui.Label {
	lb := p.ui.NewLabel(font, d2resource.PaletteStatic)
	if centre {
		lb.Alignment = d2ui.HorizontalAlignCenter
	}

	return lb
}

func (p *MercPanel) label(font string, x, y int, centre bool) *d2ui.Label {
	return p.place(p.newLabel(font, centre), x, y, font)
}

// place positions a label. The original's text y is the bottom of the line while a Label's y is its top: font is the
// font of a bottom-anchored text ("" for a label placed by its middle). The heights are the ones the character
// panel's constants differ from its tables by (font6 7, font16 11).
func (p *MercPanel) place(lb *d2ui.Label, x, y int, font string) *d2ui.Label {
	switch font {
	case d2resource.Font6:
		y -= 7
	case d2resource.Font16:
		y -= 11
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
			p.name = p.label(d2resource.Font16, r.X, r.Y, false) // left anchor (0x4869a0)
		case len(r.Name) > 6 && r.Name[:6] == "label_":
			// the static captions come from string.tbl by the id of the original's table
			id, _ := MercLabelString(r.Name[6:])
			// x2 != 0: centred between x1 and x2 (UI_DrawTextLine); else left-anchored at x1
			lb := p.label(d2resource.Font6, r.X+r.W/2, r.Y, r.W > 0)
			text, _ := p.asset.StringByID(id) // the exe id of string.tbl (TranslateString(int) is the OD2 label numbering)
			lb.SetText(text)
		case len(r.Name) > 6 && r.Name[:6] == "value_":
			// flag == 0 rows end at the table x (right-aligned); the life row is centred in its box (mercValue.Anchor)
			lb := p.ui.NewLabel(d2resource.Font6, d2resource.PaletteStatic)
			lb.Alignment = d2ui.HorizontalAlignRight

			if v, ok := mercValueByName(r.Name[6:]); ok && v.Flag != 0 {
				lb.Alignment = d2ui.HorizontalAlignCenter
			}

			p.values[r.Name[6:]] = p.place(lb, r.X, r.Y, d2resource.Font6)
		case len(r.Name) > 5 && r.Name[:5] == "slot_":
			p.slots[r.Name[5:]] = p.place(p.newLabel(d2resource.Font6, true), r.X+r.W/2, r.Y+r.H/2, "")
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
