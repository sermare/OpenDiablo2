package d2hero

// CharmActive says whether an inventory charm gives its stats to the hero.
//
// VERIFIED in Game.exe 1.14b: ITEMMODE_ApplyEquipEffectsIfEligible 0x55b780 applies an item stored
// in the inventory (anim mode 0) only when INV_ItemMeetsRequirementsNow says so. That function
// accepts the item only when it is not broken (flag 0x100), not flag 0x4000, is of item type 0xd (the
// charm type), sits on page byte 0 (the inventory, never stash or cube), and then runs
// INV_CheckItemRequirements, which needs the identified flag (0x10), the strength, dexterity,
// level and class requirements (charms have no class). So a charm that is unidentified, or whose
// level requirement the hero does not reach, gives nothing while it lies in the inventory.
//
// reqLevel is the base item's levelreq only: the affix and unique level requirements are
// UNVERIFIED here (the worn-item path in equip.go has the same limit).
func CharmActive(identified bool, reqLevel, heroLevel int) bool {
	if !identified {
		return false
	}

	return reqLevel <= 0 || heroLevel >= reqLevel
}
