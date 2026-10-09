package d2hero

import "github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"

// HydrateSkills will load the SkillRecord & SkillDescriptionRecord from the asset manager, using the skill ID.
// This is done to avoid serializing the whole record data of HeroSkill to a game save or network packets.
// We cant do this while unmarshalling because there is no reference to the asset manager.
// The points travel in the Shallow copy; they become the live SkillPoints here.
func HydrateSkills(skills map[int]*HeroSkill, asset *d2asset.AssetManager) {
	for skillID, skill := range skills {
		if skill == nil {
			delete(skills, skillID)
			continue
		}

		skill.SkillRecord = asset.Records.Skill.Details[skillID]
		skill.SkillDescriptionRecord = asset.Records.Skill.Descriptions[skill.SkillRecord.Skilldesc]

		if skill.Shallow != nil {
			skill.SkillPoints = skill.Shallow.SkillPoints
		}
	}
}
