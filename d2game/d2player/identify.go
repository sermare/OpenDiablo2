package d2player

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2trade"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// Identification: Deckard Cain's "Identify Items" window and the Scroll /
// Tome of Identify.
//
// VERIFIED (notes, inventory-trade.md and ui-npc.md): Cain charges 100 gold
// per unidentified item (0 once quest bit (4,0) or (4,1) is set); the server
// (packet 0x34) identifies every unidentified item of the inventory page and
// of the equipment/cube-less "mode 0 / mode 3" lists, the stash is not
// touched. The menu row of the real game identifies everything at once for
// count*100; the per item rows of this window are an addition for
// the engine's click UI and charge the same 100 per item.
//
// UNVERIFIED: the scroll/tome use (packet 0x27 identifies one item, packet
// 0x29 merges scrolls into a tome) follows the common game behaviour, not a
// decompiled path. The quest bit is not wired: the quest record is not
// reachable from the game screen, so the fee is always charged.

const (
	identifyScrollCode = "isc"
	identifyTomeCode   = "ibk"

	identifyPanelLeft  = 96
	identifyPanelTop   = 100
	identifyPanelWidth = 330
	identifyRowHeight  = 22
	identifyPad        = 8
)

// Errors of the identify service.
var (
	ErrNothingToIdentify = errors.New("nothing to identify")
	ErrNotAScroll        = errors.New("not a scroll or tome of identify")
	ErrScrollEmpty       = errors.New("the tome has no scrolls left")
	ErrAlreadyIdentified = errors.New("the item is already identified")
)

// IdentifyCostFor is the fee for n items (the quest bit is not wired, see above).
func IdentifyCostFor(n int, quest *d2s.QuestRecord) int {
	done := quest != nil && (quest.Get(4, 0) || quest.Get(4, 1))

	return d2trade.IdentifyCost(n, done)
}

// IdentifyWindow lists the hero's unidentified items with the fee and
// identifies on click.
type IdentifyWindow struct {
	asset *d2asset.AssetManager
	ui    *d2ui.UIManager
	inv   *Inventory
	hero  *d2mapentity.Player

	isOpen bool
	quest  *d2s.QuestRecord
	items  []*diablo2item.Item
	labels []*d2ui.Label
	title  *d2ui.Label
	hover  int

	// armed is the scroll or tome picked with a right click; the next left
	// click on an unidentified inventory item uses it.
	armed *diablo2item.Item

	onChange func()
	onClose  func()

	*d2util.Logger
}

// NewIdentifyWindow creates a closed identify window. onChange runs after
// every identification (the game screen saves the hero there).
func NewIdentifyWindow(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel,
	inv *Inventory, hero *d2mapentity.Player, onChange, onClose func()) *IdentifyWindow {
	w := &IdentifyWindow{asset: asset, ui: ui, inv: inv, hero: hero, onChange: onChange, onClose: onClose, hover: -1}
	w.Logger = d2util.NewLogger()
	w.Logger.SetLevel(l)
	w.Logger.SetPrefix(logPrefix)

	return w
}

// IsOpen reports whether the window is open.
func (w *IdentifyWindow) IsOpen() bool { return w.isOpen }

// Unidentified lists the hero's unidentified items: the inventory grid and the
// worn items (the stash is left alone, as in the original).
func (w *IdentifyWindow) Unidentified() []*diablo2item.Item {
	var out []*diablo2item.Item

	add := func(it InventoryItem) {
		if item, ok := it.(*diablo2item.Item); ok && !item.IsIdentified() {
			out = append(out, item)
		}
	}

	for _, it := range w.inv.grid.items {
		add(it)
	}

	for _, slot := range w.inv.grid.equipmentSlots {
		add(slot.item)
	}

	return out
}

// Open shows the window with the current unidentified items.
func (w *IdentifyWindow) Open(quest *d2s.QuestRecord) {
	w.quest = quest
	w.isOpen = true
	w.refresh()
	w.Infof("identify window opened: unidentified=%d fee_per_item=%d gold=%d",
		len(w.items), IdentifyCostFor(1, quest), w.hero.Gold)
}

// Close hides the window.
func (w *IdentifyWindow) Close() {
	if !w.isOpen {
		return
	}

	w.isOpen = false

	if w.onClose != nil {
		w.onClose()
	}
}

func (w *IdentifyWindow) refresh() {
	w.items = w.Unidentified()
	w.labels = w.labels[:0]

	w.title = w.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	w.title.Color[0] = color.RGBA{R: 255, G: 215, B: 0, A: 255}
	w.title.SetText(w.text("NPCIdentify1", "Identify Items"))

	all := w.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	all.SetText(fmt.Sprintf("%s (%d): %d", w.text("NPCIdentify1", "Identify Items"), len(w.items), IdentifyCostFor(len(w.items), w.quest)))
	w.labels = append(w.labels, all)

	for _, it := range w.items {
		l := w.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
		l.SetText(fmt.Sprintf("%s: %d", it.Label(), IdentifyCostFor(1, w.quest)))
		w.labels = append(w.labels, l)
	}
}

func (w *IdentifyWindow) text(key, fallback string) string {
	if s := w.asset.TranslateString(key); s != "" && s != key {
		return s
	}

	return fallback
}

func (w *IdentifyWindow) changed() {
	if w.onChange != nil {
		w.onChange()
	}
}

// IdentifyOne identifies a single item for the fee.
func (w *IdentifyWindow) IdentifyOne(item *diablo2item.Item) (cost int, err error) {
	if item.IsIdentified() {
		return 0, ErrAlreadyIdentified
	}

	cost = IdentifyCostFor(1, w.quest)

	gold, err := settle(w.hero.Gold, cost)
	if err != nil {
		return cost, err
	}

	item.Identify()
	w.hero.Gold = gold
	w.inv.SetGold(gold)
	w.changed()
	w.Infof("identified %q for %d, gold %d", item.Label(), cost, gold)

	if w.isOpen {
		w.refresh()
	}

	return cost, nil
}

// IdentifyAll identifies every unidentified item for count*fee, the way
// packet 0x34 does (nothing happens when the gold is short).
func (w *IdentifyWindow) IdentifyAll() (n, cost int, err error) {
	items := w.Unidentified()
	if len(items) == 0 {
		return 0, 0, ErrNothingToIdentify
	}

	cost = IdentifyCostFor(len(items), w.quest)

	gold, err := settle(w.hero.Gold, cost)
	if err != nil {
		return 0, cost, err
	}

	for _, it := range items {
		it.Identify()
	}

	w.hero.Gold = gold
	w.inv.SetGold(gold)
	w.changed()
	w.Infof("identified %d items for %d, gold %d", len(items), cost, gold)

	if w.isOpen {
		w.refresh()
	}

	return len(items), cost, nil
}

// IsIdentifyScroll reports whether the item is a scroll or tome of identify.
func IsIdentifyScroll(it InventoryItem) bool {
	code := it.GetItemCode()

	return code == identifyScrollCode || code == identifyTomeCode
}

// UseScroll identifies target with a scroll (consumed) or tome (one scroll
// used) of identify. The target stays unchanged on error.
func (w *IdentifyWindow) UseScroll(src, target *diablo2item.Item) error {
	if !IsIdentifyScroll(src) {
		return ErrNotAScroll
	}

	if target.IsIdentified() {
		return ErrAlreadyIdentified
	}

	if src.CommonCode == identifyTomeCode {
		if src.Quantity() < 1 {
			return ErrScrollEmpty
		}

		src.SetQuantity(src.Quantity() - 1)
	} else {
		w.inv.grid.Remove(src)
	}

	target.Identify()
	w.armed = nil
	w.changed()
	w.Infof("identify %s used on %q", src.CommonCode, target.Label())

	if w.isOpen {
		w.refresh()
	}

	return nil
}

// Arm picks a scroll or tome with the right click; the next left click on an
// unidentified inventory item uses it.
func (w *IdentifyWindow) Arm(src *diablo2item.Item) {
	w.armed = src
	w.Infof("identify %s armed", src.CommonCode)
}

func (w *IdentifyWindow) rowAt(mx, my int) int {
	if mx < identifyPanelLeft || mx >= identifyPanelLeft+identifyPanelWidth {
		return -1
	}

	idx := (my - identifyPanelTop - identifyRowHeight) / identifyRowHeight
	if my < identifyPanelTop+identifyRowHeight || idx < 0 || idx >= len(w.labels) {
		return -1
	}

	return idx
}

func (w *IdentifyWindow) contains(mx, my int) bool {
	return w.isOpen && mx >= identifyPanelLeft && mx < identifyPanelLeft+identifyPanelWidth &&
		my >= identifyPanelTop && my < identifyPanelTop+identifyRowHeight*(len(w.labels)+1)
}

// OnMouseMove highlights the row under the pointer.
func (w *IdentifyWindow) OnMouseMove(event d2interface.MouseMoveEvent) bool {
	if !w.isOpen {
		return false
	}

	w.hover = w.rowAt(event.X(), event.Y())

	return w.contains(event.X(), event.Y())
}

// OnMouseButtonDown identifies the clicked row (row 0 identifies all) or
// applies an armed scroll to the clicked inventory item. It returns true when
// it consumed the click.
func (w *IdentifyWindow) OnMouseButtonDown(event d2interface.MouseEvent) bool {
	mx, my := event.X(), event.Y()

	if w.armed != nil && event.Button() == d2enum.MouseButtonLeft && w.inv.IsOpen() && w.inv.grid.Contains(mx, my) {
		if target, ok := w.inv.grid.ItemAtScreen(mx, my).(*diablo2item.Item); ok {
			if err := w.UseScroll(w.armed, target); err != nil {
				w.Infof("identify scroll: %v", err)
			}
		}

		w.armed = nil

		return true
	}

	if !w.isOpen {
		return false
	}

	if event.Button() == d2enum.MouseButtonLeft {
		switch idx := w.rowAt(mx, my); {
		case idx == 0:
			if _, _, err := w.IdentifyAll(); err != nil {
				w.Infof("identify all: %v", err)
			}

			return true
		case idx > 0 && idx <= len(w.items):
			if _, err := w.IdentifyOne(w.items[idx-1]); err != nil {
				w.Infof("identify: %v", err)
			}

			return true
		}
	}

	return w.contains(mx, my)
}

// Render draws the list.
func (w *IdentifyWindow) Render(target d2interface.Surface) {
	if !w.isOpen {
		return
	}

	height := identifyRowHeight * (len(w.labels) + 1)

	target.PushTranslation(identifyPanelLeft-identifyPad, identifyPanelTop-identifyPad)
	target.DrawRect(identifyPanelWidth+2*identifyPad, height+2*identifyPad, color.RGBA{A: 200})
	target.Pop()

	w.title.SetPosition(identifyPanelLeft, identifyPanelTop+identifyRowHeight-4)
	w.title.Render(target)

	for i, l := range w.labels {
		top := identifyPanelTop + identifyRowHeight*(i+1)

		if i == w.hover {
			target.PushTranslation(identifyPanelLeft, top)
			target.DrawRect(identifyPanelWidth, identifyRowHeight, npcMenuHighlight)
			target.Pop()
		}

		l.SetPosition(identifyPanelLeft+4, top+identifyRowHeight-4)
		l.Render(target)
	}
}
