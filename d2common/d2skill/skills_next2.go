package d2skill

// Skills batch "next 2": the pure rules behind Energy Shield, Revive, Sanctuary
// and the knockback bit, read from Game.exe 1.14b (read-only). Addresses and
// the rules in the d2-re-notes folder, skills-batch-next2.md. VERIFIED marks
// what was read in the decompilation, U what is inferred.

// resultKnock is bit 8 of a hit's result flags: the target is knocked back
// (COMBAT_ServerHandleUnitHit 0x57ae50 sends the unit into the knockback
// mode when it is set). Sanctuary's ResultFlags (11) and Smite's (8) carry it.
const resultKnock = 8

// EnergyShieldAbsorb is the absorb callback of the Energy Shield state
// (aurafunc 24, 0x5c8840, VERIFIED): pct is calc1 (percent of each damage
// type that is absorbed, the state only works when it is above 0), ratio is
// calc2 (at least 1) and mana is the current mana in the same unit as dmg.
// A damage amount loses pct percent, limited by what the mana can pay
// (mana*16/ratio), and the shield pays absorbed*ratio/16 mana; it is gone
// when no mana is left. Returns the absorbed amount and the mana spent.
func EnergyShieldAbsorb(dmg, pct, ratio, mana int) (absorbed, cost int) {
	if dmg <= 0 || pct <= 0 || mana <= 0 {
		return 0, 0
	}

	if ratio < 1 {
		ratio = 1
	}

	absorbed = mulDiv(dmg, pct, 100)
	if limit := mulDiv(mana, 16, ratio); absorbed > limit {
		absorbed = limit
	}

	return absorbed, mulDiv(absorbed, ratio, 16)
}

// EnergyShieldTypes lists, in the order the exe walks them (table at 0x6e4670),
// the damage-struct fields the shield absorbs and whether the entry is skipped
// when the attacker is a player or a hireling: physical, fire, lightning,
// cold and magic always; the three last offsets (0x38, 0x3c, 0x40) only for
// other attackers (U: which damage types those are; the Go struct's names for
// them are life/mana/stamina leech, which cannot be right for an absorb table).
var EnergyShieldTypes = []struct {
	Offset      int
	NotVsPlayer bool
}{{0x08, false}, {0x10, false}, {0x1c, false}, {0x24, false}, {0x20, false}, {0x38, true}, {0x3c, true}, {0x40, true}}

// ReviveLife is the life a revived monster starts with (SRVDO_058 0x5c35a0,
// VERIFIED): a roll between the level-scaled minimum and maximum life of the
// corpse's monster type, evaluated at the corpse's level (stat 0xc) in the
// game's difficulty, lo + roll(hi-lo+1) whole points.
func ReviveLife(lo, hi int, roll func(n int) int) int {
	if hi < lo {
		hi = lo
	}

	v := lo
	if hi > lo && roll != nil {
		v += roll(hi - lo + 1)
	}

	if v < 1 {
		v = 1
	}

	return v
}

// ReviveCapped is the second step of Revive (VERIFIED): a corpse whose level is
// above the caster's comes back at the caster's level, its life scaled by
// caster/corpse (at least 1). Returns the level and life to use.
func ReviveCapped(corpseLevel, casterLevel, life int) (level, hp int) {
	if corpseLevel <= 0 || corpseLevel <= casterLevel {
		return corpseLevel, life
	}

	hp = mulDiv(life, casterLevel, corpseLevel)
	if hp < 1 {
		hp = 1
	}

	return casterLevel, hp
}

// Aura filter bits (SKILL_UnitMatchesTargetFilter 0x569100, VERIFIED) the engine
// tests on the monsters an aura reaches.
const (
	FilterPlayers    = 0x1     // players
	FilterMonsters   = 0x2     // monsters
	FilterUndeadOnly = 0x4     // monsters: only undead
	FilterNoBoss     = 0x4000  // monsters: not a boss
	FilterHostile    = 0x8000  // only units hostile to the caster
	FilterAllied     = 0x10000 // only allied units
)

// FilterAllowsMonster says whether an aura filter lets a monster through, given
// whether it is undead and whether it is a boss (the type bits, the undead-only
// bit and the not-a-boss bit; the room, line-of-sight and hostility bits are
// the caller's).
func FilterAllowsMonster(filter int, undead, boss bool) bool {
	if filter == 0 {
		filter = 0x583 // the exe's default when the column is empty
	}

	if filter&FilterMonsters == 0 {
		return false
	}

	if filter&FilterUndeadOnly != 0 && !undead {
		return false
	}

	if filter&FilterNoBoss != 0 && boss {
		return false
	}

	return true
}

// FilterAllowsPlayer says whether an aura filter reaches players at all.
func FilterAllowsPlayer(filter int) bool { return filter == 0 || filter&FilterPlayers != 0 }
