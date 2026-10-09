package d2hero

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// StoredProgression is the progression byte of the .d2s the hero was imported
// from (header status bits 8..12), 0 for heroes without one.
func (s *HeroState) StoredProgression() int {
	if len(s.D2SBase) < d2s.HeaderSize {
		return 0
	}

	h, err := d2s.ParseHeader(s.D2SBase)
	if err != nil {
		return 0
	}

	return d2difficulty.Progression(h.Status)
}

// questRecords returns the three quest records (zero records without progress).
func (s *HeroState) questRecords() [3]d2s.QuestRecord {
	if s.Progress == nil {
		return [3]d2s.QuestRecord{}
	}

	return s.Progress.Quests
}

// ComputedProgression is the progression derived from the quest records and
// the stored one: never lower than what the save already had.
func (s *HeroState) ComputedProgression() int {
	p := d2difficulty.ProgressionOf(s.questRecords(), s.Expansion)
	if stored := s.StoredProgression(); stored > p {
		return stored
	}

	return p
}

// UnlockedDifficulties tells which difficulties the hero may start: Normal
// always, Nightmare once Normal is finished, Hell once Nightmare is.
func (s *HeroState) UnlockedDifficulties() [3]bool {
	return d2difficulty.Unlocked(s.questRecords(), s.StoredProgression(), s.Expansion)
}

// ChooseDifficulty makes a difficulty the one the hero plays in. It refuses a
// difficulty that is not unlocked unless force is set (test scenarios). The
// choice is saved in the hero file and in the .d2s header (active byte) on the
// next export.
func (s *HeroState) ChooseDifficulty(l d2difficulty.Level, force bool) error {
	if l < d2difficulty.Normal || l >= d2difficulty.Count {
		return fmt.Errorf("difficulty %d does not exist", int(l))
	}

	if !force && !s.UnlockedDifficulties()[l] {
		return fmt.Errorf("%s is not unlocked for %s", l, s.HeroName)
	}

	s.Difficulty = d2enum.DifficultyType(l)

	// every difficulty owns its waypoints and quests: make sure they exist and
	// that the Rogue Encampment waypoint is active in the chosen one
	p := s.EnsureProgress()
	if !p.Waypoints.Has(int(l), d2s.WPRogueEncampment) {
		p.Waypoints.Set(int(l), d2s.WPRogueEncampment, true)
	}

	return nil
}
