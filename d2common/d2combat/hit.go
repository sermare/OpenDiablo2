package d2combat

// This file models the parts of hit resolution that were confirmed in the
// exe by the verify-hit-resolution pass (observations only, addresses refer to
// Game.exe 1.14):
//
//	0x579ef0  ApplyResistsToDamageStruct: reads stats 34/35, runs per type
//	0x579c90  ApplyResistToDamageType: flat, percent, absorb (per type)
//	0x57b4b0  resolves the queued hit: ApplyDamageToUnit (no resists, they were
//	          done at queue time), then item events 7 (attacker) and 3 (defender)
//	0x5bdbf0  item event func 16 (stat 136 crushing blow)
//	0x5bda80  item event func 15 (stat 135 open wounds)
//	0x57b8b0  AR/DEF operands from stats 0x73, 0x74, 0x7b, 0x7c
//	0x57ae50  reaction (hit recovery, block recovery, death, ...) selection

// DamageType indexes the 12-entry descriptor table at 0x72ff38 for the types
// that carry a flat-reduction slot.
type DamageType uint8

// Damage types handled by ReduceComponent.
const (
	TypePhysical DamageType = iota
	TypeFire
	TypeLightning
	TypeCold
	TypeMagic
	TypePoison
)

// FlatReduction picks the flat reduction (whole points, before the <<8) that
// applies to a damage type. VERIFIED from the descriptor table: physical uses
// slot 1 (stat 34, damage_reduction), fire, lightning, cold and magic use
// slot 2 (stat 35, magic_damage_reduction), poison and the length types use
// slot 0 (always 0).
func FlatReduction(t DamageType, physReduce, magicReduce int) int {
	switch t {
	case TypePhysical:
		return physReduce
	case TypeFire, TypeLightning, TypeCold, TypeMagic:
		return magicReduce
	}

	return 0
}

// ScaleFlatReduction converts a flat reduction stat to 8.8 and scales it as
// 0x579ef0 does at 0x579f74: value<<8, and when positive and the damage
// struct's dword [21] (+0x54) is > 0, MulDiv(value, struct[21], 0x400). Pass
// scale 0 for no scaling. (What fills struct[21] is not located.)
func ScaleFlatReduction(stat int, scale int32) int {
	v := stat << FixedShift
	if v > 0 && scale > 0 {
		v = v * int(scale) / 0x400
	}

	return v
}

// ReduceComponent is COMBAT_ApplyResistToDamageType (0x579c90), VERIFIED:
//
//	if dmg < 1 { return 0 }
//	if ignore { if res > 0 { res = 0 } } else { dmg -= flat }
//	if dmg > 0 && res != 0 { res = min(res, 100); dmg = dmg*(100-res)/100 }
//	if !ignore { dmg, heal = absorb(dmg) }
//
// flat is in the same unit as dmg (8.8, see ScaleFlatReduction). ignore is the
// context flag (damage struct flags 0x100/0x200/0x400 matched against the
// defender's undead/demon class). There is no floor: a flat reduction larger
// than the damage leaves a negative component (the exe stores it as is).
func ReduceComponent(dmg, flat, res int, ignore, hasAbsorb bool, absorbPct, absorbFlat int) (out, heal int) {
	if dmg < 1 {
		return 0, 0
	}

	if ignore {
		if res > 0 {
			res = 0
		}
	} else {
		dmg -= flat
	}

	if dmg > 0 && res != 0 {
		if res > ImmuneResist {
			res = ImmuneResist
		}

		dmg = dmg * (100 - res) / 100
	}

	if ignore {
		return dmg, 0
	}

	return Absorb(dmg, hasAbsorb, absorbPct, absorbFlat)
}

// ApplicableTotal reports whether the summed Total is subtracted from life:
// COMBAT_ApplyDamageToUnit (0x57a650) only subtracts when Total > 0, so
// negative components simply offset the others and never heal.
func ApplicableTotal(total int32) bool { return total > 0 }

// ---- Crushing blow (stat 136, item event func 16, VERIFIED 0x5bdbf0) ----

// CrushingBlowDefender classifies the defender for the divisor.
type CrushingBlowDefender uint8

// Defender classes of the crushing blow divisor.
const (
	// CBNormalMonster: a regular monster (divisor 4, scaled by player count).
	CBNormalMonster CrushingBlowDefender = iota
	// CBBossMonster: monstats flag byte +0xc bit 0x40 (boss) or monster data
	// flag word (+0x16) bit 2 (helpers 0x63fa40 / 0x59dd60): divisor 8, still
	// scaled by player count.
	CBBossMonster
	// CBSpecial: player, or a monster that helper 0x63fed0 reports as special
	// (class ids 0x10f, 0x152, 0x167, 0x230, 0x231): divisor 10, no scaling.
	CBSpecial
	// CBOther: any other unit type (objects): divisor 4, no scaling.
	CBOther
)

// PlayerCountBonusPercent is helper 0x571720 (VERIFIED, table at 0x6e2888 +
// formula): players<9 index [0 0 50 100 150 200 250 300 350], otherwise
// (n-2)*50. The input is stat 100 (monster_playercount) clamped to >= 1.
func PlayerCountBonusPercent(players int) int {
	if players < 1 {
		players = 1
	}

	if players < 9 {
		return [9]int{0, 0, 50, 100, 150, 200, 250, 300, 350}[players]
	}

	return (players - 2) * 50
}

// CrushingBlowDivisor returns the divisor applied to the defender's CURRENT
// life. missile is true for the missile event (event 6), which doubles it.
func CrushingBlowDivisor(def CrushingBlowDefender, players int, missile bool) int {
	d := 4

	switch def {
	case CBSpecial:
		d = 10
	case CBBossMonster:
		d = 8

		d += d * PlayerCountBonusPercent(players) / 100
	case CBNormalMonster:
		d += d * PlayerCountBonusPercent(players) / 100
	}

	if missile {
		d *= 2
	}

	return d
}

// CrushingBlowInput is one crushing blow attempt. The callback runs on the
// ATTACKER's event list, after the base damage was already subtracted, so
// DefenderLife is the life left after that damage (8.8).
type CrushingBlowInput struct {
	Chance       int // attacker stat 136
	DefenderLife int // 8.8, after the hit
	Divisor      int // CrushingBlowDivisor
	// DefenderPhysResist is the defender's RAW stat 36 (no pierce, cap or
	// difficulty), clamped to 100 and ignored when <= 0.
	DefenderPhysResist int
}

// CrushingBlowResult is the outcome of RollCrushingBlow.
type CrushingBlowResult struct {
	Triggered bool
	Removed   int // 8.8 life removed
	NewLife   int // 8.8
	Killed    bool
}

// RollCrushingBlow rolls Roll(100) < chance on the attacker's generator (one
// step when chance > 0, none otherwise) and, on success, removes
// life/divisor (truncating toward zero), reduced by the defender's physical
// resist, from the defender's current life, never below 0. VERIFIED: result bit
// 2 (will die) is set on the damage struct when the new life is <= 0.
func RollCrushingBlow(r Roller, in CrushingBlowInput) CrushingBlowResult {
	res := CrushingBlowResult{NewLife: in.DefenderLife}

	if in.Chance <= 0 {
		return res
	}

	if ok, _ := roll100(r, in.Chance); !ok {
		return res
	}

	res.Triggered = true

	cb := in.DefenderLife / in.Divisor

	pr := in.DefenderPhysResist
	if pr > ImmuneResist {
		pr = ImmuneResist
	}

	if pr > 0 {
		cb -= cb * pr / 100
	}

	life := in.DefenderLife - cb
	if life <= 0 {
		life = 0
		res.Killed = true
	}

	res.Removed = cb
	res.NewLife = life

	return res
}

// ---- Open wounds (stat 135, item event func 15, VERIFIED 0x5bda80) ----

// Open wounds timed state constants (VERIFIED from the stat-list setup at
// 0x5bdb9a): state 0x3e on the defender for 200 frames carrying stat 0x4a
// (hp regen) with a NEGATIVE value, skill 0, skill level 1.
const (
	OpenWoundsState    = 0x3e
	OpenWoundsStat     = 0x4a
	OpenWoundsDuration = 200
	OpenWoundsBase     = 40 // 0x28 added to the level term
)

// openWoundsStep is the table passed to helper 0x5bd9c0: 9, 18, 27, 36, 45.
var openWoundsStep = [5]int{9, 18, 27, 36, 45}

// OpenWoundsLevelTerm is helper 0x5bd9c0 (VERIFIED): a piecewise linear
// function of the attacker level L (stat 12, min 1). It is 0 for L <= 1, else
// the sum of (levels in each 15-level band) times that band's step, with the
// first band (L 2..15) at step 9 and steps 18, 27, 36, 45 for L 16-30, 31-45,
// 46-60 and above 60. The binary writes it as 14*s0 + 15*(s1+..) + extra.
func OpenWoundsLevelTerm(level int) int {
	if level <= 1 {
		return 0
	}

	total := 0
	prev := 1

	for i, upper := range [5]int{15, 30, 45, 60, 1 << 30} {
		if level <= prev {
			break
		}

		hi := level
		if hi > upper {
			hi = upper
		}

		total += (hi - prev) * openWoundsStep[i]
		prev = hi
	}

	return total
}

// OpenWoundsValue returns the (positive) magnitude stored, negated, as the
// hp-regen value. level is the attacker's level. defenderIsPlayer divides by 4
// (truncating toward zero); missile (event 6) then halves it again for a player
// defender. A monster defender is halved when its monster data flag word
// (+0x16) has bit 4 or 8 (mask 0xc, helper 0x59dd60), for melee and missile
// alike. The unit of the regen stat per tick is not verified.
func OpenWoundsValue(level int, defenderIsPlayer, defenderIsMonster, monsterHalved, missile bool) int {
	v := OpenWoundsLevelTerm(level) + OpenWoundsBase

	switch {
	case defenderIsPlayer:
		v /= 4

		if missile {
			v /= 2
		}
	case defenderIsMonster && monsterHalved:
		v /= 2
	}

	return v
}

// OpenWoundsEffect is the timed state a successful roll creates.
type OpenWoundsEffect struct {
	Triggered bool
	RegenStat int // value of stat 0x4a, negative
	Expires   int // frame the state ends
}

// RollOpenWounds rolls Roll(100) < chance on the attacker's generator (no step
// when chance <= 0). On success it returns the effect to attach at frame now.
// Re-applying with the same skill and level only refreshes the expiry (the
// value does not stack), which ApplyOpenWounds models.
func RollOpenWounds(r Roller, chance, value, now int) OpenWoundsEffect {
	if chance <= 0 {
		return OpenWoundsEffect{}
	}

	if ok, _ := roll100(r, chance); !ok {
		return OpenWoundsEffect{}
	}

	return OpenWoundsEffect{Triggered: true, RegenStat: -value, Expires: now + OpenWoundsDuration}
}

// ApplyOpenWounds merges a new effect into an existing one on the same unit
// (VERIFIED in 0x56c740: same state, skill and level just refreshes the expiry).
func ApplyOpenWounds(cur, add OpenWoundsEffect) OpenWoundsEffect {
	if !add.Triggered {
		return cur
	}

	if !cur.Triggered {
		return add
	}

	cur.Expires = add.Expires

	return cur
}

// ---- AR / DEF operands (0x57b8b0, VERIFIED) ----

// AROperands holds the inputs of the pre-step that adjusts AR and DEF before
// the to-hit formula. It runs only for player attackers.
type AROperands struct {
	IgnoreDefense bool // attacker stat 0x73 != 0
	// DefenderIsPlainMonster: defender is a monster that is NOT protected, i.e.
	// monster data flag (+0x16) bit 0xa clear, boss flag clear and not special
	// (0x63fed0 == 0). Only then stat 0x73 zeroes DEF.
	DefenderIsPlainMonster bool
	TargetACPct            int  // attacker stat 0x74
	HalveTargetAC          bool // defender is a player, boss, flag-2 or special monster
	DemonAR, UndeadAR      int  // attacker stats 0x7b / 0x7c
	DefenderDemon          bool
	DefenderUndead         bool // helper 0x63f9e0 (lUndead|hUndead), the same test as the state 0x2f physical-resist case
}

// AdjustAROperands applies 0x57b8b0 and returns the new AR and DEF:
//
//	if 0x73 && plain monster { DEF = 0 }
//	pct = 0x74 (halved for player/boss/special defenders), clamped to [0,100]
//	DEF -= DEF*pct/100                       (only when 0x74 > 0)
//	if defender is demon  { AR += 0x7b }
//	if defender is undead { AR += 0x7c }
func AdjustAROperands(ar, def int, o AROperands) (newAR, newDef int) {
	if o.IgnoreDefense && o.DefenderIsPlainMonster {
		def = 0
	}

	if pct := o.TargetACPct; pct > 0 {
		if o.HalveTargetAC {
			pct >>= 1
		}

		if pct > 100 {
			pct = 100
		}

		def -= def * pct / 100
	}

	if o.DefenderDemon {
		ar += o.DemonAR
	}

	if o.DefenderUndead {
		ar += o.UndeadAR
	}

	return ar, def
}

// ---- Reaction selection (0x57ae50) ----

// Reaction is what 0x57ae50 asks the defender to do.
type Reaction uint8

// Reactions in the order the player path tests them.
const (
	ReactNone Reaction = iota
	ReactDodge
	ReactAvoid
	ReactEvade
	ReactBlock // block animation (player: rate limited, monster: mode 6)
	ReactDeath
	ReactKnockback // result bit 8, mode 0x13 for players
	ReactHitRecovery
	ReactFlagOnly // room list + unit flag 0x8000, no animation
)

// ReactionInput describes the defender and the result word of the hit.
type ReactionInput struct {
	Result           uint32 // damage struct result word
	DefenderIsPlayer bool
	DefenderHasS36   bool // state 0x36 present: only death toggles state 0x5c
	// Player block recovery: frames since stat 0x5f and item_fasterblockrate.
	FramesSinceBlock int
	FasterBlockRate  int
	// Hit recovery gate, see StaggerSuppressed.
	HasState15        bool
	StaggerSuppressed bool
	// Monster defenders: the knockback bit falls back to hit when the monster
	// has no such mode.
	MonsterHasKnockbackMode bool
	// MonsterNoBlockClass: class ids 0xf3, 0x14d and 0x2c1 (and monsters
	// without a block mode) never play the block animation.
	MonsterNoBlockAnim bool
}

// BlockRecoveryReady reports whether a player's block animation may play again.
// VERIFIED (0x57ae50): it plays iff frames since stat 0x5f is > 15 + fbr/8
// (the division truncates toward zero).
func BlockRecoveryReady(framesSince, fasterBlockRate int) bool {
	return framesSince > fasterBlockRate/8+15
}

// SelectReaction mirrors 0x57ae50 (priority order VERIFIED by reading the
// branches; the animation details of each outcome are not modelled).
//
// Player: dodge 0x80, avoid 0x100, evade 0x200, block (0x10 or 0x8000, unless
// 0x4000, rate limited), death 0x2, knockback 0x8, hit recovery 0x4, then
// 0x4000 as flag only.
// Monster: death, knockback (0x8, converted to 0x4 when the monster lacks the
// mode), block 0x10, hit recovery 0x4, 0x4000 as flag only.
func SelectReaction(in ReactionInput) Reaction {
	r := in.Result

	if in.DefenderHasS36 {
		if r&0x2 != 0 {
			return ReactDeath
		}

		return ReactNone
	}

	if in.DefenderIsPlayer {
		switch {
		case r&0x80 != 0:
			return ReactDodge
		case r&0x100 != 0:
			return ReactAvoid
		case r&0x200 != 0:
			return ReactEvade
		case r&0x8010 != 0:
			if r&0x4000 != 0 || !BlockRecoveryReady(in.FramesSinceBlock, in.FasterBlockRate) {
				return ReactNone
			}

			return ReactBlock
		case r&0x2 != 0:
			return ReactDeath
		case r&0x8 != 0:
			return ReactKnockback
		case r&0x4 != 0:
			return hitOrFlag(in)
		case r&0x4000 != 0:
			return ReactFlagOnly
		}

		return ReactNone
	}

	if r&0x8 != 0 && !in.MonsterHasKnockbackMode {
		r = r&^0x8 | 0x4
	}

	switch {
	case r&0x2 != 0:
		return ReactDeath
	case r&0x8 != 0:
		return ReactKnockback
	case r&0x10 != 0:
		if r&0x4000 == 0 && !in.MonsterNoBlockAnim {
			return ReactBlock
		}

		return ReactFlagOnly
	case r&0x4 != 0:
		return hitOrFlag(in)
	case r&0x4000 != 0:
		return ReactFlagOnly
	}

	return ReactNone
}

func hitOrFlag(in ReactionInput) Reaction {
	if in.HasState15 || !in.StaggerSuppressed {
		return ReactHitRecovery
	}

	return ReactFlagOnly
}

// StaggerSuppressed is helper 0x57aa60: true means the hit is too small to
// interrupt (no hit recovery). hitClass is damage struct [24]. maxLife and
// total are 8.8. coin1 and coin2 stand for the two RAND_RollSeedPow2Mask
// calls (true = non-zero, i.e. the roll "passed"); the mask width is NOT
// verified, so callers inject them. VERIFIED structure:
//
//	frozen (state 1), or poison-only damage (field [10] == total != 0),
//	or total < 0x100                       -> suppressed
//	D = 8 for classes 2,6,10,11; 0x20 for 4,8; 0x40 for 5; else 0x10
//	maxLife/D > total                      -> suppressed
//	(maxLife/(D/2) <= total || coin1) && (maxLife/(D/4) <= total || coin2)
//	  && (not a monster || monster has hit mode) -> not suppressed
//	otherwise                              -> suppressed
func StaggerSuppressed(frozen bool, poisonDmg, total int32, hitClass int, maxLife int, monsterWithoutHitMode, coin1, coin2 bool) bool {
	if frozen || (poisonDmg != 0 && poisonDmg == total) || total < 0x100 {
		return true
	}

	d := 0x10

	switch hitClass {
	case 2, 6, 10, 11:
		d = 8
	case 4, 8:
		d = 0x20
	case 5:
		d = 0x40
	}

	t := int(total)
	if maxLife/d > t {
		return true
	}

	if (maxLife/(d/2) <= t || coin1) && (maxLife/(d/4) <= t || coin2) && !monsterWithoutHitMode {
		return false
	}

	return true
}
