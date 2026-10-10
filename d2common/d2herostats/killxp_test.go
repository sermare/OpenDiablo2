package d2herostats

import "testing"

func TestLevelScaleXP(t *testing.T) {
	for _, tc := range []struct {
		name         string
		xp, mlv, clv int
		want         int
	}{
		{"same level", 1000, 30, 30, 1000},
		{"char 5 above", 1000, 25, 30, 1000},
		{"char 6 above", 2560, 24, 30, 2070},
		{"char 7 above", 2560, 23, 30, 1590},
		{"char 10 above", 2560, 20, 30, 130},
		{"char 40 above caps at 10", 2560, 1, 41, 130},
		{"low char, monster 6 above", 2560, 16, 10, 2250},
		{"low char, monster 9 above", 2560, 19, 10, 380},
		{"low char, monster far above", 2560, 60, 10, 50},
		{"clvl 25 monster above is xp*c/m", 1000, 40, 25, 625},
		{"clvl 24 monster above uses table", 2560, 40, 24, 50},
		{"monster level 0 counts as 30 below", 100, 0, 30, 5},
	} {
		if got := LevelScaleXP(tc.xp, tc.mlv, tc.clv); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestKillXP(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		xp, m, c, max, pct, want int
	}{
		{"same level unchanged", 500, 40, 40, 99, 0, 500},
		{"item bonus", 500, 40, 40, 99, 20, 600},
		{"cap", 1 << 30, 40, 40, 99, 0, 0x7fffff},
		{"zero counts as one", 0, 40, 40, 99, 0, 1},
		{"max level gets nothing", 500, 40, 99, 99, 0, 0},
	} {
		if got := KillXP(tc.xp, tc.m, tc.c, tc.max, tc.pct); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestMercKillShareAndRatio(t *testing.T) {
	if MercKillShare(256, false) != 86 || MercKillShare(256, true) != 256 {
		t.Error("merc share")
	}

	if ExpRatioScale(1000, 1024, 10) != 1000 || ExpRatioScale(1000, 512, 10) != 500 || ExpRatioScale(-3, 1024, 10) != -3 {
		t.Error("ratio")
	}
}
