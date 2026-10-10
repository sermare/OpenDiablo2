package d2player

import "fmt"

// layoutRects returns the vendor window's grid (Inventory.txt "Monster2") and the buttons of its bottom strip
// (UI_DrawTradePanel 0x484570: four slots at screen x 195, 248, 300, 352, standing on y 477).
func (t *TradeWindow) layoutRects() []UIRect {
	l, top, r, b := t.grid.Bounds()

	rs := []UIRect{{"trade", "grid", l, top, r - l, b - top}}

	if t.tabSprite != nil && !t.gamble {
		for i := 0; i < vendorTabs; i++ {
			x, y, w, h := t.tabBox(i)
			rs = append(rs, UIRect{"trade", fmt.Sprintf("tab%d", i), x, y, w, h})
		}
	}

	if t.closeBtn != nil {
		rs = append(rs, widgetRect("trade", "close", t.closeBtn))
	}

	if t.repairBtn != nil && t.vendor.Repairs {
		x, y := t.repairBtn.GetPosition()
		rs = append(rs, UIRect{"trade", "repair_all", x, y, 0, 0})
	}

	return rs
}
