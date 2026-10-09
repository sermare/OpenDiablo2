package d2animspeed

// Action selects which speed rule a breakpoint derivation uses.
type Action int

// Actions with breakpoint tables.
const (
	// ActionHit is hit recovery (FHR), base percent 50.
	ActionHit Action = iota
	// ActionBlock is block recovery (FBR), base percent 50 (no state 0x65).
	ActionBlock
	// ActionCast is spell casting (FCR), base percent 100, capped at 175.
	ActionCast
	// ActionAttack is the attack animation with stat 0x44 = 100 (UNVERIFIED
	// composition) and the item IAS stat; clamp 15..175.
	ActionAttack
)

// MaxTableStat is the largest stat value a derived table lists. The rules
// have no upper limit for hit and block recovery (the rate only saturates at
// 0x7fff), so a cut-off is a presentation choice: the community lists stop
// at 600 (UNVERIFIED convention; it reproduces every community row length).
const MaxTableStat = 600

// ActionRate returns the exe's animation rate for the action with the given
// AnimData speed and speed stat (FHR, FBR, FCR or item IAS).
func ActionRate(act Action, speed, stat int) int {
	switch act {
	case ActionHit:
		return HitRate(speed, stat)
	case ActionBlock:
		return BlockRate(speed, stat, false)
	case ActionCast:
		return CastRate(speed, stat)
	default:
		return AttackRate(speed, AttackBasePct, stat, 0, false)
	}
}

// ActionTicks is the number of 25 Hz ticks the animation takes (0 when the
// rate is zero).
func ActionTicks(act Action, frames, speed, stat int) int {
	t, err := Ticks(frames, ActionRate(act, speed, stat))
	if err != nil {
		return 0
	}

	return t
}

// DeriveTable lists the smallest stat values at which the animation
// gets one tick shorter, starting with 0, for stats up to maxStat
// (MaxTableStat when maxStat <= 0). frames and speed are the AnimData record
// of the animation.
func DeriveTable(act Action, frames, speed, maxStat int) []int {
	if maxStat <= 0 {
		maxStat = MaxTableStat
	}

	return Breakpoints(func(stat int) int { return ActionTicks(act, frames, speed, stat) }, maxStat+1, maxStat)
}

// NextBreakpoint returns the first entry of a breakpoint table above stat.
// ok is false when stat already reaches the last entry.
func NextBreakpoint(table []int, stat int) (next int, ok bool) {
	for _, bp := range table {
		if bp > stat {
			return bp, true
		}
	}

	return 0, false
}
