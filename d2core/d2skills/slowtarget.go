package d2skills

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// The gear property "slows target by N%" (ItemStatCost item_slow, stat 0x96) is an item event of a landed hit
// (exe 0x5bdf30, VERIFIED): the amount is capped by what was hit, a timed state list is put on the target with
// stats 0x44 and 0x45 (attack rate and the other animation rates) set to -amount, and the target's velocity and
// animation rate are recalculated. Caps: a player 50; a monster with type flags champion or unique (0xc) 50, a
// special boss 50, a hireling 50; a super unique (flag 2) 75; every other monster 90.
//
// UNVERIFIED: the state id and the duration come from the caller's context in the exe and were not recovered;
// the state is named "itemslow" and lasts slowTargetFrames here. The movement speed is slowed by the same
// percent (the velocity recalculation reads the same stats; U).

const (
	// StateItemSlow is the state the slow of a weapon puts on the target.
	StateItemSlow = "itemslow"

	slowTargetFrames = 100 // UNVERIFIED duration, 4 seconds at 25 frames per second

	slowCapPlayer      = 50
	slowCapBossOrElite = 50
	slowCapSuperUnique = 75
	slowCapMonster     = 90
)

// slowTargetCap is the highest slow percent a target can get from the property.
func slowTargetCap(isPlayer, isBoss bool, typeFlags uint16) int {
	switch {
	case isPlayer, isBoss:
		return slowCapPlayer
	case typeFlags&(d2mapentity.MonTypeChampion|d2mapentity.MonTypeUnique) != 0:
		return slowCapBossOrElite
	case typeFlags&d2mapentity.MonTypeSuperUnique != 0:
		return slowCapSuperUnique
	}

	return slowCapMonster
}

// slowTargetAmount is the slow percent the property puts on a target; 0 means nothing happens.
func slowTargetAmount(stat int, isPlayer, isBoss bool, typeFlags uint16) int {
	if stat <= 0 {
		return 0
	}

	if c := slowTargetCap(isPlayer, isBoss, typeFlags); stat > c {
		return c
	}

	return stat
}

// applySlowTarget puts the weapon slow on a monster that was just hit.
func (e *Engine) applySlowTarget(m *d2mapentity.Monster, stat int) {
	boss := m.Stat != nil && m.Stat.IsSpecialBoss

	pct := slowTargetAmount(stat, false, boss, m.TypeFlags)
	if pct == 0 {
		return
	}

	e.applyMonsterState(m, d2state.Instance{
		Name: StateItemSlow, Until: e.frame + slowTargetFrames, Level: 1,
		Mods: []d2state.StatMod{{Stat: "velocitypercent", Value: -pct}, {Stat: "attackrate", Value: -pct},
			{Stat: "other_animrate", Value: -pct}},
	})
	e.emit("state", "STATE item_slow target=%s slow=%d%% until=%d", m.Label(), pct, e.frame+slowTargetFrames)
}
