package d2drop

import "testing"

// The exe's ITEMGEN_DropFromTreasureClass (0x558d80) skips the level upgrade
// for classes with monstats flag bit 2 (noRatio) or bit 6 (boss).
func TestMonsterUpgradeLevelBossAndNoRatio(t *testing.T) {
	cases := []struct {
		name            string
		exp             bool
		diff            int
		noRatio, boss   bool
		level, wantUpTo int
	}{
		{"plain nightmare", true, 1, false, false, 40, 40},
		{"plain hell", true, 2, false, false, 70, 70},
		{"plain normal", true, 0, false, false, 20, 0},
		{"classic", false, 2, false, false, 70, 0},
		{"noRatio", true, 2, true, false, 70, 0},
		{"boss", true, 2, false, true, 70, 0},
		{"boss nightmare", true, 1, false, true, 40, 0},
	}
	for _, c := range cases {
		got := MonsterUpgradeLevel(c.exp, c.diff, true, MonStatsFlags(c.noRatio, c.boss), c.level)
		if got != c.wantUpTo {
			t.Errorf("%s: got %d want %d", c.name, got, c.wantUpTo)
		}
	}
}
