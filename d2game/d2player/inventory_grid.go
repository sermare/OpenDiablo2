package d2player

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2ui"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2resource"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// images for 1x1 grid tile items (rings and stuff) are 28x28 pixel
// however, the grid cells are 29x29 pixels, this is for padding
// for each row in inventory, we need to account for this padding
const cellPadding = 1

const (
	fmtFlippyFile = "/data/global/items/inv%s.dc6"
)

// InventoryItem is an interface for an items that can be placed in the inventory grid
type InventoryItem interface {
	InventoryGridSize() (width int, height int)
	GetItemCode() string
	InventoryGridSlot() (x, y int)
	SetInventoryGridSlot(x, y int)
	GetItemDescription() []string
}

var errorInventoryFull = errors.New("inventory full")

// NewItemGrid creates a new ItemGrid instance
func NewItemGrid(asset *d2asset.AssetManager,
	ui *d2ui.UIManager,
	l d2util.LogLevel,
	record *d2records.InventoryRecord) *ItemGrid {
	grid := record.Grid

	itemGrid := &ItemGrid{
		asset:          asset,
		uiManager:      ui,
		width:          gridCells(grid.Columns, grid.Box.Width, grid.CellWidth),
		height:         gridCells(grid.Rows, grid.Box.Height, grid.CellHeight),
		originX:        grid.Box.Left,
		originY:        grid.Box.Top + (grid.Rows * cellPadding),
		slotSize:       grid.CellWidth,
		sprites:        make(map[string]*d2ui.Sprite),
		equipmentSlots: genEquipmentSlotsMap(record),
	}

	itemGrid.Logger = d2util.NewLogger()
	itemGrid.Logger.SetLevel(l)
	itemGrid.Logger.SetPrefix(logPrefix)

	return itemGrid
}

// gridCells returns the number of cells of an axis. The cell count used to be
// taken from the pixel size of the grid box, which made the grid hundreds of
// cells wide; the cell count column is used now, with the pixel size divided
// by the cell size as the fallback.
func gridCells(cells, pixels, cellSize int) int {
	if cells > 0 {
		return cells
	}

	if cellSize > 0 {
		return pixels / cellSize
	}

	return 0
}

// ItemGrid is a reusable grid for use with player and merchant inventory.
// Handles layout and rendering item icons based on code.
type ItemGrid struct {
	asset          *d2asset.AssetManager
	uiManager      *d2ui.UIManager
	items          []InventoryItem
	equipmentSlots map[d2enum.EquippedSlot]EquipmentSlot
	width          int
	height         int
	originX        int
	originY        int
	sprites        map[string]*d2ui.Sprite
	slotSize       int

	*d2util.Logger
}

// SlotToScreen translates slot coordinates to screen coordinates
func (g *ItemGrid) SlotToScreen(slotX, slotY int) (screenX, screenY int) {
	screenX = g.originX + slotX*g.slotSize
	screenY = g.originY + slotY*g.slotSize

	return screenX, screenY
}

// ScreenToSlot translates screen coordinates to slot coordinates
func (g *ItemGrid) ScreenToSlot(screenX, screenY int) (slotX, slotY int) {
	slotX = (screenX - g.originX) / g.slotSize
	slotY = (screenY - g.originY) / g.slotSize

	return slotX, slotY
}

// GetSlot returns the inventory item at a given slot (can return nil)
func (g *ItemGrid) GetSlot(x, y int) InventoryItem {
	for _, item := range g.items {
		slotX, slotY := item.InventoryGridSlot()
		width, height := item.InventoryGridSize()

		if x >= slotX && x < slotX+width && y >= slotY && y < slotY+height {
			return item
		}
	}

	return nil
}

// ChangeEquippedSlot sets the item for an equipment slot
func (g *ItemGrid) ChangeEquippedSlot(slot d2enum.EquippedSlot, item InventoryItem) {
	var curItem = g.equipmentSlots[slot]
	curItem.item = item
	g.equipmentSlots[slot] = curItem
}

// Add places a given set of items into the first available slots.
// Returns a count of the number of items which could be inserted.
func (g *ItemGrid) Add(items ...InventoryItem) (int, error) {
	added := 0

	var err error

	for _, item := range items {
		if g.add(item) {
			added++
		} else {
			err = errorInventoryFull
			break
		}
	}

	g.Load(items...)

	return added, err
}

func (g *ItemGrid) loadItem(item InventoryItem) {
	if _, exists := g.sprites[item.GetItemCode()]; !exists {
		var itemSprite *d2ui.Sprite

		imgPath := fmt.Sprintf(fmtFlippyFile, item.GetItemCode())

		// items whose invfile is not "inv"+code (potions, scrolls, tomes, ...)
		if f, ok := item.(interface{ InventoryFileName() string }); ok && f.InventoryFileName() != "" {
			imgPath = fmt.Sprintf(fmtFlippyFile, f.InventoryFileName()[len("inv"):])
		}

		itemSprite, err := g.uiManager.NewSprite(imgPath, d2resource.PaletteSky)
		if err != nil {
			g.Error("Failed to load sprite, error: " + err.Error())
		}

		g.sprites[item.GetItemCode()] = itemSprite
	}
}

// Load reads the inventory sprites for items into local cache for rendering.
func (g *ItemGrid) Load(items ...InventoryItem) {
	for _, item := range items {
		g.loadItem(item)
	}

	for _, eq := range g.equipmentSlots {
		if eq.item != nil {
			g.loadItem(eq.item)
		}
	}
}

// Walk from top left to bottom right until a position large enough to hold the item is found.
// This is inefficient but simplifies the storage.  At most a hundred or so cells will be looped, so impact is minimal.
func (g *ItemGrid) add(item InventoryItem) bool {
	for y := 0; y < g.height; y++ {
		for x := 0; x < g.width; x++ {
			if !g.canFit(x, y, item) {
				continue
			}

			g.set(x, y, item)

			return true
		}
	}

	return false
}

// canFit loops over all items to determine if any other items would overlap the given position.
func (g *ItemGrid) canFit(x, y int, item InventoryItem) bool {
	insertWidth, insertHeight := item.InventoryGridSize()
	if x+insertWidth > g.width || y+insertHeight > g.height {
		return false
	}

	for _, compItem := range g.items {
		slotX, slotY := compItem.InventoryGridSlot()
		compWidth, compHeight := compItem.InventoryGridSize()

		// rectangles overlap only when they share area; touching edges are fine
		if x+insertWidth > slotX &&
			x < slotX+compWidth &&
			y+insertHeight > slotY &&
			y < slotY+compHeight {
			return false
		}
	}

	return true
}

// Set an inventory item at the given grid coordinate
func (g *ItemGrid) Set(x, y int, item InventoryItem) error {
	if !g.canFit(x, y, item) {
		return fmt.Errorf("can not set item (%s) to position (%v, %v)", item.GetItemCode(), x, y)
	}

	g.set(x, y, item)

	return nil
}

func (g *ItemGrid) set(x, y int, item InventoryItem) {
	item.SetInventoryGridSlot(x, y)
	g.items = append(g.items, item)

	g.Load(item)
}

// Remove does an in place filter to remove the element from the slice of items.
func (g *ItemGrid) Remove(item InventoryItem) {
	n := 0

	for _, compItem := range g.items {
		if compItem == item {
			continue
		}

		g.items[n] = compItem
		n++
	}

	g.items = g.items[:n]
}

func (g *ItemGrid) renderItem(item InventoryItem, target d2interface.Surface, x, y int) {
	itemSprite := g.sprites[item.GetItemCode()]
	if itemSprite != nil {
		itemSprite.SetPosition(x, y)
		itemSprite.GetCurrentFrameSize()
		itemSprite.Render(target)
	}
}

// Render the item grid to the given surface
func (g *ItemGrid) Render(target d2interface.Surface) {
	g.renderInventoryItems(target)
	g.renderEquippedItems(target)
}

func (g *ItemGrid) renderInventoryItems(target d2interface.Surface) {
	for _, item := range g.items {
		itemSprite := g.sprites[item.GetItemCode()]
		slotX, slotY := g.SlotToScreen(item.InventoryGridSlot())
		_, h := itemSprite.GetCurrentFrameSize()
		slotY += h

		g.renderItem(item, target, slotX, slotY)
	}
}

func (g *ItemGrid) renderEquippedItems(target d2interface.Surface) {
	for _, eq := range g.equipmentSlots {
		if eq.item == nil {
			continue
		}

		itemSprite := g.sprites[eq.item.GetItemCode()]
		itemWidth, itemHeight := itemSprite.GetCurrentFrameSize()
		// nolint:gomnd // 1/2 ov width
		x := eq.x + ((eq.width - itemWidth) / 2)
		// nolint:gomnd // 1/2 ov height
		y := eq.y - ((eq.height - itemHeight) / 2)

		g.renderItem(eq.item, target, x, y)
	}
}

// newPlainItemGrid creates a grid without equipment slots (vendor storage).
func newPlainItemGrid(asset *d2asset.AssetManager, ui *d2ui.UIManager, l d2util.LogLevel,
	cols, rows, left, top, cell int) *ItemGrid {
	g := &ItemGrid{
		asset:          asset,
		uiManager:      ui,
		width:          cols,
		height:         rows,
		originX:        left,
		originY:        top + rows*cellPadding,
		slotSize:       cell,
		sprites:        make(map[string]*d2ui.Sprite),
		equipmentSlots: map[d2enum.EquippedSlot]EquipmentSlot{},
	}

	g.Logger = d2util.NewLogger()
	g.Logger.SetLevel(l)
	g.Logger.SetPrefix(logPrefix)

	return g
}

// Items returns the items on the grid (not a copy).
func (g *ItemGrid) Items() []InventoryItem {
	return g.items
}

// Bounds returns the screen rectangle of the grid's cells.
func (g *ItemGrid) Bounds() (left, top, right, bottom int) {
	return g.originX, g.originY, g.originX + g.width*g.slotSize, g.originY + g.height*g.slotSize
}

// ItemAtScreen returns the item under a screen position, or nil.
func (g *ItemGrid) ItemAtScreen(mx, my int) InventoryItem {
	left, top, right, bottom := g.Bounds()
	if mx < left || my < top || mx >= right || my >= bottom {
		return nil
	}

	x, y := g.ScreenToSlot(mx, my)

	return g.GetSlot(x, y)
}

// occupancy builds the occupancy map of the grid's items.
func (g *ItemGrid) occupancy() *d2inventory.OccupancyGrid {
	og := d2inventory.NewOccupancyGrid(g.width, g.height)

	for _, it := range g.items {
		x, y := it.InventoryGridSlot()
		w, h := it.InventoryGridSize()
		og.Fill(x, y, w, h, true)
	}

	return og
}

// CanAutoPlace reports whether the item fits anywhere (game search order,
// d2inventory.OccupancyGrid.FindFreeSlot).
func (g *ItemGrid) CanAutoPlace(item InventoryItem, playerOwner bool) bool {
	w, h := item.InventoryGridSize()
	_, _, ok := g.occupancy().FindFreeSlot(w, h, playerOwner)

	return ok
}

// AutoPlace puts the item where the original game's search for a free slot
// would (scored for the player, first fit for a vendor); false if full.
func (g *ItemGrid) AutoPlace(item InventoryItem, playerOwner bool) bool {
	w, h := item.InventoryGridSize()

	x, y, ok := g.occupancy().FindFreeSlot(w, h, playerOwner)
	if !ok {
		return false
	}

	g.set(x, y, item)

	return true
}
