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
	// Imported is set for heroes that came from a real .d2s save; nil for heroes
	// created in this engine, which are shown as Expansion characters.
	Imported *ImportedInfo `json:"imported,omitempty"`
	// Items are the items of an imported .d2s (equipped, inventory page, belt, cube
	// and stash), for the inventory panel. See ImportedItem.
	Items []ImportedItem `json:"items,omitempty"`
}

// ImportedInfo carries what the character select screen shows about a real
// Diablo II character, and the skill hotkeys of the original save.
type ImportedInfo struct {
	Hardcore  bool `json:"hardcore,omitempty"`
	Expansion bool `json:"expansion"`
	Ladder    bool `json:"ladder,omitempty"`
	Dead      bool `json:"dead,omitempty"`
	// WeaponSetII is set when the second weapon set was active when the game was saved.
	WeaponSetII bool `json:"weaponSetII,omitempty"`
	// LastPlayed is the unix time of the original's last save.
	LastPlayed uint32 `json:"lastPlayed,omitempty"`
	// Hotkeys are the skill ids on the 16 skill hotkeys, -1 for an empty slot.
	Hotkeys []int `json:"hotkeys,omitempty"`
	// SwapLeftSkill and SwapRightSkill are the mouse skills of weapon set II.
	SwapLeftSkill  int `json:"swapLeftSkill,omitempty"`
	SwapRightSkill int `json:"swapRightSkill,omitempty"`
	// Source is the .d2s path the hero was imported from (never written to).
	Source string `json:"source,omitempty"`
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
