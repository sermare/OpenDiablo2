package d2player

import (
	"fmt"
	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The geometry of the original's dialog (UI_CreateDialog 0x4b47b0 / layout 0x4b4d40, 0x4b4cd0, hit test 0x4b3a00;
// verified, see the d2-re-notes ui-layout notes): the width is the widest text plus 20, the height the sum of the row
// advances plus 15; the NPC's name line advances 0x15 and every menu line 0xf; the text is centred in the box; the
// box is clamped to the screen (10 pixels from the sides and the top, 0x3a above the bottom).
const (
	npcMenuTitleAdvance = 0x15
	npcMenuRowAdvance   = 0xf
	npcMenuWidthPad     = 20
	npcMenuHeightPad    = 15
	npcMenuEdge         = 10
	npcMenuBottomEdge   = 0x3a
	npcMenuHitInsetX    = 15
	npcMenuHitAbove     = 11
	npcMenuHitBelow     = 4
	npcMenuScreenW      = 800
	npcMenuScreenH      = 600
)

//nolint:gochecknoglobals // colours
var (
	npcMenuBackground = color.RGBA{R: 0, G: 0, B: 0, A: 200}
	npcMenuHighlight  = color.RGBA{R: 90, G: 20, B: 20, A: 220}
)

// NPCMenu is the Talk / Trade / Gamble / ... list shown next to an NPC.
type NPCMenu struct {
	asset   *d2asset.AssetManager
	ui      *d2ui.UIManager
	title   *d2ui.Label
	labels  []*d2ui.Label
	rows    []NPCMenuRow
	open    bool
	x, y    int
	width   int
	hover   int
	onClick func(NPCMenuRow)
}

// NewNPCMenu creates a closed NPC menu.
func NewNPCMenu(asset *d2asset.AssetManager, ui *d2ui.UIManager) *NPCMenu {
	return &NPCMenu{asset: asset, ui: ui, hover: -1}
}

// RowLabel returns the display text of a row, using the string table id
// the original game uses and falling back to English text.
func (m *NPCMenu) RowLabel(row NPCMenuRow) string {
	text := m.asset.TranslateString(row.Key)
	if text == "" || text == row.Key {
		return row.Fallback
	}

	return text
}

// Rows returns the rows the menu shows, with the implicit Cancel appended.
func (m *NPCMenu) Rows() []NPCMenuRow {
	return m.rows
}

// Open shows the menu for an NPC at the given screen position (the centre
// of the NPC's head). The Cancel row is appended automatically.
func (m *NPCMenu) Open(name string, rows []NPCMenuRow, anchorX, anchorY int, onClick func(NPCMenuRow)) {
	m.rows = append(append([]NPCMenuRow{}, rows...), RowCancel)
	m.onClick = onClick
	m.hover = -1
	m.open = true

	m.title = m.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	m.title.SetText(name)
	m.title.Color[0] = color.RGBA{R: 255, G: 215, B: 0, A: 255}

	m.labels = m.labels[:0]
	titleW, _ := m.title.GetTextMetrics(name)
	m.width = titleW + npcMenuWidthPad

	for _, r := range m.rows {
		l := m.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.SetText(m.RowLabel(r))

		if w, _ := l.GetTextMetrics(l.GetText()); w+npcMenuWidthPad > m.width {
			m.width = w + npcMenuWidthPad
		}

		m.labels = append(m.labels, l)
	}

	m.x, m.y = anchorX, anchorY
}

// Close hides the menu.
func (m *NPCMenu) Close() {
	m.open = false
}

// IsOpen returns whether the menu is visible.
func (m *NPCMenu) IsOpen() bool {
	return m.open
}

// SetAnchor moves the menu (called each frame so it follows the NPC).
func (m *NPCMenu) SetAnchor(x, y int) {
	m.x, m.y = x, y
}

// bounds returns the dialog box. The anchor is the NPC's screen position lifted by 0x96 (the caller does that); the
// dialog is centred on it and starts 0x15 above it (one non-selectable line; UNVERIFIED: the original takes
// height/3 when the dialog has no title line).
func (m *NPCMenu) bounds() (left, top, right, bottom int) {
	height := npcMenuTitleAdvance + npcMenuRowAdvance*len(m.rows) + npcMenuHeightPad
	left = m.x - m.width/2
	top = m.y - npcMenuTitleAdvance

	if left < npcMenuEdge {
		left = npcMenuEdge
	}

	if left+m.width > npcMenuScreenW-npcMenuEdge {
		left = npcMenuScreenW - npcMenuEdge - m.width
	}

	if top+height > npcMenuScreenH-npcMenuBottomEdge {
		top = npcMenuScreenH - height - 0x30
	}

	if top < npcMenuEdge {
		top = npcMenuEdge
	}

	return left, top, left + m.width, top + height
}

// rowBottom is the y of the bottom of the text line of menu row i (the running sum of the advances).
func (m *NPCMenu) rowBottom(i int) int {
	_, top, _, _ := m.bounds()

	return top + npcMenuTitleAdvance + npcMenuRowAdvance*(i+1)
}

func (m *NPCMenu) rowAt(mx, my int) int {
	left, _, right, _ := m.bounds()
	if mx <= left+npcMenuHitInsetX || mx >= right-npcMenuHitInsetX {
		return -1
	}

	for i := range m.rows {
		y := m.rowBottom(i)
		if my > y-npcMenuHitAbove && my < y+npcMenuHitBelow {
			return i
		}
	}

	return -1
}

// Contains reports whether the point is on the menu.
func (m *NPCMenu) Contains(mx, my int) bool {
	if !m.open {
		return false
	}

	left, top, right, bottom := m.bounds()

	return mx >= left && mx < right && my >= top && my < bottom
}

// Choose triggers a row by index (also used by the autotest).
func (m *NPCMenu) Choose(idx int) {
	if idx < 0 || idx >= len(m.rows) {
		return
	}

	row := m.rows[idx]
	if row.Action == NPCActionCancel {
		m.Close()
	}

	if m.onClick != nil {
		m.onClick(row)
	}
}

// OnMouseMove updates the highlighted row.
func (m *NPCMenu) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !m.open {
		return false
	}

	m.hover = m.rowAt(event.X(), event.Y())

	return m.Contains(event.X(), event.Y())
}

// OnMouseButtonDown handles clicks; returns true when the menu consumed it.
func (m *NPCMenu) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !m.open {
		return false
	}

	if !m.Contains(event.X(), event.Y()) {
		return false
	}

	if event.Button() == d2enum.MouseButtonLeft {
		m.Choose(m.rowAt(event.X(), event.Y()))
	}

	return true
}

// Render draws the menu.
func (m *NPCMenu) Render(target d2interface.Surface) {
	if !m.open {
		return
	}

	left, top, _, bottom := m.bounds()

	target.PushTranslation(left, top)
	target.DrawRect(m.width, bottom-top, npcMenuBackground)
	target.Pop()

	m.placeText()
	m.title.Render(target)

	for i, l := range m.labels {
		if i == m.hover {
			target.PushTranslation(left+npcMenuHitInsetX, m.rowBottom(i)-npcMenuHitAbove)
			target.DrawRect(m.width-2*npcMenuHitInsetX, npcMenuHitAbove+npcMenuHitBelow, npcMenuHighlight)
			target.Pop()
		}

		l.Render(target)
	}
}

// placeText puts the labels where the original draws them: centred in the box (x = dialog x + (width - text width +
// 1)/2 + 1, 0x4b4cd0), standing on the running sum of the row advances.
func (m *NPCMenu) placeText() {
	left, top, _, _ := m.bounds()

	put := func(l *d2ui.Label, bottom int) {
		w, h := l.GetTextMetrics(l.GetText())
		l.SetPosition(left+(m.width-w+1)/2+1, bottom-h)
	}

	put(m.title, top+npcMenuTitleAdvance)

	for i, l := range m.labels {
		put(l, m.rowBottom(i))
	}
}

// layoutRects returns the dialog box, the position of the name line and of each row for the layout log: text anchors
// are (centre x, bottom y) like the other panels'.
func (m *NPCMenu) layoutRects() []UIRect {
	left, top, right, bottom := m.bounds()
	m.placeText()

	// dialog_c: centre x, top, width minus the widest text, height (the checked form: the text widths vary with the language)
	widest := 0

	for _, l := range append([]*d2ui.Label{m.title}, m.labels...) {
		if w, _ := l.GetTextMetrics(l.GetText()); w > widest {
			widest = w
		}
	}

	rs := []UIRect{
		{"npcmenu", "dialog", left, top, right - left, bottom - top},
		{"npcmenu", "dialog_c", (left + right) / 2, top, right - left - widest, bottom - top},
		textAnchor("npcmenu", "text.title", m.title),
	}

	for i, l := range m.labels {
		rs = append(rs, textAnchor("npcmenu", fmt.Sprintf("text.row%d", i), l))
	}

	return rs
}
