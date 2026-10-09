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
	// Ignore is the single context flag ctx[5] of 0x579b10 (VERIFIED). It is
	// set when the defender is a monster that is not a mercenary (the helper at
	// 0x63fed0 recognises monster classes 271, 338, 359, 560 and 561 as
	// mercenaries). One flag does three things at once: pierce no longer
	// reduces a resist of 100 or more, the difficulty penalty is skipped and
	// the cap (75 / 50 / 75+max) is skipped. Players and mercenaries have it
	// clear. Callers with a monster defender set Ignore, with NoDifficultyPenalty
	// as the table-driven stat exemption (physical and magic) only.
	Ignore bool
	// ZeroPhysical is the state special case for physical resist (stat 36):
	// the attacker has state 0x2f and the defender is undead (helper 0x63f9e0:
	// monster unit with monstats lUndead or hUndead; not the boss flag); then a positive physical resist counts as 0.
	// VERIFIED at 0x579b10. Only the final positive-resist branch is affected.
	ZeroPhysical bool
}

// StatePhysResistZero is the attacker state (0x2f) tested by the special case.
const StatePhysResistZero = 0x2f

// AbsorbPercentCap is the cap of the absorb-percent stat (0x28), VERIFIED in
// COMBAT_ApplyDamageAbsorb 0x579c20.
const AbsorbPercentCap = 40

// EffectiveResist returns the resist percent actually applied. Verified:
//
//	res = resist
//	if hasPierce && (res < 100 || !ignore) && pierce != 0 { res -= pierce }
//	if !ignore && !noDifficultyPenalty { res += penalty }
//	if res <= 0 { return max(res, -100) }
//	if ignore { return res }
//	cap = 75 (50 for damageresist w/o max stat); with a max stat: min(75+max, 95)
//	res = min(res, cap)
//	if physical && state 0x2f on attacker && undead defender: res = 0
//
// The pierce stat is one per damage type, read from the descriptor table at
// 0x72ff38: only the passive pierce stats 333..336 are consulted here; the
// item stats 305..308 are not read by this function (see pierceOf).
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

	if in.IsPhysical && in.ZeroPhysical {
		return 0
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
		res = limit
	}

	return res
}

// ApplyResist applies a resist percent to one damage component (any unit,
// usually 8.8). Components <= 0 give 0, immunity (>= 100) gives 0, negative
// resist increases damage. VERIFIED (0x579c90 -> MulDiv 0x47f2c0): the
// component is multiplied by (100-res) and divided by 100 with a truncating
// (toward zero) integer division; res is clamped to 100 first, and a resist of
// exactly 0 leaves the value untouched.
func ApplyResist(damage, res int) int {
	return ApplyResistFull(damage, 0, res, false)
}

// ApplyResistFull is the whole COMBAT_ApplyResistToDamageType (0x579c90) for
// one damage component (VERIFIED). flat is the per-type flat reduction (the
// damage-struct array entry) that is subtracted before the percent. When
// unresistable is set (the undead/demon/beast damage flags 0x100/0x200/0x400
// matched the defender) the flat reduction is skipped, a positive resist
// counts as 0 and only a negative resist still amplifies; absorb is also
// skipped then. A component <= 0 gives 0.
func ApplyResistFull(damage, flat, res int, unresistable bool) int {
	if damage <= 0 {
		return 0
	}

	v := damage

	if unresistable {
		if res > 0 {
			res = 0
		}
	} else {
		v = damage - flat
	}

	if v > 0 && res != 0 {
		if res > 99 {
			res = 100
		}

		v = v * (100 - res) / 100
	}

	return v
}

// Absorb applies the absorb stats to a damage component in 8.8 fixed point,
// AFTER the resist (VERIFIED in COMBAT_ApplyDamageAbsorb 0x579c20, called by
// 0x579c90 once the resist was applied and only when the unresistable flag is
// clear). Only damage types whose descriptor has an absorb stat take part
// (fire, lightning, cold, magic; hasAbsorb). The percent is capped at 40, a
// percent > 0 absorbs damage*pct/100 (truncating), then the flat stat (whole
// points, times 256) absorbs at most what is left. Everything absorbed is
// returned as heal. A type without an absorb stat passes through unchanged.
func Absorb(damage int, hasAbsorb bool, absorbPct, absorbFlat int) (remaining, heal int) {
	if !hasAbsorb {
		return damage, 0
	}

	if absorbPct > AbsorbPercentCap {
		absorbPct = AbsorbPercentCap
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

// LoDResistPenalty is the resistance penalty of a difficulty in Lord of
// Destruction (DifficultyLevels.txt ResistPenalty): 0, -40, -100. The values
// are the table's documented ones (the notes read them from the table at run
// time and name -40/-100).
func LoDResistPenalty(difficulty int) int {
	switch difficulty {
	case 1:
		return -40
	case 2:
		return -100
	}

	return 0
}
