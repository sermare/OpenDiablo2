package d2hero

import "sort"

// Corpse looting (Game.exe 1.14b, read-only Ghidra, w50). Addresses only, own words.
//
//	VERIFIED:
//	- PLAYER_ServerCreateCorpse 0x57d6f0: the cursor item and the worn items (body locations 1..12) leave the hero for
//	  the corpse's inventory; one that cannot be placed there falls to the floor; the gold is dropped in piles.
//	- Interaction (SERVER_InteractWithUnitByType 0x546a30, unit type 0): within 8 table units of a unit in mode 0x11
//	  (dead), and neither player having a hostile/duel target reference, the click loots (PLAYER_ServerLootCorpse
//	  0x57db90); farther away the hero is made to run towards it first.
//	- PLAYER_CanLootCorpse 0x57db10: the looter is the owner of the corpse's inventory, or the owner's relation flags
//	  towards the looter have bit 1 (the party "loot my corpse" toggle, party_tooltips.go bit 01; the exact name of the
//	  flag is UNVERIFIED but the toggle is the only thing that sets it for a party).
//	- The owner touching a corpse whose experience stat (0xd) is non-zero gets that experience and the stat is cleared.
//	- ITEMMODE_ServerAutoEquipFromOtherUnit 0x560d30, the transfer: pass one walks the body slots 1..12 of the corpse;
//	  a worn item whose requirements the looter meets is put into the same slot of the looter when that is allowed
//	  (empty slot, hand pairing rules) or else into his inventory; the walk repeats while it moved something. Pass two
//	  takes every remaining corpse item: it must pass the quest/unique gate (0x55aa20), then it is worn if its slot is
//	  free, else belted if it is a potion the belt takes, else (second sweep) placed in the inventory; a failure marks
//	  the loot incomplete. Complete: the corpse is removed (broadcast 0x8e) and the player gets sound event 0x5d.
//	  Incomplete: the corpse stays with what is left and the player gets sound event 0x17 = the "cantcarry" hero voice.

// LootHost is what the looter's side of the transfer offers. The slots are .d2s body location numbers (1..12).
type LootHost interface {
	// MeetsRequirements says whether the looter satisfies strength, dexterity, level and class of the item.
	MeetsRequirements(it *StoredItem) bool
	// TryEquip wears the item at its own body slot when that slot accepts it now (free, rules allow) and reports it.
	TryEquip(slot int, it *StoredItem) bool
	// TryBelt puts a potion the belt takes into it.
	TryBelt(it *StoredItem) bool
	// TryInventory places the item in the inventory with the auto-placement search.
	TryInventory(it *StoredItem) bool
	// CanTake is the quest/unique gate of ITEMMODE_CanPickupQuestOrUniqueItem 0x55aa20.
	CanTake(it *StoredItem) bool
}

// CanLootCorpse is PLAYER_CanLootCorpse: the owner always, anyone else only while the owner's party relation allows
// it (partyAllows).
func CanLootCorpse(ownerID, looterID string, partyAllows bool) bool {
	return ownerID == looterID || partyAllows
}

// LootCorpse moves the corpse's items to the looter following the exe's two passes and returns the items it could not
// give. complete is true when nothing is left (the corpse then disappears); a false result is the cantcarry case. The
// corpse's Items slice is replaced with the leftovers.
func LootCorpse(c *Corpse, h LootHost) (left []StoredItem, complete bool) {
	if c == nil {
		return nil, true
	}

	items := make([]StoredItem, len(c.Items))
	copy(items, c.Items)
	sort.SliceStable(items, func(i, j int) bool { return bodyOrder(&items[i]) < bodyOrder(&items[j]) })

	taken := make([]bool, len(items))

	// pass one: worn items, the walk repeats while it moved something (a freed slot can admit another item)
	for moved := true; moved; {
		moved = false

		for i := range items {
			it := &items[i]
			if taken[i] || it.Page != PageEquipped || it.X < 1 || it.X > 12 || !h.MeetsRequirements(it) || !h.CanTake(it) {
				continue
			}

			if h.TryEquip(it.X, it) || h.TryInventory(it) {
				taken[i], moved = true, true
			}
		}
	}

	// pass two: everything left, worn if possible, belt, and on the second sweep the inventory
	for sweep := 0; sweep < 2; sweep++ {
		for i := range items {
			it := &items[i]
			if taken[i] || !h.CanTake(it) {
				continue
			}

			switch {
			case it.Page == PageEquipped && it.X >= 1 && it.X <= 12 && h.TryEquip(it.X, it):
				taken[i] = true
			case h.TryBelt(it):
				taken[i] = true
			case sweep == 1 && h.TryInventory(it):
				taken[i] = true
			}
		}
	}

	for i := range items {
		if !taken[i] {
			left = append(left, items[i])
		}
	}

	c.Items = left

	return left, len(left) == 0
}

// bodyOrder sorts worn items by slot first, the cursor/inventory items after them.
func bodyOrder(s *StoredItem) int {
	if s.Page == PageEquipped {
		return s.X
	}

	return 100 + s.X
}
