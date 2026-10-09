package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// specialACClasses are the monster classes of helper 0x63fed0 (also used by
// crushing blow): stat 0x73 does not zero their defense and stat 0x74 is halved.
var specialACClasses = map[int]bool{0x10f: true, 0x152: true, 0x167: true, 0x230: true, 0x231: true}

// HeroAROperands applies 0x57b8b0 (d2combat.AdjustAROperands, VERIFIED by
// disassembly) for a hero attacking monster m: stat 0x73 zeroes the defense of
// plain monsters, 0x74 removes a percent of it, 0x7b / 0x7c add attack rating
// against demons / undead. UNVERIFIED mapping: the monster data flags
// +0x16 & 0xa (plain test) and +0x16 & 2 (halving) are not modelled, only the
// boss flag and the special classes. A hero without these stats (or without
// Totals) gets ar and def back unchanged.
func HeroAROperands(t *d2statlist.Totals, m *d2mapentity.Monster, ar, def int) (newAR, newDef int) {
	if t == nil || t.Stats == nil || m == nil || m.Stat == nil {
		return ar, def
	}

	s := t.Stats
	special := specialACClasses[m.MonstatID()]
	o := d2combat.AROperands{
		IgnoreDefense:          s.Get(d2statlist.StatIgnoreDef) != 0,
		TargetACPct:            int(s.Get(d2statlist.StatTargetACPct)),
		DemonAR:                int(s.Get(d2statlist.StatDemonAR)),
		UndeadAR:               int(s.Get(d2statlist.StatUndeadAR)),
		DefenderIsPlainMonster: !m.Stat.IsSpecialBoss && !special,
		HalveTargetAC:          m.Stat.IsSpecialBoss || special,
		DefenderDemon:          m.Stat.IsDemon,
		DefenderUndead:         m.Stat.IsUndeadLow || m.Stat.IsUndeadHigh,
	}

	return d2combat.AdjustAROperands(ar, def, o)
}
