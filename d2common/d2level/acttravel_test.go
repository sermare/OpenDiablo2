package d2level

import (
	"errors"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

func TestCheckActTravel(t *testing.T) {
	done := func(slot int) *d2s.QuestRecord {
		q := &d2s.QuestRecord{}
		q.Set(slot, d2s.QuestBitDone)

		return q
	}

	tests := []struct {
		name      string
		from, to  int
		q         *d2s.QuestRecord
		expansion bool
		free      bool
		want      error
	}{
		{"1->2 without Andariel", 1, 2, &d2s.QuestRecord{}, true, false, ErrQuestNotDone},
		{"1->2 with Andariel", 1, 2, done(slotAndariel), false, false, nil},
		{"1->2 reward pending counts", 1, 2, func() *d2s.QuestRecord {
			q := &d2s.QuestRecord{}
			q.Set(slotAndariel, d2s.QuestBitRewardPending)
			return q
		}(), false, false, nil},
		{"2->3 without Duriel", 2, 3, done(slotAndariel), false, false, ErrQuestNotDone},
		{"2->3 with Duriel", 2, 3, done(slotDuriel), false, false, nil},
		{"3->4 portal needs Mephisto", 3, 4, &d2s.QuestRecord{}, false, false, ErrQuestNotDone},
		{"3->4 with Mephisto", 3, 4, done(slotMephisto), false, false, nil},
		{"4->5 classic", 4, 5, done(slotDiablo), false, false, ErrNeedLoD},
		{"4->5 expansion", 4, 5, done(slotDiablo), true, false, nil},
		{"4->5 expansion, Diablo alive", 4, 5, &d2s.QuestRecord{}, true, false, ErrQuestNotDone},
		{"2->1 always", 2, 1, &d2s.QuestRecord{}, false, false, nil},
		{"3->2 always", 3, 2, nil, false, false, nil},
		{"5->4 has no way", 5, 4, done(slotDiablo), true, false, ErrNoRoute},
		{"4->3 has no way", 4, 3, nil, true, false, ErrNoRoute},
		{"5->4 free", 5, 4, nil, false, true, nil},
		{"1->5 free", 1, 5, nil, false, true, nil},
		{"1->3 skips an act", 1, 3, nil, true, false, ErrNoRoute},
		{"same act", 2, 2, nil, true, true, ErrNoRoute},
	}

	for _, tc := range tests {
		_, err := CheckActTravel(tc.from, tc.to, tc.q, tc.expansion, tc.free)
		if !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.want)
		}
	}
}

func TestRuleForNPC(t *testing.T) {
	for class, want := range map[int][2]int{
		ClassWarriv1: {1, 2}, ClassWarriv2: {2, 1}, ClassMeshif1: {2, 3}, ClassMeshif2: {3, 2}, ClassTyrael2: {4, 5},
	} {
		r, ok := RuleForNPC(class)
		if !ok || r.From != want[0] || r.To != want[1] {
			t.Errorf("class %d: %+v ok=%v", class, r, ok)
		}
	}

	if _, ok := RuleForNPC(148); ok {
		t.Error("Akara must not travel")
	}

	if _, ok := RuleForNPC(0); ok {
		t.Error("class 0 must not match the portal rule")
	}
}

func TestMarkActFinished(t *testing.T) {
	q := &d2s.QuestRecord{}

	if got := MarkActFinished(q, 1); got != d2s.QuestSlotAct1Finished || !q.Get(7, d2s.QuestBitDone) {
		t.Fatalf("act 1: slot %d", got)
	}

	if MarkActFinished(q, 2) != 15 || MarkActFinished(q, 4) != 28 || MarkActFinished(q, 3) != -1 {
		t.Fatal("finished slots")
	}
}

func TestActStartLevelsOfRules(t *testing.T) {
	for _, r := range ActRules {
		if ActStartLevel(r.To) == 0 || !IsTown(ActStartLevel(r.To)) {
			t.Errorf("rule %d->%d has no town destination", r.From, r.To)
		}
	}
}
