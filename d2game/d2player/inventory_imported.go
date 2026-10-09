package d2player

import (
	"regexp"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// d2sEquipSlots maps the equipment slot ids of a .d2s to the slots of the panel.
// RightArm is the box on the left of the screen (the weapon), LeftArm the one on
// the right (shield), and the ring "hands" follow inventory.txt (rHand = right ring).
var d2sEquipSlots = map[uint8]d2enum.EquippedSlot{
	1:  d2enum.EquippedSlotHead,
	2:  d2enum.EquippedSlotNeck,
	3:  d2enum.EquippedSlotTorso,
	4:  d2enum.EquippedSlotRightArm,
	5:  d2enum.EquippedSlotLeftArm,
	6:  d2enum.EquippedSlotRightHand,
	7:  d2enum.EquippedSlotLeftHand,
	8:  d2enum.EquippedSlotBelt,
	9:  d2enum.EquippedSlotLegs,
	10: d2enum.EquippedSlotGloves,
}

// colorTokens matches the [gold]-style color markers of tooltip lines.
var colorTokens = regexp.MustCompile(`\[[a-z]+\]`)

// SetImportedItems makes the panel show the equipment and the inventory page of an
// imported .d2s hero instead of the placeholder test items. It must be called
// before Load. Items of the other pages (belt, cube, stash) are not placed here.
func (g *Inventory) SetImportedItems(items []d2hero.ImportedItem, weaponSetII bool) {
	g.imported = true
	g.importedItems = items
	g.importedSetII = weaponSetII
}

// placeImportedItems puts the imported equipped items and page 1 items in place.
func (g *Inventory) placeImportedItems() {
	for i := range g.importedItems {
		src := &g.importedItems[i]

		item, err := g.item.NewItem(src.Codes...)
		if err != nil {
			g.Errorf("imported item %v: %v", src.Codes, err)
			continue
		}

		if src.Identified {
			item.Identify()
		}

		switch {
		case src.Location == d2s.LocationEquipped:
			slot := d2hero.ActiveWeaponSetSlot(src.Equipped, g.importedSetII)
			if enumSlot, ok := d2sEquipSlots[slot]; ok {
				g.grid.ChangeEquippedSlot(enumSlot, item)
			}
		case src.Location == d2s.LocationStored && src.Page == 1 && src.AutoPlace:
			if _, err := g.grid.Add(item); err != nil {
				g.Errorf("imported item %v: %v", src.Codes, err)
			}
		case src.Location == d2s.LocationStored && src.Page == 1:
			if err := g.grid.Set(int(src.X), int(src.Y), item); err != nil {
				g.Errorf("imported item %v: %v", src.Codes, err)
			}
		}
	}

	g.grid.Load()
}

// EquippedSummary lists the item names in the equipment boxes and the number of
// items in the inventory grid, for the autotest PANEL line.
func (g *Inventory) EquippedSummary() string {
	order := []struct {
		name string
		slot d2enum.EquippedSlot
	}{
		{"head", d2enum.EquippedSlotHead}, {"neck", d2enum.EquippedSlotNeck},
		{"torso", d2enum.EquippedSlotTorso}, {"weapon", d2enum.EquippedSlotRightArm},
		{"shield", d2enum.EquippedSlotLeftArm}, {"ringR", d2enum.EquippedSlotRightHand},
		{"ringL", d2enum.EquippedSlotLeftHand}, {"belt", d2enum.EquippedSlotBelt},
		{"feet", d2enum.EquippedSlotLegs}, {"gloves", d2enum.EquippedSlotGloves},
	}

	out := ""

	for _, o := range order {
		if it := g.grid.equipmentSlots[o.slot].item; it != nil {
			name := it.GetItemCode()
			if lines := it.GetItemDescription(); len(lines) > 0 {
				name = colorTokens.ReplaceAllString(lines[0], "") // the first tooltip line is the item name
			}

			out += " " + o.name + "=" + name
		}
	}

	return out
}
