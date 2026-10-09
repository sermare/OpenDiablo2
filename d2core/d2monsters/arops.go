package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// specialACClasses are the monster classes of helper 0x63fed0 (VERIFIED: it
// returns nonzero for classes 0x10f, 0x152, 0x167, 0x230, 0x231).
var specialACClasses = map[int]bool{0x10f: true, 0x152: true, 0x167: true, 0x230: true, 0x231: true}

// IsMercenaryClass reports whether a monstats id is one of the five classes of helper 0x63fed0
// (MONSTER_GetHirelingClassIndex): the rogue archer 0x10f, desert guard 0x152, iron wolf 0x167 and
// the barbarians 0x230 / 0x231. The code and the notes call them "special" classes (attack rating
// halving, the 13 frame stun cut, the crushing blow divisor of 10): they are the mercenaries.
func IsMercenaryClass(class int) bool { return specialACClasses[class] }

// HeroAROperands applies 0x57b8b0 (d2combat.AdjustAROperands, VERIFIED by
// disassembly) for a hero attacking monster m: stat 0x73 zeroes the defense of
// plain monsters, 0x74 removes a percent of it, 0x7b / 0x7c add attack rating
// against demons / undead.
//
// The two conditions are VERIFIED (see verify-monster-flag16.md). The byte
// at +0x16 is not a monstats column: 0x59dd60 reads word +0x16 of the unit's
// monster data (unit+0x14), a per-spawn type mask. "Plain" (0x73) needs
// TypeFlags&0xa == 0 (unique / super unique), the monstats flag dword byte
// +0xc & 0x40 == 0 (0x63fa40, the boss column, bit 6 of the loader table) and
// 0x63fed0 == 0 (special classes). Halving (0x74) applies when TypeFlags&2,
// or boss, or a special class. A hero without these stats (or without Totals)
// gets ar and def back unchanged.
func HeroAROperands(t *d2statlist.Totals, m *d2mapentity.Monster, ar, def int) (newAR, newDef int) {
	if t == nil || t.Stats == nil || m == nil || m.Stat == nil {
		return ar, def
	}

	s := t.Stats
	special := specialACClasses[m.MonstatID()]
	boss := m.Stat.IsSpecialBoss
	o := d2combat.AROperands{
		IgnoreDefense:          s.Get(d2statlist.StatIgnoreDef) != 0,
		TargetACPct:            int(s.Get(d2statlist.StatTargetACPct)),
		DemonAR:                int(s.Get(d2statlist.StatDemonAR)),
		UndeadAR:               int(s.Get(d2statlist.StatUndeadAR)),
		DefenderIsPlainMonster: m.TypeFlags&0xa == 0 && !boss && !special,
		HalveTargetAC:          m.TypeFlags&d2mapentity.MonTypeSuperUnique != 0 || boss || special,
		DefenderDemon:          m.Stat.IsDemon,
		DefenderUndead:         m.Stat.IsUndeadLow || m.Stat.IsUndeadHigh,
	}

	return d2combat.AdjustAROperands(ar, def, o)
}
