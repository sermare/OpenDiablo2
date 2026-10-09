package d2quest

// Quest ids as the original game numbers its quest nodes (d2-re-notes
// quests.md section 2). The record slot of a quest is given separately
// because it differs from the id for the intro quests and the later acts.
const (
	QuestA1Prologue      = 0  // Warriv's welcome, slot 0
	QuestDenOfEvil       = 1  // slot 1
	QuestBurial          = 2  // Sisters' Burial Grounds, slot 2
	QuestTools           = 3  // Tools of the Trade, slot 3
	QuestCain            = 4  // The Search for Cain, slot 4
	QuestTower           = 5  // The Forgotten Tower, slot 5
	QuestAndariel        = 6  // Sisters to the Slaughter, slot 6
	QuestA2Prologue      = 7  // Jerhyn's welcome, slot 8
	QuestRadament        = 8  // Radament's Lair, slot 9
	QuestHoradricStaff   = 9  // slot 10
	QuestTaintedSun      = 10 // slot 11
	QuestArcane          = 11 // slot 12
	QuestSummoner        = 12 // slot 13
	QuestSevenTombs      = 13 // slot 14
	QuestA3Prologue      = 14 // Hratli's welcome, slot 16
	QuestLamEsen         = 15 // slot 17
	QuestKhalim          = 16 // slot 18
	QuestBlade           = 17 // slot 19
	QuestGoldenBird      = 18 // slot 20
	QuestBlackenedTemple = 19 // slot 21
	QuestGuardian        = 20 // slot 22
	QuestA4Prologue      = 21 // Tyrael's welcome, slot 24
	QuestFallenAngel     = 22 // slot 25
	QuestTerrorsEnd      = 23 // slot 26
	QuestHellforge       = 24 // slot 27
	QuestNavi            = 25
	QuestMalachai        = 29 // Malachai's stone gossip, slot 33
	QuestSiege           = 31 // slot 35
	QuestRescue          = 32 // slot 36
	QuestPrison          = 33 // slot 37
	QuestBetrayal        = 34 // slot 38
	QuestRite            = 35 // slot 39
	QuestEve             = 36 // slot 40
	QuestA1Intro         = 37 // the four intro quests are ids 37..40, slot 41 (outside the record)
	QuestA2Intro         = 38
	QuestA3Intro         = 39
	QuestA5Intro         = 40
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
	// Classes of the monsters quests kill or talk to (monstats.txt rows of
	// the 1.14b classic+expansion table; the expansion divider row is not
	// counted, which matches the NPC ids above). Only the ones below were
	// cross-checked against the binary notes; NPCBaal is UNVERIFIED (the row
	// named "Baal Throne" of monstats.txt).
	NPCSummoner       = 250
	NPCDuriel         = 211
	NPCMephisto       = 242
	NPCDiablo         = 243
	NPCIzualGhost     = 406
	NPCCouncilA       = 345
	NPCCouncilB       = 346
	NPCCouncilC       = 347
	NPCPrisonDoor     = 434
	NPCNihlathakBoss  = 526
	NPCAnyaFrozen     = 527 // "Drehya outside town", the frozen Anya (UNVERIFIED)
	NPCAncientStatue1 = 537
	NPCAncientStatue2 = 538
	NPCAncientStatue3 = 539
	NPCAncient1       = 540
	NPCAncient2       = 541
	NPCAncient3       = 542
	NPCBaal           = 543
	NPCGuard2         = 331 // ACT2GUARD2 of the speech table (which guard class it is is unverified)
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
	LevelHarem1          = 50
	LevelCanyon          = 46
	LevelLostCity        = 44
	LevelValleySnakes    = 45
	LevelArcane          = 74
	LevelTalRashaFirst   = 66
	LevelTalRashaLast    = 72
	LevelDurielLair      = 73
	LevelKurastDocktown  = 75
	LevelLowerKurast     = 79
	LevelFlayerDungeon1  = 88
	LevelFlayerDungeon2  = 89
	LevelFlayerDungeon3  = 91
	LevelSewers2Kurast   = 93
	LevelRuinedTemple    = 94
	LevelSpiderCavern    = 85
	LevelTravincal       = 83
	LevelDurance1        = 100
	LevelDurance3        = 102
	LevelPandemonium     = 103
	LevelPlainsDespair   = 105
	LevelRiverOfFlame    = 107
	LevelChaosSanctum    = 108
	LevelHarrogath       = 109
	LevelBloodyFoothills = 110
	LevelFrigidHighlands = 111
	LevelArreatPlateau   = 112
	LevelFrozenRiver     = 114 // levels.txt calls it "Cellar of Pity" (internal name mixup)
	LevelArreatSummit    = 120
	LevelNihlathakTemple = 121
	LevelWorldstone1     = 128
	LevelThrone          = 131
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
	ItemViperAmulet       = "vip"
	ItemHoradricStaff     = "hst"
	ItemLamEsenTome       = "bbb"
	ItemKhalimEye         = "qey"
	ItemKhalimHeart       = "qhr"
	ItemKhalimBrain       = "qbr"
	ItemKhalimFlail       = "qf1"
	ItemKhalimWill        = "qf2"
	ItemGidbinn           = "g33"
	ItemJadeFigurine      = "j34"
	ItemGoldenBird        = "g34"
	ItemMephistoSoulstone = "mss"
	ItemHellforgeHammer   = "hfh"
	ItemMalahScroll       = "tr2"
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
