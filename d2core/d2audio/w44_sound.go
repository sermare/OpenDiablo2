package d2audio

// Wave 44 sound audit (Game.exe 1.14b, Ghidra D2_1_14b). Everything here was read in the exe unless marked UNVERIFIED.

// swingByHitClass: the weapon swing a hero plays on an A1 / A2 mode, by the weapons.txt "hit class" of the
// right hand weapon. UNITSND_PlayHitSoundForUnit 0x4c8150 (player branch) reads the table at 0x726bc8
// (8 byte records indexed by the low nibble of the unit hit class, valid below 0xe): Sounds.txt rows
// 0, 0xfb, 0x106, 0x10c, 0x112, 0x118, 0x11e, 0x124, 0x10c, 0x12a for classes 0..9 and 0 for 10 and up
// (VERIFIED). The class NAMES to those numbers are UNVERIFIED (inferred from the row names, which fit
// the order hth, 1hss, 1hsl, 2hss, 2hsl, 1ht, 2ht, club, staf); bows and crossbows (the last classes)
// have no swing sound, their shot sounds come from the skill and missile rows.
var swingByHitClass = map[string]string{
	"hth": "weapon_punch_1", "1hss": "weapon_1hs_small_1", "1hsl": "weapon_1hs_large_1",
	"2hss": "weapon_2hs_small_1", "2hsl": "weapon_2hs_large_1", "1ht": "weapon_1ht_1",
	"2ht": "weapon_2ht_1", "club": "weapon_1hs_large_1", "staf": "weapon_staff_1",
}

// SwingHandle returns the swing handle of a weapons.txt hit class ("" when none plays, ok false when the
// class is unknown). An unarmed hero has class hth.
func SwingHandle(hitClass string) (handle string, ok bool) {
	if hitClass == "" {
		hitClass = "hth"
	}

	if h, found := swingByHitClass[hitClass]; found {
		return h, true
	}

	switch hitClass {
	case "bow", "xbow":
		return "", true
	}

	return "", false
}

// HeroFootstepWeight is the footstep set of a hero class: the exe takes it from the class record (dword +0x28,
// Sounds.txt rows 0xaa0 light, 0xad0 medium, 0xb00 heavy; UNITSND_GetStepSoundId 0x4c7970, VERIFIED), NOT from the armor
// worn: amazon and paladin medium, barbarian heavy, necromancer, sorceress, assassin and druid light.
func HeroFootstepWeight(classPrefix string) string {
	switch classPrefix {
	case "amazon", "paladin":
		return "medium"
	case "barbarian":
		return "heavy"
	}

	return "light"
}

// Vocal spacing of a monster branch hit sound (UNITSND_PlayHitSoundForUnit 0x4c8150, VERIFIED): when the
// MonSounds attack probability is below 100, the vocal plays only if more than 14 ticks passed since the last
// monster-branch sound (global DAT_007c0944, set by every vocal, hit, death and skill sound) and no speech
// entry is playing; at 100 it is unconditional. The weapon sound always plays.
const (
	VocalSpacingTicks = 15
	// SkillRepeatTicks: the Skill1-4 sound row 0xa84 plays at most once per 25 ticks per unit
	// (UNITSND_PlayMonsterEventSound 0x4c8340, VERIFIED).
	SkillRepeatTicks = 25
	// ThrottledSkillSoundRow is that row.
	ThrottledSkillSoundRow = 0xa84
)

// VocalAllowed applies the spacing rule.
func VocalAllowed(probability int, now, lastMonsterSound int64, speechPlaying bool) bool {
	if probability >= 100 {
		return true
	}

	return now-lastMonsterSound >= VocalSpacingTicks && !speechPlaying
}

// RefusalVoice maps the reason a skill cast was refused to the hero voice id. The exe asks
// SKILL_GetSkillUsabilityStatus 0x648d40 (1 mana or skill level requirement, 2 weapon / ammo / charges,
// 3 not castable, 4 state restriction, 5 and 6 passive and similar, 7 no skill level, 8 delay not over) and
// PLAYER_PlayCastFailSound 0x45c830 looks the status up in the table at 0x7118f8: status 1 = needmana (0x15),
// 2, 3 and 4 = impossible (0x13), every other status is silent (VERIFIED). A cast in town without the InTown
// flag says not_in_town (0x18, SKILL_ClientStartSkill 0x4c2b90 and PLAYER_ClickSkillTargetAction 0x45d920).
// The reasons are d2skill.Reason* (strings kept here to avoid an import).
func RefusalVoice(reason string) int {
	switch reason {
	case "mana":
		return VoiceNeedMana
	case "town":
		return VoiceNotInTown
	case "no_ammo", "no_throwable", "no_shield", "restricted":
		return VoiceImpossible
	}

	return 0
}
