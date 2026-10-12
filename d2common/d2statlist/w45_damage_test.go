package d2statlist

import "testing"

// The character screen damage box (SKILLDESC_ComputeWeaponDamageRange 0x4e5860, VERIFIED): the unit range is floored
// at 1 / 2, then scaled by the percent pool (floored at -90), then the weapon's own item_normaldamage is added flat.
func TestWeaponDamageFlatAfterPercent(t *testing.T) {
	mk := func(b WeaponBase, props ...Prop) []Item {
		return []Item{{Slot: SlotRightHand, Weapon: &b, Props: props}}
	}

	cases := []struct {
		name         string
		active       []Item
		str          int
		dmgPct       int64
		wantMin, max int
	}{
		{"no flat", mk(WeaponBase{Min: 10, Max: 20}), 0, 0, 10, 20},
		{"flat after percent", mk(WeaponBase{Min: 10, Max: 20}, Prop{ID: StatNormalDamage, Value: 5}), 0, 100, 25, 45},
		{"flat of a huge runeword", mk(WeaponBase{Min: 10, Max: 20}, Prop{ID: StatNormalDamage, Value: 340}), 0, 0, 350, 360},
		{"percent floor -90 keeps a tenth", mk(WeaponBase{Min: 100, Max: 200}), 0, -300, 10, 20},
	}

	for _, tc := range cases {
		list := &List{}
		if tc.dmgPct != 0 {
			list.Add(StatDamagePct, 0, tc.dmgPct)
		}

		gotMin, gotMax := weaponDamage(tc.active, list, tc.str, 0)
		if gotMin != tc.wantMin || gotMax != tc.max {
			t.Errorf("%s: %d-%d, want %d-%d", tc.name, gotMin, gotMax, tc.wantMin, tc.max)
		}
	}
}

func TestWeaponPartsFlat(t *testing.T) {
	b := WeaponBase{Min: 3, Max: 7}
	active := []Item{{Slot: SlotRightHand, Weapon: &b, Props: []Prop{{ID: StatNormalDamage, Value: 12}}}}

	if wp := weaponParts(active, &List{}, 1, 0, 0); wp.Flat != 12 || wp.Min != 3 || wp.Max != 7 {
		t.Errorf("parts %+v", wp)
	}

	// another worn item's "+damage" is not the weapon's: the box reads the weapon item only
	ring := Item{Slot: SlotRingLeft, Props: []Prop{{ID: StatNormalDamage, Value: 9}}}

	if wp := weaponParts(append(active, ring), &List{}, 1, 0, 0); wp.Flat != 12 {
		t.Errorf("flat %d, want the weapon's 12", wp.Flat)
	}
}
