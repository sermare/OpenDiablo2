package d2statlist

import "testing"

// TestResistClassicPenalty: classic uses -20/-50, expansion -40/-100; physical
// and magic resist are exempt; the floor is -100 and the cap 75 + max stat.
func TestResistClassicPenalty(t *testing.T) {
	it := Item{Slot: SlotAmulet, Props: []Prop{
		{ID: StatFireResist, Value: 80}, {ID: StatColdResist, Value: 30},
		{ID: StatDamageResist, Value: 20}, {ID: StatMagicResist, Value: 20},
	}}

	tests := []struct {
		classic    bool
		diff       int
		fire, cold int
	}{
		{true, 0, 75, 30},
		{true, 1, 60, 10},
		{true, 2, 30, -20},
		{false, 1, 40, -10},
		{false, 2, -20, -70},
	}

	for _, tc := range tests {
		h := hero()
		h.Difficulty, h.Classic = tc.diff, tc.classic
		got := Compute(h, []Item{it}, nil)

		if got.ResistShown[ResFire] != tc.fire || got.ResistShown[ResCold] != tc.cold {
			t.Errorf("%+v: fire %d cold %d", tc, got.ResistShown[ResFire], got.ResistShown[ResCold])
		}

		if got.PhysResist != 20 || got.MagicResist != 20 {
			t.Errorf("%+v: phys %d magic %d should ignore the penalty", tc, got.PhysResist, got.MagicResist)
		}
	}
}
