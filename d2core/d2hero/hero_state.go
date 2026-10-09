package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2inventory"
)

// HeroState stores the state of the player
type HeroState struct {
	HeroName   string                         `json:"heroName"`
	HeroType   d2enum.Hero                    `json:"heroType"`
	Act        int                            `json:"act"`
	FilePath   string                         `json:"-"`
	Equipment  d2inventory.CharacterEquipment `json:"equipment"`
	Stats      *HeroStatsState                `json:"stats"`
	Skills     map[int]*HeroSkill             `json:"skills"`
	X          float64                        `json:"x"`
	Y          float64                        `json:"y"`
	LeftSkill  int                            `json:"leftSkill"`
	RightSkill int                            `json:"rightSkill"`
	Gold       int                            `json:"Gold"`
	Difficulty d2enum.DifficultyType          `json:"difficulty"`
	// MapSeed is the level generator seed of an imported .d2s (header 0xAB).
	MapSeed uint32 `json:"mapSeed,omitempty"`
	// Progress is the quest/waypoint/NPC state imported from a .d2s; nil for
	// heroes that have none (older hero files omit it).
	Progress *HeroProgress `json:"progress,omitempty"`
}

// HeroProgress carries the story progress of a .d2s save: the quest records,
// activated waypoints and the NPC introduction/return flags of the three
// difficulties (0 normal, 1 nightmare, 2 hell).
type HeroProgress struct {
	Quests    [3]d2s.QuestRecord `json:"quests"`
	Waypoints d2s.Waypoints      `json:"waypoints"`
	NPC       d2s.NPCBlock       `json:"npc"`
}

// QuestRecord returns the quest record of a difficulty, or nil.
func (p *HeroProgress) QuestRecord(difficulty int) *d2s.QuestRecord {
	if p == nil || difficulty < 0 || difficulty >= len(p.Quests) {
		return nil
	}

	return &p.Quests[difficulty]
}
