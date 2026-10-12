package d2player

import (
	"errors"
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2itemdesc"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
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
	tradeGoldY       = 420
	tradeRepairX     = 195 // slot 0 of the strip of 0x721608: panel x 0x74 + 80 - 1
	tradeRepairY     = 445 // the strip stands on y 477 (UI_DrawTradePanel 0x484570), 32 high
	tradeRepairW     = 120
	tradeRepairH     = 24
	tradeCloseX      = 352 // slot 3: panel x 0x111 + 80 - 1
	tradeCloseY      = 445
	tradePanelRight  = 400
	tradePanelBottom = 492 // 60 + 256 + 176
)

// Errors of a transaction.
var (
	ErrNotEnoughGold = errors.New("not enough gold")
	ErrNoRoom        = errors.New("no room in the inventory")
	ErrNotSellable   = errors.New("the vendor does not buy this")
	ErrNoRepair      = errors.New("nothing to repair")
	ErrNotRepairer   = errors.New("this vendor does not repair")
	ErrCursorBusy    = errors.New("the cursor holds an item")
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

	// compareHook adds the lines that compare a vendor item with the worn one (before the price line).
	compareHook func(InventoryItem) []string

	panelGroup *d2ui.WidgetGroup
	grid       *ItemGrid
	tooltip    *d2ui.Tooltip
	title      *d2ui.Label
	goldLabel  *d2ui.Label
	msgDlg     *NpcMessageDialog // the vendor's reply message, a modal dialog (UI_ApplyNpcTransactionResultA 0x4b1c50)
	repairBtn  *d2ui.Button
	closeBtn   *d2ui.Button
	tabSprite  *d2ui.Sprite // buyselltabs.dc6: frames 0..3 selected, 4..7 other
	tabLabels  [vendorTabs]*d2ui.Label

	// tab is the selected vendor tab (storepage row, DAT_007b4c8c).
	tab int

	isOpen  bool
	vendor  d2vendor.Vendor
	stock   *d2vendor.Stock
	entries map[InventoryItem]*d2vendor.Item
	npc     d2trade.NPC

	// LevelOverride, when > 0, replaces the hero's level for stock generation
	// (autotest only: a level 94 hero only ever sees magic items, see
	// d2vendor.Generate).
	LevelOverride int

	// stashGold, when set, is the stash gold the vendor payments fall back on
	// (TRADE_DeductGold spends the inventory gold first, then the stash).
	stashGold *int

	difficulty int
	// classic is a game without the expansion: no Version >= 100 stock, no gamble upgrades (w43)
	classic bool
	mouseX  int
	mouseY  int

	onChange func()
	onClose  func()
	// OnImpossible plays the hero's "impossible" line (voice 0x13): ctrl-clicking an item the vendor does not buy
	// (INV_HandleGridClick 0x48c3c0, TRADE_CheckItemSellable false).
	OnImpossible func()
	// OnCantCarry plays the hero's "cantcarry" line (voice 0x17) when a purchase finds no room. The exe says it
	// for trade command 9 / 10 (UI_HandleTradeUiCommand 0x4b5640, VERIFIED); using it for a vendor purchase
	// without room is UNVERIFIED.
	OnCantCarry func()
	// autoBelt puts a bought potion on the belt when it has room (TRADE_ServerBuyItem tries ITEMACT_ServerItemToBelt
	// before the inventory); nil: inventory only.
	autoBelt func(InventoryItem) bool
	stocks   map[string]*d2vendor.Stock

	// gamble is true while the window shows the vendor's gamble stock.
	gamble bool
	// built records when each stock was generated, restocks counts the
	// regenerations (they change the seed); now is the clock (tests replace it).
	built    map[string]time.Time
	restocks map[string]uint32
	// due flags the stocks that the town-leaving check marked for a restock
	// (the original's per vendor flag +0x27), stamp is its timestamp (+0x28)
	// and keyVendor the vendor of each stock key.
	due       map[string]bool
	stamp     map[string]time.Time
	keyVendor map[string]d2vendor.Vendor
	now       func() time.Time
	logLevel  d2util.LogLevel

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
		due: make(map[string]bool), stamp: make(map[string]time.Time), keyVendor: make(map[string]d2vendor.Vendor),
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
	t.closeBtn = closeBtn
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

	t.msgDlg = NewNpcMessageDialog(t.asset, t.ui)
	t.msgDlg.Load()

	t.goldLabel = t.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	t.goldLabel.Alignment = d2ui.HorizontalAlignLeft
	t.goldLabel.SetPosition(left, tradeGoldY)
	t.panelGroup.AddWidget(t.goldLabel)

	if sp, err := t.ui.NewSprite(vendorTabsSprite, d2resource.PaletteSky); err == nil {
		t.tabSprite = sp
	}

	for i, key := range vendorTabKeys {
		l := t.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.SetText(t.text(key, vendorTabFallback[i]))
		t.tabLabels[i] = l
	}

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
	if stock != nil && t.due[key] {
		t.Infof("vendor restock: vendor=%s gamble=%v flagged when the hero left town", v.Name, gamble)

		stock = nil
		t.due[key] = false
		t.restocks[key]++

		if !gamble {
			// TRADE_OpenVendorSession 0x577240 frees the vendor's lists (0x5346d0), the gamble inventories included.
			gk := v.Name + gambleKeySuffix
			if t.stocks[gk] != nil {
				delete(t.stocks, gk)
				t.restocks[gk]++
			}
		}
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
			PlayerLevel: t.playerLevel(), Tier: d2vendor.ActIndex(v), Difficulty: t.difficulty, Resolve: d2vendor.Resolver(t.asset.Records), Classic: t.classic,
		})
		t.realise(stock, seed)
	}

	if t.stocks[key] != stock {
		t.stocks[key] = stock
		t.built[key] = t.now()
		t.stamp[key] = t.built[key]
		t.keyVendor[key] = v
	}

	t.stock = stock
	t.tab = 0

	if !gamble {
		t.tab = t.firstTab(initialVendorTab(v.ClassID))
	}

	t.rebuildGrid()
	t.title.SetText(v.Name)
	t.repairBtn.SetVisible(t.showsRepair())
	t.panelGroup.SetVisible(true)
	t.clearMessage()
	t.isOpen = true
	t.inv.SetPriceHook(t.inventoryPriceLines)
	t.syncGold()
	t.Infof("trade window opened: vendor=%s gamble=%v items=%d gold=%d tab=%d pages=%d/%d/%d/%d", v.Name, gamble,
		len(stock.Items), t.hero.Gold, t.tab, t.pageCount(0), t.pageCount(1), t.pageCount(2), t.pageCount(3))
}

// LeaveTown is the town-leaving restock check (SUNITPROXY_RefreshVendorsByTier 0x534c20, called from the town
// transition 0x534d40 with the act of the town being left, VERIFIED). othersInTown says whether another player
// still stands in that town: when nobody does (always, in a one player game) every vendor of the act that was
// opened before is flagged whatever the age of its stock; with players left behind only the vendors whose stamp
// is more than d2vendor.RestockInterval old are flagged and restamped (d2vendor.LeaveTownRestock). The flag is
// consumed when the vendor's regular window is next opened, which also drops its gamble stock (the per player
// gamble inventories are freed with the vendor's lists); a gamble window never consumes it. Entering a town, or
// staying in it, never restocks.
func (t *TradeWindow) LeaveTown(act int, othersInTown bool) {
	now := t.now()

	for key, v := range t.keyVendor {
		if d2vendor.ActIndex(v) != act || strings.HasSuffix(key, gambleKeySuffix) {
			continue
		}

		if d2vendor.LeaveTownRestock(!othersInTown, now.Sub(t.stamp[key]).Milliseconds()) {
			t.due[key] = true
			t.stamp[key] = now
		}
	}
}

// Close hides the window.
func (t *TradeWindow) Close() {
	if !t.isOpen {
		return
	}

	t.isOpen = false
	t.clearMessage()
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
	t.factory.Classic = t.classic

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
		if !ok || e.Page != t.visiblePage() {
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
	t.goldLabel.SetText(fmt.Sprintf("%s %s", t.text("strGold", "Gold:"), t.asset.Locale().FormatInt(t.hero.Gold)))
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

// vendorSells is TRADE_CheckVendorSellsItem for the open vendor: its
// permanent stock, and the tier 4 and 5 potions on the harder difficulties.
func (t *TradeWindow) vendorSells(code string) bool {
	var perm []string

	for _, b := range d2vendor.BasesFor(t.asset.Records, t.vendor) {
		if b.Permanent {
			perm = append(perm, b.Code)
		}
	}

	return d2vendor.VendorSells(t.difficulty, code, perm)
}

// SetStashGold gives the window the stash gold that pays what the inventory
// gold cannot (nil: inventory gold only).
func (t *TradeWindow) SetStashGold(p *int) { t.stashGold = p }

// stash is the stash gold the window can spend.
func (t *TradeWindow) stash() int {
	if t.stashGold == nil {
		return 0
	}

	return *t.stashGold
}

// canPay reports whether the inventory and stash gold together cover a price.
func (t *TradeWindow) canPay(price int) bool {
	_, _, ok := d2trade.DeductGold(t.hero.Gold, t.stash(), price)

	return price >= 0 && ok
}

// pay deducts a price: inventory gold first, then the stash (TRADE_DeductGold
// 0x574bc0, VERIFIED).
func (t *TradeWindow) pay(price int) error {
	if !t.canPay(price) {
		return ErrNotEnoughGold
	}

	inv, stash, _ := d2trade.DeductGold(t.hero.Gold, t.stash(), price)
	t.hero.Gold = inv

	if t.stashGold != nil {
		*t.stashGold = stash
	}

	return nil
}

// earn adds sale gold up to the carry cap of the hero's level; what does not
// fit is lost here (the original drops it as gold piles, 0x558e40).
func (t *TradeWindow) earn(amount int) (overflow int) {
	total, over := d2inventory.AddGold(t.hero.Gold, amount, d2inventory.InventoryGoldLimit(t.heroLevel()))
	t.hero.Gold = total

	return over
}

// heroLevel is the character level (not the stock level override).
func (t *TradeWindow) heroLevel() int {
	if t.hero != nil && t.hero.Stats != nil {
		return t.hero.Stats.Level
	}

	return 1
}

// Buy purchases a vendor item with a plain click (see BuyMulti).
func (t *TradeWindow) Buy(item *diablo2item.Item) (price int, err error) {
	return t.BuyMulti(item, false)
}

// BuyMulti is TRADE_ServerBuyItem 0x575640 (VERIFIED unless said otherwise). The cursor must be empty. Gold is the
// inventory plus the stash gold. An item the vendor always sells (permanent stock, and the tier 4 and 5 potions on
// Nightmare and Hell) is not taken from the stock: the player gets a copy. Anything else leaves the stock. A scroll
// goes into a tome of its kind that has room (one scroll, or with shift as many as the gold pays and the tome
// holds); a potion goes on the belt before the inventory. Shift (the packet's bit 31) is only sent for items.txt
// "multibuy" rows and only kept for items the vendor always sells: an auto stack item (arrows, bolts, throwing
// weapons) then tops up the player's partial stack or makes a full stack as far as the gold goes, a beltable potion
// is bought again and again until the belt, the inventory or the gold gives out. The price paid is the sum.
// Failures show the vendor's message (d2trade.ResultMessage).
func (t *TradeWindow) BuyMulti(item *diablo2item.Item, shift bool) (price int, err error) {
	t.clearMessage()

	e := t.entries[item]
	if e == nil {
		return 0, t.fail(ErrNotSellable, d2trade.ResultBusy)
	}

	if t.inv.CursorItem() != nil {
		return 0, t.fail(ErrCursorBusy, d2trade.ResultBusy)
	}

	unit := t.BuyPrice(item)

	if !t.canPay(unit) {
		return unit, t.fail(ErrNotEnoughGold, d2trade.ResultNoGold)
	}

	keep := !t.gamble && t.vendorSells(e.Code)
	multi := d2trade.MultiBuy(shift, item.CommonRecord().Multibuy, keep)

	var total int

	switch {
	case t.buyIntoTome(item, unit, multi, &total):
	case multi && t.isAutoStack(item) && t.buyStack(item, unit, &total):
	default:
		total, err = t.buyCopies(item, e, unit, keep, multi && t.isBeltable(item))
		if err != nil {
			return unit, err
		}
	}

	if total == 0 {
		return unit, t.fail(ErrNotEnoughGold, d2trade.ResultNoGold)
	}

	if !keep {
		t.stock.Remove(e)
		t.grid.Remove(item)
		delete(t.entries, item)
		t.fixTab()
	}

	if t.gamble {
		t.reveal(item)
	}

	t.syncGold()
	t.changed()
	t.Infof("bought %q for %d, gold %d", item.Label(), total, t.hero.Gold)

	return total, nil
}

// buyCopies delivers the bought item (the stock item itself when it leaves the stock, else a copy) and repeats the
// purchase for a multi buy while the next one still fits the belt or inventory and the gold. It reports the gold paid.
func (t *TradeWindow) buyCopies(item *diablo2item.Item, e *d2vendor.Item, unit int, keep, repeat bool) (int, error) {
	total := 0

	for {
		piece := item

		if keep {
			c, err := t.copyOf(item, e)
			if err != nil {
				t.Errorf("vendor copy %q: %v", e.Code, err)

				if total == 0 {
					return 0, t.fail(ErrNoRoom, d2trade.ResultInvalid)
				}

				break
			}

			piece = c
		}

		if !t.canPay(unit) {
			if total == 0 {
				return 0, t.fail(ErrNotEnoughGold, d2trade.ResultNoGold)
			}

			break
		}

		if !t.deliver(piece) {
			if total == 0 {
				return 0, t.fail(ErrNoRoom, d2trade.ResultNoRoom)
			}

			break
		}

		if err := t.pay(unit); err != nil {
			return total, err
		}

		total += unit

		if !repeat {
			break
		}
	}

	return total, nil
}

// deliver puts a bought item on the belt (potions) or in the first free spot of the inventory (the original puts it
// on the cursor; the engine has no cursor item yet).
func (t *TradeWindow) deliver(it *diablo2item.Item) bool {
	if t.autoBelt != nil && t.autoBelt(it) {
		return true
	}

	return t.inv.grid.AutoPlace(it, true)
}

// copyOf makes the copy of a permanent stock item that the player receives (ITEM_CloneItemViaSavedBits): the same
// base, quality, level, seed and stack size.
func (t *TradeWindow) copyOf(item *diablo2item.Item, e *d2vendor.Item) (*diablo2item.Item, error) {
	t.factory.Difficulty = t.difficulty
	t.factory.Classic = t.classic

	c, err := t.factory.ItemFromCodeForVendor(e.Code, e.Quality, e.ILvl, uint32(item.Seed), t.difficulty)
	if err != nil {
		return nil, err
	}

	c.Identify()

	if q := item.Quantity(); q > 0 {
		c.SetQuantity(q)
	}

	return c, nil
}

func (t *TradeWindow) itemType(item *diablo2item.Item) *d2records.ItemTypeRecord {
	return t.asset.Records.Item.Types[item.CommonRecord().Type]
}

func (t *TradeWindow) isAutoStack(item *diablo2item.Item) bool {
	ty := t.itemType(item)

	return item.IsStackable() && ty != nil && ty.AutoStack
}

func (t *TradeWindow) isBeltable(item *diablo2item.Item) bool {
	ty := t.itemType(item)

	return ty != nil && ty.Beltable && item.CommonRecord().AutoBelt
}

// tomeFor is the player's tome that takes a bought scroll: the same kind (d2player tomeOf) with room left
// (TRADE_FindStackableTomeInInventory).
func (t *TradeWindow) tomeFor(item *diablo2item.Item) *diablo2item.Item {
	code, ok := tomeOf[item.GetItemCode()]
	if !ok {
		return nil
	}

	for _, it := range t.inv.grid.items {
		tome, isItem := it.(*diablo2item.Item)
		if !isItem || tome.GetItemCode() != code || d2trade.TomeRoom(tome.Quantity(), tome.StackLimit()) == 0 {
			continue
		}

		// ITEM_HaveSameAffixes 0x55d400: the exe pairs by affixes, tomeOf is the code stand-in for the suffix
		if d2trade.TomeMatches(itemAffixes(item), itemAffixes(tome)) {
			return tome
		}
	}

	return nil
}

// itemAffixes is the affix identity of an item for d2trade.TomeMatches.
func itemAffixes(it *diablo2item.Item) d2trade.AffixSet {
	return d2trade.AffixesOf(it.PrefixCodes, it.SuffixCodes, "")
}

// buyIntoTome adds the scrolls to a tome of the inventory; false when the item is no scroll or no tome has room.
func (t *TradeWindow) buyIntoTome(item *diablo2item.Item, unit int, multi bool, paid *int) bool {
	tome := t.tomeFor(item)
	if tome == nil {
		return false
	}

	qty := 1

	if multi {
		room := tome.StackLimit() - tome.Quantity()
		if q, _, ok := d2trade.StackBuy(unit, t.hero.Gold+t.stash(), room, true); ok {
			qty = q
		}
	}

	cost := qty * unit
	if t.pay(cost) != nil {
		return false
	}

	tome.SetQuantity(tome.Quantity() + qty)
	*paid = cost

	return true
}

// buyStack is the shift buy of an auto stack item: it tops up the player's partial stack of the same kind, or makes
// a new stack of up to the maximum, as far as the gold goes (the unit price is the price of one piece).
func (t *TradeWindow) buyStack(item *diablo2item.Item, price int, paid *int) bool {
	qty := item.Quantity()
	if qty < 1 {
		qty = 1
	}

	unit := price / qty
	if unit < 1 {
		unit = 1
	}

	limit := item.StackLimit()
	room, topUp := limit, false

	var target *diablo2item.Item

	for _, it := range t.inv.grid.items {
		if o, ok := it.(*diablo2item.Item); ok && o != item && o.GetItemCode() == item.GetItemCode() &&
			o.Quality() == item.Quality() && o.Quantity() < o.StackLimit() {
			target, room, topUp = o, o.StackLimit()-o.Quantity(), true

			break
		}
	}

	n, cost, ok := d2trade.StackBuy(unit, t.hero.Gold+t.stash(), room, topUp)
	if !ok || !t.canPay(cost) {
		return false
	}

	if target != nil {
		if t.pay(cost) != nil {
			return false
		}

		target.SetQuantity(target.Quantity() + n)
		*paid = cost

		return true
	}

	e := t.entries[item]

	piece, err := t.copyOf(item, e)
	if err != nil {
		return false
	}

	piece.SetQuantity(n)

	if !t.deliver(piece) || t.pay(cost) != nil {
		return false
	}

	*paid = cost

	return true
}

// fail shows the vendor's message for a reply code and returns err.
func (t *TradeWindow) fail(err error, code int) error {
	key, text := d2trade.ResultMessage(code)
	if key != "" {
		msg := t.text(key, text)
		t.Infof("vendor message: code=%#x %q (%v)", code, msg, err)

		t.msgDlg.Open(msg)
	}

	return err
}

// ShowReply shows the modal message of a trade reply code (scripted scenarios).
func (t *TradeWindow) ShowReply(code int) {
	_ = t.fail(nil, code)
	t.Infof("vendor dialog open=%v text=%q", t.msgDlg.IsOpen(), t.msgDlg.Text())
}

func (t *TradeWindow) clearMessage() {
	t.msgDlg.Close()
}

// Sell sells one of the player's inventory items (TRADE_ServerSellItem
// 0x577320). Quest items are refused. The item joins the vendor's stock only
// when d2trade.Resellable allows it and the vendor has room; it is then
// refilled (durability, thrown quantity) and the player gets the lower of the
// price before and after the refill. The gold is capped by the hero's carry
// limit.
func (t *TradeWindow) Sell(item *diablo2item.Item) (price int, err error) {
	if !d2trade.Sellable(item.SellInfo()) {
		return 0, ErrNotSellable
	}

	price = t.SellPrice(item)

	info := item.ResaleInfo()
	info.GambleMode = t.gamble
	info.VendorSells = t.vendorSells(item.CommonCode)

	t.inv.grid.Remove(item)

	if d2trade.Resellable(info) {
		e := &d2vendor.Item{Code: item.CommonCode, SoldByPlayer: true, Payload: item}
		e.W, e.H = item.InventoryGridSize()
		e.Page = d2vendor.StorePageOf(t.asset.Records, item.CommonCode)

		// the weapons page overflows to the magic page, an unpaged type is not kept (PlaceSold, 0x577320)
		if t.stock.PlaceSold(e) {
			if e.Page == t.visiblePage() {
				if err := t.grid.Set(e.X, e.Y, item); err == nil {
					t.entries[item] = e
				}
			}

			item.Refill()
			price = d2trade.ResalePaid(price, t.SellPrice(item))
		}
	}

	if over := t.earn(price); over > 0 {
		t.Infof("sale gold over the carry cap: %d lost", over)
	}

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
	if !t.showsRepair() {
		return 0, ErrNotRepairer
	}

	items := t.repairable()
	if len(items) == 0 {
		return 0, ErrNoRepair
	}

	t.clearMessage()

	for _, it := range items {
		total += t.RepairPrice(it)
	}

	if err := t.pay(total); err != nil {
		return total, t.fail(err, d2trade.ResultNoGold)
	}

	for _, it := range items {
		it.Refill()
	}

	t.syncGold()
	t.changed()
	t.Infof("repaired %d items for %d, gold %d", len(items), total, t.hero.Gold)

	return total, nil
}

// Repair repairs one item (TRADE_ServerRepairItem 0x575e80, single item
// path). When the full price is out of reach and the item is not broken, all
// the inventory gold buys as many durability points as it covers
// (d2trade.PartialRepair); the returned price is what was paid.
func (t *TradeWindow) Repair(item *diablo2item.Item) (paid int, err error) {
	if !t.vendor.Repairs {
		return 0, ErrNotRepairer
	}

	price := t.RepairPrice(item)
	if price <= 0 {
		return 0, ErrNoRepair
	}

	t.clearMessage()

	if t.canPay(price) {
		if err := t.pay(price); err != nil {
			return price, err
		}

		item.Refill()
		t.syncGold()
		t.changed()
		t.Infof("repaired %q for %d, gold %d", item.Label(), price, t.hero.Gold)

		return price, nil
	}

	cur, maxDur := item.Durability()

	if !item.IsBroken() && maxDur > 0 {
		if dur, spent, ok := d2trade.PartialRepair(price, cur, maxDur, t.hero.Gold); ok {
			if err := t.pay(spent); err != nil {
				return price, err
			}

			item.SetDurability(dur)
			t.syncGold()
			t.changed()
			t.Infof("partly repaired %q to %d/%d for %d, gold %d", item.Label(), dur, maxDur, spent, t.hero.Gold)

			return spent, nil
		}
	}

	return price, t.fail(ErrNotEnoughGold, d2trade.ResultNoGold)
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

// showsRepair reports whether the Repair All button belongs to the open window: the gamble window of a vendor
// that repairs (Jamella, Halbu...) has no repair strip (w51; the exe shows the strip only for the regular shop).
func (t *TradeWindow) showsRepair() bool {
	return t.vendor.Repairs && !t.gamble
}

// OnMouseMove tracks the pointer and shows vendor item tooltips.
func (t *TradeWindow) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !t.isOpen {
		return false
	}

	t.mouseX, t.mouseY = event.X(), event.Y()
	t.refreshTooltip()

	return t.Contains(t.mouseX, t.mouseY)
}

// refreshTooltip shows the tooltip of the vendor item under the last pointer position and hides it when
// there is none. Also called after a click that changed the grid (a bought item must not keep its tooltip).
func (t *TradeWindow) refreshTooltip() {
	it := t.grid.ItemAtScreen(t.mouseX, t.mouseY)
	if item, ok := it.(*diablo2item.Item); ok {
		lines := item.GetItemDescription()
		if t.compareHook != nil {
			lines = append(append([]string{}, lines...), t.compareHook(item)...)
		}

		lines = append(lines, t.costLine("cost", "Cost: ", t.BuyPrice(item)))

		t.tooltip.SetTextLines(lines)
		t.tooltip.SetPosition(t.mouseX, t.mouseY-vendorCell/2)
		t.tooltip.SetVisible(true)
	} else {
		t.tooltip.SetVisible(false)
	}
}

// OnMouseButtonDown buys on a click on a vendor item and sells on a click on
// one of the player's inventory items. Returns true when it consumed the click.
func (t *TradeWindow) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	if !t.isOpen {
		return false
	}

	defer t.refreshTooltip()

	t.mouseX, t.mouseY = event.X(), event.Y()

	// the reply message is modal: the shop ignores the mouse until Continue is pressed
	if t.msgDlg.IsOpen() {
		return true
	}

	mx, my := event.X(), event.Y()

	if event.Button() == d2enum.MouseButtonLeft {
		if i := t.tabAt(mx, my); i >= 0 {
			t.selectTab(i)

			return true
		}

		if item, ok := t.grid.ItemAtScreen(mx, my).(*diablo2item.Item); ok {
			if _, err := t.BuyMulti(item, event.KeyMod() == d2enum.KeyModShift); err != nil {
				t.Infof("buy %q: %v", item.Label(), err)

				if errors.Is(err, ErrNoRoom) && t.OnCantCarry != nil {
					t.OnCantCarry()
				}
			}

			return true
		}

		// INV_HandleGridClick 0x48c3c0: with an empty cursor, ctrl held and a vendor open the
		// click on an inventory item sells it; without ctrl it picks the item up.
		if t.inv.IsOpen() && event.KeyMod() == d2enum.KeyModControl && t.inv.CursorItem() == nil {
			if item, ok := t.inv.grid.ItemAtScreen(mx, my).(*diablo2item.Item); ok {
				if _, err := t.Sell(item); err != nil {
					t.Infof("sell %q: %v", item.Label(), err)

					if errors.Is(err, ErrNotSellable) && t.OnImpossible != nil {
						t.OnImpossible()
					}
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
	t.renderTabs(target)
}

// The item tabs of a vendor window (UI_DrawNpcShopPanelAndTabs 0x484570, verified unless said otherwise): four tabs
// of buyselltabs.dc6 standing on y 91 (H + oy - 0x1c1) at x = 80 + 0x50*i, frame i for the selected tab and i+4 for
// the others, captioned with the strings 0xfc4..0xfc7 (Armor, Weapons, Magic, Misc: the static table repeats 0xfc5 for
// the third entry, which is UNVERIFIED and read as Magic); the caption stands at y 80 (UNVERIFIED anchor: bottom of
// the text). The stock is split into pages by the item type's StorePage (d2vendor.Item.Page); a tab without items is not drawn.
const (
	vendorTabs       = 4
	vendorTabsSprite = "/data/global/ui/panel/buyselltabs.dc6"
	vendorTabX       = 80
	vendorTabPitch   = 0x50
	vendorTabBottom  = 91
	vendorTabTextY   = 80
)

//nolint:gochecknoglobals // tables
var (
	vendorTabKeys     = [vendorTabs]string{"strBSArmor", "strBSWeapons", "strBSMagic", "strBSMisc"}
	vendorTabFallback = [vendorTabs]string{"Armor", "Weapons", "Magic", "Misc"}
)

// tabBox is the box of vendor tab i.
func (t *TradeWindow) tabBox(i int) (x, top, w, h int) {
	w, h = vendorTabPitch, 0

	if t.tabSprite != nil {
		if fw, fh, err := t.tabSprite.GetFrameSize(i); err == nil {
			w, h = fw, fh
		}
	}

	return vendorTabX + vendorTabPitch*i, vendorTabBottom - h, w, h
}

// pageCount is the number of stock items on a tab.
func (t *TradeWindow) pageCount(page int) int {
	n := 0

	if t.stock == nil {
		return 0
	}

	for _, e := range t.stock.Items {
		if e.Page == page {
			n++
		}
	}

	return n
}

// visiblePage is the page the grid shows: the gamble stock has a single page.
func (t *TradeWindow) visiblePage() int {
	if t.gamble {
		return 0
	}

	return t.tab
}

// initialVendorTab is the tab a vendor window opens on (UI_OpenNpcTradePanelInit 0x48dd70, VERIFIED switch on the
// NPC class): Charsi (0x9a) and Larzuk (0x1ff) open on Weapons, the classes listed as Misc open there (empty tabs
// fall through to the next, see firstTab), any other class starts on the first tab.
func initialVendorTab(class int) int {
	switch class {
	case 0x9a, 0x1ff:
		return 1
	case 0x94, 0xb1, 0xb2, 0xca, 0xfc, 0xfd, 0x101, 0x195, 0x201, 0xff:
		return 3
	}

	return 0
}

// firstTab is the tab to show when tab want is asked for: itself when it holds items, else the next one that does,
// wrapping after the last (UI_BuildNpcTabItemCounts 0x483b60, VERIFIED loop of at most four steps).
func (t *TradeWindow) firstTab(want int) int {
	for i := 0; i < vendorTabs; i++ {
		p := (want + i) % vendorTabs
		if t.pageCount(p) > 0 {
			return p
		}
	}

	return 0
}

// fixTab leaves a tab that has just run empty (the grid is rebuilt by the caller when it changed).
func (t *TradeWindow) fixTab() {
	if !t.gamble && t.pageCount(t.tab) == 0 {
		t.tab = t.firstTab(t.tab + 1)
		t.rebuildGrid()
	}
}

// selectTab shows tab i if it holds items and is not shown yet (click handler 0x48e340 region, VERIFIED: an empty
// tab is neither drawn nor clickable).
func (t *TradeWindow) selectTab(i int) bool {
	if t.gamble || i < 0 || i >= vendorTabs || i == t.tab || t.pageCount(i) == 0 {
		return false
	}

	t.tab = i
	t.rebuildGrid()
	t.tooltip.SetVisible(false)
	t.Infof("vendor tab: vendor=%s tab=%d items=%d", t.vendor.Name, i, t.pageCount(i))

	return true
}

// tabAt returns the tab under a screen point, or -1.
func (t *TradeWindow) tabAt(mx, my int) int {
	if t.gamble || t.tabSprite == nil {
		return -1
	}

	for i := 0; i < vendorTabs; i++ {
		x, top, w, h := t.tabBox(i)
		if t.pageCount(i) > 0 && mx >= x && mx < x+w && my >= top && my < top+h {
			return i
		}
	}

	return -1
}

func (t *TradeWindow) renderTabs(target d2interface.Surface) {
	if t.tabSprite == nil || t.gamble {
		return
	}

	for i := 0; i < vendorTabs; i++ {
		if t.pageCount(i) == 0 {
			continue // empty tabs are not drawn (flag +0x16 of the tab record)
		}

		frame := i + vendorTabs
		if i == t.tab {
			frame = i
		}

		x, top, w, h := t.tabBox(i)

		if t.tabSprite.SetCurrentFrame(frame) == nil {
			t.tabSprite.SetPosition(x, top+h)
			t.tabSprite.Render(target)
		}

		l := t.tabLabels[i]
		if i == t.tab {
			l.Color[0] = color.RGBA{R: 255, G: 215, B: 0, A: 255} // font colour 4 (gold) for the selected tab
		} else {
			l.Color[0] = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}

		tw, th := l.GetTextMetrics(l.GetText())
		l.SetPosition(x+(w-tw)/2, vendorTabTextY-th)
		l.Render(target)
	}
}

// SetClassic tells the shop that the game is a classic (non expansion) one.
func (t *TradeWindow) SetClassic(classic bool) { t.classic = classic }
