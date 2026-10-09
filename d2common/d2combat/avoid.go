package d2combat

// AvoidOutcome is the value returned by the game's dodge/avoid/evade roll
// (COMBAT_RollDodgeAvoidEvade, 0x57bd60).
type AvoidOutcome uint8

// Raw outcome codes as returned by the game.
const (
	AvoidNone     AvoidOutcome = 0
	AvoidAvoided  AvoidOutcome = 2    // passive_avoid, vs missiles
	AvoidDodged   AvoidOutcome = 4    // passive_dodge, vs melee
	AvoidEvaded   AvoidOutcome = 8    // passive_evade, only while moving
	AvoidMonBlock AvoidOutcome = 0x10 // animation block
)

// AvoidInput holds the inputs of the dodge/avoid/evade chain.
type AvoidInput struct {
	// Moving: players in modes 3 or 2, monsters in modes 15 or 2.
	Moving bool
	// EvadeChance is stat 0x154, DodgeChance 0x152, AvoidChance 0x153.
	EvadeChance, DodgeChance, AvoidChance int
	// AnimBlockChance is the value of 0x57bcb0 (VERIFIED: NOT a monster stat:
	// the largest value among the entries of the layered stat 0x15c whose item
	// type matches the weapon in either hand, i.e. the Weapon Block passive;
	// 0 without a match) and AnimIsBlock says the unit's current animation
	// action is 0xd (block). Both must hold to roll. Applies to players and
	// monsters alike (the same code path); evade uses mode 3 or 2 for players
	// and 15 or 2 for monsters (VERIFIED), returns code 8 and has no result bit.
	AnimBlockChance int
	AnimIsBlock     bool
	// IsMissile selects avoid (true) over dodge (false).
	IsMissile bool
}

// RollAvoid runs the chain exactly as the binary does:
//
//  1. Moving: only evade is rolled (if EvadeChance > 0, one step); a failed
//     roll or no evade ends the chain, dodge/avoid are NOT tried.
//  2. Not moving: animation block (chance > 0 and animation is block, one
//     step), then dodge (melee) or avoid (missile), each only if the chance
//     is > 0. Every roll is Roll(100) < chance.
//
// NOTE (binary): the notes describe evade, block, dodge and avoid as a flat
// list; the decompile shows evade replaces the rest while moving.
func RollAvoid(r Roller, in AvoidInput) AvoidOutcome {
	if in.Moving {
		if in.EvadeChance < 1 {
			return AvoidNone
		}

		if ok, _ := roll100(r, in.EvadeChance); ok {
			return AvoidEvaded
		}

		return AvoidNone
	}

	if in.AnimBlockChance > 0 && in.AnimIsBlock {
		if ok, _ := roll100(r, in.AnimBlockChance); ok {
			return AvoidMonBlock
		}
	}

	if in.IsMissile {
		if in.AvoidChance > 0 {
			if ok, _ := roll100(r, in.AvoidChance); ok {
				return AvoidAvoided
			}
		}

		return AvoidNone
	}

	if in.DodgeChance > 0 {
		if ok, _ := roll100(r, in.DodgeChance); ok {
			return AvoidDodged
		}
	}

	return AvoidNone
}
