package d2skill

import "testing"

func TestChargeStatesOf(t *testing.T) {
	reg := NewRegistry()
	reg.Add(&Skill{ID: 10, AuraState: "progressive_fire", SrvDoFunc: 34, PrgDam: 1, Params: [9]int{0, 12, 4}})
	reg.Add(&Skill{ID: 11, AuraState: "progressive_cold", SrvDoFunc: 99}) // not a builder

	got := ChargeStatesOf(reg, map[string]int{"progressive_fire": 2, "progressive_cold": 1, "progressive_damage": 0},
		func(id int) int { return map[int]int{10: 5}[id] })

	want := ChargeState{PrgDam: 1, Param1: 12, Param2: 4, Level: 5, Count: 2}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	// a builder the hero has no points in counts as level 1
	got = ChargeStatesOf(reg, map[string]int{"progressive_fire": 1}, nil)
	if len(got) != 1 || got[0].Level != 1 {
		t.Fatalf("level floor: %+v", got)
	}
}
