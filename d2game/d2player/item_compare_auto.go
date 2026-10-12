package d2player

import (
	"fmt"
	"strings"
)

// Console aids for the item comparison scenarios (hover an item through the real input path, log the lines).

// compareGrid returns the grid of a container by name: inv, stash, cube or vendor.
func (g *GameControls) compareGrid(where string) (*ItemGrid, error) {
	switch where {
	case "inv", "inventory":
		return g.inventory.grid, nil
	case "stash":
		return g.stash.grid, nil
	case "cube":
		return g.cube.grid, nil
	case "vendor", "trade":
		return g.Trade.grid, nil
	}

	return nil, fmt.Errorf("unknown container %q (inv, stash, cube, vendor)", where)
}

func (g *GameControls) itemWithCode(where, code string) (InventoryItem, *ItemGrid, error) {
	grid, err := g.compareGrid(where)
	if err != nil {
		return nil, nil, err
	}

	for _, it := range grid.Items() {
		if it.GetItemCode() == code || code == "any" {
			return it, grid, nil
		}
	}

	return nil, grid, fmt.Errorf("no %s in the %s", code, where)
}

// HoverItem moves the pointer onto the first item with this base code in a container, like the player's hand.
func (g *GameControls) HoverItem(where, code string) error {
	it, grid, err := g.itemWithCode(where, code)
	if err != nil {
		return err
	}

	// the panel has to be open for the pointer to reach the item
	switch {
	case (where == "inv" || where == "inventory") && !g.inventory.isOpen:
		g.inventory.Open()
	case where == "stash" && !g.stash.IsOpen():
		g.OpenStash()
	case where == "cube" && !g.cube.IsOpen():
		g.OpenCube()
	}

	x, y := it.InventoryGridSlot()
	w, h := it.InventoryGridSize()
	sx, sy := grid.SlotToScreen(x, y)
	px, py := sx+w*grid.slotSize/2, sy+h*grid.slotSize/2

	g.synthMove(px, py)
	g.Infof("HOVERITEM where=%s code=%s at=(%d,%d)", where, code, px, py)

	return nil
}

// CompareLog logs the comparison lines of the first item with this base code in a container (plain text, one line
// per row, "|" between them) and returns them.
func (g *GameControls) CompareLog(where, code string) ([]string, error) {
	it, _, err := g.itemWithCode(where, code)
	if err != nil {
		return nil, err
	}

	var plain []string

	for _, l := range g.compareLines(it) {
		plain = append(plain, stripColorTokens(l))
	}

	g.Infof("COMPARE where=%s code=%s lines=%d [%s]", where, code, len(plain), strings.Join(plain, " | "))

	return plain, nil
}

// GiveItemToStash creates an identified item and puts it into the stash (scenarios).
func (g *GameControls) GiveItemToStash(code string) error {
	item, err := g.inventory.item.NewItem(code)
	if err != nil {
		return err
	}

	item.Identify()

	w, h := item.InventoryGridSize()

	x, y, ok := g.stash.grid.occupancy().FindFreeSlot(w, h, true)
	if !ok {
		return fmt.Errorf("no room in the stash for %s", code)
	}

	return g.stash.Place(item, x, y)
}

// HoverGround moves the pointer onto the ground item whose label contains the text (scenarios).
func (g *GameControls) HoverGround(label string) error {
	spec, _, err := g.resolveEntitySpec("left@item:" + label)
	if err != nil {
		return err
	}

	_, _, x, y, err := g.parseClick(spec)
	if err != nil {
		return err
	}

	g.synthMove(x, y)
	g.Infof("HOVERGROUND label=%q at=(%d,%d)", label, x, y)

	return nil
}

// wornSlotWithCode finds the equipment slot that holds an item with this base code.
func (g *GameControls) wornSlotWithCode(code string) (EquipmentSlot, error) {
	if !g.inventory.isOpen {
		g.inventory.Open()
	}

	for _, slot := range g.inventory.grid.equipmentSlots {
		if slot.item != nil && slot.item.GetItemCode() == code {
			return slot, nil
		}
	}

	return EquipmentSlot{}, fmt.Errorf("no worn item with code %s", code)
}

// HoverWorn moves the pointer onto the worn item with this base code (scenarios).
func (g *GameControls) HoverWorn(code string) error {
	slot, err := g.wornSlotWithCode(code)
	if err != nil {
		return err
	}

	// slot.y is the bottom edge of the slot (checkEquippedSlotsHover)
	px, py := slot.x+slot.width/2, slot.y-slot.height/2

	g.synthMove(px, py)
	g.Infof("HOVERWORN code=%s at=(%d,%d)", code, px, py)

	return nil
}

// TooltipLog logs the item tooltip text of a worn item (where "worn") or of the first item with this base code
// in a container, one line per row with its colour token ("[green]Name | [white]..."), for the tooltip scenarios.
func (g *GameControls) TooltipLog(where, code string) ([]string, error) {
	var it InventoryItem

	if where == "worn" {
		slot, err := g.wornSlotWithCode(code)
		if err != nil {
			return nil, err
		}

		it = slot.item
	} else {
		found, _, err := g.itemWithCode(where, code)
		if err != nil {
			return nil, err
		}

		it = found
	}

	lines := it.GetItemDescription()
	g.Infof("TOOLTIP where=%s code=%s lines=%d [%s]", where, code, len(lines), strings.Join(lines, " | "))

	return lines, nil
}
