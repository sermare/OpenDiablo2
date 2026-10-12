package d2hero

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// A character created in the lobby is a 335 byte .d2s without items; the game
// hands out the class' starting items when the hero first enters a game
// (CharStats.txt item1..item10 with item<n>loc and item<n>count). The weapon
// and armor rows (locations rarm, larm, ...) are the worn equipment the hero
// state already starts with; what is left are the potions and scrolls of the
// belt and the inventory.

const (
	// startBeltCells is the number of cells of a belt-less hero (one row).
	startBeltCells = 4
	// startInventoryCols is the width of the inventory grid.
	startInventoryCols = 10
)

// StartingContainers returns the inventory and belt a new hero of a class
// starts with, from CharStats.txt: potions go to the belt (in the order of the
// rows), everything else is laid out in the inventory from the top left.
// Rows whose location is a body slot are the equipment, not part of the result.
func (f *HeroStateFactory) StartingContainers(hero d2enum.Hero) *HeroContainers {
	rec := f.asset.Records.Character.Stats[hero]
	if rec == nil {
		return nil
	}

	startSkill := f.startSkillID(rec, hero)

	return startingKitStacks(rec.StartItem[:], rec.StartItemLocation[:], rec.StartItemCount[:], func(code string) (int, bool) {
		item := f.asset.Records.Item.All[code]
		if item == nil {
			return 0, false
		}

		return item.InventoryWidth, true
	}, func(code string) int {
		if item := f.asset.Records.Item.All[code]; item != nil && item.Stackable {
			return item.MaxStack
		}

		return 0
	}, startSkill)
}

// startingKit lays out the item1..item10 columns of a CharStats row: the items
// whose location is a body slot are worn (Equipped, in the order of the rows),
// potions go to the belt, everything else to the inventory from the top left.
// widthOf gives the inventory width of a base item and false for a code the
// tables do not have (such a row is skipped). The worn items are listed in
// Equipped with EquippedSet, so the body of a new hero is exactly the kit and
// nothing else (no placeholder gear); all items are identified, normal quality.
func startingKit(codes, locs []string, counts []int, widthOf func(code string) (int, bool)) *HeroContainers {
	return startingKitStacks(codes, locs, counts, widthOf, nil, 0)
}

// startingKitStacks is startingKit with the stack size of stackable base items: the exe gives a stackable starting
// item a full stack (PLAYER_CreateStartingItem 0x5326c0: ITEM_IsStackable -> stat 0x46 quantity = ITEM_GetMaxStack,
// VERIFIED in the decompilation), so the Amazon's javelin starts as a stack of 60 (weapons.txt maxstack). maxStackOf
// gives the max stack of a stackable base item and 0 for any other. Every item is flagged Starter (0x20000), and the
// item of the first row gets +1 to startSkill (a skill id, 0 for none): PLAYER_CreateStartingItem 0x5326c0 attaches
// stat 107 item_singleskill (param = CharStats StartSkill, value 1) to the item of row 0 (the Sorceress' Fire Bolt on the
// staff, the Necromancer's Raise Skeleton on the wand).
func startingKitStacks(codes, locs []string, counts []int, widthOf func(code string) (int, bool), maxStackOf func(code string) int,
	startSkill int) *HeroContainers {
	stack := func(code string) int {
		if maxStackOf == nil {
			return 0
		}

		return maxStackOf(code)
	}

	rowSkill := func(row, skill int) int {
		if row == 0 {
			return skill
		}

		return 0
	}

	out := &HeroContainers{Items: []StoredItem{}, Equipped: []StoredItem{}, EquippedSet: true}
	belt, invX := 0, 0

	for i, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || code == "0" {
			continue
		}

		w, known := widthOf(code)
		if !known {
			continue
		}

		loc := ""
		if i < len(locs) {
			loc = locs[i]
		}

		if bodySlotLocation(loc) {
			if slot := starterSlot(loc); slot > 0 {
				out.Equipped = append(out.Equipped, StoredItem{
					Code: code, Page: PageEquipped, X: slot, ILvl: 1, Quality: int(d2drop.QualityNormal),
					Seed: int64(1000*i + 1), Identified: true, Origin: true, Quantity: stack(code),
					Starter: true, StartSkill: rowSkill(i, startSkill),
				})
			}

			continue
		}

		count := 1
		if i < len(counts) && counts[i] > 1 {
			count = counts[i]
		}

		if w < 1 {
			w = 1
		}

		for n := 0; n < count; n++ {
			s := StoredItem{Code: code, ILvl: 1, Quality: int(d2drop.QualityNormal), Seed: int64(1000*i + n + 1), Identified: true, Origin: true, Quantity: stack(code), Starter: true,
				StartSkill: rowSkill(i, startSkill)}

			if isPotion(code) && belt < startBeltCells {
				s.Page, s.X = PageBelt, belt
				belt++
			} else {
				if invX+w > startInventoryCols {
					break // does not fit the first rows; a new hero's kit never needs more
				}

				s.Page, s.X, s.Y = PageInventory, invX, 0
				invX += w
			}

			out.Items = append(out.Items, s)
		}
	}

	return out
}

// bodySlotLocation reports a CharStats item location that is a body slot.
func bodySlotLocation(loc string) bool {
	switch strings.ToLower(strings.TrimSpace(loc)) {
	case "head", "neck", "tors", "rarm", "larm", "lrin", "rrin", "belt", "feet", "glov":
		return true
	}

	return false
}

// isPotion says whether a code is a healing, mana or rejuvenation potion
// (misc.txt hp1..hp5, mp1..mp5, rvs, rvl).
func isPotion(code string) bool {
	switch {
	case len(code) == 3 && (strings.HasPrefix(code, "hp") || strings.HasPrefix(code, "mp")) && code[2] >= '1' && code[2] <= '5':
		return true
	case code == "rvs" || code == "rvl":
		return true
	}

	return false
}

// freshHero says whether a hero was never played: level 1, no experience and no
// item in any container or on the body. The lobby writes a character with a
// body but without items (the game's own first save, OD2_AUTONEWCHAR's
// "first-save reparse"), and the starting items are handed out at the first
// game, so such a hero gets them here.
func freshHero(state *HeroState) bool {
	st := state.Stats
	if st == nil || st.Level != 1 || st.Experience != 0 {
		return false
	}

	c := state.Containers

	return c == nil || (len(c.Items) == 0 && len(c.Equipped) == 0 && c.BeltCode == "")
}

// giveStartingItemsToFreshHero puts the starting potions and scrolls of the
// class into a hero that has never been played (see freshHero).
func (f *HeroStateFactory) giveStartingItemsToFreshHero(state *HeroState, hero d2enum.Hero) bool {
	if !freshHero(state) {
		return false
	}

	if c := f.StartingContainers(hero); c != nil && len(c.Items)+len(c.Equipped) > 0 {
		state.Containers = c
	}

	f.wearStartingEquipment(state, hero)

	return true
}

// startSkillRight puts the class' start skill (the Sorceress' Fire Bolt, the Necromancer's Raise Skeleton) on the
// right button of a hero that begins its first game, as SERVER_CreateFreshPlayer 0x52ff20 does with
// SERVER_SetPlayerActiveSkill(hand 0 = right); the left button stays Attack. The exe does it only when the hero
// has the skill (the +1 of the first starting item) - StartingContainers gives that item the bonus.
func (f *HeroStateFactory) startSkillRight(state *HeroState, hero d2enum.Hero) {
	id := f.startSkillID(f.asset.Records.Character.Stats[hero], hero)
	if id <= 0 || state.Skills[id] == nil {
		return
	}

	state.RightSkill = id

	if state.SkillBar == nil {
		state.SkillBar = NewSkillBar()
	}

	state.SkillBar.Right.Skill = id
}

// starterBodyItem is a starting item of CharStats.txt that is worn (location rarm, larm, tors, ...).
type starterBodyItem struct {
	Code string
	Slot int // the .d2s equipment slot
}

// starterBodyItems lists the worn starting items of a class from the item1..item10 columns, in order.
func starterBodyItems(codes, locs []string) []starterBodyItem {
	var out []starterBodyItem

	for i, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || code == "0" || i >= len(locs) {
			continue
		}

		if slot := starterSlot(locs[i]); slot > 0 {
			out = append(out, starterBodyItem{Code: code, Slot: slot})
		}
	}

	return out
}

// wearStartingEquipment puts the worn starting items of the class (the Paladin's sword and buckler, the
// Sorceress' short staff, ...) on the body of a brand new hero. The lobby's 335 byte .d2s has no items, and the
// game hands out the kit at the first game (StartingContainers does the potions and scrolls); without it a new
// Paladin could not cast Smite (a shield is its requirement) and every new hero fought bare-handed. Only the
// combat model (state.Equipment) is filled: the item-less file stays item-less, so the first save exports no
// worn item that "is not from the save".
func (f *HeroStateFactory) wearStartingEquipment(state *HeroState, hero d2enum.Hero) {
	rec := f.asset.Records.Character.Stats[hero]
	if rec == nil {
		return
	}

	// the class kit replaces the placeholder gear of CreateHeroState (DefaultHeroItems); it used to be added on top,
	// which left a second short staff in a Sorceress' left hand that the first .d2s export reported as unwritable
	state.Equipment = d2inventory.CharacterEquipment{}

	for _, it := range starterBodyItems(rec.StartItem[:], rec.StartItemLocation[:]) {
		switch {
		case f.asset.Records.Item.Armors[it.Code] != nil:
			f.equipArmor(&state.Equipment, uint8(it.Slot), it.Code, false)
		case f.asset.Records.Item.Weapons[it.Code] != nil:
			f.equipWeapon(&state.Equipment, uint8(it.Slot), it.Code, false)
		}
	}
}
