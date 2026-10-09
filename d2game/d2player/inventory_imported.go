package d2player

import (
	"regexp"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

// colorTokens matches the [gold]-style color markers of tooltip lines.
var colorTokens = regexp.MustCompile(`\[[a-z]+\]`)

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
