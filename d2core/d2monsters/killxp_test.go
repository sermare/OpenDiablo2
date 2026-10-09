package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func xpHero(exp int) *d2mapentity.Player {
	return &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Experience: exp}}
}

// TestAwardKillXP pins what the engine does with a kill's experience: the
// shrine bonus is percent of the base (truncated), a party hook that accepts the
// kill takes it over, and there is no level-difference penalty (UNVERIFIED
// original rule, 0x0057c300; this documents the current simplification).
func TestAwardKillXP(t *testing.T) {
	for _, tc := range []struct {
		name         string
		start, base  int
		bonus        int // -1: no hook
		partyTakes   bool
		wantXP, want int
	}{
		{"plain", 100, 30, -1, false, 30, 130},
		{"shrine +50%", 0, 30, 50, false, 45, 45},
		{"shrine truncates", 0, 7, 50, false, 10, 10},
		{"shrine 0%", 5, 30, 0, false, 30, 35},
		{"party takes the kill", 100, 30, -1, true, 30, 100},
		{"bonus goes to the party too", 100, 30, 100, true, 60, 100},
		{"zero xp monster", 9, 0, -1, false, 0, 9},
	} {
		d := &Director{}
		if tc.bonus >= 0 {
			b := tc.bonus
			d.ExpBonusPct = func() int { return b }
		}

		var offered int
		if tc.partyTakes {
			d.PartyXP = func(_ *d2mapentity.Player, xp int, _ string) bool { offered = xp; return true }
		}

		h := xpHero(tc.start)
		if got := d.awardKillXP(h, tc.base, "x"); got != tc.wantXP || h.Stats.Experience != tc.want {
			t.Errorf("%s: xp=%d exp=%d, want %d %d", tc.name, got, h.Stats.Experience, tc.wantXP, tc.want)
		}

		if tc.partyTakes && offered != tc.wantXP {
			t.Errorf("%s: party offered %d", tc.name, offered)
		}
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
