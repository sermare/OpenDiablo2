package d2hero

import "testing"

// TestLoadedGold is the differential check of the d2s loader against the
// original (0x531a50, VERIFIED): stored gold above level*10000 loads as 0, at
// or below the cap it is kept untouched.
func TestLoadedGold(t *testing.T) {
	tests := []struct {
		name   string
		stored uint64
		level  int
		want   int
	}{
		{"zero", 0, 1, 0},
		{"within", 9999, 1, 9999},
		{"at the cap", 10000, 1, 10000},
		{"over the cap", 10001, 1, 0},
		{"level 94 cap", 940000, 94, 940000},
		{"level 94 over", 940001, 94, 0},
		{"25-bit maximum, low level", 1<<25 - 1, 30, 0},
		{"absurd", 1 << 40, 99, 0},
	}

	for _, tc := range tests {
		if got := loadedGold(tc.stored, tc.level); got != tc.want {
			t.Errorf("%s: loadedGold(%d, %d) = %d, want %d", tc.name, tc.stored, tc.level, got, tc.want)
		}
	}
}

// TestRealSaveGoldSurvivesLoader makes sure the real level-94 oracle keeps its
// gold through the loader rule (so the byte-exact save-back roundtrip, which
// writes state.Gold back, still sees an unchanged value).
func TestRealSaveGoldSurvivesLoader(t *testing.T) {
	_, _, state := realSave(t)

	if got := loadedGold(uint64(state.Gold), state.Stats.Level); got != state.Gold {
		t.Errorf("the real save's gold %d (level %d) would be zeroed by the loader", state.Gold, state.Stats.Level)
	}
}
