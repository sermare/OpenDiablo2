package d2player

import (
	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// HireListEntry is one entry of the hire list: the two text lines of an offer and the slot it stands for.
type HireListEntry struct {
	Line1, Line2 string
	Row          NPCMenuRow
}

//nolint:gochecknoglobals // colours
var (
	hireListBack   = color.RGBA{R: 0, G: 0, B: 0, A: 225}
	hireListSelect = color.RGBA{R: 90, G: 20, B: 20, A: 220}
	hireListLine2  = color.RGBA{R: 170, G: 170, B: 170, A: 255}
	hireListTitle  = color.RGBA{R: 255, G: 215, B: 0, A: 255}
)

// HireList is the hire dialog of the executable (UI_OpenMercHireDialog 0x4b24c0): a box with the gold title, a list of six rows
// of two lines (name and numbers, then the HireDesc label) that scrolls, and a Cancel entry under it. A click on an entry hands its
// row to the callback, like the callback of the original's list box.
type HireList struct {
	ui      *d2ui.UIManager
	open    bool
	title   string
	cancel  string
	entries []HireListEntry
	first   int
	hover   int // entry index, -1 none, len(entries) = Cancel
	onClick func(NPCMenuRow)
	lab     map[string]*d2ui.Label
}

// NewHireList creates a closed hire list.
func NewHireList(ui *d2ui.UIManager) *HireList {
	return &HireList{ui: ui, hover: -1, lab: map[string]*d2ui.Label{}}
}

// Open shows the list. cancel is the text of the Cancel entry; onClick receives the row of a chosen entry (a NPCActionCancel row
// for Cancel).
func (h *HireList) Open(title, cancel string, entries []HireListEntry, onClick func(NPCMenuRow)) {
	h.title, h.cancel, h.entries, h.onClick = title, cancel, entries, onClick
	h.first, h.hover, h.open = 0, -1, true
}

// Close hides the list.
func (h *HireList) Close() {
	if h != nil {
		h.open = false
	}
}

// IsOpen reports whether the list is shown.
func (h *HireList) IsOpen() bool { return h != nil && h.open }

// Entries returns the entries shown (for logs and tests).
func (h *HireList) Entries() []HireListEntry { return h.entries }

// Choose triggers an entry by index (len(entries) is Cancel). The list closes first.
func (h *HireList) Choose(i int) {
	if !h.IsOpen() {
		return
	}

	var row NPCMenuRow

	switch {
	case i == len(h.entries):
		row = RowCancel
	case i >= 0 && i < len(h.entries):
		row = h.entries[i].Row
	default:
		return
	}

	h.Close()

	if h.onClick != nil {
		h.onClick(row)
	}
}

func (h *HireList) at(mx, my int) int {
	if i := HireListRowAt(npcMenuScreenW, npcMenuScreenH, h.first, len(h.entries), mx, my); i >= 0 {
		return i
	}

	dx, dy, dw, _ := HireListDialogRect(npcMenuScreenW, npcMenuScreenH)
	cy := dy + HireListCancelY

	if mx >= dx && mx < dx+dw && my >= cy-hireListRowH/2 && my < cy+hireListRowH/2 {
		return len(h.entries)
	}

	return -1
}

// Contains reports whether the point is on the dialog.
func (h *HireList) Contains(mx, my int) bool {
	if !h.IsOpen() {
		return false
	}

	dx, dy, dw, dh := HireListDialogRect(npcMenuScreenW, npcMenuScreenH)

	return mx >= dx && mx < dx+dw && my >= dy && my < dy+dh
}

// OnKeyDown moves the highlight with Up/Down, chooses with Enter or Space and cancels with Escape.
func (h *HireList) OnKeyDown(event d2interface.KeyEvent) bool {
	if !h.IsOpen() {
		return false
	}

	n := len(h.entries) + 1 // Cancel is the last entry

	switch event.Key() {
	case d2enum.KeyEscape:
		h.Choose(len(h.entries))
	case d2enum.KeyUp:
		h.hover = NextNPCMenuRow(h.hover, n, -1)
		h.scrollTo(h.hover)
	case d2enum.KeyDown:
		h.hover = NextNPCMenuRow(h.hover, n, 1)
		h.scrollTo(h.hover)
	case d2enum.KeyEnter, d2enum.KeySpace:
		if h.hover >= 0 {
			h.Choose(h.hover)
		}
	default:
		return false
	}

	return true
}

func (h *HireList) scrollTo(i int) {
	if i < 0 || i >= len(h.entries) {
		return
	}

	if i < h.first {
		h.first = i
	}

	if i >= h.first+HireListRows {
		h.first = i - HireListRows + 1
	}
}

// OnMouseMove highlights the entry under the pointer.
func (h *HireList) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !h.IsOpen() {
		return false
	}

	h.hover = h.at(event.X(), event.Y())

	return h.Contains(event.X(), event.Y())
}

// OnMouseButtonDown picks the entry under the pointer.
func (h *HireList) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !h.IsOpen() || !h.Contains(event.X(), event.Y()) {
		return false
	}

	if event.Button() == d2enum.MouseButtonLeft {
		h.Choose(h.at(event.X(), event.Y()))
	}

	return true
}

// Scroll scrolls the list by whole entries.
func (h *HireList) Scroll(delta int) {
	if h.IsOpen() {
		h.first = HireListClampFirst(h.first+delta, len(h.entries))
	}
}

func (h *HireList) label(key, text string, c color.RGBA) *d2ui.Label {
	l := h.lab[key]
	if l == nil {
		l = h.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		h.lab[key] = l
	}

	l.SetText(text)
	l.Color[0] = c

	return l
}

func (h *HireList) put(target d2interface.Surface, l *d2ui.Label, cx, bottom int) {
	w, hh := l.GetTextMetrics(l.GetText())
	l.SetPosition(cx-w/2, bottom-hh)
	l.Render(target)
}

// Render draws the dialog.
func (h *HireList) Render(target d2interface.Surface) {
	if !h.IsOpen() {
		return
	}

	dx, dy, dw, dh := HireListDialogRect(npcMenuScreenW, npcMenuScreenH)
	lx, ly, lw, _ := HireListRect(npcMenuScreenW, npcMenuScreenH)
	cx := dx + dw/2

	target.PushTranslation(dx, dy)
	target.DrawRect(dw, dh, hireListBack)
	target.Pop()

	h.put(target, h.label("title", h.title, hireListTitle), cx, dy+npcMenuTitleAdvance)

	for r := 0; r < HireListRows && h.first+r < len(h.entries); r++ {
		i := h.first + r
		top := ly + r*hireListRowH

		if i == h.hover {
			target.PushTranslation(lx, top)
			target.DrawRect(lw, hireListRowH, hireListSelect)
			target.Pop()
		}

		e := h.entries[i]
		h.put(target, h.label("a"+string(rune('0'+r)), e.Line1, color.RGBA{R: 255, G: 255, B: 255, A: 255}), cx, top+hireListRowH/2)

		if e.Line2 != "" {
			h.put(target, h.label("b"+string(rune('0'+r)), e.Line2, hireListLine2), cx, top+hireListRowH-4)
		}
	}

	cy := dy + HireListCancelY

	if h.hover == len(h.entries) {
		target.PushTranslation(dx+npcMenuHitInsetX, cy-hireListRowH/2)
		target.DrawRect(dw-2*npcMenuHitInsetX, hireListRowH, hireListSelect)
		target.Pop()
	}

	h.put(target, h.label("cancel", h.cancel, color.RGBA{R: 255, G: 255, B: 255, A: 255}), cx, cy+8)
}
