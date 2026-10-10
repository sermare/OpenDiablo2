package d2player

import (
	"fmt"
	"strings"
)

// MoveToCube moves one inventory item of each base code into the Horadric Cube (the debug counterpart of
// dragging the items in with the mouse; scripts use it to cube Khalim's Will from the Flail and the three
// organs). The items stay where they are when one of them is missing or the cube has no room.
func (g *GameControls) MoveToCube(codes ...string) error {
	var picked []InventoryItem

	for _, code := range codes {
		var found InventoryItem

		for _, it := range g.inventory.grid.Items() {
			if strings.TrimSpace(it.GetItemCode()) != code {
				continue
			}

			already := false

			for _, p := range picked {
				already = already || p == it
			}

			if !already {
				found = it

				break
			}
		}

		if found == nil {
			return fmt.Errorf("no %s in the inventory", code)
		}

		picked = append(picked, found)
	}

	for i, it := range picked {
		g.inventory.grid.Remove(it)

		if !g.cube.grid.AutoPlace(it, true) {
			g.inventory.grid.AutoPlace(it, true)

			for _, back := range picked[:i] {
				g.cube.grid.Remove(back)
				g.inventory.grid.AutoPlace(back, true)
			}

			return fmt.Errorf("no room in the cube for %s", codes[i])
		}
	}

	return nil
}
