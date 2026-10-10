package d2combat

// Damage is the 0x70-byte damage struct that every skill function memsets
// to zero and fills (SUnitDmg). Values are 8.8 fixed point unless noted;
// field names are chosen here, the layout comes from the notes and the
// decompile of COMBAT_BuildAttackerDamage (0x5794e0). The Go struct has the
// same size and field order as the C one (28 dwords).
type Damage struct {
	Flags        uint32   // [0]  DamageFlag*
	Result       uint32   // [1]  low 16 bits are the Result* word
	Physical     int32    // [2]
	DamagePct    int32    // [3]  percent / demon-bonus accumulator
	Fire         int32    // [4]
	Burn         int32    // [5]  per tick
	BurnLen      int32    // [6]
	Lightning    int32    // [7]
	Magic        int32    // [8]
	Cold         int32    // [9]
	Poison       int32    // [10]
	PoisonLen    int32    // [11]
	ColdLen      int32    // [12]
	FreezeLen    int32    // [13]
	LifeLeech    int32    // [14] (+0x38)
	ManaLeech    int32    // [15]
	StaminaLeech int32    // [16]
	StunLen      int32    // [17]
	Heal         int32    // [18] (+0x48) life gained from absorb/leech
	Total        int32    // [19] (+0x4c) total after resists
	Unknown20    [4]int32 // [20..23] not located
	HitClass     int32    // [24] (+0x60)
	ForcedClass  int32    // [25] (+0x64, only the low byte is used)
	Unknown26    [2]int32 // [26..27]
}

// DamageSize is the size of the C struct in bytes.
const DamageSize = 0x70

// Damage.Flags bits (notes; partly inferred).
const (
	DamageFlagNoPhysical   uint32 = 0x1 // skip the physical roll (checked before deadly strike)
	DamageFlagNoLifeLeech  uint32 = 0x4
	DamageFlagNoManaLeech  uint32 = 0x8
	DamageFlagNoStamLeech  uint32 = 0x10
	DamageFlagComputed     uint32 = 0x20 // set at the start of BuildAttackerDamage
	DamageFlagIgnoreUndead uint32 = 0x100
	DamageFlagIgnoreDemon  uint32 = 0x200
	DamageFlagIgnoreBeast  uint32 = 0x400
)

// Damage.Result bits (notes).
const (
	ResultHit      uint32 = 0x1
	ResultDied     uint32 = 0x2
	ResultBlocked  uint32 = 0x10
	ResultDodged   uint32 = 0x80
	ResultAvoided  uint32 = 0x100
	ResultHitReact uint32 = 0x4    // landed on a unit without state 0x36 (verified in 0x57cc10)
	ResultCritical uint32 = 0x2000 // deadly strike / critical strike applied
	ResultMonBlock uint32 = 0x8000
)

// SumTotal adds the damage components the game sums into Total: physical,
// fire, lightning, magic, cold and poison (notes, from 579ef0/57bc00), plus
// the [14] field when the defender is a monster (notes, not re-checked).
func (d *Damage) SumTotal(defenderIsMonster bool) int32 {
	t := d.Physical + d.Fire + d.Lightning + d.Magic + d.Cold + d.Poison
	if defenderIsMonster {
		t += d.LifeLeech
	}

	return t
}

// ScaleBySrcDam scales a value by a skill's SrcDam byte (128 == 100%), as
// BuildAttackerDamage does for elemental, leech, length and stun values:
// value*scale/128 truncating toward zero. A scale of 0 means 128. Verified.
func ScaleBySrcDam(value int32, scale uint8) int32 {
	if scale == 0 {
		scale = 0x80
	}

	return value * int32(scale) / 128
}

// StrikeInput holds the chances (percent) that can double physical damage.
type StrikeInput struct {
	// WeaponChance is the weapon-based deadly strike (646bc0 mode 2); it is
	// only rolled when SkipWeapon is false (callers pass 0 without a weapon).
	WeaponChance int
	SkipWeapon   bool
	// CriticalChance is passive_critical_strike (stat 0x151).
	CriticalChance int
	// DeadlyChance is item_deadlystrike (stat 0x8d).
	DeadlyChance int
}

// RollStrike rolls deadly/critical strike. Verified in 0x5794e0: three
// independent chances evaluated in this order with short-circuit OR, each
// rolled only if > 0, each hit iff Roll(100) < chance. Returns true when the
// physical damage must double. It consumes 0 to 3 steps. (That the roll is
// Roll(100) from a zero base is inferred from the helper's name.)
func RollStrike(r Roller, in StrikeInput) bool {
	if !in.SkipWeapon && in.WeaponChance > 0 {
		if ok, _ := roll100(r, in.WeaponChance); ok {
			return true
		}
	}

	if in.CriticalChance > 0 {
		if ok, _ := roll100(r, in.CriticalChance); ok {
			return true
		}
	}

	if in.DeadlyChance > 0 {
		if ok, _ := roll100(r, in.DeadlyChance); ok {
			return true
		}
	}

	return false
}

// RollMissileStrike is the missile-side crit helper (FUN_0064ba70, called from
// MISSILE_BuildDamageDescriptor 0x64cce0 / 0x64cf2a; verified): the order is
// passive_critical_strike (0x151), item_deadlystrike (0x8d), then the weapon
// mastery (646bc0 mode 2, only with a weapon: pass 0 or SkipWeapon without
// one); the first success wins. Each chance is rolled only when > 0.
// DEVIATION (no gameplay effect): the exe rolls the 0x151 step even when the
// chance is 0 (no test before the roll at 0x64ba7d..0x64baae), consuming one
// random step per missile; this port does not, so a hero without crit stats
// keeps an identical random stream.
func RollMissileStrike(r Roller, in StrikeInput) bool {
	if r == nil {
		return false
	}

	if in.CriticalChance > 0 {
		if ok, _ := roll100(r, in.CriticalChance); ok {
			return true
		}
	}

	if in.DeadlyChance > 0 {
		if ok, _ := roll100(r, in.DeadlyChance); ok {
			return true
		}
	}

	if !in.SkipWeapon && in.WeaponChance > 0 {
		if ok, _ := roll100(r, in.WeaponChance); ok {
			return true
		}
	}

	return false
}

// ApplyStrike rolls a strike and, on success, doubles the physical damage and
// sets ResultCritical. Nothing happens (and no step is consumed) when
// DamageFlagNoPhysical is set. Critical and deadly strike both double and do
// not stack beyond x2. Reports whether it doubled.
func (d *Damage) ApplyStrike(r Roller, in StrikeInput) bool {
	if d.Flags&DamageFlagNoPhysical != 0 {
		return false
	}

	if !RollStrike(r, in) {
		return false
	}

	d.Result |= ResultCritical
	d.Physical *= 2

	return true
}

// RollMonsterDouble implements COMBAT_RollMonsterCriticalDouble (0x5a2f90,
// verified): if the monster's monstats byte +0xa6 (chance) is non-zero, roll
// Roll(100) < chance; on success physical, fire, lightning, magic, cold and
// poison damage all double. No step is consumed for chance 0.
func (d *Damage) RollMonsterDouble(r Roller, chance uint8) bool {
	if chance == 0 {
		return false
	}

	if ok, _ := roll100(r, int(chance)); !ok {
		return false
	}

	d.Physical *= 2
	d.Fire *= 2
	d.Lightning *= 2
	d.Magic *= 2
	d.Cold *= 2
	d.Poison *= 2

	return true
}

// Damage scale percentages from the pre-step 0x579d70 of
// COMBAT_ApplyResistsToDamageStruct.
const (
	ScaleNormal               = 100
	ScalePlayerOnPlayer       = 17 // 0x11
	ScaleBothPlayerControlled = 25 // 0x19
)

// CombatantScalePercent returns the damage percent applied before resists.
//
// NOTE (binary): the notes call 25 the player-vs-player constant. The decompile
// of 0x579d70 shows: when the DEFENDER is a player (unit type 0) the result is
// 17 (0x11), or 100 for a plain monster attacker; 25 (0x19) is returned only
// when the defender is NOT a player and both units are "player controlled"
// (helper 0x63fed0, meaning inferred: player, mercenary or pet). This
// function models those branches; the hireling-difficulty (63fa40) and
// monster-vs-class (x4/x2) branches are not modelled. The helper 0x44d580
// check in the monster-attacker branch is ignored (inferred).
func CombatantScalePercent(sameUnit, attackerIsPlayer, defenderIsPlayer, attackerPlayerControlled, defenderPlayerControlled bool) int {
	if sameUnit {
		return ScaleNormal
	}

	if defenderIsPlayer {
		if !attackerIsPlayer && !attackerPlayerControlled {
			return ScaleNormal
		}

		return ScalePlayerOnPlayer
	}

	if attackerPlayerControlled && defenderPlayerControlled {
		return ScaleBothPlayerControlled
	}

	return ScaleNormal
}

// ScaleByPercent applies pct to a damage component: v*pct/100.
func ScaleByPercent(v int32, pct int) int32 {
	return v * int32(pct) / 100
}
