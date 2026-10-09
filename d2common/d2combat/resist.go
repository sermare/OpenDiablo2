package d2combat

// Resistance limits (verified, COMBAT_GetEffectiveResist 0x579b10).
const (
	DefaultMaxResist        = 75   // base cap
	MaxResistCeiling        = 95   // 75 + bonus is capped at 95
	DamageResistCap         = 50   // physical (stat 36) has no max stat; cap 50
	MinResist               = -100 // floor
	ImmuneResist            = 100
	ClassicNightmarePenalty = -20
	ClassicHellPenalty      = -50
)

// ClassicResistPenalty returns the classic-mode difficulty penalty for a
// difficulty byte (0 normal, 1 nightmare, 2 hell). LoD reads the penalty
// from DifficultyLevels.txt instead (-40/-100) and passes it directly.
func ClassicResistPenalty(difficulty int) int {
	switch difficulty {
	case 1:
		return ClassicNightmarePenalty
	case 2:
		return ClassicHellPenalty
	}

	return 0
}

// ResistInput describes one damage type for EffectiveResist.
type ResistInput struct {
	Resist int // defender's resist stat for the type
	// Pierce is the attacker's pierce stat; HasPierce says the type has one
	// (descriptor pierceStat != -1).
	Pierce    int
	HasPierce bool
	// MaxResistBonus is the defender's max-resist stat; HasMaxResist says the
	// type has one (descriptor maxResistStat != -1).
	MaxResistBonus int
	HasMaxResist   bool
	// IsPhysical: the type is damageresist (stat 36): cap 50 without a max stat.
	IsPhysical bool
	// NoDifficultyPenalty: damageresist (36) and magicresist (37) are exempt.
	NoDifficultyPenalty bool
	// DifficultyPenalty is the signed value added after pierce (e.g. -40,
	// -100, or ClassicResistPenalty).
	DifficultyPenalty int
	// Ignore is the context flag that disables pierce on immunities, the
	// difficulty penalty and the cap altogether (ctx[5]).
	Ignore bool
}

// EffectiveResist returns the resist percent actually applied. Verified:
//
//	res = resist
//	if hasPierce && (res < 100 || !ignore) && pierce != 0 { res -= pierce }
//	if !ignore && !noDifficultyPenalty { res += penalty }
//	if res <= 0 { return max(res, -100) }
//	if ignore { return res }
//	cap = 75 (50 for damageresist w/o max stat); with a max stat: min(75+max, 95)
//	return min(res, cap)
//
// Not modelled: the physical-resist special case of state 0x2f (returns 0).
func EffectiveResist(in ResistInput) int {
	res := in.Resist

	if in.HasPierce && (res < ImmuneResist || !in.Ignore) && in.Pierce != 0 {
		res -= in.Pierce
	}

	if !in.Ignore && !in.NoDifficultyPenalty {
		res += in.DifficultyPenalty
	}

	if res <= 0 {
		if res < MinResist {
			return MinResist
		}

		return res
	}

	if in.Ignore {
		return res
	}

	limit := DefaultMaxResist

	switch {
	case in.HasMaxResist:
		limit = in.MaxResistBonus + DefaultMaxResist
		if limit > MaxResistCeiling {
			limit = MaxResistCeiling
		}
	case in.IsPhysical:
		limit = DamageResistCap
	}

	if res > limit {
		return limit
	}

	return res
}

// ApplyResist applies a resist percent to one damage component (any unit,
// usually 8.8). Components <= 0 give 0, immunity (>= 100) gives 0, negative
// resist increases damage. The exact MulDiv operands are hidden in the
// decompile of 0x579c90, so the formula damage*(100-res)/100 is inferred.
func ApplyResist(damage, res int) int {
	if damage <= 0 || res >= ImmuneResist {
		return 0
	}

	return damage * (100 - res) / 100
}

// Absorb applies the absorb stats to a damage component in 8.8 fixed point.
// Verified in COMBAT_ApplyDamageAbsorb (0x579c20): if the type has an absorb
// stat, absorbPct > 0 absorbs damage*pct/100 (inferred MulDiv operands),
// then absorbFlat*256 more (capped at what is left). Everything absorbed is
// returned as heal. A type without an absorb stat passes through unchanged.
func Absorb(damage int, hasAbsorb bool, absorbPct, absorbFlat int) (remaining, heal int) {
	if !hasAbsorb {
		return damage, 0
	}

	if absorbPct > 0 {
		x := damage * absorbPct / 100
		heal += x
		damage -= x
	}

	if flat := absorbFlat * 256; flat > 0 {
		if flat > damage {
			flat = damage
		}

		heal += flat
		damage -= flat
	}

	return damage, heal
}
