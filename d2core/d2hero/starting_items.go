package d2hero

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
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

	out := &HeroContainers{Items: []StoredItem{}}
	belt, invX := 0, 0

	for i, code := range rec.StartItem {
		code = strings.TrimSpace(code)
		if code == "" || code == "0" || bodySlotLocation(rec.StartItemLocation[i]) {
			continue
		}

		item := f.asset.Records.Item.All[code]
		if item == nil {
			continue
		}

		count := rec.StartItemCount[i]
		if count < 1 {
			count = 1
		}

		w := item.InventoryWidth
		if w < 1 {
			w = 1
		}

		for n := 0; n < count; n++ {
			s := StoredItem{Code: code, ILvl: 1, Quality: int(d2drop.QualityNormal), Seed: int64(1000*i + n + 1), Identified: true, Origin: true}

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
func (f *HeroStateFactory) giveStartingItemsToFreshHero(state *HeroState, hero d2enum.Hero) {
	if !freshHero(state) {
		return
	}

	if c := f.StartingContainers(hero); c != nil && len(c.Items) > 0 {
		state.Containers = c
	}
}
