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
	// D2SBase is the .d2s the hero was imported from. ExportD2S starts from it
	// so everything the engine does not model survives a save back.
	D2SBase []byte `json:"d2sBase,omitempty"`
	// Progress is the quest/waypoint/NPC state imported from a .d2s; nil for
	// heroes that have none (older hero files omit it).
	Progress *HeroProgress `json:"progress,omitempty"`
	// Containers is the inventory, belt, cube and stash content; nil for
	// heroes that never saved any (older hero files omit it).
	Containers *HeroContainers `json:"containers,omitempty"`
	// Merc is the hired mercenary (the d2s header fields); nil when the hero
	// has none. The merc level is not stored: it is derived from Experience.
	Merc *MercState `json:"merc,omitempty"`
}

// MercState is the persistent part of a mercenary, as the .d2s header keeps it
// (0xB1 dead, 0xB3 id, 0xB7 name id, 0xB9 type, 0xBB experience).
type MercState struct {
	Dead       bool   `json:"dead,omitempty"`
	ID         uint32 `json:"id"`
	NameID     uint16 `json:"nameId"`
	Type       uint16 `json:"type"`
	Experience uint32 `json:"experience"`
	// Replaced is set when a new merc was hired over an imported one: the old
	// merc's items are dropped on export.
	Replaced bool `json:"replaced,omitempty"`
}

// MercFromHeader converts the d2s header fields; nil when there is no merc.
func MercFromHeader(m d2s.Mercenary) *MercState {
	if m.ID == 0 {
		return nil
	}

	return &MercState{Dead: m.Dead, ID: m.ID, NameID: m.NameID, Type: m.Type, Experience: m.Experience}
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
