package d2s

import (
	"os"
	"testing"
)

func TestQuestSlot(t *testing.T) {
	tests := []struct {
		act, quest, slot int
		ok               bool
	}{
		{1, 0, 0, true}, {1, 1, 1, true}, {1, 6, 6, true},
		{2, 0, 8, true}, {2, 1, 9, true}, {2, 6, 14, true},
		{3, 0, 16, true}, {3, 6, 22, true},
		{4, 0, 24, true}, {4, 3, 27, true}, {4, 4, 0, false},
		{5, 1, 35, true}, {5, 6, 40, true}, {5, 0, 0, false}, {5, 7, 0, false},
		{0, 1, 0, false}, {6, 1, 0, false}, {1, 7, 0, false},
	}
	for _, tc := range tests {
		slot, ok := QuestSlot(tc.act, tc.quest)
		if slot != tc.slot || ok != tc.ok {
			t.Errorf("QuestSlot(%d,%d) = %d,%v want %d,%v", tc.act, tc.quest, slot, ok, tc.slot, tc.ok)
		}
	}
}

func TestQuestRecordBits(t *testing.T) {
	var b Body

	q := b.QuestRecord(1)
	q.Set(5, 3)
	q.Set(5, 15)

	if b.Quests[1][10] != 0x08 || b.Quests[1][11] != 0x80 {
		t.Fatalf("little endian layout wrong: % x", b.Quests[1][8:14])
	}

	if !q.Get(5, 3) || q.Get(5, 4) || b.QuestRecord(0).Get(5, 3) {
		t.Fatal("Get mismatch")
	}

	q.Set(1, QuestBitRewardPending)
	q.Set(1, 2)
	q.Set(1, 11)
	q.Set(1, QuestBitUpdated)

	if q.Progress(1) != 0x201 {
		t.Fatalf("progress %x", q.Progress(1))
	}

	if !q.RewardPending(1, 1) || q.Completed(1, 1) {
		t.Fatal("reward pending state wrong")
	}

	q.SetCompleted(1, 1)

	if !q.Completed(1, 1) || q.RewardPending(1, 1) || q.Progress(1) != 0 || !q.Get(1, QuestBitUpdated) {
		t.Fatalf("SetCompleted: slot=%016b", q.Slot(1))
	}

	q.SetCompleted(5, 6)
	q.Set(QuestSlotAkaraRespec, QuestBitRewardPending)
	q.Set(QuestSlotAct2Finished, QuestBitDone)

	if q.CompletedCount(1) != 1 || q.CompletedCount(5) != 1 || !q.AkaraRespecAvailable() ||
		!q.ActFinished(2) || q.ActFinished(1) || b.QuestRecord(3) != nil {
		t.Fatal("helpers wrong")
	}
}

func TestWaypoints(t *testing.T) {
	var w Waypoints

	w.Set(2, WPTravincal, true)
	w.Set(2, WPHarrogath, true)

	if !w.Has(2, WPTravincal) || w.Has(1, WPTravincal) || w.Has(2, NumWaypoints) {
		t.Fatal("Has mismatch")
	}

	if got := w.ActiveInAct(2, 3); len(got) != 1 || got[0] != WPTravincal {
		t.Fatalf("act 3: %v", got)
	}

	if WPHarrogath.Act() != 5 || WPRiverOfFlame.Act() != 4 || WPRogueEncampment.Act() != 1 || len(WaypointsOfAct(4)) != 3 {
		t.Fatal("act mapping")
	}

	total := 0
	for a := 1; a <= NumActs; a++ {
		total += len(WaypointsOfAct(a))
	}

	if total != 39 {
		t.Fatalf("total %d", total)
	}

	w.Set(2, WPTravincal, false)

	if w.Has(2, WPTravincal) {
		t.Fatal("clear")
	}
}

func TestNPCBlockBits(t *testing.T) {
	var b Body

	n := b.NPCFlags()
	n.SetIntroBit(1, 9, true)
	n.SetReturnBit(2, 33, true)

	if b.NPC[2+8+1] != 0x02 || b.NPC[2+24+16+4] != 0x02 {
		t.Fatalf("layout: % x", b.NPC)
	}

	if !n.IntroBit(1, 9) || n.IntroBit(0, 9) || !n.ReturnBit(2, 33) || n.ReturnBit(2, 64) {
		t.Fatal("bit access")
	}

	n.SetIntroBit(1, 9, false)

	if n.Intro(1) != 0 {
		t.Fatal("clear")
	}
}

// TestSampleQuests reads a real save; NokkaSorc only uses bit 0 (plus 0x0080 on
// slot 37), so it is a weak reference.
func TestSampleQuests(t *testing.T) {
	path := os.Getenv("D2S_SAMPLE_BODY")
	if path == "" {
		t.Skip("set D2S_SAMPLE_BODY to run")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	body, err := ParseBody(data, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := body.QuestRecord(0)
	if q.Slot(37) != 0x0080 || !q.Completed(1, 1) {
		t.Errorf("unexpected quests: slot37=%#x", q.Slot(37))
	}

	for d := 0; d < 3; d++ {
		if body.Waypoints[d] != 0x7FFFFFFFFF {
			t.Errorf("difficulty %d waypoints %#x", d, body.Waypoints[d])
		}

		if body.NPCFlags().Intro(d) != 0 || body.NPCFlags().Return(d) != 0 {
			t.Errorf("difficulty %d NPC bits not zero", d)
		}
	}
}
