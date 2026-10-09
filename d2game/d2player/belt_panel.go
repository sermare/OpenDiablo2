package d2player

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The belt (inventory-trade.md, Inventory.cpp): 16 cells, 4 columns by up to 4
// rows, cell = row*4 + column, the equipped belt decides how many rows can be
// used. belts.txt gives the box positions; its second set of rows ("girdle2" ...)
// is the 800x600 control panel, the first set moved right by 80 and down by 120.
const (
	// fallback geometry (belts.txt "girdle2") when the record is missing
	beltBox1Left   = 423
	beltBox1Top    = 562
	beltStrideX    = 31
	beltStrideY    = 32
	beltBoxSize    = 29
	beltRecordName = "girdle2"
	beltPopFrame   = 0
)

// BeltPanel is the potion belt shown in the control panel.
type BeltPanel struct {
	asset  *d2asset.AssetManager
	ui     *d2ui.UIManager
	loader *ItemGrid // only loads and caches the item sprites
	cursor *Inventory

	items    [d2inventory.BeltCells]InventoryItem
	boxes    func() int // number of cells the equipped belt allows
	expanded bool

	left0, top0, strideX, strideY int

	background *d2ui.Sprite
	tooltip    *d2ui.Tooltip
	mouseX     int
	mouseY     int
	hoverCell  int

	onChange func()

	*d2util.Logger
}

// NewBeltPanel creates the belt. boxes reports the number of usable cells of
// the equipped belt; onChange runs after the content changed.
func NewBeltPanel(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel, cursor *Inventory,
	boxes func() int, onChange func()) *BeltPanel {
	b := &BeltPanel{
		asset: asset, ui: ui, cursor: cursor, boxes: boxes, onChange: onChange, hoverCell: -1,
		loader: newPlainItemGrid(asset, ui, l, d2inventory.BeltColumns, d2inventory.BeltCells/d2inventory.BeltColumns,
			0, 0, beltBoxSize),
		left0: beltBox1Left, top0: beltBox1Top, strideX: beltStrideX, strideY: beltStrideY,
	}

	if rec := asset.Records.Item.Belts[beltRecordName]; rec != nil && rec.NumBoxes >= d2inventory.BeltCells {
		b.left0, b.top0 = rec.Box1Left, rec.Box1Top

		if s := rec.Box2Left - rec.Box1Left; s > 0 {
			b.strideX = s
		}

		if s := rec.Box1Top - rec.Box5Top; s > 0 {
			b.strideY = s
		}
	}

	b.Logger = d2util.NewLogger()
	b.Logger.SetLevel(l)
	b.Logger.SetPrefix(logPrefix)

	return b
}

// Load creates the resources.
func (b *BeltPanel) Load() {
	var err error

	if b.background, err = b.ui.NewSprite(beltBackground, d2resource.PaletteSky); err != nil {
		b.Errorf("belt picture: %v", err)
	}

	b.tooltip = b.ui.NewTooltip(d2resource.FontFormal11, d2resource.PaletteStatic, d2ui.TooltipXCenter, d2ui.TooltipYBottom)
}

const beltBackground = "/data/global/ui/PANEL/ctrlpnl_popbelt.DC6"

// Boxes returns the number of usable cells.
func (b *BeltPanel) Boxes() int {
	n := d2inventory.BeltDefaultBoxes
	if b.boxes != nil {
		n = b.boxes()
	}

	if n > d2inventory.BeltCells {
		n = d2inventory.BeltCells
	}

	return n
}

// Rows returns the number of rows the belt has.
func (b *BeltPanel) Rows() int { return d2inventory.BeltRows(b.Boxes()) }

// Expanded reports whether all rows are shown (otherwise only the front row).
func (b *BeltPanel) Expanded() bool { return b.expanded }

// SetExpanded shows all rows or only the front row.
func (b *BeltPanel) SetExpanded(v bool) { b.expanded = v }

// Toggle flips between the front row and all rows.
func (b *BeltPanel) Toggle() { b.expanded = !b.expanded }

// Get returns the item in a cell or nil.
func (b *BeltPanel) Get(cell int) InventoryItem {
	if cell < 0 || cell >= len(b.items) {
		return nil
	}

	return b.items[cell]
}

// Set puts an item into a cell. It returns an error for a cell that is taken
// or outside the 16 cells. A cell beyond the equipped belt's size is accepted
// (a saved belt may hold more than the current belt shows) but not shown.
func (b *BeltPanel) Set(cell int, item InventoryItem) error {
	if cell < 0 || cell >= len(b.items) {
		return fmt.Errorf("belt cell %d out of range", cell)
	}

	if b.items[cell] != nil {
		return fmt.Errorf("belt cell %d is taken", cell)
	}

	b.items[cell] = item
	b.loader.Load(item)

	return nil
}

// kindOf is the potion kind of an item: its item type.
func (b *BeltPanel) kindOf(item InventoryItem) string {
	if it, ok := item.(*diablo2item.Item); ok {
		return it.CommonRecord().Type
	}

	return item.GetItemCode()
}

func (b *BeltPanel) kinds() *d2inventory.BeltKinds {
	var k d2inventory.BeltKinds

	for i, it := range b.items {
		if it != nil {
			k[i] = b.kindOf(it)
		}
	}

	return &k
}

// Beltable reports whether an item may be put on the belt (ItemTypes.txt Beltable).
func (b *BeltPanel) Beltable(item InventoryItem) bool {
	it, ok := item.(*diablo2item.Item)
	if !ok {
		return false
	}

	t := b.asset.Records.Item.Types[it.CommonRecord().Type]

	return t != nil && t.Beltable
}

// AutoBelt puts a picked-up item on the belt when its misc.txt row has
// autobelt (potions) and the belt has room, and reports the cell.
func (b *BeltPanel) AutoBelt(item InventoryItem) (cell int, ok bool) {
	it, isItem := item.(*diablo2item.Item)
	if !isItem || !it.CommonRecord().AutoBelt || !b.Beltable(item) {
		return 0, false
	}

	cell, ok = d2inventory.FindBeltSlot(b.kinds(), b.Boxes(), b.kindOf(item))
	if !ok {
		return 0, false
	}

	if err := b.Set(cell, item); err != nil {
		return 0, false
	}

	b.changed()

	return cell, true
}

// Use removes and returns the potion a hotkey (column 0..3) drinks: the front
// item of the column. The items behind it move forward.
func (b *BeltPanel) Use(col int) (InventoryItem, bool) {
	cell, ok := d2inventory.FrontOfColumn(b.kinds(), b.Boxes(), col)
	if !ok {
		return nil, false
	}

	item := b.items[cell]
	b.items[cell] = nil

	kinds := b.kinds()
	for _, mv := range d2inventory.CompactColumn(kinds, b.Boxes(), col) {
		b.items[mv[1]], b.items[mv[0]] = b.items[mv[0]], nil
	}

	return item, true
}

func (b *BeltPanel) changed() {
	if b.onChange != nil {
		b.onChange()
	}
}

// cellRect returns the screen rectangle of a cell.
func (b *BeltPanel) cellRect(cell int) (x, y, w, h int) {
	row, col := d2inventory.BeltCell(cell)

	return b.left0 + col*b.strideX, b.top0 - row*b.strideY, beltBoxSize, beltBoxSize
}

func (b *BeltPanel) visibleRows() int {
	if b.expanded {
		return b.Rows()
	}

	return 1
}

// cellAt returns the visible cell under a screen point, or -1.
func (b *BeltPanel) cellAt(mx, my int) int {
	for cell := 0; cell < b.visibleRows()*d2inventory.BeltColumns; cell++ {
		x, y, w, h := b.cellRect(cell)
		if cell < b.Boxes() && mx >= x && mx < x+w && my >= y && my < y+h {
			return cell
		}
	}

	return -1
}

// Contains reports whether the point is on a visible belt cell.
func (b *BeltPanel) Contains(mx, my int) bool { return b.cellAt(mx, my) >= 0 }

// HandleClick handles a left click on the belt: an empty cursor picks the
// potion up, a held beltable item is put into an empty cell or swapped with
// the potion in it. It reports whether the click was on the belt.
func (b *BeltPanel) HandleClick(mx, my int) bool {
	cell := b.cellAt(mx, my)
	if cell < 0 {
		return false
	}

	cur := b.cursor.CursorItem()

	switch {
	case cur == nil:
		if it := b.items[cell]; it != nil {
			b.items[cell] = nil
			b.cursor.SetCursorItem(it)
			b.Infof("picked up %s from belt cell %d", it.GetItemCode(), cell)
			b.changed()
		}
	case !b.Beltable(cur):
		b.Infof("%s does not fit on the belt", cur.GetItemCode())
	default:
		old := b.items[cell]
		b.items[cell] = cur
		b.loader.Load(cur)
		b.cursor.SetCursorItem(old)
		b.Infof("belt cell %d: %s", cell, cur.GetItemCode())
		b.changed()
	}

	return true
}

// OnMouseMove shows the tooltip of the potion under the pointer.
func (b *BeltPanel) OnMouseMove(mx, my int) {
	b.mouseX, b.mouseY = mx, my
	b.hoverCell = b.cellAt(mx, my)

	if b.tooltip == nil {
		return
	}

	if b.hoverCell >= 0 && b.items[b.hoverCell] != nil {
		b.tooltip.SetTextLines(b.items[b.hoverCell].GetItemDescription())
		b.tooltip.SetPosition(mx, my-beltBoxSize)
		b.tooltip.SetVisible(true)

		return
	}

	b.tooltip.SetVisible(false)
}

// Render draws the visible rows and their potions.
func (b *BeltPanel) Render(target d2interface.Surface) {
	for row := 0; row < b.visibleRows(); row++ {
		if row > 0 && b.background != nil {
			x, y, _, _ := b.cellRect(row * d2inventory.BeltColumns)
			_ = b.background.SetCurrentFrame(beltPopFrame)
			_, h := b.background.GetCurrentFrameSize()
			b.background.SetPosition(x-1, y-1+h)
			b.background.Render(target)
		}

		for col := 0; col < d2inventory.BeltColumns; col++ {
			cell := row*d2inventory.BeltColumns + col
			if cell >= b.Boxes() || b.items[cell] == nil {
				continue
			}

			x, y, _, _ := b.cellRect(cell)
			b.renderItem(target, b.items[cell], x, y)
		}
	}
}

func (b *BeltPanel) renderItem(target d2interface.Surface, item InventoryItem, x, y int) {
	sprite := b.loader.sprites[item.GetItemCode()]
	if sprite == nil {
		return
	}

	w, h := sprite.GetCurrentFrameSize()
	sprite.SetPosition(x+(beltBoxSize-w)/2, y+(beltBoxSize+h)/2)
	sprite.Render(target)
}

// Describe lists the belt content for the autotest log, in cell order.
func (b *BeltPanel) Describe() []string {
	var out []string

	for cell, it := range b.items {
		if it == nil {
			continue
		}

		row, col := d2inventory.BeltCell(cell)
		out = append(out, fmt.Sprintf("code=%s cell=%d row=%d col=%d name=%q", it.GetItemCode(), cell, row, col, itemName(it)))
	}

	return out
}

// Count returns the number of potions on the belt.
func (b *BeltPanel) Count() int {
	n := 0

	for _, it := range b.items {
		if it != nil {
			n++
		}
	}

	return n
}
