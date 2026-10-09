package d2player

import (
	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

const (
	npcMenuRowHeight = 24
	npcMenuPadding   = 10
	npcMenuMinWidth  = 130
	npcMenuTitleGap  = 4
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
	m.width = npcMenuMinWidth

	titleW, _ := m.title.GetTextMetrics(name)
	if titleW+2*npcMenuPadding > m.width {
		m.width = titleW + 2*npcMenuPadding
	}

	for _, r := range m.rows {
		l := m.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.SetText(m.RowLabel(r))

		if w, _ := l.GetTextMetrics(l.GetText()); w+2*npcMenuPadding > m.width {
			m.width = w + 2*npcMenuPadding
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

func (m *NPCMenu) bounds() (left, top, right, bottom int) {
	height := npcMenuRowHeight*(len(m.rows)+1) + npcMenuTitleGap
	left = m.x - m.width/2
	top = m.y - height

	// keep the menu on the 800x600 screen
	if left < 0 {
		left = 0
	} else if left+m.width > 800 {
		left = 800 - m.width
	}

	if top < 0 {
		top = 0
	}

	return left, top, left + m.width, top + height
}

func (m *NPCMenu) rowAt(mx, my int) int {
	left, top, right, bottom := m.bounds()
	if mx < left || mx >= right || my < top || my >= bottom {
		return -1
	}

	idx := (my - top - npcMenuRowHeight - npcMenuTitleGap) / npcMenuRowHeight
	if my-top < npcMenuRowHeight+npcMenuTitleGap || idx >= len(m.rows) {
		return -1
	}

	return idx
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

	left, top, _, _ := m.bounds()

	target.PushTranslation(left, top)
	target.DrawRect(m.width, npcMenuRowHeight*(len(m.rows)+1)+npcMenuTitleGap, npcMenuBackground)
	target.Pop()

	tw, _ := m.title.GetTextMetrics(m.title.GetText())
	m.title.SetPosition(left+(m.width-tw)/2, top+npcMenuRowHeight-2)
	m.title.Render(target)

	for i, l := range m.labels {
		rowTop := top + npcMenuRowHeight + npcMenuTitleGap + i*npcMenuRowHeight

		if i == m.hover {
			target.PushTranslation(left, rowTop)
			target.DrawRect(m.width, npcMenuRowHeight, npcMenuHighlight)
			target.Pop()
		}

		w, _ := l.GetTextMetrics(l.GetText())
		l.SetPosition(left+(m.width-w)/2, rowTop+npcMenuRowHeight-4)
		l.Render(target)
	}
}
