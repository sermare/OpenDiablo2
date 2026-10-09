package d2quest

import "strings"

// Boss kills of the later acts: Duriel (A2Q6, The Seven Tombs), Mephisto (A3Q6,
// The Guardian), Diablo (A4Q2, Terror's End) and Baal (A5Q6, Eve of
// Destruction). Andariel (A1Q6) lives in a1q5q6.go. The nodes below only
// react to the boss's death (event 8); the rest of those quests (the talks,
// the Horadric Staff, the Hellforge...) is not modelled yet.
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
// Nodes have no log index: the quest log panel has no text for them yet.

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
	NPCDiablo   = 243
	NPCBaalCrab = 544 // exe class of the Baal who dies in the Worldstone Chamber (row 545 "Baal Crab")
)

type bossQuest struct {
	id, slot, act int
	name, label   string
	// matches reports whether the kill is the boss.
	matches func(e *Event) bool
	// extra sets quest specific bits.
	extra func(g *Game, q *Quest)
}

var bossQuests = []bossQuest{
	{QuestSevenTombs, SlotSevenTombs, 1, "The Seven Tombs", "A2Q6",
		func(e *Event) bool { return e.Monster == NPCDuriel || nameIs(e, "duriel") },
		func(g *Game, q *Quest) {
			// VERIFIED (0x59ac40): bit 5 when none of 0, 3, 4, 5 is set
			if !g.get(q, FlagRewardGranted) && !g.get(q, FlagLeaveTown) && !g.get(q, FlagEnterArea) && !g.get(q, FlagCustom1) {
				g.set(q, FlagCustom1, "Duriel killed (bit 5)")
			}
		}},
	{QuestGuardian, SlotGuardian, 2, "The Guardian", "A3Q6",
		func(e *Event) bool { return e.Monster == NPCMephisto || nameIs(e, "mephisto") }, nil},
	{QuestTerrorsEnd, SlotTerrorsEnd, 3, "Terror's End", "A4Q2",
		func(e *Event) bool { return e.Monster == NPCDiablo || nameIs(e, "diablo") }, nil},
	{QuestEveOfDestruction, SlotEveOfDestruction, 4, "Eve of Destruction", "A5Q6",
		func(e *Event) bool { return e.Monster == NPCBaalCrab || nameIs(e, "baal") }, nil},
}

func nameIs(e *Event, name string) bool { return strings.EqualFold(strings.TrimSpace(e.Name), name) }

func newBossQuest(b bossQuest) *Quest {
	q := &Quest{ID: b.id, Slot: b.slot, Act: b.act, Name: b.name, Label: b.label, Active: true, NotIntro: true,
		SeqID: -1}

	q.on[EvMonsterKilled] = func(g *Game, q *Quest, e *Event) {
		if !b.matches(e) {
			return
		}

		if !g.get(q, FlagRewardGranted) && !g.get(q, FlagRewardPending) {
			g.set(q, FlagPrimaryGoal, b.name+" killed")
			g.set(q, FlagRewardPending, b.name+" killed")
		}

		if b.extra != nil {
			b.extra(g, q)
		}

		g.globalDone(q)
		g.setState(q, 1)
	}

	return q
}

// BossQuests returns the nodes of the boss quests.
func newBossQuests() []*Quest {
	out := make([]*Quest, 0, len(bossQuests))
	for _, b := range bossQuests {
		out = append(out, newBossQuest(b))
	}

	return out
}
