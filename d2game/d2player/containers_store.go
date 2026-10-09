package d2player

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2equip"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// This file connects the hero's saved containers (d2hero.HeroContainers: what
// the hero file and a .d2s import hold) with the inventory grid, the stash, the
// cube and the belt.

// realiseStored builds the item a stored entry describes.
func realiseStored(f *diablo2item.ItemFactory, s *d2hero.StoredItem) (*diablo2item.Item, error) {
	spec := diablo2item.Spec{
		Code: s.Code, Quality: s.Quality, ILvl: s.ILvl, Seed: s.Seed,
		Unique: s.Unique, SetItem: s.SetItem, Set: s.Set, Prefixes: s.Prefixes, Suffixes: s.Suffixes,
		Identified: s.Identified, Ethereal: s.Ethereal, Quantity: s.Quantity, Durability: -1,
	}

	if s.Durability != nil {
		spec.Durability = *s.Durability
	}

	if s.Origin {
		// imported from a .d2s: roll the base item with the item generator (an
		// approximation, see StoredItem.Origin), then keep what that rolled
		it, err := f.ItemFromCode(s.Code, d2drop.Quality(s.Quality), s.ILvl, uint32(s.Seed))
		if err != nil {
			return nil, err
		}

		rolled := it.Spec()
		rolled.Identified, rolled.Ethereal = spec.Identified, spec.Ethereal
		rolled.Quantity, rolled.Durability = spec.Quantity, spec.Durability

		spec = rolled
	}

	item, err := f.ItemFromSpec(spec)
	if err != nil {
		return nil, err
	}

	if s.D2S != nil {
		item.SetOrigin(s.D2S) // the tooltip is built from the saved item
	}

	return item, nil
}

// storedFromItem describes a placed item.
func storedFromItem(it *diablo2item.Item, page, x, y int, orig *d2s.Item) d2hero.StoredItem {
	spec := it.Spec()
	s := d2hero.StoredItem{
		Code: spec.Code, Page: page, X: x, Y: y,
		Quality: spec.Quality, ILvl: spec.ILvl, Seed: spec.Seed,
		Unique: spec.Unique, SetItem: spec.SetItem, Set: spec.Set, Prefixes: spec.Prefixes, Suffixes: spec.Suffixes,
		Identified: spec.Identified, Ethereal: spec.Ethereal, Quantity: spec.Quantity,
		D2S: orig,
	}

	if spec.Durability >= 0 {
		d := spec.Durability
		s.Durability = &d
	}

	return s
}

// loadContainers fills the inventory, stash, cube and belt from the hero's
// saved containers. Items that cannot be built are skipped with a warning.
func (g *GameControls) loadContainers() {
	c := g.hero.Containers
	if c == nil {
		return
	}

	factory := g.inventory.item
	g.itemOrigin = make(map[InventoryItem]*d2s.Item)

	g.loadEquipment()

	if c.BeltCode != "" && g.inventory.WornAt(d2equip.LocBelt) == nil {
		if belt, err := factory.NewItem(c.BeltCode); err == nil {
			belt.Identify()
			g.inventory.grid.ChangeEquippedSlot(d2enum.EquippedSlotBelt, belt)
			g.inventory.grid.Load()
		}
	}

	counts := map[int]int{}

	for i := range c.Items {
		s := &c.Items[i]

		item, err := realiseStored(factory, s)
		if err != nil {
			g.Warningf("skipping stored item %q (page %d): %v", s.Code, s.Page, err)
			continue
		}

		if s.D2S != nil {
			g.itemOrigin[item] = s.D2S
		}

		if err := g.placeStored(item, s); err != nil {
			g.Warningf("stored item %q does not fit (page %d, %d,%d): %v", s.Code, s.Page, s.X, s.Y, err)
			continue
		}

		counts[s.Page]++
	}

	g.Infof("containers loaded: belt=%q boxes=%d inventory=%d belt_items=%d cube=%d stash=%d", c.BeltCode, g.beltBoxes(), counts[d2hero.PageInventory],
		counts[d2hero.PageBelt], counts[d2hero.PageCube], counts[d2hero.PageStash])
}

func (g *GameControls) placeStored(item *diablo2item.Item, s *d2hero.StoredItem) error {
	switch s.Page {
	case d2hero.PageInventory:
		return g.inventory.grid.Set(s.X, s.Y, item)
	case d2hero.PageStash:
		return g.stash.Place(item, s.X, s.Y)
	case d2hero.PageCube:
		return g.cube.Place(item, s.X, s.Y)
	case d2hero.PageBelt:
		return g.belt.Set(s.X, item)
	default:
		return fmt.Errorf("unknown page %d", s.Page)
	}
}

// snapshotContainers captures the current content of every container. An item
// on the cursor is kept too: it is stored at the spot the auto-placement
// search finds in the inventory (the grids themselves are left alone).
func (g *GameControls) snapshotContainers() *d2hero.HeroContainers {
	out := &d2hero.HeroContainers{Items: []d2hero.StoredItem{}}

	if g.hero.Containers != nil {
		out.BeltCode = g.hero.Containers.BeltCode
	}

	if belt, ok := g.inventory.grid.equipmentSlots[d2enum.EquippedSlotBelt].item.(*diablo2item.Item); ok {
		out.BeltCode = belt.CommonCode
	}

	add := func(it InventoryItem, page, x, y int) {
		item, ok := it.(*diablo2item.Item)
		if !ok {
			return
		}

		out.Items = append(out.Items, storedFromItem(item, page, x, y, g.itemOrigin[it]))
	}

	grids := []struct {
		page int
		grid *ItemGrid
	}{
		{d2hero.PageInventory, g.inventory.grid},
		{d2hero.PageCube, g.cube.grid},
		{d2hero.PageStash, g.stash.grid},
	}

	for _, gr := range grids {
		for _, it := range gr.grid.items {
			x, y := it.InventoryGridSlot()
			add(it, gr.page, x, y)
		}
	}

	for cell, it := range g.belt.items {
		if it != nil {
			add(it, d2hero.PageBelt, cell, 0)
		}
	}

	if g.equipOn() {
		out.Equipped, out.EquippedSet, out.ActiveArms = g.snapshotEquipped(), true, g.inventory.activeArms
		if out.Equipped == nil {
			out.Equipped = []d2hero.StoredItem{}
		}
	}

	if cur := g.inventory.CursorItem(); cur != nil {
		w, h := cur.InventoryGridSize()
		if x, y, ok := g.inventory.grid.occupancy().FindFreeSlot(w, h, true); ok {
			add(cur, d2hero.PageInventory, x, y)
		}
	}

	return out
}

// SyncContainers refreshes the hero's saved containers from the panels; the
// game calls it before sending a save to the server.
func (g *GameControls) SyncContainers() {
	g.hero.Containers = g.snapshotContainers()
}

// ContainerPositions lists a container's items with their grid positions
// (inventory, stash, cube, belt), for the autotest log.
func (g *GameControls) ContainerPositions(name string) (lines []string, size string, err error) {
	switch name {
	case "inventory":
		return describeGrid(g.inventory.grid), fmt.Sprintf("%dx%d", g.inventory.grid.width, g.inventory.grid.height), nil
	case "stash":
		cols, rows := g.stash.Size()

		return g.stash.Describe(), fmt.Sprintf("%dx%d", cols, rows), nil
	case "cube":
		cols, rows := g.cube.Size()

		return g.cube.Describe(), fmt.Sprintf("%dx%d", cols, rows), nil
	case "belt":
		return g.belt.Describe(), fmt.Sprintf("%dx%d boxes=%d", d2inventory.BeltColumns, g.belt.Rows(), g.belt.Boxes()), nil
	}

	return nil, "", fmt.Errorf("unknown container %q", name)
}

// SpecRoundTripMismatches rebuilds every container item from its spec and
// counts the items whose description differs, for the autotest log: saved
// items must come back identical.
func (g *GameControls) SpecRoundTripMismatches() (checked, mismatched int) {
	factory := g.inventory.item

	check := func(it InventoryItem) {
		item, ok := it.(*diablo2item.Item)
		if !ok {
			return
		}

		checked++

		again, err := factory.ItemFromSpec(item.Spec())
		if err != nil {
			g.Infof("AUTOPANEL spec mismatch code=%s: %v", item.CommonCode, err)
			mismatched++

			return
		}

		if a, b := strings.Join(item.GetItemDescription(), "|"), strings.Join(again.GetItemDescription(), "|"); a != b {
			g.Infof("AUTOPANEL spec mismatch code=%s: %q != %q", item.CommonCode, a, b)
			mismatched++
		}
	}

	for _, grid := range []*ItemGrid{g.inventory.grid, g.stash.grid, g.cube.grid} {
		for _, it := range grid.items {
			check(it)
		}
	}

	for _, it := range g.belt.items {
		if it != nil {
			check(it)
		}
	}

	return checked, mismatched
}

// beltBoxes is the number of belt cells the equipped belt gives
// (belts.txt numboxes via the armor.txt belt column); without a belt it is the
// "default" row of belts.txt.
func (g *GameControls) beltBoxes() int {
	rec := g.inventory.grid.equipmentSlots[d2enum.EquippedSlotBelt].item

	item, ok := rec.(*diablo2item.Item)
	if !ok {
		return d2inventory.BeltDefaultBoxes
	}

	if b := g.asset.Records.Item.Belts.ByIndex(item.CommonRecord().BeltIndex); b != nil {
		return b.NumBoxes
	}

	return d2inventory.BeltDefaultBoxes
}
