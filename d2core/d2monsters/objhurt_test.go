package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestHurtHero(t *testing.T) {
	tests := []struct {
		name       string
		hp         int
		totals     *d2statlist.Totals
		dmg        int
		kind       string
		wantDamage int
		wantHP     int
		wantDeaths int
	}{
		{"plain physical", 100, nil, 30, "phys", 30, 70, 0},
		{"zero damage", 100, nil, 0, "phys", 0, 100, 0},
		{"physical resist 50%", 100, &d2statlist.Totals{PhysResist: 50}, 40, "phys", 20, 80, 0},
		{"flat reduction", 100, &d2statlist.Totals{DamageReduction: 10}, 25, "phys", 15, 85, 0},
		{"reduction swallows the hit", 100, &d2statlist.Totals{DamageReduction: 50}, 25, "phys", 0, 100, 0},
		{"fire ignores physical resist", 100, &d2statlist.Totals{PhysResist: 90}, 40, "fire", 40, 60, 0},
		{"kills at zero", 20, nil, 90, "phys", 90, 0, 1},
		{"dead hero is left alone", 0, nil, 10, "phys", 0, 0, 0},
	}

	for _, tc := range tests {
		d := testDirector()
		p := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Health: tc.hp, MaxHealth: 100, Totals: tc.totals}}

		got := d.HurtHero(p, tc.dmg, tc.kind, "test")
		if got != tc.wantDamage || p.Stats.Health != tc.wantHP || d.Counters.HeroDeaths != tc.wantDeaths {
			t.Errorf("%s: applied %d hp %d deaths %d, want %d %d %d", tc.name, got, p.Stats.Health, d.Counters.HeroDeaths,
				tc.wantDamage, tc.wantHP, tc.wantDeaths)
		}
	}

	if d := testDirector(); d.HurtHero(nil, 5, "phys", "x") != 0 {
		t.Error("nil hero")
	}
}
