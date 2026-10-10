package d2animspeed

// This file models the whole mode dispatch of the exe's
// PATH_UpdateUnitVelocityAndAnimRate (0x624150) as observed by running the
// real function in the emulator oracle (testdata/anim_golden.json, test
// oracle_test.go). Everything here is VERIFIED against that oracle for fake
// units whose stats and MonStats fields were supplied by the harness, except
// where marked UNVERIFIED.

// UnitKind is the unit type field of the exe (player 0, monster 1).
type UnitKind int

// Unit kinds that have animation rates.
const (
	KindPlayer  UnitKind = 0
	KindMonster UnitKind = 1
)

// ModeRule is the rate rule an animation mode falls under.
type ModeRule int

// Actions of the mode dispatch.
const (
	// RuleKeep plays at the AnimData speed unchanged (player DT and DD, monster
	// DT, DD and the sequence mode 16).
	RuleKeep ModeRule = iota
	// RuleWalk is walk or run: pct = max(25, stat 0x43 + diminished FRW), applied
	// to a base rate and a base velocity.
	RuleWalk
	// RuleHit is hit recovery (FHR), RuleBlock is block recovery (FBR).
	RuleHit
	RuleBlock
	// RuleCast is spell cast (FCR).
	RuleCast
	// RuleAttack is a weapon attack (IAS).
	RuleAttack
	// RuleKnock is knock back: a fixed rate and velocity.
	RuleKnock
	// RuleOther is every other mode: stat 0x45 clamped to 15..175.
	RuleOther
)

// Fixed values the exe uses for players (VERIFIED by the oracle; the rate bases
// do not come from AnimData and the velocity base is the same for walk and run).
const (
	// PlayerWalkBaseRate is the base rate of player walk (modes 2 and 6).
	PlayerWalkBaseRate = 213
	// PlayerRunBaseRate is the base rate of player run (mode 3).
	PlayerRunBaseRate = 101
	// PlayerWalkVelocity is the 8.8 base path velocity of player walk and run.
	PlayerWalkVelocity = 6 << 8
	// KnockVelocity is the 8.8 path velocity set for knock back (both kinds).
	KnockVelocity = 16 << 8
	// PlayerKnockRate is the fixed rate of player knock back.
	PlayerKnockRate = 213
)

// ModeAction maps a unit kind and animation mode to its rate rule. Player modes
// follow the PlrMode order (DT NU WL RN GH TN TW A1 A2 BL SC TH KK S1 S2 S3 S4
// DD SQ KB); monster modes the MonMode order (DT NU WL GH A1 A2 BL SC S1 S2 S3
// S4 DD KB SQ RN). Modes above these ranges are RuleOther.
//
// Monster skill modes 8..11 become RuleWalk when MonStats flag bit 0 is clear
// (see UnitInputs.MonFlagBit0); UnitRate applies that, ModeAction does not.
//
// UNVERIFIED: a player in the sequence mode 18 casts (RuleCast) instead when
// the oracle's weapon-type probe says so; the oracle does not model that probe.
func ModeAction(k UnitKind, mode int) ModeRule {
	if k == KindPlayer {
		switch mode {
		case 0, 17:
			return RuleKeep
		case 2, 3, 6:
			return RuleWalk
		case 4:
			return RuleHit
		case 9:
			return RuleBlock
		case 10:
			return RuleCast
		case 7, 8, 11, 12:
			return RuleAttack
		case 19:
			return RuleKnock
		}

		return RuleOther
	}

	switch mode {
	case 0, 12, 16:
		return RuleKeep
	case 2, 15:
		return RuleWalk
	case 3:
		return RuleHit
	case 6:
		return RuleBlock
	case 7:
		return RuleCast
	case 4, 5:
		return RuleAttack
	case 13:
		return RuleKnock
	}

	return RuleOther
}

// UnitInputs are the stats and table values the dispatch reads.
type UnitInputs struct {
	// Speed is the AnimData speed of the unit's current animation record.
	Speed int
	// IAS, FHR, FCR, FBR and FRW are the item stats 93, 99, 105, 102 and 96.
	IAS, FHR, FCR, FBR, FRW int
	// Stat43, Stat44 and Stat45 are the walk, attack and other percent stats.
	Stat43, Stat44, Stat45 int
	// MonWalkRate, MonRunRate and MonVelocity are the MonStats fields at record
	// offsets 0x36, 0x38 and 0x32 (monsters only). UNVERIFIED: which monstats.txt
	// columns those are (rate bases per whole number, velocity in whole units).
	MonWalkRate, MonRunRate, MonVelocity int
	// MonFlagBit0 is bit 0 of the MonStats byte at record offset 0xd (masked
	// with the exe global 0x6cf250 = 1). UNVERIFIED meaning. When it is clear
	// the skill modes S1..S4 of a monster move it like a walk (ModeRuleWalk with
	// the walk rate and velocity); when set only walk and run do.
	MonFlagBit0 bool
}

// UnitRate returns the animation rate (8.8) of a unit in a mode, and for walk,
// run and knock back the path velocity (hasVel true).
func UnitRate(k UnitKind, mode int, in UnitInputs) (rate, vel int, hasVel bool) {
	pct := func() int {
		p := in.Stat43 + Diminish(in.FRW, KRunWalk)
		if p < MinWalkPct {
			p = MinWalkPct
		}

		return p
	}

	rule := ModeAction(k, mode)
	if k == KindMonster && !in.MonFlagBit0 && mode >= 8 && mode <= 11 {
		rule = RuleWalk
	}

	switch rule {
	case RuleKeep:
		return in.Speed, 0, false
	case RuleHit:
		return HitRate(in.Speed, in.FHR), 0, false
	case RuleBlock:
		return BlockRate(in.Speed, in.FBR, false), 0, false
	case RuleCast:
		return CastRate(in.Speed, in.FCR), 0, false
	case RuleAttack:
		return AttackRate(in.Speed, in.Stat44, in.IAS, 0, false), 0, false
	case RuleKnock:
		if k == KindPlayer {
			return PlayerKnockRate, KnockVelocity, true
		}

		return capRate(in.MonWalkRate), KnockVelocity, true
	case RuleWalk:
		p := pct()

		if k == KindPlayer {
			base := PlayerWalkBaseRate
			if mode == 3 {
				base = PlayerRunBaseRate
			}

			return capRate(base * p / 100), PlayerWalkVelocity * p / 100, true
		}

		base := in.MonWalkRate
		if mode == 15 {
			base = in.MonRunRate
		}

		return capRate(base * p / 100), (in.MonVelocity << 8) * p / 100, true
	}

	return OtherRate(in.Speed, in.Stat45), 0, false
}

func capRate(r int) int {
	if r < 0 {
		return 0
	}

	if r > MaxRate {
		return MaxRate
	}

	return r
}
