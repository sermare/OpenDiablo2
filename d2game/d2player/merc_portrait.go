package d2player

import (
	"strings"

	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The hireling portrait of the HUD (UI_DrawHirelingPortraitStrip 0x4904f0 -> UI_DrawHirelingPortraitEntry 0x4902d0): a
// small picture of the mercenary class at the top left of the screen (x 15, bottom edge 60, 0x38 pixels between
// entries) with a life bar over its top (UI_DrawPortraitLifeBar 0x48feb0). The name under the picture
// (UI_DrawHirelingTypeName 0x48f4a0) is the pettype name column, which is empty for the hireable type in the shipped
// pettype.txt (patch_d2), so nothing is drawn there.
const (
	mercPortraitX       = 15 // local_210 of the strip
	mercPortraitBottom  = 60 // GFX_DrvSlot84(.., x, 0x3c): the bottom edge of the picture
	mercPortraitBarTop  = 14 // the bar covers y 0xe..0x13
	mercPortraitBarH    = 5
	mercPortraitBarW    = 46 // 0x2e
	mercPortraitHealthy = 50 // the bar is green from 50 percent
	mercPortraitLow     = 25 // and red below 25 percent

	mercPortraitDir = "/data/global/ui/HIREABLES/"
)

// bar colours: the exe looks up the palette entries nearest to (0,128,0), (192,192,0) and (255,0,0).
var (
	mercBarGreen  = color.RGBA{0, 128, 0, 255}   //nolint:gochecknoglobals // static
	mercBarYellow = color.RGBA{192, 192, 0, 255} //nolint:gochecknoglobals // static
	mercBarRed    = color.RGBA{255, 0, 0, 255}   //nolint:gochecknoglobals // static
	mercBarBack   = color.RGBA{0, 0, 0, 160}     //nolint:gochecknoglobals // static: draw mode 5 black
)

// MercPortraitBar is the life bar of the portrait: the filled width in pixels and its colour for a life. A percentage of 0
// draws no bar at all (the exe returns before painting); 100 and above is clamped to a full bar.
func MercPortraitBar(hp, maxHP int) (fill int, c color.RGBA, drawn bool) {
	if maxHP <= 0 || hp <= 0 {
		return 0, color.RGBA{}, false
	}

	pct := hp * 100 / maxHP
	if pct > 99 {
		pct = 100
	}

	if pct <= 0 {
		return 0, color.RGBA{}, false
	}

	fill = pct * mercPortraitBarW / 100
	c = mercBarGreen

	if pct < mercPortraitHealthy {
		c = mercBarYellow
	}

	if pct < mercPortraitLow {
		c = mercBarRed
	}

	return fill, c, true
}

// MercPortraitIcon is the picture file (under data\global\ui\hireables) of a hireling class. The exe reads the file name
// from the pettype record (UI_LoadHirelingIconForRecord 0x48f700, "hireables\%s"): the icon of the first of mclass1..4
// that equals the unit's class, else baseicon. The hireable row of pettype.txt (patch_d2) reads baseicon rogueicon,
// mclass1 338 act2hireableicon, mclass2 359 act3hireableicon, mclass3 561 barbhirable_icon (hireling.txt has the
// classes 271, 338, 359, 561 only). VERIFIED. An unknown class would get baseicon in the exe; "" here means "not a
// hireling" for the callers.
func MercPortraitIcon(class int) string {
	switch class {
	case 271: // Rogue Scout
		return "rogueicon"
	case 338: // Desert Mercenary
		return "act2hireableicon"
	case 359: // Iron Wolf
		return "act3hireableicon"
	case 561: // Barbarian (Act 5)
		return "barbhirable_icon"
	}

	return ""
}

// MercPortrait draws the portrait strip.
type MercPortrait struct {
	ui    *d2ui.UIManager
	icons map[string]*d2ui.Sprite
	tip   []*d2ui.Label
	name  *d2ui.Label
}

// NewMercPortrait creates the portrait.
func NewMercPortrait(_ *d2asset.AssetManager, ui *d2ui.UIManager) *MercPortrait {
	return &MercPortrait{ui: ui, icons: map[string]*d2ui.Sprite{}}
}

func (p *MercPortrait) icon(name string) *d2ui.Sprite {
	if s, ok := p.icons[name]; ok {
		return s
	}

	s, err := p.ui.NewSprite(mercPortraitDir+name+".DC6", d2resource.PaletteSky)
	if err != nil {
		s = nil
	}

	p.icons[name] = s

	return s
}

// Render draws the portrait of the view (nothing for a class without a picture).
func (p *MercPortrait) Render(target d2interface.Surface, v MercView) {
	name := MercPortraitIcon(v.Class)
	if name == "" {
		return
	}

	if s := p.icon(name); s != nil {
		s.SetPosition(mercPortraitX, mercPortraitBottom)
		s.Render(target)
	}

	p.renderName(target, v.TypeName)

	hp := v.HP
	if v.Dead {
		hp = 0
	}

	if fill, c, ok := MercPortraitBar(hp, v.MaxHP); ok {
		target.PushTranslation(mercPortraitX, mercPortraitBarTop)
		target.DrawRect(fill, mercPortraitBarH, c)
		target.PushTranslation(fill, 0)
		target.DrawRect(mercPortraitBarW-fill, mercPortraitBarH, mercBarBack)
		target.Pop()
		target.Pop()
	}
}

// MercPortraitHit reports whether a point in the interface column is on the portrait picture (a drop target for a
// potion or an item, and a right-click target for the inventory of the merc).
func MercPortraitHit(x, y int) bool {
	return x >= mercPortraitX && x < mercPortraitX+mercPortraitBarW && y >= mercPortraitBarTop && y < mercPortraitBottom
}

// MercPortraitTooltip is the text over the portrait (UI_DrawHirelingPortraitStrip: string 0x58a3 with the name of the
// command key, a line break and string 0x58a2): the two lines are given by the caller's string lookup
// (hireiconinfo2 holds a %s for the key, hireiconinfo1 is the potion hint).
func MercPortraitTooltip(open, potion, key string) []string {
	if i := strings.Index(open, "%s"); i >= 0 {
		open = open[:i] + key + open[i+2:]
	}

	return []string{open, potion}
}

// RenderTooltip draws the hover text under the mouse.
func (p *MercPortrait) RenderTooltip(target d2interface.Surface, lines []string, x, y int) {
	for len(p.tip) < len(lines) {
		l := p.ui.NewLabel(d2resource.Font6, d2resource.PaletteStatic)
		p.tip = append(p.tip, l)
	}

	w := 0

	for i, t := range lines {
		p.tip[i].SetText(t)

		if tw, _ := p.tip[i].GetTextMetrics(t); tw > w {
			w = tw
		}
	}

	target.PushTranslation(x+8, y+16)
	target.DrawRect(w+8, 14*len(lines)+4, color.RGBA{A: 200})
	target.Pop()

	for i := range lines {
		p.tip[i].SetPosition(x+12, y+28+14*i)
		p.tip[i].Render(target)
	}
}

// mercPortraitNameW is the widest the type name under the picture may be (UI_DrawClippedPortraitName 0x48f420 cuts characters off
// the end until the text is at most 0x42 pixels wide).
const mercPortraitNameW = 0x42

// ClipPortraitName cuts the name at the end until measure says it fits in 0x42 pixels.
func ClipPortraitName(name string, measure func(string) int) string {
	r := []rune(name)
	for len(r) > 0 && measure(string(r)) > mercPortraitNameW {
		r = r[:len(r)-1]
	}

	return string(r)
}

// renderName draws the type name (UI_DrawHirelingTypeName 0x48f4a0: the name string of the pettype record, nothing for string id 0
// or 0x1506 "an evil force") right under the picture. The row of the text is UNVERIFIED (the decompiler lost the argument).
func (p *MercPortrait) renderName(target d2interface.Surface, name string) {
	if name == "" {
		return
	}

	if p.name == nil {
		p.name = p.ui.NewLabel(d2resource.Font6, d2resource.PaletteStatic)
	}

	name = ClipPortraitName(name, func(s string) int { w, _ := p.name.GetTextMetrics(s); return w })
	p.name.SetText(name)
	p.name.SetPosition(mercPortraitX, mercPortraitBottom+14)
	p.name.Render(target)
}
