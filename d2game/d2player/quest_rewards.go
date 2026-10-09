package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// Item access for quest rewards (Larzuk, Anya) and the Horadric Cube recipes of
// the quests and the Pandemonium event.

// FindItem returns the first item (the cursor first, then the inventory grid)
// the predicate accepts, or nil.
func (g *GameControls) FindItem(pred func(*diablo2item.Item) bool) *diablo2item.Item {
	if it, ok := g.inventory.CursorItem().(*diablo2item.Item); ok && it != nil && pred(it) {
		return it
	}

	for _, it := range g.inventory.grid.Items() {
		if d, ok := it.(*diablo2item.Item); ok && pred(d) {
			return d
		}
	}

	return nil
}

// CubeCodes lists the base codes of the items in the Horadric Cube.
func (g *GameControls) CubeCodes() []string {
	var out []string

	for _, it := range g.cube.grid.Items() {
		out = append(out, strings.TrimSpace(it.GetItemCode()))
	}

	return out
}

// CubeClear removes every item from the Horadric Cube (a transmutation
// consumed them) and returns how many there were.
func (g *GameControls) CubeClear() int {
	items := append([]InventoryItem{}, g.cube.grid.Items()...)
	for _, it := range items {
		g.cube.grid.Remove(it)
	}

	return len(items)
}

// CubePut places an item in the Horadric Cube at the first free spot; it
// reports whether it fitted.
func (g *GameControls) CubePut(it InventoryItem) bool {
	w, h := it.InventoryGridSize()

	x, y, ok := g.cube.grid.occupancy().FindFreeSlot(w, h, true)
	if !ok {
		return false
	}

	return g.cube.Place(it, x, y) == nil
}

// GiveItemToCube creates an item and puts it into the Horadric Cube (a console
// aid for the autotests; the item is identified).
func (g *GameControls) GiveItemToCube(code string) error {
	item, err := g.inventory.item.NewItem(code)
	if err != nil {
		return err
	}

	item.Identify()

	if !g.CubePut(item) {
		return fmt.Errorf("no room in the Horadric Cube for %s", code)
	}

	return nil
}

// SaveItems stores the hero after an item was changed by a reward.
func (g *GameControls) SaveItems() { g.saveHero() }

// GiveItemQuality puts a new item of the given quality (d2drop.Quality number)
// and item level into the inventory (a console aid for the autotests) and
// returns its name.
func (g *GameControls) GiveItemQuality(code string, quality, ilvl int) (string, error) {
	item, err := g.inventory.item.ItemFromCode(code, d2drop.Quality(quality), ilvl, 4242)
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
