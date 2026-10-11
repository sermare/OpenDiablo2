package d2skill

// Client skill start gates (the CLTST_* functions of the exe; notes in gaps-slice-G.md). The exe refuses or
// shapes a cast on the client before the server sees it. These are the pure rules, applied by Pipeline.Start
// when the engine supplies the data they need: zero values never refuse, so a caller that knows nothing is
// unchanged.

// Server do-function ids of the skills gated here (skills.txt srvdofunc).
const (
	doBoneWall  = 60
	doGolemIron = 57
	doRevive    = 58
)

// FendEnemyMask is the enemy filter of the Fend and Zeal target search (CLTST_014, 0x4f1840).
const FendEnemyMask = 0x20003

// FendReachBonus is added to the melee range for the Fend and Zeal search (UNIT_GetMeleeRangePlus4Fend, 0x4c46a0).
const FendReachBonus = 4

// FendReach is the radius of the Fend / Zeal enemy search: melee range plus 4.
func FendReach(meleeRange int) int { return meleeRange + FendReachBonus }

// StrikeCount is the hit count of Fend and Zeal at the start: calc1, capped by the enemies found in reach.
// CLTST_014 stores min(calc1, found); with nothing in reach the count is 0 and the start still succeeds.
func StrikeCount(calc1, found int) int {
	if found < calc1 {
		return found
	}

	return calc1
}

// BoneWallAllowed is CLTST_022 (0x4f13f0): the target room must exist, and a town room is refused unless the
// skill row has bit 0x100 of dwBits4 (allowInTown).
func BoneWallAllowed(roomFound, townRoom, allowInTown bool) bool {
	return roomFound && (allowInTown || !townRoom)
}

// ItemTarget describes the item under the cursor for Iron Golem.
type ItemTarget struct {
	OnGround  bool // mode 3
	GolemItem bool // the item type may become a golem (bitfield test of its txt row)
	FlagOK    bool // ITEM_TestFlags(item, 0x10) (meaning unverified)
	InRoom    bool
	Owned     bool // a stat list owner exists (the item is in someone's inventory)
}

// IronGolemTargetOK is CLTST_023 (0x4f1490): an item on the ground, of a golem type, in a room, not owned.
func IronGolemTargetOK(t ItemTarget) bool {
	return t.GolemItem && t.OnGround && t.FlagOK && t.InRoom && !t.Owned
}

// ReviveTarget describes the corpse under the cursor for Revive.
type ReviveTarget struct {
	IsMonster    bool
	Flag8        bool // monstats2 flag bit 8 (revivable class)
	UsableCorpse bool
	Revivable    bool
}

// ReviveTargetOK is the corpse test of CLTST_024 (0x4f1540).
func ReviveTargetOK(t ReviveTarget) bool {
	return t.IsMonster && t.Flag8 && t.UsableCorpse && t.Revivable
}

// clientGate applies the gates to a cast; it returns the refusal reason and false when refused.
func clientGate(sk *Skill, tgt Target) (string, bool) {
	switch sk.SrvDoFunc {
	case doBoneWall:
		if tgt.TownRoom && !sk.AllowTownRoom {
			return ReasonTown, false
		}
	case doGolemIron:
		if tgt.Item != nil && !IronGolemTargetOK(*tgt.Item) {
			return ReasonTarget, false
		}
	case doRevive:
		if tgt.Revive != nil && !ReviveTargetOK(*tgt.Revive) {
			return ReasonNoCorpse, false
		}
	}

	return "", true
}
