package d2player

// Gold amount line and coin button of the original (UI_DrawGoldAmountAndTooltip 0x484250, called from
// UI_DrawInventoryAndTradePanels 0x48b150; read from the decompiler, not wired into the Go panels yet).
//
// The original has no separate "move gold" window in this function: the coin button (pictures "Panel\inv_goldbtn",
// "Panel\goldcoinbtn") and the amount text are what it draws. Where the original's drop/withdraw gold dialog is drawn
// was NOT located (the gold-click path of INV_HandleMouseDown 0x48d6c0 was not read): the Go MoveGoldPanel layout
// stays unverified.
//
// y values are screen pixels; the text y is the baseline argument of the text call, the button picture stands on
// its bottom edge (button bottom - 2, plus 1 while pressed). The text x is a register argument (UNVERIFIED).

// GoldKind selects the variant (the function's argument).
type GoldKind int

// The three variants of the gold line.
const (
	GoldStash   GoldKind = iota // stat 15 (banked gold), stash panel
	GoldCarried                 // stat 14 (gold in the inventory), inventory panel
	GoldTrade                   // stat 15 with the string 0xcf3 label, third variant
)

// GoldLayout is the result for one variant.
type GoldLayout struct {
	TextY       int // baseline argument of the amount text
	ButtonX     int // left edge of the coin button (0 for GoldTrade, which draws no button)
	ButtonY     int // bottom edge of the coin button picture before the pressed shift
	TooltipY    int // y of the "gold" tooltip (string 0x101c), only for GoldStash
	HitW, HitH  int // UI_IsCursorInBox size of the stash button
	TooltipText int // string.tbl id of the tooltip
}

// Gold returns the layout of the gold line. otherPanelOpen is UI_GetOpenPanelMask() != 0 (stash variant only).
func (m Mode) Gold(kind GoldKind, otherPanelOpen bool) GoldLayout {
	ox, oy := m.PanelOffset()
	e4 := -oy

	switch kind {
	case GoldStash:
		g := GoldLayout{
			TextY: e4 - 0xf4 + m.H, ButtonY: e4 - 0xf2 + m.H - 2, TooltipY: e4 - 0x106 + m.H,
			ButtonX: ox + 0x49 + 2, HitW: 0x14, HitH: 0x12, TooltipText: 0x101c,
		}
		if otherPanelOpen {
			g.TextY, g.ButtonY, g.TooltipY = e4-0x1b8+m.H, e4-0x1b6+m.H-2, e4-0x1ca+m.H
		}

		return g
	case GoldCarried:
		return GoldLayout{TextY: e4 - 0x48 + m.H, ButtonY: e4 - 0x45 + m.H - 2, ButtonX: m.W - ox - 0xed + 1}
	default:
		return GoldLayout{TextY: e4 - 0x6a + m.H}
	}
}
