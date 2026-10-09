package d2hero

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// Equipment slots of a .d2s file (Item.Equipped).
const (
	d2sSlotHead     = 1
	d2sSlotTorso    = 3
	d2sSlotRightArm = 4 // primary weapon
	d2sSlotLeftArm  = 5 // shield or off-hand weapon
	d2sSlotFeet     = 9
	d2sSlotGloves   = 10
)

// Excel tables that describe items in a save, and where the game keeps them.
const (
	excelDir        = "data/global/excel/"
	itemStatCostBin = excelDir + "itemstatcost.bin"
	itemStatCostTxt = excelDir + "ItemStatCost.txt"
	armorTxt        = excelDir + "armor.txt"
	weaponsTxt      = excelDir + "weapons.txt"
	miscTxt         = excelDir + "misc.txt"
	itemTypesTxt    = excelDir + "ItemTypes.txt"
)

// loadD2SItemTables builds the item tables a .d2s needs from the game data.
func (f *HeroStateFactory) loadD2SItemTables() (*d2s.ItemTables, error) {
	statCost, err := f.asset.LoadFile(itemStatCostBin)
	if err != nil {
		if statCost, err = f.asset.LoadFile(itemStatCostTxt); err != nil {
			return nil, err
		}
	}

	files := make([][]byte, 0, 4)

	for _, name := range []string{armorTxt, weaponsTxt, miscTxt, itemTypesTxt} {
		data, err := f.asset.LoadFile(name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}

		files = append(files, data)
	}

	return d2s.NewItemTables(statCost, files[0], files[1], files[2], files[3])
}

// applyD2SEquipment places the equipped items of a save into the hero's
// equipment. Slots OpenDiablo2 does not model yet (amulet, rings, belt) and
// items that cannot be resolved are skipped.
func (f *HeroStateFactory) applyD2SEquipment(state *HeroState, items []d2s.Item, tables *d2s.ItemTables, weaponSetII bool) {
	for i := range items {
		item := &items[i]
		if item.Location != d2s.LocationEquipped {
			continue
		}

		// weapons of the inactive weapon set are not worn
		slot := ActiveWeaponSetSlot(item.Equipped, weaponSetII)
		if slot == 0 {
			continue
		}

		eqItem := *item
		eqItem.Equipped = slot
		item = &eqItem

		code := strings.TrimSpace(item.Code)

		switch tables.ItemKindOf(code) {
		case d2s.KindArmor:
			f.equipArmor(&state.Equipment, item.Equipped, code)
		case d2s.KindWeapon:
			f.equipWeapon(&state.Equipment, item.Equipped, code)
		}
	}
}

func (f *HeroStateFactory) equipArmor(eq *d2inventory.CharacterEquipment, slot uint8, code string) {
	armor, err := f.GetArmorItemByCode(code)
	if err != nil {
		return
	}

	switch slot {
	case d2sSlotHead:
		eq.Head = armor
	case d2sSlotTorso:
		eq.Torso = armor
	case d2sSlotFeet:
		eq.Legs = armor
	case d2sSlotGloves:
		eq.RightArm = armor
	case d2sSlotLeftArm:
		eq.Shield = armor
	}
}

func (f *HeroStateFactory) equipWeapon(eq *d2inventory.CharacterEquipment, slot uint8, code string) {
	weapon, err := f.GetWeaponItemByCode(code)
	if err != nil {
		return
	}

	switch slot {
	case d2sSlotRightArm:
		eq.RightHand = weapon
	case d2sSlotLeftArm:
		eq.LeftHand = weapon
	}
}
