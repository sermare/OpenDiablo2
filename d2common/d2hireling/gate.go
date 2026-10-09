package d2hireling

// Quest and level gates of the hire NPCs and the revive allow-list (V,
// Game.exe HIRE_ProcessHireOffer 0x574f00, level helper 0x5746a0,
// HIRE_ServerHandleReviveMercenary 0x577a10).

const (
	// SellerKashya, SellerQualKehk and SellerTyrael are monstats class ids.
	SellerKashya   = 150
	SellerQualKehk = 515
	SellerTyrael   = 367 // 0x16f: Tyrael (act 4) may only revive, never hire

	// QuestSlotBurial is the quest record slot (bit 0 = done) Kashya wants
	// from characters below level 8: Sisters' Burial Grounds.
	QuestSlotBurial = 2
	// QuestSlotRescue is the slot (0x24) Qual-Kehk always wants: Rescue on
	// Mount Arreat.
	QuestSlotRescue = 36

	kashyaFreeLevel = 8
)

// normalLevelCap is the table at 0x5746a0, indexed by the byte at +0x22 of
// the vendor record (0..4, probably the act): in Normal difficulty the level
// used by the Kashya gate is min(level, cap).
var normalLevelCap = [5]int{12, 20, 28, 36, 45}

// GateLevel is the level the Kashya check compares (helper 0x5746a0, V):
// only in Normal difficulty and for a vendor index 0..4 is the level capped.
// Since every cap is above 8 the cap never changes the Kashya outcome.
func GateLevel(difficulty0, vendorIdx, level int) int {
	if difficulty0 == 0 && vendorIdx >= 0 && vendorIdx < len(normalLevelCap) && level > normalLevelCap[vendorIdx] {
		return normalLevelCap[vendorIdx]
	}

	return level
}

// HireAllowed is the quest gate of HIRE_ProcessHireOffer (V): Qual-Kehk needs
// bit 0 of quest slot 36 (Rescue on Mount Arreat) in the quest record of the
// game difficulty; Kashya needs bit 0 of slot 2 (Sisters' Burial Grounds)
// while the gate level is below 8; every other seller has no gate. done
// reports bit 0 of a quest record slot.
func HireAllowed(seller, difficulty0, level int, done func(slot int) bool) bool {
	switch seller {
	case SellerQualKehk:
		return done(QuestSlotRescue)
	case SellerKashya:
		if GateLevel(difficulty0, 0, level) < kashyaFreeLevel {
			return done(QuestSlotBurial)
		}
	}

	return true
}

// CanRevive reports whether an NPC class may revive a merc: the four hire
// NPCs and Tyrael (0x16f), V.
func CanRevive(npcClass int) bool {
	switch npcClass {
	case SellerKashya, 198, 252, SellerQualKehk, SellerTyrael:
		return true
	}

	return false
}

// BossDamage scales the damage of a hireling to a boss: the damage percent
// helper 0x579d70 returns the DifficultyLevels row's HireableBossDamagePercent
// (offset 0x38) when the attacker is a hireling (classes 0x10f, 0x152, 0x167,
// 0x230, 0x231) and the target's monstats record has flag 0x40 (the boss
// column, an inference from the level rule), V. The damage is multiplied by
// the percentage (the exe uses MulDiv(x, pct, 100)).
func BossDamage(dmg, hireableBossDamagePercent int) int {
	return dmg * hireableBossDamagePercent / 100
}
