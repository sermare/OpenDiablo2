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

func TestLarzukSockets(t *testing.T) {
	base := SocketItem{Code: "7cr", ItemLevel: 30, BaseSockets: 4, MaxSock1: 3, MaxSock25: 4, MaxSock40: 6}

	tests := []struct {
		name string
		mod  func(i *SocketItem)
		diff int
		want int
		err  bool
	}{
		{"normal difficulty capped at 3", func(i *SocketItem) {}, 0, 3, false},
		{"nightmare gives the bracket maximum", func(i *SocketItem) {}, 1, 4, false},
		{"low ilvl uses MaxSock1", func(i *SocketItem) { i.ItemLevel = 10 }, 2, 3, false},
		{"high ilvl limited by the base", func(i *SocketItem) { i.ItemLevel = 60 }, 2, 4, false},
		{"base with 6 slots in hell", func(i *SocketItem) { i.ItemLevel = 60; i.BaseSockets = 6 }, 2, 6, false},
		{"already socketed", func(i *SocketItem) { i.Sockets = 2 }, 2, 0, true},
		{"no socket slots", func(i *SocketItem) { i.BaseSockets = 0 }, 2, 0, true},
		{"quest item", func(i *SocketItem) { i.Quest = true }, 2, 0, true},
		{"type allows none", func(i *SocketItem) { i.MaxSock1 = 0; i.ItemLevel = 5 }, 2, 0, true},
		{"difficulty out of range is clamped", func(i *SocketItem) { i.ItemLevel = 60; i.BaseSockets = 6 }, 9, 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			it := base
			tt.mod(&it)

			n, err := LarzukSockets(it, tt.diff)
			if (err != nil) != tt.err || n != tt.want {
				t.Fatalf("got %d, %v", n, err)
			}
		})
	}
}

func TestPersonalize(t *testing.T) {
	tests := []struct {
		it  PersonalizeItem
		err bool
	}{
		{PersonalizeItem{MagicOrBetter: true}, false},
		{PersonalizeItem{MagicOrBetter: true}, false},
		{PersonalizeItem{}, true},
		{PersonalizeItem{Nameable: true}, false},
		{PersonalizeItem{MagicOrBetter: true, Personalized: true}, true},
	}

	for i, tt := range tests {
		if err := CanPersonalize(tt.it); (err != nil) != tt.err {
			t.Errorf("case %d: %v", i, err)
		}
	}

	if got := PersonalName("Nokka", "Windforce"); got != "Nokka's Windforce" {
		t.Errorf("name %q", got)
	}

	if got := PersonalName(" ", "Windforce"); got != "Windforce" {
		t.Errorf("empty hero name: %q", got)
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
