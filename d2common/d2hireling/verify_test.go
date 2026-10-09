package d2hireling

import "testing"

// Rules verified in Game.exe (see gate.go and stats.go for the addresses).

func TestVerifyLevelCaps(t *testing.T) {
	tab := loadReal(t)

	if MaxLevel != 98 || MaxLoadLevel != 99 {
		t.Fatalf("caps %d/%d", MaxLevel, MaxLoadLevel)
	}

	for id := 0; id < 30; id++ {
		huge := uint32(0xffffffff)

		// a save with a huge experience loads at 99 ...
		if got := tab.LevelFromExp(id, huge); got != 99 {
			t.Errorf("id %d: load level %d, want 99", id, got)
		}

		// ... but experience can never take a merc past 98
		if got := tab.LevelAfterGain(id, 90, huge); got != 98 {
			t.Errorf("id %d: gain level %d, want 98", id, got)
		}

		if s, _ := tab.StatsFor(id, 98); s.NextXP != 0 {
			t.Errorf("id %d: next xp at 98 = %d, want 0", id, s.NextXP)
		}

		if s, _ := tab.StatsFor(id, 97); s.NextXP == 0 {
			t.Errorf("id %d: next xp at 97 is 0", id)
		}
	}
}

func TestVerifyLevelAfterGain(t *testing.T) {
	tab := loadReal(t)

	r := tab.Find(6, 20)
	need := ExpThreshold(23, r.ExpPerLvl)

	tests := []struct {
		exp  int64
		want int
	}{
		{ExpThreshold(21, r.ExpPerLvl) - 1, 20},
		{ExpThreshold(21, r.ExpPerLvl), 21},
		{need - 1, 22},
		{need, 23},
	}

	for _, tc := range tests {
		if got := tab.LevelAfterGain(6, 20, uint32(tc.exp)); got != tc.want {
			t.Errorf("exp %d: level %d, want %d", tc.exp, got, tc.want)
		}
	}
}

func TestVerifyHireGate(t *testing.T) {
	done := func(slots ...int) func(int) bool {
		return func(s int) bool {
			for _, x := range slots {
				if x == s {
					return true
				}
			}

			return false
		}
	}

	tests := []struct {
		name              string
		seller, diff, lvl int
		done              func(int) bool
		want              bool
	}{
		{"kashya level 7 needs burial", SellerKashya, 0, 7, done(), false},
		{"kashya level 7 burial done", SellerKashya, 0, 7, done(QuestSlotBurial), true},
		{"kashya level 8 free", SellerKashya, 0, 8, done(), true},
		{"kashya hell level 1", SellerKashya, 2, 1, done(), false},
		{"qual-kehk needs rescue", SellerQualKehk, 0, 90, done(QuestSlotBurial), false},
		{"qual-kehk rescue done", SellerQualKehk, 1, 10, done(QuestSlotRescue), true},
		{"greiz no gate", 198, 0, 1, done(), true},
		{"asheara no gate", 252, 2, 1, done(), true},
	}

	for _, tc := range tests {
		if got := HireAllowed(tc.seller, tc.diff, tc.lvl, tc.done); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestVerifyGateLevelAndRevive(t *testing.T) {
	tests := []struct{ diff, idx, lvl, want int }{
		{0, 0, 30, 12}, {0, 1, 30, 20}, {0, 2, 30, 28}, {0, 3, 40, 36}, {0, 4, 99, 45},
		{0, 4, 10, 10}, {1, 0, 30, 30}, {0, 5, 30, 30}, {0, -1, 30, 30},
	}

	for _, tc := range tests {
		if got := GateLevel(tc.diff, tc.idx, tc.lvl); got != tc.want {
			t.Errorf("GateLevel(%d,%d,%d) = %d, want %d", tc.diff, tc.idx, tc.lvl, got, tc.want)
		}
	}

	for _, c := range []int{150, 198, 252, 367, 515} {
		if !CanRevive(c) {
			t.Errorf("class %d should revive", c)
		}
	}

	for _, c := range []int{148, 251, 0, 0x203 + 1} {
		if CanRevive(c) {
			t.Errorf("class %d should not revive", c)
		}
	}
}

func TestVerifyBossDamage(t *testing.T) {
	tests := []struct{ dmg, pct, want int }{{100, 50, 50}, {100, 35, 35}, {100, 25, 25}, {7, 25, 1}, {0, 25, 0}}

	for _, tc := range tests {
		if got := BossDamage(tc.dmg, tc.pct); got != tc.want {
			t.Errorf("BossDamage(%d,%d) = %d, want %d", tc.dmg, tc.pct, got, tc.want)
		}
	}
}
