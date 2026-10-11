package d2skill

// ConcentrationHolder is implemented by a Unit that can report the damage
// percent (stat 0x19) held by its Concentration state (state 0x2a): the exe's
// SRVDO_BlessedHammer_ScaleBySkillRecord (0x647550) reads that state's own
// statlist, not the unit's total.
type ConcentrationHolder interface {
	ConcentrationDamagePct() int
}

// HammerConcentrationScale is the percent a Blessed Hammer gains from an
// active Concentration state (0x647550, VERIFIED): Param1 of the hammer skill
// times the state's damage percent, divided by 8 (toward zero). The missile's
// stats 0x34 / 0x35 (magic damage) become base * (100 + scale) / 100.
func HammerConcentrationScale(param1, damagePct int) int {
	return param1 * damagePct / 8
}
