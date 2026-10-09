package d2level

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// gates.go: which level links need a quest, and the links that are portals
// rather than warp tiles. The level graph audit (audit_test.go) checks every
// table here against Levels.txt.
//
// The QuestFlag / QuestFlagEx columns of Levels.txt name a quest RECORD SLOT
// (the same slot numbers as d2s.QuestSlot): the values match the slots of
// quests.md: 11 = The Tainted Sun, 13 = The Summoner, 21 = The Blackened
// Temple, 26 = Terror's End (classic Cow Level), 39 = Rite of Passage, 40 = Eve
// of Destruction (expansion Cow Level).
//
// VERIFIED in Game.exe: the Levels.txt loader (DATATBL_LoadLevelsTable_Layout
// 0x61dd60) reads QuestFlag and QuestFlagEx as the first two words of the
// record, and the portal-use handler SERVER_UsePortalObject (0x582700) feeds
// them (QuestFlagEx when the game is an expansion game) to QUESTREC_GetFlag
// (0x65e820) with bit 0, the "done" bit: a town portal object (class 59) whose
// level has a QuestFlag > 0 is refused until that slot is done. The column is
// NOT consulted by the warp-tile handler (see the gates below).

// levelQuestFlag is {QuestFlag, QuestFlagEx} of the levels that have one
// (Levels.txt, patch_d2).
var levelQuestFlag = map[int][2]int{
	39: {26, 40},                                                         // Moo Moo Farm
	46: {13, 13},                                                         // Canyon of the Magi
	50: {11, 11}, 51: {11, 11}, 52: {11, 11}, 53: {11, 11}, 54: {11, 11}, // palace
	66: {13, 13}, 67: {13, 13}, 68: {13, 13}, 69: {13, 13}, 70: {13, 13}, 71: {13, 13}, 72: {13, 13}, // Tal Rasha's tombs
	73:  {13, 13},                               // Duriel's Lair
	74:  {11, 11},                               // Arcane Sanctuary
	100: {21, 21}, 101: {21, 21}, 102: {21, 21}, // Durance of Hate
	128: {39, 39}, 129: {39, 39}, 130: {39, 39}, 131: {39, 39}, 132: {39, 39}, // Worldstone Keep
}

// LevelQuestFlag returns the quest slot a level needs (0 if none), classic or
// expansion column.
func LevelQuestFlag(level int, expansion bool) int {
	f := levelQuestFlag[level]
	if expansion {
		return f[1]
	}

	return f[0]
}

// portalLinks are the links no Levels.txt Vis slot carries: the level is
// entered through a portal object or a script. From is the level that holds the
// portal. The portal factory is QUEST_Func_56ae80 (0x56ae80): it is called with
// (room, x, y, destination level, ..., object class, permanent flag).
//
//   - 1 -> 39: the Cow Level portal. VERIFIED that the factory lets a portal to
//     level 39 be made in town (a whitelist of destinations 39 and 133..136, and
//     only for object class 60, the permanent portal); the call site that makes
//     it was not located (no literal 0x27 among the 11 callers), so the quest
//     rules (Terror's End or Eve of Destruction done, Wirt's Leg) are play rules.
//   - 4 -> 38: VERIFIED. The Cairn Stones solved: the Search for Cain code
//     (call at 0x590b92) makes a permanent portal (class 60) to level 0x26 =
//     Tristram beside the Cairn Stone in Stony Field.
//   - 54 -> 74: UNVERIFIED. No portal factory call names level 74. D2MOO calls
//     the Palace Cellar 3 to Arcane Sanctuary link an object (class 298), not a
//     portal; where the object stands is a play rule.
//   - 74 -> 46: VERIFIED. The Arcane Sanctuary quest code on the journal message
//     (call at 0x598d56) makes a permanent portal to level 0x2e = Canyon of the
//     Magi beside the player (who is in the sanctuary).
//   - 66..72 -> 73: UNVERIFIED. D2MOO opens a "portal to Duriel's lair" object
//     (class 100) at the staff orifice; no factory call names level 73. Which of
//     the seven is the true tomb depends on the seed (tombA of the act), so every
//     tomb is listed.
//   - 109 -> 121: VERIFIED. Anya's red portal: two quest call sites (0x588ea9,
//     0x589b7d) make a permanent portal (class 60) to level 0x79 = Nihlathak's
//     Temple in Harrogath.
//   - 109 -> 133..136: the Pandemonium portals (not in this table): the factory
//     whitelist lets class 60 portals to 133..136 be made in Harrogath, the
//     creating code (key/Hell rules) was not located.
//
//nolint:gochecknoglobals // static data
var portalLinks = []struct{ From, To int }{
	{1, 39}, {4, 38}, {54, 74}, {74, 46},
	{66, 73}, {67, 73}, {68, 73}, {69, 73}, {70, 73}, {71, 73}, {72, 73},
	{109, 121},
}

// Gate says that entering To (from From, or from anywhere when From is 0)
// needs a quest.
type Gate struct {
	From, To int
	// Act and Quest name the quest (d2s.QuestSlot arguments); Slot is its
	// record slot.
	Act, Quest, Slot int
	Why              string
	// Verified is true when the gate is read from Levels.txt (every
	// levelQuestFlag gate) or the quest notes name the level.
	Verified bool
}

// extraGates are the gates Levels.txt does not state. Each says why.
//
//nolint:gochecknoglobals // static data
var extraGates = []Gate{
	{From: LevelTravincal, To: LevelDurance1, Act: 3, Quest: 2,
		Verified: true,
		Why:      "VERIFIED: SERVER_EnterWarpTile 0x553140 refuses a warp into level 100 (0x5b9b60) while the Blackened Temple node (quest id 19, slot 21) has its private byte +0xc clear; the byte is set when the Orb is smashed and restored on join from the Khalim's Will slot (18) bit 0. Coming from level 101 is always allowed (CheckActThreeWarp)"},
	{From: 66, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 67, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 68, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 69, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 70, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 71, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 72, To: 73, Act: 2, Quest: 6, Verified: true, Why: "warp into Duriel's lair refused while the Seven Tombs node (id 13, slot 14) is active and its private byte +0xb is clear (0x59b700); that the Horadric Staff in the orifice sets the byte was not traced"},
	{From: 4, To: 38, Act: 1, Quest: 4, Verified: true, Why: "the Search for Cain code makes the portal when the stones are solved (0x590b92)"},
	{From: 109, To: 121, Act: 5, Quest: 4, Verified: true, Why: "Betrayal of Harrogath code makes Anya's portal (0x588ea9, 0x589b7d)"},
	{From: 120, To: 118, Act: 5, Quest: 5, Verified: true, Why: "warp from Arreat Summit refused until the Rite of Passage node (id 35) private byte +0 is set (0x58ae70)"},
	{From: 120, To: 128, Act: 5, Quest: 5, Verified: true, Why: "same Rite of Passage check as 120 -> 118 (0x58ae70)"},
	{From: 131, To: 132, Act: 5, Quest: 6, Verified: true, Why: "warp into the Worldstone Chamber refused unless the Eve of Destruction node (id 36) private byte +0x86 is 1 (0x58c3f0)"},
	// Level features that are gated inside a level rather than on a link: To is the
	// level, From 0, Why starts with the feature.
	{To: 107, Act: 4, Quest: 3, Verified: true,
		Why: "Hellforge object (objects 376 operates quest id 24, quests.md)"},
	// River of Flame -> Chaos Sanctuary needs no quest: level 108 is in neither
	// the warp-tile gate list (0x543a70) nor Levels.txt QuestFlag (VERIFIED). The
	// seals only call up Diablo (D2MOO, UNVERIFIED in the exe); Terror's End just
	// watches levels 0x67 and 0x6c = 108 (0x5b2ad0). This entry is a feature
	// marker, not an entry gate.
	{To: 108, Act: 4, Quest: 2, Why: "Chaos Sanctuary seals: feature marker only, not an entry gate; Terror's End watches level 0x6c (0x5b2ad0)"},
}

// slotQuest finds the (act, quest) of a record slot.
func slotQuest(slot int) (act, quest int, ok bool) {
	for a := 1; a <= NumActs; a++ {
		for q := 0; q <= d2s.QuestCount(a); q++ {
			if s, found := d2s.QuestSlot(a, q); found && s == slot {
				return a, q, true
			}
		}
	}

	return 0, 0, false
}

// Gates lists every gate: one per flagged level from Levels.txt (From 0) plus
// the extra ones.
func Gates() []Gate {
	var out []Gate

	for level := 1; level <= 136; level++ {
		f, ok := levelQuestFlag[level]
		if !ok {
			continue
		}

		act, q, found := slotQuest(f[0])
		g := Gate{To: level, Slot: f[0], Why: "Levels.txt QuestFlag", Verified: true}

		if found {
			g.Act, g.Quest = act, q
		}

		out = append(out, g)
	}

	for _, g := range extraGates {
		if s, ok := d2s.QuestSlot(g.Act, g.Quest); ok {
			g.Slot = s
		}

		out = append(out, g)
	}

	return out
}

// GatesOf returns the gates on the move from -> to. A level-wide gate (From 0)
// applies to every move into the level from a level that has not the same
// flag, so walking inside the palace or the Durance is free.
func GatesOf(from, to int) []Gate {
	var out []Gate

	for _, g := range Gates() {
		if g.To != to {
			continue
		}

		switch {
		case g.From == from && from != 0:
			out = append(out, g)
		case g.From == 0 && g.Why == "Levels.txt QuestFlag":
			if LevelQuestFlag(from, false) != g.Slot {
				out = append(out, g)
			}
		}
	}

	return out
}

// TownPortalDestination is where a town portal cast in a level leads: the town
// of that level's act (0 for ids outside the game). VERIFIED: the cast routine
// (0x5bbe10) asks DRLG_GetActStartLevel (0x61a8b0, table 0x6e92c8 = 1, 40, 75,
// 103, 109) for the act of the room's level.
func TownPortalDestination(level int) int {
	return ActStartLevel(ActOfLevel(level))
}

// TownPortalCastable reports whether the town portal scroll/tome works in the
// level. VERIFIED (0x5bbe10): not in a town room and not in level 136
// (Pandemonium Finale, 0x88).
func TownPortalCastable(level int) bool {
	return level >= 1 && level <= 136 && !IsTown(level) && level != 136
}
