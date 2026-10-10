package d2player

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"
)

// The stash and the Horadric Cube are the same thing to the interface: a left
// panel with one item grid, whose layout is a row of Inventory.txt, whose items
// live on one item page of the hero (inventory-trade.md: the panel pages 3 =
// cube, 4 = stash of the client are .d2s pages 4 and 5). The grids behave like
// the inventory grid (ItemGrid.ClickWith) and exchange items through the
// cursor, which the inventory owns.

const (
	panelArtTop      = 64 // the 4 quadrant pictures start this far below the screen top, like invchar6
	panelCloseOffset = 208
	panelCloseY      = 453
	panelTitleY      = 84

	cubeItemCode = "box"
)

// containerKind describes one container panel.
type containerKind struct {
	name    string
	title   string // fallback English title
	page    int    // d2hero.Page*
	art     string // the quadrant picture (4 frames: top-left, top-right, bottom-left, bottom-right)
	records []string
	// fallback layout when no record is found: columns, rows and the pixel rect of the grid
	cols, rows, left, top int
	acceptsCube           bool
}

var (
	stashKind = containerKind{
		name: "stash", title: "Stash", page: d2hero.PageStash,
		art: "/data/global/ui/PANEL/bank.DC6",
		// the expansion stash (Big Bank, 6x8) is used when the game data has it; the classic one is 6x4
		records: []string{"Big Bank Page2", "Bank Page2"},
		cols:    6, rows: 8, left: 154, top: 142,
		acceptsCube: true,
	}
	cubeKind = containerKind{
		name: "cube", title: "Horadric Cube", page: d2hero.PageCube,
		art:     "/data/global/ui/PANEL/supertransmogrifier.dc6",
		records: []string{"Transmogrify Box2"},
		cols:    3, rows: 4, left: 198, top: 199,
		acceptsCube: false, // the cube cannot hold itself (INV_GetSingleOverlappedItem rejects "box ")
	}
)

// ContainerPanel is the stash or the cube panel.
type ContainerPanel struct {
	asset *d2asset.AssetManager
	ui    *d2ui.UIManager
	kind  containerKind

	panelGroup *d2ui.WidgetGroup
	art        *d2ui.Sprite
	title      *d2ui.Label
	tooltip    *d2ui.Tooltip
	grid       *ItemGrid
	originX    int

	isOpen         bool
	mouseX, mouseY int

	// cursor is the inventory that owns the cursor item.
	cursor *Inventory
	// onChange runs after the content changed (the game saves the hero).
	onChange func()
	onClose  func()

	// transmute is the cube's Transmute button (nil on the stash); onTransmute
	// runs when it is pressed.
	transmute   *d2ui.Button
	onTransmute func()
	status      *d2ui.Label
	statusText  string

	*d2util.Logger
}

// NewContainerPanel creates a closed panel of a kind ("stash" or "cube").
func NewContainerPanel(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel, cursor *Inventory,
	kind containerKind, onChange func()) *ContainerPanel {
	p := &ContainerPanel{asset: asset, ui: ui, kind: kind, cursor: cursor, onChange: onChange}

	cols, rows, left, top, cell, panelLeft := kind.cols, kind.rows, kind.left, kind.top, vendorCell, 80

	for _, name := range kind.records {
		if rec := asset.Records.Layout.Inventory[name]; rec != nil && rec.Grid != nil && rec.Grid.Columns > 0 {
			cols, rows = rec.Grid.Columns, rec.Grid.Rows
			left, top, cell = rec.Grid.Box.Left, rec.Grid.Box.Top, rec.Grid.CellWidth
			panelLeft = rec.Panel.Left

			break
		}
	}

	p.originX = panelLeft
	p.grid = newPlainItemGrid(asset, ui, l, cols, rows, left, top, cell)

	p.Logger = d2util.NewLogger()
	p.Logger.SetLevel(l)
	p.Logger.SetPrefix(logPrefix)

	return p
}

// Load creates the widgets.
func (p *ContainerPanel) Load() {
	p.panelGroup = p.ui.NewWidgetGroup(d2ui.RenderPriorityInventory)
	p.panelGroup.AddWidget(p.ui.NewUIFrame(d2ui.FrameLeft))

	var err error

	if p.art, err = p.ui.NewSprite(p.kind.art, d2resource.PaletteSky); err != nil {
		p.Errorf("%s panel art: %v", p.kind.name, err)
	}

	closeButton := p.ui.NewButton(d2ui.ButtonTypeSquareClose, "")
	closeButton.SetVisible(false)
	closeButton.SetPosition(p.originX+panelCloseOffset, panelCloseY)
	closeButton.OnActivated(func() { p.Close() })
	p.panelGroup.AddWidget(closeButton)

	p.title = p.ui.NewLabel(d2resource.Font16, d2resource.PaletteStatic)
	p.title.Color[0] = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	p.title.SetText(p.kind.title)

	tw, _ := p.title.GetTextMetrics(p.kind.title)
	p.title.SetPosition(p.originX+(320-tw)/2, panelTitleY) //nolint:gomnd // the panel is 320 wide
	p.panelGroup.AddWidget(p.title)

	if p.kind.page == d2hero.PageCube {
		p.transmute = p.ui.NewButton(d2ui.ButtonTypeShort, "Transmute")
		p.transmute.SetVisible(false)
		// centred under the grid
		p.transmute.SetPosition(p.originX+(320-cubeButtonWidth)/2, p.grid.originY+p.grid.height*p.grid.slotSize+cubeButtonGap+cubeButtonHeight)
		p.transmute.OnActivated(func() {
			if p.onTransmute != nil {
				p.onTransmute()
			}
		})
		p.panelGroup.AddWidget(p.transmute)

		p.status = p.ui.NewLabel(d2resource.FontFormal11, d2resource.PaletteStatic)
		p.status.Color[0] = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		p.status.SetPosition(p.originX+panelStatusX, p.grid.originY+p.grid.height*p.grid.slotSize+cubeButtonGap+cubeButtonHeight+panelStatusDY)
		p.panelGroup.AddWidget(p.status)
	}

	p.tooltip = p.ui.NewTooltip(d2resource.FontFormal11, d2resource.PaletteStatic, d2ui.TooltipXCenter, d2ui.TooltipYBottom)
	p.panelGroup.SetVisible(false)
}

// The size of the short button (the picture is 96 by 22 pixels in the sprite sheet) and its gap to the grid.
const (
	cubeButtonWidth  = 96
	cubeButtonHeight = 22
	cubeButtonGap    = 14
)

const (
	panelStatusX  = 20
	panelStatusDY = 22
)

// SetStatus shows a line of text under the Transmute button (the reason a
// transmute did nothing, or what it made).
func (p *ContainerPanel) SetStatus(text string) {
	p.statusText = text

	if p.status != nil {
		p.status.SetText(text)
	}
}

// Status returns the last status line.
func (p *ContainerPanel) Status() string { return p.statusText }

// SetOnTransmute sets the function the Transmute button runs.
func (p *ContainerPanel) SetOnTransmute(f func()) { p.onTransmute = f }

// PressTransmute presses the Transmute button (autotests do it without a mouse).
func (p *ContainerPanel) PressTransmute() {
	if p.transmute != nil {
		p.transmute.Activate()
	}
}

// SetOnClose sets the callback run when the panel closes.
func (p *ContainerPanel) SetOnClose(cb func()) { p.onClose = cb }

// Name returns "stash" or "cube".
func (p *ContainerPanel) Name() string { return p.kind.name }

// Page returns the hero item page of the panel's items.
func (p *ContainerPanel) Page() int { return p.kind.page }

// Grid returns the item grid.
func (p *ContainerPanel) Grid() *ItemGrid { return p.grid }

// Size returns the grid size in cells.
func (p *ContainerPanel) Size() (cols, rows int) { return p.grid.width, p.grid.height }

// IsOpen reports whether the panel is open.
func (p *ContainerPanel) IsOpen() bool { return p.isOpen }

// Open shows the panel.
func (p *ContainerPanel) Open() {
	p.isOpen = true
	p.panelGroup.SetVisible(true)
}

// Close hides the panel.
func (p *ContainerPanel) Close() {
	if !p.isOpen {
		return
	}

	p.isOpen = false
	p.panelGroup.SetVisible(false)
	p.tooltip.SetVisible(false)

	if p.onClose != nil {
		p.onClose()
	}

	p.changed()
}

func (p *ContainerPanel) changed() {
	if p.onChange != nil {
		p.onChange()
	}
}

// Contains reports whether the point is on the panel.
func (p *ContainerPanel) Contains(x, y int) bool {
	return p.isOpen && x >= p.originX && x < p.originX+320 && y >= panelArtTop && y < panelArtTop+432
}

// Place puts an item at a cell (used when the hero is loaded).
func (p *ContainerPanel) Place(item InventoryItem, x, y int) error {
	return p.grid.Set(x, y, item)
}

// Accepts reports whether the panel takes the item at all.
func (p *ContainerPanel) Accepts(item InventoryItem) bool {
	return p.kind.acceptsCube || item.GetItemCode() != cubeItemCode
}

// HandleClick handles a left click while the panel is open and reports whether
// the click was consumed. The cursor item lives in the inventory.
func (p *ContainerPanel) HandleClick(mx, my int, ctrl bool) bool {
	if !p.isOpen {
		return false
	}

	if !p.grid.Contains(mx, my) {
		return p.Contains(mx, my)
	}

	cur := p.cursor.CursorItem()
	if cur != nil && !p.Accepts(cur) {
		p.Infof("%s refuses %s", p.kind.name, cur.GetItemCode())
		return true
	}

	held, act, x, y := p.grid.ClickWith(cur, mx, my, ctrl)

	switch act {
	case ClickPickup:
		p.Infof("picked up %s from the %s at (%d,%d)", held.GetItemCode(), p.kind.name, x, y)
	case ClickPlace, ClickSwap, ClickAuto, ClickMerge:
		p.Infof("%s: %s at (%d,%d)", p.kind.name, act, x, y)
	}

	p.cursor.SetCursorItem(held)

	if act != ClickNone && act != ClickNoTarget && act != ClickRefused {
		p.changed()
	}

	return true
}

// OnMouseMove tracks the pointer and shows item tooltips.
func (p *ContainerPanel) OnMouseMove(mx, my int) {
	p.mouseX, p.mouseY = mx, my

	if !p.isOpen {
		return
	}

	if it := p.grid.ItemAtScreen(mx, my); it != nil {
		p.tooltip.SetTextLines(it.GetItemDescription())
		p.tooltip.SetPosition(mx, my-vendorCell/2)
		p.tooltip.SetVisible(true)

		return
	}

	p.tooltip.SetVisible(false)
}

// Render draws the panel art and the items.
func (p *ContainerPanel) Render(target d2interface.Surface) {
	if !p.isOpen {
		return
	}

	if p.art != nil {
		// quadrants: top-left, top-right, bottom-right, bottom-left (frames 0, 1, 3, 2)
		x, y := p.originX+1, panelArtTop

		for _, frame := range []int{0, 1, 3, 2} {
			if err := p.art.SetCurrentFrame(frame); err != nil {
				p.Error(err.Error())
				return
			}

			w, h := p.art.GetCurrentFrameSize()
			p.art.SetPosition(x, y+h)
			p.art.Render(target)

			switch frame {
			case 0:
				x += w
			case 1:
				y += h
			case 3:
				x = p.originX + 1
			}
		}
	}

	p.grid.Render(target)
}

// Describe lists the items with their grid positions, for the autotest log.
func (p *ContainerPanel) Describe() []string {
	return describeGrid(p.grid)
}

// describeGrid formats the items of a grid, sorted by position.
func describeGrid(g *ItemGrid) []string {
	items := append([]InventoryItem(nil), g.items...)

	// by row, then column
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && lessSlot(items[j], items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}

	out := make([]string, 0, len(items))

	for _, it := range items {
		x, y := it.InventoryGridSlot()
		w, h := it.InventoryGridSize()
		out = append(out, fmt.Sprintf("code=%s x=%d y=%d w=%d h=%d name=%q", it.GetItemCode(), x, y, w, h, itemName(it)))
	}

	return out
}

func lessSlot(a, b InventoryItem) bool {
	ax, ay := a.InventoryGridSlot()
	bx, by := b.InventoryGridSlot()

	if ay != by {
		return ay < by
	}

	return ax < bx
}

// itemName returns the plain (colour code free) first line of an item's description.
func itemName(it InventoryItem) string {
	lines := it.GetItemDescription()
	if len(lines) == 0 {
		return it.GetItemCode()
	}

	return stripColorTokens(lines[0])
}

// stripColorTokens removes the "[gold]" style colour tokens of the engine's strings.
func stripColorTokens(s string) string {
	for {
		i := strings.Index(s, "[")
		j := strings.Index(s, "]")

		if i < 0 || j < i {
			return strings.TrimSpace(s)
		}

		s = s[:i] + s[j+1:]
	}
}
