package d2player

import (
	"errors"
	"fmt"
	"image/color"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2itemdesc"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2vendor"
)

// The vendor grid is the "Monster" record of Inventory.txt in its 800x600
// form (VERIFIED in the notes: 10 x 10 cells of 29 px, 640x480 rect
// 16,305,63,352, shifted by +80,+60). It is used when the layout record is
// missing from the loaded tables.
const (
	vendorGridLeft = 96
	vendorGridTop  = 123
	vendorCell     = 29
	vendorRecord   = "Monster2"

	tradePanelPad    = 8
	tradeTitleY      = 100
	tradeGoldY       = 440
	tradeRepairX     = 100
	tradeRepairY     = 500
	tradeRepairW     = 120
	tradeRepairH     = 24
	tradeCloseX      = 360
	tradeCloseY      = 500
	tradePanelRight  = 400
	tradePanelBottom = 530
)

// Errors of a transaction.
var (
	ErrNotEnoughGold = errors.New("not enough gold")
	ErrNoRoom        = errors.New("no room in the inventory")
	ErrNotSellable   = errors.New("the vendor does not buy this")
	ErrNoRepair      = errors.New("nothing to repair")
	ErrNotRepairer   = errors.New("this vendor does not repair")
)

// settle subtracts a price from a gold amount.
func settle(gold, price int) (int, error) {
	if price < 0 || price > gold {
		return gold, ErrNotEnoughGold
	}

	return gold - price, nil
}

// StockLine describes one stock entry for logs and tests.
type StockLine struct {
	Name     string
	Code     string
	Quality  d2drop.Quality
	ILvl     int
	Quantity int
	Price    int
}

// TradeWindow is the vendor window: the vendor's grid on the left, the
// player's inventory panel on the right, prices from d2trade.
type TradeWindow struct {
	asset   *d2asset.AssetManager
	ui      *d2ui.UIManager
	inv     *Inventory
	hero    *d2mapentity.Player
	factory *diablo2item.ItemFactory

	panelGroup *d2ui.WidgetGroup
	grid       *ItemGrid
	tooltip    *d2ui.Tooltip
	title      *d2ui.Label
	goldLabel  *d2ui.Label
	repairBtn  *d2ui.Button

	isOpen  bool
	vendor  d2vendor.Vendor
	stock   *d2vendor.Stock
	entries map[InventoryItem]*d2vendor.Item
	npc     d2trade.NPC

	// LevelOverride, when > 0, replaces the hero's level for stock generation
	// (autotest only: a level 94 hero only ever sees magic items, see
	// d2vendor.Generate).
	LevelOverride int

	difficulty int
	mouseX     int
	mouseY     int

	onChange func()
	onClose  func()
	stocks   map[string]*d2vendor.Stock

	// gamble is true while the window shows the vendor's gamble stock.
	gamble bool
	// built records when each stock was generated, restocks counts the
	// regenerations (they change the seed); now is the clock (tests replace it).
	built    map[string]time.Time
	restocks map[string]uint32
	now      func() time.Time
	logLevel d2util.LogLevel

	*d2util.Logger
}

// NewTradeWindow creates a closed trade window. onChange runs after every
// transaction (the game screen saves the hero there), onClose when the window
// closes.
func NewTradeWindow(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel,
	inv *Inventory, hero *d2mapentity.Player, onChange, onClose func()) (*TradeWindow, error) {
	factory, err := diablo2item.NewItemFactory(asset)
	if err != nil {
		return nil, fmt.Errorf("creating the item factory: %w", err)
	}

	t := &TradeWindow{
		asset: asset, ui: ui, inv: inv, hero: hero, factory: factory,
		entries: make(map[InventoryItem]*d2vendor.Item), stocks: make(map[string]*d2vendor.Stock),
		built: make(map[string]time.Time), restocks: make(map[string]uint32), now: time.Now,
		onChange: onChange, onClose: onClose, logLevel: l,
	}

	t.Logger = d2util.NewLogger()
	t.Logger.SetLevel(l)
	t.Logger.SetPrefix(logPrefix)

	return t, nil
}

// Load creates the widgets.
func (t *TradeWindow) Load() {
	t.panelGroup = t.ui.NewWidgetGroup(d2ui.RenderPriorityInventory)
	t.panelGroup.AddWidget(t.ui.NewUIFrame(d2ui.FrameLeft))

	left, top, cell, cols, rows := vendorGridLeft, vendorGridTop, vendorCell, d2vendor.GridCols, d2vendor.GridRows
	if rec := t.asset.Records.Layout.Inventory[vendorRecord]; rec != nil && rec.Grid != nil {
		left, top, cell = rec.Grid.Box.Left, rec.Grid.Box.Top, rec.Grid.CellWidth
	}

	t.grid = newPlainItemGrid(t.asset, t.ui, t.logLevel, cols, rows, left, top, cell)

	closeBtn := t.ui.NewButton(d2ui.ButtonTypeSquareClose, "")
	closeBtn.SetVisible(false)
	closeBtn.SetPosition(tradeCloseX, tradeCloseY)
	closeBtn.OnActivated(func() { t.Close() })
	t.panelGroup.AddWidget(closeBtn)

	t.repairBtn = t.ui.NewButton(d2ui.ButtonTypeShort, t.text("NPCRepairItems", "Repair All"))
	t.repairBtn.SetVisible(false)
	t.repairBtn.SetPosition(tradeRepairX, tradeRepairY)
	t.repairBtn.OnActivated(func() {
		if _, err := t.RepairAll(); err != nil {
			t.Infof("repair: %v", err)
		}
	})
	t.panelGroup.AddWidget(t.repairBtn)

	t.title = t.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	t.title.SetPosition(left, tradeTitleY)
	t.title.Color[0] = color.RGBA{R: 255, G: 215, B: 0, A: 255}
	t.panelGroup.AddWidget(t.title)

	t.goldLabel = t.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	t.goldLabel.Alignment = d2ui.HorizontalAlignLeft
	t.goldLabel.SetPosition(left, tradeGoldY)
	t.panelGroup.AddWidget(t.goldLabel)

	t.tooltip = t.ui.NewTooltip(d2resource.FontFormal11, d2resource.PaletteStatic, d2ui.TooltipXCenter, d2ui.TooltipYBottom)

	t.panelGroup.SetVisible(false)
}

// text translates a string table key, falling back to English.
func (t *TradeWindow) text(key, fallback string) string {
	if s := t.asset.TranslateString(key); s != "" && s != key {
		return s
	}

	return fallback
}

// IsOpen reports whether the window is open.
func (t *TradeWindow) IsOpen() bool { return t.isOpen }

// Vendor returns the vendor of the open window.
func (t *TradeWindow) Vendor() d2vendor.Vendor { return t.vendor }

// Contains reports whether the point is on the vendor panel.
func (t *TradeWindow) Contains(x, y int) bool {
	return t.isOpen && x >= 0 && y >= 0 && x < tradePanelRight && y < tradePanelBottom
}

// Open shows the vendor window. The stock is generated the first time a
// vendor is opened in this game session and regenerated when the window is
// opened more than four minutes after the stock was built (VERIFIED in the
// binary: a per frame loop sets the vendor's "needs refresh" flag every
// 240000 ms and the next open consumes it, see d2vendor.RestockInterval; the
// original's timer is global per vendor record, this one starts at the
// generation, an approximation). seed seeds that generation.
func (t *TradeWindow) Open(v d2vendor.Vendor, seed uint32, quests *d2s.QuestRecord) {
	t.open(v, seed, quests, false)
}

func (t *TradeWindow) open(v d2vendor.Vendor, seed uint32, quests *d2s.QuestRecord, gamble bool) {
	t.vendor = v
	t.gamble = gamble
	t.npc = d2vendor.NPCPricing(t.asset.Records, v, quests)

	key := t.stockKey(v)

	stock := t.stocks[key]
	if stock != nil && d2vendor.RestockDue(t.now().Sub(t.built[key]).Milliseconds()) {
		t.Infof("vendor restock: vendor=%s gamble=%v after %dms", v.Name, gamble, d2vendor.RestockInterval)

		stock = nil
		t.restocks[key]++
	}

	seed += t.restocks[key]

	switch {
	case stock != nil:
	case gamble:
		stock = t.generateGamble(v, seed)
	default:
		bases := d2vendor.BasesFor(t.asset.Records, v)
		// Tier is the vendor's act (VERIFIED item level cap index, normal
		// difficulty only; Options ignores it when Difficulty != 0).
		stock = d2vendor.GenerateSeeded(seed, bases, d2vendor.Options{
			PlayerLevel: t.playerLevel(), Tier: d2vendor.ActIndex(v), Difficulty: t.difficulty, Resolve: d2vendor.Resolver(t.asset.Records),
		})
		t.realise(stock, seed)
	}

	if t.stocks[key] != stock {
		t.stocks[key] = stock
		t.built[key] = t.now()
	}

	t.stock = stock
	t.rebuildGrid()
	t.title.SetText(v.Name)
	t.repairBtn.SetVisible(v.Repairs)
	t.panelGroup.SetVisible(true)
	t.isOpen = true
	t.inv.SetPriceHook(t.inventoryPriceLines)
	t.syncGold()
	t.Infof("trade window opened: vendor=%s gamble=%v items=%d gold=%d", v.Name, gamble, len(stock.Items), t.hero.Gold)
}

// Close hides the window.
func (t *TradeWindow) Close() {
	if !t.isOpen {
		return
	}

	t.isOpen = false
	t.panelGroup.SetVisible(false)
	t.repairBtn.SetVisible(false)
	t.tooltip.SetVisible(false)
	t.inv.SetPriceHook(nil)

	if t.onClose != nil {
		t.onClose()
	}
}

func (t *TradeWindow) playerLevel() int {
	if t.LevelOverride > 0 {
		return t.LevelOverride
	}

	if t.hero != nil && t.hero.Stats != nil {
		return t.hero.Stats.Level
	}

	return 1
}

// realise creates the item objects of a freshly generated stock.
func (t *TradeWindow) realise(stock *d2vendor.Stock, seed uint32) {
	entries := append([]*d2vendor.Item{}, stock.Items...)
	t.factory.Difficulty = t.difficulty

	for idx, e := range entries {
		item, err := t.factory.ItemFromCodeForVendor(e.Code, e.Quality, e.ILvl, seed+uint32(idx)+1, t.difficulty)
		if err != nil {
			t.Errorf("vendor item %q: %v", e.Code, err)
			stock.Remove(e)

			continue
		}

		item.Identify()

		// UNVERIFIED for anything but ammo (a full stack is verified there):
		// other stackable items (throwing weapons, keys) are made full stacks
		// too. Tomes are skipped (their scroll count is not modelled).
		if rec := item.CommonRecord(); e.Quantity == 0 && rec.Stackable && rec.MaxStack > 0 && rec.Type != "book" {
			e.Quantity = rec.MaxStack
		}

		if e.Quantity > 0 {
			item.SetQuantity(e.Quantity)
		}

		e.Payload = item
	}
}

// rebuildGrid mirrors the stock onto the drawing grid.
func (t *TradeWindow) rebuildGrid() {
	t.grid.items = t.grid.items[:0]
	t.entries = make(map[InventoryItem]*d2vendor.Item)

	for _, e := range t.stock.Items {
		item, ok := e.Payload.(*diablo2item.Item)
		if !ok {
			continue
		}

		if err := t.grid.Set(e.X, e.Y, item); err != nil {
			t.Warningf("vendor grid: %v", err)
			continue
		}

		t.entries[item] = e
	}
}

func (t *TradeWindow) syncGold() {
	t.inv.SetGold(t.hero.Gold)
	t.goldLabel.SetText(fmt.Sprintf("%s %d", t.text("strGold", "Gold:"), t.hero.Gold))
}

func (t *TradeWindow) params(mode d2trade.Mode) d2trade.Params {
	return d2trade.Params{Mode: mode, Difficulty: t.difficulty, PlayerLevel: t.playerLevel(), NPC: t.npc}
}

// BuyPrice is what the player pays for a vendor item.
func (t *TradeWindow) BuyPrice(item *diablo2item.Item) int {
	if t.gamble {
		return t.gamblePrice(item)
	}

	return d2trade.ItemPrice(item.TradeItem(), t.params(d2trade.ModeBuy))
}

// SellPrice is what the vendor pays for one of the player's items.
func (t *TradeWindow) SellPrice(item *diablo2item.Item) int {
	return d2trade.ItemPrice(item.TradeItem(), t.params(d2trade.ModeSell))
}

// RepairPrice is the cost of repairing an item (0 if nothing to repair or the
// vendor does not repair).
func (t *TradeWindow) RepairPrice(item *diablo2item.Item) int {
	if !t.vendor.Repairs {
		return 0
	}

	return d2trade.ItemPrice(item.TradeItem(), t.params(d2trade.ModeRepair))
}

// Stock lists the vendor's items with their buy prices.
func (t *TradeWindow) Stock() []StockLine {
	out := make([]StockLine, 0, len(t.stock.Items))

	for _, e := range t.stock.Items {
		item, ok := e.Payload.(*diablo2item.Item)
		if !ok {
			continue
		}

		out = append(out, StockLine{
			Name: item.Label(), Code: e.Code, Quality: e.Quality, ILvl: e.ILvl,
			Quantity: item.Quantity(), Price: t.BuyPrice(item),
		})
	}

	return out
}

// Buy purchases a vendor item: the price is paid, the item leaves the stock
// and goes to the first free spot of the inventory (the original puts it on
// the cursor; the engine has no cursor item yet).
func (t *TradeWindow) Buy(item *diablo2item.Item) (price int, err error) {
	e := t.entries[item]
	if e == nil {
		return 0, ErrNotSellable
	}

	price = t.BuyPrice(item)

	gold, err := settle(t.hero.Gold, price)
	if err != nil {
		return price, err
	}

	if !t.inv.grid.CanAutoPlace(item, true) {
		return price, ErrNoRoom
	}

	t.stock.Remove(e)
	t.grid.Remove(item)
	delete(t.entries, item)

	t.inv.grid.AutoPlace(item, true)
	t.hero.Gold = gold

	if t.gamble {
		t.reveal(item)
	}

	t.syncGold()
	t.changed()
	t.Infof("bought %q for %d, gold %d", item.Label(), price, gold)

	return price, nil
}

// Sell sells one of the player's inventory items; it joins the vendor's
// stock if there is room (the original has more conditions, see
// inventory-trade.md: unverified here).
func (t *TradeWindow) Sell(item *diablo2item.Item) (price int, err error) {
	price = t.SellPrice(item)

	e := &d2vendor.Item{Code: item.CommonCode, SoldByPlayer: true, Payload: item}
	e.W, e.H = item.InventoryGridSize()

	t.inv.grid.Remove(item)

	if !t.gamble && t.stock.Place(e) {
		if err := t.grid.Set(e.X, e.Y, item); err == nil {
			t.entries[item] = e
		}
	}

	t.hero.Gold += price
	t.syncGold()
	t.changed()
	t.Infof("sold %q for %d, gold %d", item.Label(), price, t.hero.Gold)

	return price, nil
}

// repairable returns the player's damaged items.
func (t *TradeWindow) repairable() []*diablo2item.Item {
	out := make([]*diablo2item.Item, 0)

	collect := func(it InventoryItem) {
		if item, ok := it.(*diablo2item.Item); ok && t.RepairPrice(item) > 0 {
			out = append(out, item)
		}
	}

	for _, it := range t.inv.grid.items {
		collect(it)
	}

	for _, slot := range []d2enum.EquippedSlot{
		d2enum.EquippedSlotHead, d2enum.EquippedSlotTorso, d2enum.EquippedSlotLegs, d2enum.EquippedSlotRightArm,
		d2enum.EquippedSlotLeftArm, d2enum.EquippedSlotGloves, d2enum.EquippedSlotBelt,
	} {
		if it := t.inv.grid.equipmentSlots[slot].item; it != nil {
			collect(it)
		}
	}

	return out
}

// RepairAll repairs every damaged item of the player and returns the total
// price (the server sums the per item prices, VERIFIED in the notes).
func (t *TradeWindow) RepairAll() (total int, err error) {
	if !t.vendor.Repairs {
		return 0, ErrNotRepairer
	}

	items := t.repairable()
	if len(items) == 0 {
		return 0, ErrNoRepair
	}

	for _, it := range items {
		total += t.RepairPrice(it)
	}

	gold, err := settle(t.hero.Gold, total)
	if err != nil {
		return total, err
	}

	for _, it := range items {
		_, maxDur := it.Durability()
		it.SetDurability(maxDur)
	}

	t.hero.Gold = gold
	t.syncGold()
	t.changed()
	t.Infof("repaired %d items for %d, gold %d", len(items), total, gold)

	return total, nil
}

func (t *TradeWindow) changed() {
	if t.onChange != nil {
		t.onChange()
	}
}

func (t *TradeWindow) costLine(key, fallback string, price int) string {
	if tables := t.factory.DescriptionTables(); tables != nil {
		kind := map[string]d2itemdesc.PriceKind{"cost": d2itemdesc.PriceBuy, "Sell": d2itemdesc.PriceSell,
			"Repair": d2itemdesc.PriceRepair}[key]

		return diablo2item.Tokenize([]d2itemdesc.Line{tables.PriceLine(kind, price, t.hero.Gold)})[0]
	}

	token := d2ui.ColorTokenGold
	if price > t.hero.Gold && key == "cost" {
		token = d2ui.ColorTokenRed
	}

	return d2ui.ColorTokenize(fmt.Sprintf("%s%d", t.text(key, fallback), price), token)
}

// inventoryPriceLines are the extra tooltip lines for the player's own items
// while the window is open: "Sell value:" and, for damaged items, "Repair
// cost:" (the real labels, string table keys "Sell" and "Repair").
func (t *TradeWindow) inventoryPriceLines(it InventoryItem) []string {
	item, ok := it.(*diablo2item.Item)
	if !ok || !t.isOpen {
		return nil
	}

	lines := []string{t.costLine("Sell", "Sell value: ", t.SellPrice(item))}

	if r := t.RepairPrice(item); r > 0 {
		lines = append(lines, t.costLine("Repair", "Repair cost: ", r))
	}

	return lines
}

// OnMouseMove tracks the pointer and shows vendor item tooltips.
func (t *TradeWindow) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !t.isOpen {
		return false
	}

	t.mouseX, t.mouseY = event.X(), event.Y()

	it := t.grid.ItemAtScreen(t.mouseX, t.mouseY)
	if item, ok := it.(*diablo2item.Item); ok {
		lines := item.GetItemDescription()
		lines = append(lines, t.costLine("cost", "Cost: ", t.BuyPrice(item)))

		t.tooltip.SetTextLines(lines)
		t.tooltip.SetPosition(t.mouseX, t.mouseY-vendorCell/2)
		t.tooltip.SetVisible(true)
	} else {
		t.tooltip.SetVisible(false)
	}

	return t.Contains(t.mouseX, t.mouseY)
}

// OnMouseButtonDown buys on a click on a vendor item and sells on a click on
// one of the player's inventory items. Returns true when it consumed the click.
func (t *TradeWindow) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !t.isOpen {
		return false
	}

	mx, my := event.X(), event.Y()

	if event.Button() == d2enum.MouseButtonLeft {
		if item, ok := t.grid.ItemAtScreen(mx, my).(*diablo2item.Item); ok {
			if _, err := t.Buy(item); err != nil {
				t.Infof("buy %q: %v", item.Label(), err)
			}

			return true
		}

		if t.inv.IsOpen() {
			if item, ok := t.inv.grid.ItemAtScreen(mx, my).(*diablo2item.Item); ok {
				if _, err := t.Sell(item); err != nil {
					t.Infof("sell %q: %v", item.Label(), err)
				}

				return true
			}
		}
	}

	return t.Contains(mx, my)
}

// Render draws the vendor panel.
func (t *TradeWindow) Render(target d2interface.Surface) {
	if !t.isOpen {
		return
	}

	left, top, right, bottom := t.grid.Bounds()

	target.PushTranslation(left-tradePanelPad, top-tradePanelPad)
	target.DrawRect(right-left+2*tradePanelPad, bottom-top+2*tradePanelPad, color.RGBA{A: 200})
	target.Pop()

	t.grid.Render(target)
}
