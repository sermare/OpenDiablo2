package d2quest

import "strings"

// Boss kills of the later acts: Duriel (A2Q6, The Seven Tombs), Mephisto (A3Q6,
// The Guardian), Diablo (A4Q2, Terror's End) and Baal (A5Q6, Eve of
// Destruction). Andariel (A1Q6) lives in a1q5q6.go. The quest nodes
// themselves are the full quests of a2more.go / a345.go; this file keeps the
// verified ids, slots and classes and the kill matching by monster name.
//
// Evidence (d2-re-notes/boss-encounters.md, Game.exe 1.14b read-only):
//
//	quest id / record slot: Seven Tombs 13 / 14, Guardian 20 / 22, Terror's End
//	23 / 26, Eve of Destruction 36 / 40 (VERIFIED by the init functions' slot
//	field and by the QUESTREC_GetFlag(slot) calls in the kill handlers).
//	VERIFIED bits: Duriel's kill sets bit 5 of slot 14 when none of the bits
//	0, 3, 4, 5 is set (0x59ac40); Mephisto's kill sets the per-player quest
//	flag 0xd (primary goal) of slot 22 (0x5ba440).
//	UNVERIFIED: that the kills of Diablo and Baal set the same primary-goal
//	bit, and that all four also raise "reward pending" (bit 1) the way
//	Andariel's does (D2MOO convention, not read from the binary for these).
//

// Quest ids / slots of the boss quests.
const (
	QuestSevenTombs       = 13
	QuestGuardian         = 20
	QuestTerrorsEnd       = 23
	QuestEveOfDestruction = 36

	SlotSevenTombs       = 14
	SlotGuardian         = 22
	SlotTerrorsEnd       = 26
	SlotEveOfDestruction = 40
)

// Monster classes of the bosses (monstats.txt rows).
const (
	NPCDuriel   = 211
	NPCMephisto = 242
	// NPCHephasto is monstats row 409 "hephasto" (the Hell Forge smith demon); VERIFIED: the monster-create hook 0x5af8c0 attaches
	// the Hellforge quest node (id 24) to class 0x199 (= 409).
	NPCHephasto = 409
	NPCDiablo   = 243
	NPCBaalCrab = 544 // exe class of the Baal who dies in the Worldstone Chamber (row 545 "Baal Crab")
)

func nameIs(e *Event, name string) bool { return strings.EqualFold(strings.TrimSpace(e.Name), name) }
