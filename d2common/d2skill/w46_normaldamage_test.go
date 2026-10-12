package d2skill

import "testing"

// COMBAT_RollPhysicalDamage 0x579120 (VERIFIED wave 46) adds item_normaldamage (stat 0x6f) to both ends of the
// weapon range before the floors and the percent pool; the roll is 8.8 fixed point.
func TestStrikeAddsNormalDamage(t *testing.T) {
	cases := []struct {
		name         string
		wmin, wmax   int
		flat, pctDmg int
		want         int32 // roll 0 -> the minimum, 8.8
	}{
		{"none", 10, 20, 0, 0, 10 << 8},
		{"flat before percent", 10, 20, 5, 100, 30 << 8},
		{"flat lifts a zero min", 0, 0, 3, 0, 3 << 8},
		{"negative flat floors at one", 2, 4, -10, 0, 1 << 8},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := newUnit(&Registry{}, nil)
			u.wmin, u.wmax = tc.wmin, tc.wmax
			u.stats["item_normaldamage"] = tc.flat
			u.stats["damagepercent"] = tc.pctDmg

			tg := &testTarget{id: "m", alive: true, level: 1}
			mr := (&Pipeline{}).strike(u, &Skill{}, 1, tg, nil, strikeOpt{autoHit: true})

			if mr.Damage.Physical != tc.want {
				t.Fatalf("physical = %d (%.2f), want %d", mr.Damage.Physical, float64(mr.Damage.Physical)/256, tc.want)
			}
		})
	}
}

// Missiles and thrown weapons never read stat 0x6f (0x64ca60, 0x4e7470 VERIFIED wave 46).
func TestMissileWeaponIgnoresNormalDamage(t *testing.T) {
	u := newUnit(&Registry{}, nil)
	u.wmin, u.wmax = 10, 20
	u.stats["item_normaldamage"] = 50

	if lo, hi := missileWeapon(u); lo != 10 || hi != 20 {
		t.Fatalf("missile weapon range %d-%d, want 10-20", lo, hi)
	}
}
