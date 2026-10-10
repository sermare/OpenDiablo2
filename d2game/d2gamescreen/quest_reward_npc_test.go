package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

func rewardTestGame() (*Game, *d2s.QuestRecord) {
	var rec d2s.QuestRecord

	g := d2quest.New(&rec, &d2s.NPCBlock{}, 0)
	g.Start()

	return &Game{questRT: &questRuntime{g: g}}, &rec
}

func actions(rows []d2player.NPCMenuRow) []d2player.NPCMenuAction {
	var out []d2player.NPCMenuAction
	for _, r := range rows {
		out = append(out, r.Action)
	}

	return out
}

func TestRewardRows(t *testing.T) {
	v, rec := rewardTestGame()
	base := []d2player.NPCMenuRow{{Action: d2player.NPCActionTalk}}

	// nothing is owed: the menus are the table's rows
	for _, class := range []int{d2quest.NPCAkara, d2quest.NPCLarzuk, d2quest.NPCDrehya, d2quest.NPCCharsi, d2quest.NPCKashya} {
		if got := actions(v.withRewardRows(class, base)); len(got) != 1 {
			t.Errorf("class %d with nothing owed: %v", class, got)
		}
	}

	// Akara: the reset row follows the slot-41 bit (the Den of Evil claim sets it, a save keeps it)
	rec.Set(d2s.QuestSlotAkaraRespec, d2quest.FlagRewardPending)

	if got := actions(v.withRewardRows(d2quest.NPCAkara, base)); len(got) != 2 || got[1] != d2player.NPCActionRespec {
		t.Errorf("Akara with the reset available: %v", got)
	}

	if got := actions(v.withRewardRows(d2quest.NPCCharsi, base)); len(got) != 1 {
		t.Errorf("Charsi has no reset row: %v", got)
	}

	// Larzuk and Anya: a row while sockets / personalisation are owed
	v.questRT.rewards.SocketPending = 1
	v.questRT.rewards.PersonalizePending = 1

	for _, class := range []int{d2quest.NPCLarzuk, d2quest.NPCDrehya} {
		got := v.withRewardRows(class, base)
		if len(got) != 2 || got[1].Action != d2player.NPCActionReward || got[1].Fallback == "" {
			t.Errorf("class %d with a reward owed: %+v", class, got)
		}
	}

	// the other way round: Larzuk's reward is not Anya's
	v.questRT.rewards.PersonalizePending = 0

	if got := v.withRewardRows(d2quest.NPCDrehya, base); len(got) != 1 {
		t.Errorf("Anya owes nothing: %+v", got)
	}

	// Charsi: the imbue is owed from the moment the Malus is handed in (reward pending in slot 3) until the claim
	if v.imbueOwed() {
		t.Error("imbue owed on a fresh record")
	}

	rec.Set(3, d2quest.FlagRewardPending)

	if got := actions(v.withRewardRows(d2quest.NPCCharsi, base)); len(got) != 2 || got[1] != d2player.NPCActionReward {
		t.Errorf("Charsi with the imbue owed: %v", got)
	}

	rec.Set(3, d2quest.FlagRewardGranted)
	rec.Clear(3, d2quest.FlagRewardPending)

	if v.imbueOwed() {
		t.Error("imbue still owed after the claim")
	}
}

func TestRewardRowsWithoutQuests(t *testing.T) {
	v := &Game{}

	// no quest system (title screen, tests): no rows, no panic
	if v.respecAvailable() || v.imbueOwed() {
		t.Error("rewards without a quest system")
	}
}
