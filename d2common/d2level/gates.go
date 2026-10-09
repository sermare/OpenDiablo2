package d2level

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"

// gates.go: which level links need a quest, and the links that are portals
// rather than warp tiles. The level graph audit (audit_test.go) checks every
// table here against Levels.txt.
//
// The QuestFlag / QuestFlagEx columns of Levels.txt name a quest RECORD SLOT
// (the same slot numbers as d2s.QuestSlot): the level only loads once that slot
// shows the quest done. The values match the slots of quests.md: 11 = The
// Tainted Sun, 13 = The Summoner, 21 = The Blackened Temple, 26 = Terror's End
// (classic Cow Level), 39 = Rite of Passage, 40 = Eve of Destruction (expansion
// Cow Level). That reading is an inference from the numbers (it fits all of
// them) and UNVERIFIED in the binary.

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
// portal. All are Source notes.
//
//   - 1 -> 39: the Cow Level portal is opened in the Rogue Encampment (play rule).
//   - 4 -> 38: the Cairn Stones portal in Stony Field leads to Tristram.
//   - 54 -> 74: Palace Cellar 3 holds the portal to the Arcane Sanctuary (the
//     quest record asks for level 74; where the portal stands is a play rule).
//   - 74 -> 46: the Summoner's death opens the portal to the Canyon of the
//     Magi; Levels.txt gives the Canyon QuestFlag 13 (The Summoner), and no
//     warp leads there.
//   - 66..72 -> 73: the Horadric orifice of the true tomb opens the way to
//     Duriel's Lair. Which of the seven is the true tomb depends on the seed
//     (tombA of the act), so every tomb is listed.
//   - 109 -> 121: Anya's red portal to Nihlathak's Temple (play rule; the
//     Betrayal of Harrogath quest names level 0x79 = 121, quests.md).
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
		Why: "the stairs stay sealed until Khalim's Will smashes the Compelling Orb (CheckActThreeWarp)"},
	{From: 66, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 67, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 68, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 69, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 70, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 71, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 72, To: 73, Act: 2, Quest: 2, Why: "Horadric Staff in the orifice (play rule)"},
	{From: 4, To: 38, Act: 1, Quest: 4, Why: "the Cairn Stones open once the Search for Cain is under way (play rule)"},
	{From: 109, To: 121, Act: 5, Quest: 4, Why: "Anya opens the portal for Betrayal of Harrogath (play rule; Prison of Ice comes first)"},
	// Level features that are gated inside a level rather than on a link: To is the
	// level, From 0, Why starts with the feature.
	{To: 107, Act: 4, Quest: 3, Verified: true,
		Why: "Hellforge object (objects 376 operates quest id 24, quests.md)"},
	{To: 108, Act: 4, Quest: 2, Why: "Chaos Sanctuary seals: Terror's End names levels 0x67 and 0x6c = 108 (quests.md); that the seals themselves use this quest's record is UNVERIFIED"},
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
// of that level's act (0 for ids outside the game). The original opens the
// portal to the act's start level; this is the play rule, UNVERIFIED in the
// binary.
func TownPortalDestination(level int) int {
	return ActStartLevel(ActOfLevel(level))
}
