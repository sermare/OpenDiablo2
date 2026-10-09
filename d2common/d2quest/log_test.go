package d2quest

import "testing"

func TestLogSeenSetsTheUpdateBit(t *testing.T) {
	g, _ := newGame(t)
	den := g.Quest(QuestDenOfEvil)

	g.Rec.Set(1, FlagRewardGranted) // as if Akara's reward had been taken

	var entry QuestLog

	for _, l := range g.Log() {
		if l.Act == 1 && l.Index == 1 {
			entry = l
		}
	}

	if entry.Status != LogCompleted || !entry.Unseen {
		t.Fatalf("a completed quest nobody looked at is unseen: %+v", entry)
	}

	g.LogSeen(1, 1)

	if !g.get(den, FlagUpdateLog) {
		t.Fatal("bit 12 not set")
	}

	for _, l := range g.Log() {
		if l.Act == 1 && l.Index == 1 && l.Unseen {
			t.Fatal("still unseen")
		}
	}

	// a quest that is not completed is not marked
	g.LogSeen(1, 2)

	if g.get(g.Quest(QuestBurial), FlagUpdateLog) {
		t.Fatal("an unfinished quest must not get the bit")
	}
}

func TestTraceKeepsBitsAcrossQuestsIndependent(t *testing.T) {
	g, _ := newGame(t)

	g.Quest(QuestBurial).State = 1
	talk(g, NPCAkara)
	talk(g, NPCKashya)

	if slot(g, QuestDenOfEvil) != 1<<FlagStarted || slot(g, QuestBurial) != 1<<FlagStarted {
		t.Fatalf("den=%#x burial=%#x", slot(g, QuestDenOfEvil), slot(g, QuestBurial))
	}

	if slot(g, QuestCain) != 0 || slot(g, QuestTools) != 0 {
		t.Fatal("other quests must stay untouched")
	}
}
