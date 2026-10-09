package d2quest

// Quest ids as the original game numbers its quest nodes (d2-re-notes
// quests.md section 2). The record slot of a quest is given separately
// because it differs from the id for the intro quests and the later acts.
const (
	QuestA1Prologue = 0 // Warriv's welcome, slot 0
	QuestDenOfEvil  = 1 // slot 1
	QuestBurial     = 2 // Sisters' Burial Grounds, slot 2
	QuestTools      = 3 // Tools of the Trade, slot 3
	QuestCain       = 4 // The Search for Cain, slot 4
	QuestTower      = 5 // The Forgotten Tower, slot 5
	QuestAndariel   = 6 // Sisters to the Slaughter, slot 6
	QuestA2Prologue = 7 // Jerhyn's welcome, slot 8
	QuestRadament   = 8 // Radament's Lair, slot 9
	QuestNavi       = 25
	QuestA1Intro    = 37 // the four intro quests are ids 37..40, slot 41 (outside the record)
	QuestA2Intro    = 38
	QuestA3Intro    = 39
	QuestA5Intro    = 40
)

// Monster class ids (monstats.txt rows), verified against the class
// numbering in the notes (quests-2.md section 0 point 6) and the D2MOO
// MonsterIds.h enum order.
const (
	NPCCain1      = 146 // Cain in Tristram
	NPCGheed      = 147
	NPCAkara      = 148
	NPCKashya     = 150
	NPCCharsi     = 154
	NPCWarriv1    = 155
	NPCAndariel   = 156
	NPCWarriv2    = 175
	NPCAtma       = 176
	NPCDrognan    = 177
	NPCFara       = 178
	NPCGreiz      = 198
	NPCElzix      = 199
	NPCGeglash    = 200
	NPCJerhyn     = 201
	NPCLysander   = 202
	NPCMeshif1    = 210
	NPCRadament   = 229
	NPCCain2      = 244
	NPCCain3      = 245
	NPCCain4      = 246
	NPCTyrael1    = 251
	NPCAsheara    = 252
	NPCHratli     = 253
	NPCAlkor      = 254
	NPCOrmus      = 255
	NPCIzual      = 256
	NPCMeshif2    = 264
	NPCCain5      = 265 // Cain in the Rogue Encampment
	NPCNavi       = 266
	NPCBloodRaven = 267
	NPCNatalya    = 297
	NPCTyrael2    = 367
	NPCMalachai   = 408
	NPCLarzuk     = 511
	NPCDrehya     = 512
	NPCMalah      = 513
	NPCNihlathak  = 514
	NPCQualKehk   = 515
	NPCCain6      = 520
	NPCTyrael3    = 521
)

// Level ids (levels.txt).
const (
	LevelRogueEncampment = 1
	LevelDenOfEvil       = 8
	LevelBurialGrounds   = 17
	LevelForgottenTower  = 20
	LevelTowerCellar5    = 25
	LevelCatacombs1      = 34
	LevelCatacombs4      = 37
	LevelTristram        = 38
	LevelMooMooFarm      = 39
	LevelLutGholein      = 40
	LevelSewers3         = 49
)

// Player classes in the .d2s numbering.
const (
	ClassAmazon = iota
	ClassSorceress
	ClassNecromancer
	ClassPaladin
	ClassBarbarian
	ClassDruid
	ClassAssassin
)

// Item codes of quest items (four-character codes without the padding).
const (
	ItemHoradricMalus     = "hdm"
	ItemScrollOfInifuss   = "bks"
	ItemDecipheredScroll  = "bkd"
	ItemHoradricScroll    = "tr1"
	ItemBookOfSkill       = "ass"
	ItemReward            = "rin"
	ItemHoradricCube      = "box"
	ItemStaffOfKingsShaft = "msf"
)

// Difficulty indexes of the record.
const (
	Normal = iota
	Nightmare
	Hell
)

// actOfLevel returns the act (0..4) of a level id, using the act borders of
// levels.txt (Act 1: 1-39, Act 2: 40-74, Act 3: 75-102, Act 4: 103-108,
// Act 5: 109+). -1 for level 0.
func actOfLevel(level int) int {
	switch {
	case level <= 0:
		return -1
	case level < 40:
		return 0
	case level < 75:
		return 1
	case level < 103:
		return 2
	case level < 109:
		return 3
	}

	return 4
}
