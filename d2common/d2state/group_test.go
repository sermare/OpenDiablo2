package d2state

import (
	"reflect"
	"testing"
)

// Pins 0x56a480 (buff casts) and 0x56c740 (timed statlist): the States.txt
// group is cleared by ClearGroup, never by Apply/ApplyTimed.
func groupDefs() Defs {
	return Defs{
		"frozenarmor": {Name: "frozenarmor", Group: 1}, "shiverarmor": {Name: "shiverarmor", Group: 1},
		"chillingarmor": {Name: "chillingarmor", Group: 1}, "justhit": {Name: "justhit", Group: 1},
		"quickness": {Name: "quickness", Group: 2}, "fade": {Name: "fade", Group: 2},
		"might": {Name: "might"},
	}
}

func TestApplyDoesNotUseGroup(t *testing.T) {
	s := New()
	s.SetDefs(groupDefs())
	s.Apply(0, Instance{Name: "frozenarmor"})
	s.Apply(1, Instance{Name: "shiverarmor"})
	s.Apply(2, Instance{Name: "quickness"})
	s.Apply(3, Instance{Name: "fade"})

	if got := s.Names(4); !reflect.DeepEqual(got, []string{"fade", "frozenarmor", "quickness", "shiverarmor"}) {
		t.Errorf("Apply must not end same-group states: %v", got)
	}
}

func TestClearGroup(t *testing.T) {
	s := New()
	s.SetDefs(groupDefs())
	s.Apply(0, Instance{Name: "frozenarmor"})
	s.Apply(0, Instance{Name: "justhit"})
	s.Apply(0, Instance{Name: "quickness"})
	s.Apply(0, Instance{Name: "might"})

	// casting Shiver Armor ends the group-1 states
	if !s.ClearGroup(1, "shiverarmor") {
		t.Fatal("nothing ended")
	}

	s.Apply(1, Instance{Name: "shiverarmor"})

	if got := s.Names(2); !reflect.DeepEqual(got, []string{"might", "quickness", "shiverarmor"}) {
		t.Errorf("after Shiver Armor: %v", got)
	}

	// the same state is ended too, so a recast rebuilds it
	if !s.ClearGroup(2, "shiverarmor") || s.Active(2, "shiverarmor") {
		t.Error("own state not ended")
	}

	// Burst of Speed ends Fade
	s.Apply(2, Instance{Name: "fade"})
	s.ClearGroup(3, "quickness")

	if s.Active(3, "fade") || s.Active(3, "quickness") {
		t.Errorf("group 2 not ended: %v", s.Names(3))
	}

	// group 0 and unknown states do nothing
	if s.ClearGroup(3, "might") || s.ClearGroup(3, "nosuch") || !s.Active(3, "might") {
		t.Error("group 0 must be a no-op")
	}
}

func TestDefensePctSumsListsOfAllStates(t *testing.T) {
	s := New() // no ClearGroup: both lists exist, the exe adds stat 171 of every list
	s.Apply(0, Instance{Name: "frozenarmor", Mods: []StatMod{{Stat: "skill_armor_percent", Value: 30}}})
	s.Apply(0, Instance{Name: "shiverarmor", Mods: []StatMod{{Stat: "skill_armor_percent", Value: 45}}})

	if got := s.DefensePct(1); got != 75 {
		t.Errorf("defense pct = %d, want 75", got)
	}
}
