package d2hero

import (
	"encoding/json"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// HeroSkill stores additional payload for a skill of a hero.
type HeroSkill struct {
	*d2records.SkillRecord
	*d2records.SkillDescriptionRecord
	SkillPoints int
	Shallow     *shallowHeroSkill
}

// An auxiliary struct which only stores the ID of the SkillRecord, instead of the whole SkillRecord
// and SkillDescrptionRecord.
type shallowHeroSkill struct {
	SkillID     int `json:"skillId"`
	SkillPoints int `json:"skillPoints"`
}

// MarshalJSON overrides the default logic used when the HeroSkill is serialized to a byte array.
func (hs *HeroSkill) MarshalJSON() ([]byte, error) {
	// only serialize the Shallow object instead of the SkillRecord & SkillDescriptionRecord
	// (the live points win: some code adds points without touching Shallow)
	shallow := hs.Shallow
	if hs.SkillRecord != nil {
		shallow = &shallowHeroSkill{SkillID: hs.SkillRecord.ID, SkillPoints: hs.SkillPoints}
	}

	bytes, err := json.Marshal(shallow)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// UnmarshalJSON overrides the default logic used when the HeroSkill is deserialized from a byte array.
func (hs *HeroSkill) UnmarshalJSON(data []byte) error {
	shallow := &shallowHeroSkill{}
	if err := json.Unmarshal(data, shallow); err != nil {
		return err
	}

	hs.Shallow = shallow

	return nil
}

// SetPoints sets the points of the skill, keeping the serialised copy
// (Shallow, the part a save keeps) in step with the live value.
func (hs *HeroSkill) SetPoints(n int) {
	hs.SkillPoints = n

	if hs.Shallow == nil {
		hs.Shallow = &shallowHeroSkill{}
		if hs.SkillRecord != nil {
			hs.Shallow.SkillID = hs.SkillRecord.ID
		}
	}

	hs.Shallow.SkillPoints = n
}
