package d2combat

// AttackInput describes one attack for ResolveAttack, which chains the steps
// of COMBAT_RollAttackOutcome (0x57cc10) in the order the notes verified:
// to-hit, then shield block, then (only if not blocked) the avoid chain.
type AttackInput struct {
	ToHit ToHitInput
	// BlockChance is the defender's block percent (0 = cannot block); the
	// shield roll is skipped without consuming a step when it is below 1.
	BlockChance int
	// DefenderMoving divides a player's block chance by 3 (inferred meaning).
	DefenderMoving bool
	Avoid          AvoidInput
}

// AttackResult is the outcome flag word plus the to-hit percent.
type AttackResult struct {
	Result uint32 // Result* bits; ResultHit set only when the attack lands
	Chance int    // to-hit percent
}

// ResolveAttack runs to-hit, shield block, then dodge/avoid/evade. A miss
// consumes one step and stops. Any block/dodge/avoid clears ResultHit and
// sets the matching flag (blocked 0x10, dodge 0x80, avoid 0x100, monster
// animation block 0x8000). Evade has no result bit in the notes' mapping, so
// it only clears the hit (UNVERIFIED).
func ResolveAttack(r Roller, in AttackInput) AttackResult {
	hit, chance, _ := RollToHit(r, in.ToHit)
	res := AttackResult{Chance: chance}

	if !hit {
		return res
	}

	if RollShieldBlock(r, in.BlockChance, in.DefenderMoving) {
		res.Result = ResultBlocked

		return res
	}

	switch RollAvoid(r, in.Avoid) {
	case AvoidAvoided:
		res.Result = ResultAvoided
	case AvoidDodged:
		res.Result = ResultDodged
	case AvoidMonBlock:
		res.Result = ResultMonBlock
	case AvoidEvaded:
		res.Result = 0
	default:
		res.Result = ResultHit
	}

	return res
}

// ReduceHit applies the engine's current order: flat reduction, then the
// physical resist percent, floored at 0. ORDER UNVERIFIED: 0x579ef0 applies
// only percent resists/absorb (checked in the decompile); the flat stats
// 34/35 are not touched there, so they are applied elsewhere (confirm at
// 0x57b4b0).
func ReduceHit(dmg, flatReduce, resistPct int) int {
	d := ApplyResist(dmg-flatReduce, resistPct)
	if d < 0 {
		return 0
	}

	return d
}
