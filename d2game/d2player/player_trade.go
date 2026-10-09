package d2player

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2playertrade"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The player-to-player trade window. The rules and the item transfer are in
// package d2playertrade (and run on the server); this is only what a player
// sees and clicks. The original draws the two offers as two item grids, the
// rows "Trade Page 1" and "Trade Page 2" of Inventory.txt (10 x 4 cells of 29
// pixels, stacked in the left panel; verified in the notes: inventory-trade.md).
// Which page is whose is UNVERIFIED: here the partner's offer is the upper page
// and the player's own the lower one. An inventory item is put into (or taken
// out of) the offer with a right click, a click on an item of the player's own
// page takes it back, the gold line adds 100 on a left click and takes 100
// back on a right click, "Accept" accepts the unchanged offers, "Cancel" (or
// Escape) ends the trade.

const (
	ptradeCell     = 29
	ptradeGoldSt   = 100
	ptradeRecTheir = "Trade Page 1-2" // 800x600 rows: panel rect 80,60..401,502
	ptradeRecYour  = "Trade Page 2-2"
	ptradePanelL   = 80
	ptradePanelT   = 60
	ptradePanelR   = 401
	ptradePanelB   = 502
)

// PlayerTradeHandler is what the window asks of the game: send the offer, the
// acceptance or the cancellation to the server.
type PlayerTradeHandler interface {
	TradeOffer(offer d2playertrade.Offer)
	TradeAccept()
	TradeCancel()
}

// ptradeHit is a clickable text region of the window.
type ptradeHit struct {
	x, y, w, h int
	action     int // 1 gold, 2 accept, 3 cancel
}

type ptradeText struct {
	label *d2ui.Label
	x, y  int
}

// PlayerTradeWindow shows a trade with another player.
type PlayerTradeWindow struct {
	asset    *d2asset.AssetManager
	ui       *d2ui.UIManager
	gc       *GameControls
	logLevel d2util.LogLevel

	handler PlayerTradeHandler
	isOpen  bool
	partner string

	offer     d2playertrade.Offer // what this player offers now
	theirGold int
	youOK     bool
	theyOK    bool
	status    string

	group      *d2ui.WidgetGroup
	tooltip    *d2ui.Tooltip
	yourGrid   *ItemGrid
	theirGrid  *ItemGrid
	yourStored map[InventoryItem]d2hero.StoredItem
	yourNames  []string
	theirNames []string

	texts []ptradeText
	hits  []ptradeHit
	hover int

	*d2util.Logger
}

func newPlayerTradeWindow(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel, gc *GameControls) *PlayerTradeWindow {
	w := &PlayerTradeWindow{asset: asset, ui: ui, gc: gc, hover: -1, logLevel: l}
	w.Logger = d2util.NewLogger()
	w.Logger.SetLevel(l)
	w.Logger.SetPrefix(logPrefix)

	return w
}

// IsOpen reports whether the window is open.
func (w *PlayerTradeWindow) IsOpen() bool { return w.isOpen }

// Offer returns what this player currently offers.
func (w *PlayerTradeWindow) Offer() d2playertrade.Offer { return w.offer }

// gridFor builds an empty grid for a record of Inventory.txt (fallback: the
// numbers of the 800x600 rows).
func (w *PlayerTradeWindow) gridFor(record string, fallbackTop int) *ItemGrid {
	cols, rows, left, top, cell := 10, 4, 100, fallbackTop, ptradeCell

	if rec := w.asset.Records.Layout.Inventory[record]; rec != nil && rec.Grid != nil && rec.Grid.Columns > 0 {
		cols, rows = rec.Grid.Columns, rec.Grid.Rows
		left, top, cell = rec.Grid.Box.Left, rec.Grid.Box.Top, rec.Grid.CellWidth
	}

	return newPlainItemGrid(w.asset, w.ui, w.logLevel, cols, rows, left, top, cell)
}

// Open shows the window for a trade with the named player.
func (w *PlayerTradeWindow) Open(partner string) {
	w.isOpen, w.partner = true, partner
	w.offer = d2playertrade.Offer{}
	w.theirGold, w.youOK, w.theyOK = 0, false, false
	w.status = "Choose what to offer"

	if w.group == nil {
		w.group = w.ui.NewWidgetGroup(d2ui.RenderPriorityInventory)
		w.group.AddWidget(w.ui.NewUIFrame(d2ui.FrameLeft))
		w.tooltip = w.ui.NewTooltip(d2resource.FontFormal11, d2resource.PaletteStatic, d2ui.TooltipXCenter, d2ui.TooltipYBottom)
		w.group.AddWidget(w.tooltip)
	}

	w.group.SetVisible(true)
	w.tooltip.SetVisible(false)
	w.gc.inventory.Open()
	w.gc.updateLayout()
	w.fillGrids(nil, nil)
	w.rebuild()
}

// Close hides the window.
func (w *PlayerTradeWindow) Close() {
	if !w.isOpen {
		return
	}

	w.isOpen = false
	w.offer = d2playertrade.Offer{}
	w.texts, w.hits = nil, nil
	w.group.SetVisible(false)
	w.tooltip.SetVisible(false)
}

// Update shows the server's view of the trade.
func (w *PlayerTradeWindow) Update(yours, theirs d2playertrade.Offer, youOK, theyOK bool, status string) {
	w.offer = yours
	w.theirGold = theirs.Gold
	w.youOK, w.theyOK = youOK, theyOK
	w.fillGrids(yours.Items, theirs.Items)

	if status != "" {
		w.status = status
	}

	w.rebuild()
	w.Infof("TRADE WINDOW pages yours=%q theirs=%q gold_yours=%d gold_theirs=%d", w.yourNames, w.theirNames, w.offer.Gold, w.theirGold)
}

// fillGrids puts the offered items on the two pages.
func (w *PlayerTradeWindow) fillGrids(yours, theirs []d2hero.StoredItem) {
	w.yourGrid, w.theirGrid = w.gridFor(ptradeRecYour, 315), w.gridFor(ptradeRecTheir, 101)
	w.yourStored = make(map[InventoryItem]d2hero.StoredItem)
	w.yourNames, w.theirNames = nil, nil

	put := func(g *ItemGrid, list []d2hero.StoredItem, names *[]string, keep bool) {
		for i := range list {
			s := list[i]

			item, err := realiseStored(w.gc.inventory.item, &s)
			if err != nil {
				w.Warningf("trade: cannot show %q: %v", s.Code, err)
				continue
			}

			g.Load(item)

			if !g.AutoPlace(item, true) {
				w.Warningf("trade: %q does not fit the trade page", s.Code)
				continue
			}

			*names = append(*names, oneLine(itemName(item)))

			if keep {
				w.yourStored[item] = s
			}
		}
	}

	put(w.yourGrid, yours, &w.yourNames, true)
	put(w.theirGrid, theirs, &w.theirNames, false)
}

func (w *PlayerTradeWindow) mark(ok bool) string {
	if ok {
		return " (accepted)"
	}

	return ""
}

// Names lists the names of the offered items of both pages, for the autotests.
func (w *PlayerTradeWindow) Names() (yours, theirs []string) { return w.yourNames, w.theirNames }

func (w *PlayerTradeWindow) rebuild() {
	gold := color.RGBA{R: 255, G: 215, B: 0, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	green := color.RGBA{R: 24, G: 255, B: 0, A: 255}
	red := color.RGBA{R: 255, G: 60, B: 60, A: 255}

	w.texts, w.hits = nil, nil

	add := func(text string, x, y int, c color.Color, action int) {
		l := w.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.SetText(text)
		l.Color[0] = c // after SetText: it resets the colour (and reads [..] as colour tokens)
		w.texts = append(w.texts, ptradeText{label: l, x: x, y: y})

		if action != 0 {
			tw, _ := l.GetTextMetrics(text)
			w.hits = append(w.hits, ptradeHit{x: x - 4, y: y - 18, w: tw + 8, h: 22, action: action})
		}
	}

	tl, tt, _, tb := w.theirGrid.Bounds()
	yl, yt, _, yb := w.yourGrid.Bounds()

	add(fmt.Sprintf("Trade with %s", w.partner), tl, ptradePanelT+22, gold, 0)
	add(w.partner+" offers"+w.mark(w.theyOK), tl, tt-6, gold, 0)
	add(fmt.Sprintf("Gold: %d", w.theirGold), tl, tb+22, white, 0)
	add(w.status, tl, tb+48, white, 0)
	add("You offer"+w.mark(w.youOK), yl, yt-34, gold, 0)
	add(fmt.Sprintf("Gold: %d (click +%d, right click -%d)", w.offer.Gold, ptradeGoldSt, ptradeGoldSt), yl, yt-12, white, 1)
	add("> Accept <", yl, yb+26, green, 2)
	add("> Cancel <", yl+150, yb+26, red, 3)
}

func (w *PlayerTradeWindow) hitAt(mx, my int) int {
	for i, h := range w.hits {
		if mx >= h.x && mx < h.x+h.w && my >= h.y && my < h.y+h.h {
			return i
		}
	}

	return -1
}

func (w *PlayerTradeWindow) contains(mx, my int) bool {
	return w.isOpen && mx >= ptradePanelL && mx < ptradePanelR && my >= ptradePanelT && my < ptradePanelB
}

// OnMouseMove highlights the text under the pointer and shows item tooltips.
func (w *PlayerTradeWindow) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !w.isOpen {
		return false
	}

	mx, my := event.X(), event.Y()
	w.hover = w.hitAt(mx, my)

	it := w.yourGrid.ItemAtScreen(mx, my)
	if it == nil {
		it = w.theirGrid.ItemAtScreen(mx, my)
	}

	if it != nil {
		w.tooltip.SetTextLines(it.GetItemDescription())
		w.tooltip.SetPosition(mx, my-ptradeCell/2)
		w.tooltip.SetVisible(true)
	} else {
		w.tooltip.SetVisible(false)
	}

	return w.contains(mx, my)
}

// OnMouseButtonDown handles the gold, accept and cancel lines and the player's own page.
func (w *PlayerTradeWindow) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !w.isOpen {
		return false
	}

	mx, my := event.X(), event.Y()

	if it := w.yourGrid.ItemAtScreen(mx, my); it != nil {
		if s, ok := w.yourStored[it]; ok {
			w.Toggle(s)
		}

		return true
	}

	idx := w.hitAt(mx, my)
	if idx < 0 {
		return w.contains(mx, my)
	}

	switch w.hits[idx].action {
	case 1:
		delta := ptradeGoldSt
		if event.Button() == d2enum.MouseButtonRight {
			delta = -ptradeGoldSt
		}

		w.SetGold(w.offer.Gold + delta)
	case 2:
		if w.handler != nil {
			w.handler.TradeAccept()
		}
	case 3:
		w.Cancel()
	}

	return true
}

// Cancel asks the server to end the trade.
func (w *PlayerTradeWindow) Cancel() {
	if w.handler != nil {
		w.handler.TradeCancel()
	}
}

// SetGold changes the gold of the offer (within what the hero has) and sends it.
func (w *PlayerTradeWindow) SetGold(n int) {
	if n < 0 {
		n = 0
	}

	if n > w.gc.hero.Gold {
		n = w.gc.hero.Gold
	}

	w.offer.Gold = n
	w.send()
}

func (w *PlayerTradeWindow) send() {
	w.youOK = false
	w.fillGrids(w.offer.Items, nil)
	w.rebuild()

	if w.handler != nil {
		w.handler.TradeOffer(w.offer)
	}
}

// Toggle puts an inventory item into the offer, or takes it out again.
func (w *PlayerTradeWindow) Toggle(s d2hero.StoredItem) {
	for i, it := range w.offer.Items {
		if it.X == s.X && it.Y == s.Y && it.Code == s.Code {
			w.offer.Items = append(append([]d2hero.StoredItem{}, w.offer.Items[:i]...), w.offer.Items[i+1:]...)
			w.send()

			return
		}
	}

	w.offer.Items = append(append([]d2hero.StoredItem{}, w.offer.Items...), s)
	w.send()
}

// Render draws the window.
func (w *PlayerTradeWindow) Render(target d2interface.Surface) {
	if !w.isOpen || w.yourGrid == nil {
		return
	}

	target.PushTranslation(ptradePanelL, ptradePanelT)
	target.DrawRect(ptradePanelR-ptradePanelL, ptradePanelB-ptradePanelT, color.RGBA{R: 16, G: 14, B: 12, A: 255})
	target.Pop()

	for _, g := range []*ItemGrid{w.theirGrid, w.yourGrid} {
		left, top, right, bottom := g.Bounds()
		target.PushTranslation(left-2, top-2)
		target.DrawRect(right-left+4, bottom-top+4, color.RGBA{A: 200})
		target.Pop()
		g.Render(target)
	}

	for i, h := range w.hits {
		if i == w.hover {
			target.PushTranslation(h.x, h.y)
			target.DrawRect(h.w, h.h, npcMenuHighlight)
			target.Pop()
		}
	}

	for _, t := range w.texts {
		t.label.SetPosition(t.x, t.y)
		t.label.Render(target)
	}
}

// ---- GameControls side: the hero's items for the trade ----

// SetTradeHandler sets who sends the trade's offers, acceptance and cancellation.
func (g *GameControls) SetTradeHandler(h PlayerTradeHandler) {
	g.PTrade.handler = h
}

// InventoryStored lists the items of the inventory page as the hero file keeps
// them (positions included); these are what a trade offer names.
func (g *GameControls) InventoryStored() []d2hero.StoredItem {
	var out []d2hero.StoredItem

	for _, s := range g.snapshotContainers().Items {
		if s.Page == d2hero.PageInventory {
			out = append(out, s)
		}
	}

	return out
}

// TradeItemName is the name of a stored item as the inventory shows it.
func (g *GameControls) TradeItemName(s d2hero.StoredItem) string {
	it, err := realiseStored(g.inventory.item, &s)
	if err != nil {
		return s.Code
	}

	return oneLine(itemName(it))
}

// oneLine joins the lines of an item's name (magic items show their affix name
// and base type on two lines) for the window and the logs.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(s, "\n", " ")), " ")
}

// TradeItemNames names several stored items.
func (g *GameControls) TradeItemNames(items []d2hero.StoredItem) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = g.TradeItemName(s)
	}

	return out
}

// FindInventoryItem returns the first inventory item with the code that is not
// in the list already.
func (g *GameControls) FindInventoryItem(code string, except []d2hero.StoredItem) (d2hero.StoredItem, bool) {
	for _, s := range g.InventoryStored() {
		if !strings.EqualFold(s.Code, code) {
			continue
		}

		taken := false

		for _, e := range except {
			taken = taken || (e.X == s.X && e.Y == s.Y)
		}

		if !taken {
			return s, true
		}
	}

	return d2hero.StoredItem{}, false
}

// GiveItem creates an item and puts it into the inventory (a console aid for
// the autotests; the item is identified).
func (g *GameControls) GiveItem(code string) (string, error) {
	item, err := g.inventory.item.NewItem(code)
	if err != nil {
		return "", err
	}

	item.Identify()

	if !g.inventory.grid.AutoPlace(item, true) {
		return "", fmt.Errorf("no room in the inventory for %s", code)
	}

	g.saveHero()

	return oneLine(itemName(item)), nil
}

// ApplyTrade carries out a finished trade on the hero's panels: the items it
// gave leave the inventory, the items it got appear at the places the server
// chose, the gold is set to the server's total. It returns the names of the
// items that left and arrived.
func (g *GameControls) ApplyTrade(m *d2playertrade.Moved) (gave, got []string) {
	for _, s := range m.Gave {
		for _, it := range g.inventory.grid.items {
			x, y := it.InventoryGridSlot()
			if x == s.X && y == s.Y && it.GetItemCode() == s.Code {
				gave = append(gave, oneLine(itemName(it)))
				g.inventory.grid.Remove(it)

				break
			}
		}
	}

	if g.itemOrigin == nil {
		g.itemOrigin = make(map[InventoryItem]*d2s.Item)
	}

	for i := range m.Got {
		s := &m.Got[i]

		item, err := realiseStored(g.inventory.item, s)
		if err != nil {
			g.Warningf("trade: cannot build %q: %v", s.Code, err)
			continue
		}

		if s.D2S != nil {
			g.itemOrigin[item] = s.D2S
		}

		if err := g.inventory.grid.Set(s.X, s.Y, item); err != nil && !g.inventory.grid.AutoPlace(item, true) {
			g.Warningf("trade: %q does not fit at %d,%d: %v", s.Code, s.X, s.Y, err)
			continue
		}

		got = append(got, oneLine(itemName(item)))
	}

	g.AddGold(m.GoldAfter - g.hero.Gold)
	g.saveHero()

	return gave, got
}

// OpenPlayerTrade opens the trade window (the game screen calls it when both
// players agreed to trade).
func (g *GameControls) OpenPlayerTrade(partner string) {
	g.NPCMenu.Close()
	g.PTrade.Open(partner)
}

// ClosePlayerTrade closes the trade window and the inventory beside it.
func (g *GameControls) ClosePlayerTrade() {
	if !g.PTrade.IsOpen() {
		return
	}

	g.PTrade.Close()
	g.inventory.Close()
	g.updateLayout()
}

// tradeRightClick puts the clicked inventory item into the open trade offer.
func (g *GameControls) tradeRightClick(mx, my int) bool {
	if !g.PTrade.IsOpen() || !g.inventory.IsOpen() || !g.inventory.grid.Contains(mx, my) {
		return false
	}

	item := g.inventory.grid.ItemAtScreen(mx, my)
	if item == nil {
		return true
	}

	x, y := item.InventoryGridSlot()

	for _, s := range g.InventoryStored() {
		if s.X == x && s.Y == y {
			g.PTrade.Toggle(s)
			break
		}
	}

	return true
}
