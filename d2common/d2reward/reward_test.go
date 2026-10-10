package d2reward

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

func TestApply(t *testing.T) {
	var s State

	tests := []struct {
		code  string
		value int
		check func(o Outcome) bool
	}{
		{"stat-points", 5, func(o Outcome) bool { return o.StatPoints == 5 }},
		{"life-boost", 20, func(o Outcome) bool { return o.LifeBonus == 20 }},
		{"resist-bonus", 10, func(o Outcome) bool { return o.ResistBonus == 10 }},
		{"socket-quest", 1, func(o Outcome) bool { return o.PendingSocket }},
		{"personalize", 1, func(o Outcome) bool { return o.PendingPersonalize }},
		{"hire-ironwolves", 0, func(o Outcome) bool { return o.Hire == "ironwolves" }},
		{"hire-barbarians", 0, func(o Outcome) bool { return o.Hire == "barbarians" }},
		{"unlock-difficulty", 0, func(o Outcome) bool { return o.UnlockDifficulty }},
		{"game-complete", 0, func(o Outcome) bool { return o.GameComplete }},
	}

	for _, tt := range tests {
		if o := s.Apply(d2quest.Effect{Kind: d2quest.EffectReward, Code: tt.code, Value: tt.value}); !tt.check(o) || o.Log == "" {
			t.Errorf("%s: %+v", tt.code, o)
		}
	}

	if s.StatPoints != 5 || s.LifeBonus != 20 || s.ResistBonus != 10 || s.SocketPending != 1 || s.PersonalizePending != 1 ||
		!s.Hired["ironwolves"] || !s.Hired["barbarians"] || !s.DifficultyUnlocked || !s.Complete {
		t.Errorf("state %+v", s)
	}

	// the three difficulties' scrolls add up
	s.Apply(d2quest.Effect{Code: "resist-bonus", Value: 10})
	s.Apply(d2quest.Effect{Code: "resist-bonus", Value: 10})

	if s.ResistBonus != 30 {
		t.Errorf("resist bonus %d", s.ResistBonus)
	}

	if o := s.Apply(d2quest.Effect{Code: "nonsense"}); o.Log == "" || o.StatPoints != 0 {
		t.Errorf("unknown reward %+v", o)
	}
}

func TestQuestDrops(t *testing.T) {
	none := func(string) bool { return false }
	both := func(string) bool { return true }

	if d := DropsFor("Hephasto the Armorer", none, false); len(d) != 1 || d[0].Code != "hfh" {
		t.Errorf("hephasto %+v", d)
	}

	if d := DropsFor("Mephisto", none, false); len(d) != 1 || d[0].Code != "mss" {
		t.Errorf("mephisto %+v", d)
	}

	if d := DropsFor("Mephisto", both, false); len(d) != 0 {
		t.Errorf("second soulstone %+v", d)
	}

	if d := DropsFor("Mephisto", none, true); len(d) != 0 {
		t.Errorf("after the quest %+v", d)
	}

	if d := DropsFor("Andariel", none, false); len(d) != 0 {
		t.Errorf("andariel %+v", d)
	}

	if _, ok := HellforgeSmash(none); ok {
		t.Error("smashed without items")
	}

	if c, ok := HellforgeSmash(both); !ok || len(c) != 1 || c[0] != "mss" {
		t.Errorf("smash %v %v", c, ok)
	}
}
