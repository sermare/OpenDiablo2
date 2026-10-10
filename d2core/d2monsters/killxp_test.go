package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func xpHero(exp int) *d2mapentity.Player {
	return &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Experience: exp}}
}

// TestAwardKillXP pins awardKillXP: the shrine bonus is percent of the base
// (truncated) and added to the hero.
func TestAwardKillXP(t *testing.T) {
	for _, tc := range []struct {
		name         string
		start, base  int
		bonus        int // -1: no hook
		wantXP, want int
	}{
		{"plain", 100, 30, -1, 30, 130},
		{"shrine +50%", 0, 30, 50, 45, 45},
		{"shrine truncates", 0, 7, 50, 10, 10},
		{"shrine 0%", 5, 30, 0, 30, 35},
		{"zero xp monster", 9, 0, -1, 0, 9},
	} {
		d := &Director{}
		if tc.bonus >= 0 {
			b := tc.bonus
			d.ExpBonusPct = func() int { return b }
		}

		h := xpHero(tc.start)
		if got := d.awardKillXP(h, tc.base, "x"); got != tc.wantXP || h.Stats.Experience != tc.want {
			t.Errorf("%s: xp=%d exp=%d, want %d %d", tc.name, got, h.Stats.Experience, tc.wantXP, tc.want)
		}
	}
}

// TestCreditKill: a solo hero is exactly as before (scaled by level, shrine
// after); a party hook that accepts the kill gets the UNSCALED xp plus the
// shrine bonus and the monster level, and the hero is not credited here.
func TestCreditKill(t *testing.T) {
	hero := func(lvl, exp int) *d2mapentity.Player {
		return &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Level: lvl, Experience: exp}}
	}

	// solo: same level, 5 below, 10 below (unchanged vs scaleKillXP + award)
	for _, tc := range []struct{ hlvl, mlvl, xp, want int }{
		{30, 30, 300, 300}, {30, 25, 300, 300}, {30, 20, 2560, 130}, {99, 99, 300, 0},
	} {
		d := &Director{}
		h := hero(tc.hlvl, 7)

		if got := d.creditKill(h, tc.xp, tc.mlvl, "x"); got != tc.want || h.Stats.Experience != 7+tc.want {
			t.Errorf("solo %+v: got %d exp %d", tc, got, h.Stats.Experience)
		}
	}

	var gotXP, gotLvl int

	d := &Director{ExpBonusPct: func() int { return 100 }}
	d.PartyXP = func(_ *d2mapentity.Player, xp, mlvl int, _ string) bool { gotXP, gotLvl = xp, mlvl; return true }
	h := hero(30, 0)

	if got := d.creditKill(h, 2560, 20, "x"); got != 5120 || gotXP != 5120 || gotLvl != 20 || h.Stats.Experience != 0 {
		t.Errorf("party: got %d offered %d lvl %d exp %d", got, gotXP, gotLvl, h.Stats.Experience)
	}

	d.PartyXP = func(*d2mapentity.Player, int, int, string) bool { return false }
	h = hero(30, 0)

	if got := d.creditKill(h, 2560, 20, "x"); got != 260 || h.Stats.Experience != 260 {
		t.Errorf("party refused: got %d exp %d", got, h.Stats.Experience)
	}
}

// TestScaleKillXPClampAndItemBonus: the 0x7fffff clamp holds even without
// stats, and the item bonus only applies when the hero has it.
func TestScaleKillXPClampAndItemBonus(t *testing.T) {
	if got := scaleKillXP(1<<30, 0, nil); got != 0x7fffff {
		t.Errorf("clamp without stats: %d", got)
	}

	if got := scaleKillXP(1<<30, 50, &d2hero.HeroStatsState{Level: 50}); got != 0x7fffff {
		t.Errorf("clamp: %d", got)
	}

	plain := &d2hero.HeroStatsState{Level: 30}
	if got := scaleKillXP(300, 30, plain); got != 300 {
		t.Errorf("hero without the bonus changed: %d", got)
	}

	bonus := &d2hero.HeroStatsState{Level: 30, Totals: &d2statlist.Totals{Stats: d2statlist.NewList()}}
	bonus.Totals.Stats.Add(d2statlist.StatAddExp, 0, 25)

	if got := scaleKillXP(300, 30, bonus); got != 375 {
		t.Errorf("+25%% experience: %d", got)
	}
}

func TestMercGetsKillXP(t *testing.T) {
	for _, tc := range []struct {
		merc, owner int
		want        bool
	}{
		{1, 2, true}, {5, 5, false}, {9, 5, false}, {d2hireling.MaxLevel, 99, false}, {d2hireling.MaxLevel - 1, 99, true},
	} {
		if got := mercGetsKillXP(tc.merc, tc.owner); got != tc.want {
			t.Errorf("merc %d owner %d: %v", tc.merc, tc.owner, got)
		}
	}
}

// TestScaleKillXP is the regression for the level scaling: a kill at the
// hero's level (or up to 5 levels below, or unknown levels) is unchanged.
func TestScaleKillXP(t *testing.T) {
	st := func(l int) *d2hero.HeroStatsState { return &d2hero.HeroStatsState{Level: l} }

	for _, tc := range []struct {
		name     string
		xp, mlvl int
		st       *d2hero.HeroStatsState
		want     int
	}{
		{"same level unchanged", 300, 30, st(30), 300},
		{"5 below unchanged", 300, 25, st(30), 300},
		{"low level hero, monster above by 3", 300, 6, st(3), 300},
		{"unknown monster level", 300, 0, st(30), 300},
		{"no stats", 300, 30, nil, 300},
		{"10 below is nearly nothing", 2560, 20, st(30), 130},
		{"clvl 40 vs mlvl 50", 1000, 50, st(40), 800},
		{"level 99 hero gets nothing", 300, 99, st(99), 0},
	} {
		if got := scaleKillXP(tc.xp, tc.mlvl, tc.st); got != tc.want {
			t.Errorf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}
