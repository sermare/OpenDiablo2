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
	// AutoHit: VERIFIED in 0x57cc10. When the defender is a player whose mode
	// is 3, the hit bit is set without rolling to-hit (no step consumed).
	AutoHit bool
	// DefenderLacksState36: VERIFIED. A landed hit on a unit without state
	// 0x36 also gets result bit 0x4 (ResultHitReact), which is what makes
	// 0x57ae50 choose the hit-recovery animation.
	DefenderLacksState36 bool
}

// AttackResult is the outcome flag word plus the to-hit percent.
type AttackResult struct {
	Result uint32 // Result* bits; ResultHit set only when the attack lands
	Chance int    // to-hit percent
}

// ResolveAttack runs to-hit, shield block, then dodge/avoid/evade. A miss
// consumes one step and stops. Any block/dodge/avoid clears ResultHit and
// sets the matching flag (blocked 0x10, dodge 0x80, avoid 0x100, monster
// animation block 0x8000). Evade (code 8) has no result bit: VERIFIED in
// 0x57cc10, any non-zero avoid code clears bit 0x1 and only 2, 4, 0x10 and the
// shield-block code 1 map to flag bits, so an evade leaves Result == 0.
func ResolveAttack(r Roller, in AttackInput) AttackResult {
	hit, chance := true, 0

	if !in.AutoHit {
		hit, chance, _ = RollToHit(r, in.ToHit)
	}

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

	if res.Result&ResultHit != 0 && in.DefenderLacksState36 {
		res.Result |= ResultHitReact
	}

	return res
}

// ReduceHit applies the order VERIFIED in COMBAT_ApplyResistToDamageType
// (0x579c90, called per damage type from 0x579ef0 at queue time, not from
// 0x57b4b0): flat reduction first (stat 34 for physical, 35 for fire,
// lightning, cold and magic; none for poison), then the percent resist, then
// absorb. The result is floored at 0 here; the exe does not floor a single
// component, but ApplyDamageToUnit only subtracts when the summed Total is
// above 0, so the floor is equivalent for a single type. See ReduceComponent
// for the full per-type step.
func ReduceHit(dmg, flatReduce, resistPct int) int {
	d := ApplyResist(dmg-flatReduce, resistPct)
	if d < 0 {
		return 0
	}

	return d
}
