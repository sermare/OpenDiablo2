package d2statlist

import "testing"

// TestComputeRegenStats: the regeneration stats of the worn items reach the
// totals that the natural regeneration reads (replenish life 74, regenerate
// mana percent 27, raw mana per frame 26), summed over the items.
func TestComputeRegenStats(t *testing.T) {
	ring := Item{Slot: SlotRingLeft, Props: []Prop{{ID: StatManaRecovery, Value: 20}, {ID: StatHPRegen, Value: 5}}}
	amulet := Item{Slot: SlotAmulet, Props: []Prop{{ID: StatManaRecovery, Value: 15}, {ID: StatHPRegen, Value: 10}}}
	stash := Item{Slot: 0, Props: []Prop{{ID: StatManaRecovery, Value: 99}, {ID: StatHPRegen, Value: 99}}}

	cases := []struct {
		name             string
		items            []Item
		pct, flat, regen int
	}{
		{"nothing worn", nil, 0, 0, 0},
		{"one ring", []Item{ring}, 20, 0, 5},
		{"ring and amulet add", []Item{ring, amulet}, 35, 0, 15},
		{"an item that is not worn gives nothing", []Item{ring, stash}, 20, 0, 5},
	}

	for _, c := range cases {
		got := Compute(hero(), c.items, nil)
		if got.ManaRecoveryPct != c.pct || got.ManaRecoveryRaw != c.flat || got.LifeRegen != c.regen {
			t.Errorf("%s: pct=%d flat=%d regen=%d, want %d %d %d", c.name,
				got.ManaRecoveryPct, got.ManaRecoveryRaw, got.LifeRegen, c.pct, c.flat, c.regen)
		}
	}
}
