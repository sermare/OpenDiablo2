package d2statlist

import "testing"

// Bare hands: strength counts as a damage percent and the percent is floored
// at -90 (COMBAT_RollPhysicalDamage 0x579120, verified in the emulator).
func TestWeaponDamageUnarmedAndFloor(t *testing.T) {
	sword := func(b WeaponBase) []Item { return []Item{{Slot: SlotRightHand, Weapon: &b}} }

	cases := []struct {
		name     string
		active   []Item
		str, dex int
		dmgPct   int64
		wantMin  int
		wantMax  int
	}{
		{"unarmed no strength", nil, 0, 0, 0, 1, 2},
		{"unarmed 100 strength doubles 1-2", nil, 100, 0, 0, 2, 4},
		{"unarmed strength and percent", nil, 50, 0, 50, 2, 4},
		{"unarmed floor -90", nil, 0, 0, -200, 1, 2},
		{"sword str bonus", sword(WeaponBase{Min: 10, Max: 20, StrBonus: 100}), 50, 0, 0, 15, 30},
		{"bow dex bonus", sword(WeaponBase{Min: 10, Max: 20, DexBonus: 100}), 99, 30, 0, 13, 26},
		{"75/75 weapon", sword(WeaponBase{Min: 100, Max: 100, StrBonus: 75, DexBonus: 75}), 40, 20, 0, 145, 145},
		{"zero bonus weapon ignores stats", sword(WeaponBase{Min: 10, Max: 20}), 100, 100, 0, 10, 20},
	}

	for _, tc := range cases {
		list := &List{}
		if tc.dmgPct != 0 {
			list.Add(StatDamagePct, 0, tc.dmgPct)
		}

		gotMin, gotMax := weaponDamage(tc.active, list, tc.str, tc.dex)
		if gotMin != tc.wantMin || gotMax != tc.wantMax {
			t.Errorf("%s: %d-%d, want %d-%d", tc.name, gotMin, gotMax, tc.wantMin, tc.wantMax)
		}
	}
}
