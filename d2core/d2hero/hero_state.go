package d2hero

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
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
	// SkillBar holds the 16 skill hotkeys and the swap-set skills (with the
	// active left/right skill, which LeftSkill and RightSkill mirror); nil for
	// heroes that never had any (older hero files omit it).
	SkillBar   *SkillBar             `json:"skillBar,omitempty"`
	Difficulty d2enum.DifficultyType `json:"difficulty"`
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
	// Imported is set for heroes that came from a real .d2s save; nil for heroes
	// created in this engine, which are shown as Expansion characters.
	Imported *ImportedInfo `json:"imported,omitempty"`
	// Expansion, Hardcore and Ladder are the choices made when the character
	// was created (the .d2s status bits 0x20, 0x04 and 0x40).
	Expansion bool `json:"expansion,omitempty"`
	Hardcore  bool `json:"hardcore,omitempty"`
	Ladder    bool `json:"ladder,omitempty"`
	// Death is the record of the hero's deaths and corpse; nil if the hero
	// never died.
	Death *DeathState `json:"death,omitempty"`

	// statEquipped caches the equipped items as the stat list sees them.
	statEquipped []d2statlist.Item
	// Merc is the hired mercenary (the d2s header fields); nil when the hero
	// has none. The merc level is not stored: it is derived from Experience.
	Merc *MercState `json:"merc,omitempty"`
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
	// Source is the .d2s path the hero was imported from (never written to).
	Source string `json:"source,omitempty"`
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

// EnsureProgress returns the hero's progress, creating it for heroes that have
// none (new characters): like a fresh character of the original, the
// Rogue Encampment waypoint (bit 0) starts activated in every difficulty.
func (s *HeroState) EnsureProgress() *HeroProgress {
	if s.Progress == nil {
		s.Progress = &HeroProgress{}

		for d := 0; d < len(s.Progress.Waypoints); d++ {
			s.Progress.Waypoints.Set(d, d2s.WPRogueEncampment, true)
		}
	}

	return s.Progress
}
