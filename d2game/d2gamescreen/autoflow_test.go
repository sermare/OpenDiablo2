package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
)

func TestNewHeroProblem(t *testing.T) {
	free := func(string) bool { return false }
	used := func(n string) bool { return n == "Taken" }

	tests := []struct {
		name      string
		hero      d2enum.Hero
		expansion bool
		taken     func(string) bool
		wantBad   bool
	}{
		{"", d2enum.HeroSorceress, true, free, false}, // empty: no message, OK stays disabled
		{"x", d2enum.HeroSorceress, true, free, true},
		{"Flowy", d2enum.HeroSorceress, true, free, false},
		{"Flow y", d2enum.HeroSorceress, true, free, true},
		{"Flow1", d2enum.HeroSorceress, true, free, true},
		{"-Flow", d2enum.HeroSorceress, true, free, true},
		{"Fl-ow", d2enum.HeroSorceress, true, free, false},
		{"Fl-o-w", d2enum.HeroSorceress, true, free, true},
		{"ABCDEFGHIJKLMNOP", d2enum.HeroSorceress, true, free, true},
		{"Druid", d2enum.HeroDruid, false, free, true},
		{"Druid", d2enum.HeroDruid, true, free, false},
		{"Taken", d2enum.HeroPaladin, true, used, true},
	}

	for _, tt := range tests {
		got := newHeroProblem(tt.name, tt.hero, tt.expansion, tt.taken)
		if (got != "") != tt.wantBad {
			t.Errorf("newHeroProblem(%q, %v, %v) = %q, want problem=%v", tt.name, tt.hero, tt.expansion, got, tt.wantBad)
		}
	}
}

func TestAutoFlowSteps(t *testing.T) {
	autoFlow = autoFlowState{loaded: true, steps: []string{"single", "create:Sorceress:Flowy:hardcore", "wait:2", "exit"}}

	if _, ok := autoFlowNext(0.5, "single"); ok {
		t.Fatal("a step must wait for the screen to settle")
	}

	if args, ok := autoFlowNext(2, "play", "single"); !ok || args[0] != "single" {
		t.Fatalf("single not offered: %v %v", args, ok)
	}

	if _, ok := autoFlowNext(0, "create"); ok {
		t.Fatal("create offered while single is the head")
	}

	autoFlowDone()

	if _, ok := autoFlowNext(2, "create"); !ok {
		t.Fatal("create not offered")
	}

	autoFlowDone()

	if autoFlowWait(1, "2") || !autoFlowWait(1, "2") {
		t.Fatal("wait:2 should need two seconds")
	}

	autoFlowDone()
	autoFlowDone()

	if autoFlowActive() {
		t.Fatal("flow should be finished")
	}

	if _, ok := heroByName("sorceress"); !ok {
		t.Fatal("heroByName")
	}
}
