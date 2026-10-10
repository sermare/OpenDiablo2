package d2hero

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"

const (
	// prisonSlot and scrollReadBit locate Malah's Scroll of Resistance in a quest record: A5Q3 (Prison of
	// Ice) is slot 37 and the exe sets bit 7 when the scroll is READ (VERIFIED Game.exe 0x55bfd0, verify-quest-rewards.md).
	prisonSlot    = 37
	scrollReadBit = 7
	// resistPerScroll is the resistance every read scroll adds (one per difficulty record).
	resistPerScroll = 10
)

// ResistScrollBonus is the Scroll of Resistance bonus (fire, lightning, cold, poison) of the hero: 10 for every
// difficulty record whose scroll-read bit is set, so 10/20/30. Like the exe (FUN_00587f90, re-run on every game
// join) it is DERIVED from the quest records and never stored as a stat, so saving and loading cannot count it twice.
func (p *HeroProgress) ResistScrollBonus() int {
	if p == nil {
		return 0
	}

	n := 0

	for d := range p.Quests {
		if p.Quests[d].Get(prisonSlot, scrollReadBit) {
			n += resistPerScroll
		}
	}

	return n
}

// resistScrollItem is the bonus as a stat-list source (the exe attaches a separate stat list with base stats
// 39/41/43/45 = the sum); nil when there is no bonus.
func resistScrollItem(bonus int) []d2statlist.Item {
	if bonus <= 0 {
		return nil
	}

	v := int64(bonus)

	return []d2statlist.Item{{Code: "tr2", Charm: true, Props: []d2statlist.Prop{
		{ID: d2statlist.StatFireResist, Value: v}, {ID: d2statlist.StatLightResist, Value: v},
		{ID: d2statlist.StatColdResist, Value: v}, {ID: d2statlist.StatPoisonResist, Value: v},
	}}}
}

// AddLifeBonus adds a permanent max-life reward (the Potion of Life, +20): the bonus is part of the base maximum
// that a .d2s stores, and an import derives LifeBonus from the stored value, so it is counted exactly once.
// The current life rises with the maximum.
func (s *HeroStatsState) AddLifeBonus(n int) {
	if s == nil || n == 0 {
		return
	}

	if !s.StatsBonusInit && s.Recalc != nil {
		s.Recalc() // derive an imported save's existing bonus before adding to it
	}

	s.LifeBonus += n
	s.Health += n

	if s.Recalc != nil {
		s.Recalc()
		return
	}

	s.MaxHealth += n

	if s.BaseMaxHealth > 0 { // 0 means "not computed": the export then stores MaxHealth
		s.BaseMaxHealth += n
	}
}
