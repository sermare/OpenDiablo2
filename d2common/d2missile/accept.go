package d2missile

// Target acceptance of the missile collision scan, verified against the
// predicates of Game.exe in the emulator (golden collide_golden.json). The sim
// applies the same rules inline in Sim.process and testUnits; this pure form is
// the oracle-checked statement of them. Observations only.

// Addresses of the per CollideType predicates (table 0x739948).
const (
	predMonsters  = 0x5a61d0 // CollideType 2 and 5: monsters only
	predPlayers   = 0x5a6210 // CollideType 1: players (and aligned monsters)
	predUnitTypes = 0x5a6270 // CollideType 3 and 8: players and monsters
)

// AcceptIn describes one candidate target of a moving missile.
type AcceptIn struct {
	TargetFlags   uint32 // unit flags (+0xc4): bits 3 and 2 must both be set
	TargetKind    int    // unit type: 0 player, 1 monster
	NextHit       bool   // missile record NextHit (+0x188)
	TargetState56 bool   // target carries the NextHit state 0x56
	LastHit       bool   // 0x64b700: this target was the last one hit by the missile
	OwnerPresent  bool
	OwnerEnemy    bool // 0x552270(owner, target)
	CollideFriend bool // missile record CollideFriend (+0x187)
	Aligned       bool // 0x625c10(target) == 2 (CollideType 1 only)
}

// AcceptCommon is the common predicate 0x5a6160: the target must be flagged
// (bits 3 and 2), not under state 0x56 when the missile has NextHit, not the
// last unit hit, and, when the missile has an owner, an enemy of it or the
// missile must have CollideFriend. A missile without an owner takes any
// flagged target.
func AcceptCommon(in AcceptIn) bool {
	if in.TargetFlags&8 == 0 || in.TargetFlags&4 == 0 {
		return false
	}

	if in.NextHit && in.TargetState56 {
		return false
	}

	if in.LastHit {
		return false
	}

	if in.OwnerPresent && !in.OwnerEnemy && !in.CollideFriend {
		return false
	}

	return true
}

// AcceptTyped is the predicate of a CollideType: pred is one of the predMonsters
// (types 2, 5), predPlayers (type 1) and predUnitTypes (types 3, 8) addresses.
func AcceptTyped(pred int, in AcceptIn) bool {
	switch pred {
	case predMonsters:
		if in.TargetKind != 1 {
			return false
		}
	case predUnitTypes:
		if in.TargetKind != 0 && in.TargetKind != 1 {
			return false
		}
	case predPlayers:
		if in.TargetKind != 0 && !(in.TargetKind == 1 && in.Aligned) {
			return false
		}
	}

	return AcceptCommon(in)
}
