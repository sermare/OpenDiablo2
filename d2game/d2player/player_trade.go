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
// sees and clicks. UNVERIFIED design: the original draws two inventory-like
// grids ("Trade Page 1/2" of inventory.txt) that items are dragged into; this
// window lists both offers as text rows, an inventory item is put into (or
// taken out of) the offer with a right click, the gold row adds 100 on a left
// click and takes 100 back on a right click, "Accept" accepts the unchanged
// offers, "Cancel" (or Escape) ends the trade.

const (
	ptradeLeft   = 96
	ptradeTop    = 100
	ptradeWidth  = 360
	ptradeRowH   = 22
	ptradePad    = 8
	ptradeGoldSt = 100
)

// PlayerTradeHandler is what the window asks of the game: send the offer, the
// acceptance or the cancellation to the server.
type PlayerTradeHandler interface {
	TradeOffer(offer d2playertrade.Offer, seq uint32)
	TradeAccept()
	TradeCancel()
}

type ptradeRow struct {
	text   string
	action int // 0 none, 1 gold, 2 accept, 3 cancel
	color  color.Color
}

// PlayerTradeWindow shows a trade with another player.
type PlayerTradeWindow struct {
	asset *d2asset.AssetManager
	ui    *d2ui.UIManager
	gc    *GameControls

	handler PlayerTradeHandler
	isOpen  bool
	partner string

	pend       d2playertrade.Pending // what this player offers now, and the number of its latest edit
	yourNames  []string
	theirNames []string
	theirGold  int
	youOK      bool
	theyOK     bool
	status     string

	rows   []ptradeRow
	labels []*d2ui.Label
	hover  int

	*d2util.Logger
}

func newPlayerTradeWindow(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel, gc *GameControls) *PlayerTradeWindow {
	w := &PlayerTradeWindow{asset: asset, ui: ui, gc: gc, hover: -1}
	w.Logger = d2util.NewLogger()
	w.Logger.SetLevel(l)
	w.Logger.SetPrefix(logPrefix)

	return w
}

// IsOpen reports whether the window is open.
func (w *PlayerTradeWindow) IsOpen() bool { return w.isOpen }

// Offer returns what this player currently offers.
func (w *PlayerTradeWindow) Offer() d2playertrade.Offer { return w.pend.Offer }

// Open shows the window for a trade with the named player.
func (w *PlayerTradeWindow) Open(partner string) {
	w.isOpen, w.partner = true, partner
	w.pend.Reset()
	w.yourNames, w.theirNames, w.theirGold, w.youOK, w.theyOK = nil, nil, 0, false, false
	w.status = "Choose what to offer"
	w.gc.inventory.Open()
	w.gc.updateLayout()
	w.rebuild()
}

// Close hides the window.
func (w *PlayerTradeWindow) Close() {
	if !w.isOpen {
		return
	}

	w.isOpen = false
	w.pend.Reset()
	w.labels, w.rows = nil, nil
}

// Update shows the server's view of the trade.
// ack is the sequence number of our last offer the server had applied; an
// update older than our latest edit keeps the local offer (see
// d2playertrade.Pending) and does not show us as accepted. It reports whether
// the server's copy of our offer was taken.
func (w *PlayerTradeWindow) Update(yours, theirs d2playertrade.Offer, youOK, theyOK bool, status string, ack uint32) bool {
	taken := w.pend.Apply(ack, yours)
	w.yourNames = w.gc.TradeItemNames(w.pend.Offer.Items)
	w.theirNames, w.theirGold = w.gc.TradeItemNames(theirs.Items), theirs.Gold
	w.youOK, w.theyOK = youOK && taken, theyOK

	if status != "" {
		w.status = status
	}

	w.rebuild()

	return taken
}

func (w *PlayerTradeWindow) mark(ok bool) string {
	if ok {
		return " (accepted)"
	}

	return ""
}

func (w *PlayerTradeWindow) rebuild() {
	gold := color.RGBA{R: 255, G: 215, B: 0, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	rows := []ptradeRow{{text: fmt.Sprintf("Trade with %s", w.partner), color: gold}}
	rows = append(rows, ptradeRow{text: "You offer" + w.mark(w.youOK), color: gold})

	for _, n := range w.yourNames {
		rows = append(rows, ptradeRow{text: "  " + n, color: white})
	}

	rows = append(rows, ptradeRow{text: fmt.Sprintf("  Gold: %d (click +%d, right click -%d)", w.pend.Offer.Gold, ptradeGoldSt, ptradeGoldSt),
		action: 1, color: white})
	rows = append(rows, ptradeRow{text: w.partner + " offers" + w.mark(w.theyOK), color: gold})

	for _, n := range w.theirNames {
		rows = append(rows, ptradeRow{text: "  " + n, color: white})
	}

	rows = append(rows, ptradeRow{text: fmt.Sprintf("  Gold: %d", w.theirGold), color: white})
	rows = append(rows, ptradeRow{text: "> Accept <", action: 2, color: color.RGBA{R: 24, G: 255, B: 0, A: 255}})
	rows = append(rows, ptradeRow{text: "> Cancel <", action: 3, color: color.RGBA{R: 255, G: 60, B: 60, A: 255}})
	rows = append(rows, ptradeRow{text: w.status, color: white})

	w.rows = rows
	w.labels = w.labels[:0]

	for _, r := range rows {
		l := w.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.SetText(r.text)
		l.Color[0] = r.color // after SetText: it resets the colour (and reads [..] as colour tokens)
		w.labels = append(w.labels, l)
	}
}

func (w *PlayerTradeWindow) rowAt(mx, my int) int {
	if mx < ptradeLeft || mx >= ptradeLeft+ptradeWidth || my < ptradeTop {
		return -1
	}

	idx := (my - ptradeTop) / ptradeRowH
	if idx < 0 || idx >= len(w.rows) {
		return -1
	}

	return idx
}

func (w *PlayerTradeWindow) contains(mx, my int) bool {
	return w.isOpen && mx >= ptradeLeft-ptradePad && mx < ptradeLeft+ptradeWidth+ptradePad &&
		my >= ptradeTop-ptradePad && my < ptradeTop+ptradeRowH*len(w.rows)+ptradePad
}

// OnMouseMove highlights the row under the pointer.
func (w *PlayerTradeWindow) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !w.isOpen {
		return false
	}

	w.hover = w.rowAt(event.X(), event.Y())

	return w.contains(event.X(), event.Y())
}

// OnMouseButtonDown handles the gold, accept and cancel rows.
func (w *PlayerTradeWindow) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !w.isOpen {
		return false
	}

	idx := w.rowAt(event.X(), event.Y())
	if idx < 0 {
		return w.contains(event.X(), event.Y())
	}

	switch w.rows[idx].action {
	case 1:
		delta := ptradeGoldSt
		if event.Button() == d2enum.MouseButtonRight {
			delta = -ptradeGoldSt
		}

		w.SetGold(w.pend.Offer.Gold + delta)
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

	w.pend.SetGold(n)
	w.send()
}

func (w *PlayerTradeWindow) send() {
	w.youOK = false
	w.yourNames = w.gc.TradeItemNames(w.pend.Offer.Items)
	w.rebuild()

	if w.handler != nil {
		w.handler.TradeOffer(w.pend.Offer, w.pend.Seq)
	}
}

// Toggle puts an inventory item into the offer, or takes it out again.
func (w *PlayerTradeWindow) Toggle(s d2hero.StoredItem) {
	w.pend.Toggle(s)
	w.send()
}

// Render draws the window.
func (w *PlayerTradeWindow) Render(target d2interface.Surface) {
	if !w.isOpen {
		return
	}

	target.PushTranslation(ptradeLeft-ptradePad, ptradeTop-ptradePad)
	target.DrawRect(ptradeWidth+2*ptradePad, ptradeRowH*len(w.rows)+2*ptradePad, color.RGBA{A: 200})
	target.Pop()

	for i, l := range w.labels {
		top := ptradeTop + ptradeRowH*i

		if i == w.hover && w.rows[i].action != 0 {
			target.PushTranslation(ptradeLeft, top)
			target.DrawRect(ptradeWidth, ptradeRowH, npcMenuHighlight)
			target.Pop()
		}

		l.SetPosition(ptradeLeft+4, top+ptradeRowH-4)
		l.Render(target)
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

// DropInventoryItem removes the first inventory item with the given base code
// (the player dropping it; the scripted export scenario makes room this way)
// and saves the hero. It returns the item's name.
func (g *GameControls) DropInventoryItem(code string) (string, error) {
	for _, it := range g.inventory.grid.items {
		if it.GetItemCode() == code {
			name := oneLine(itemName(it))
			g.inventory.grid.Remove(it)
			g.saveHero()

			return name, nil
		}
	}

	return "", fmt.Errorf("no %s in the inventory", code)
}
