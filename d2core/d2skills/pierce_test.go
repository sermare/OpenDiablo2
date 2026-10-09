package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestPierceOf(t *testing.T) {
	l := d2statlist.NewList()
	l.Add(333, 0, 10) // passive_fire_pierce
	l.Add(d2statlist.StatPierceFire, 0, 5)
	l.Add(334, 0, 20)

	p := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Totals: &d2statlist.Totals{Stats: l}}}

	tests := []struct {
		src      *d2mapentity.Player
		kind     string
		want     int
		wantHas  bool
		wantResp int // resist of an immune (100) monster after pierce
	}{
		{p, "fire", 10, true, 100}, // 306 (item) is not added: only 333..336 are read
		{p, "ltng", 20, true, 100}, // monster immunity cannot be pierced
		{p, "cold", 0, true, 100},
		{p, "phys", 0, false, 100},
		{p, "mag", 0, false, 100},
		{nil, "fire", 0, true, 100},
		{&d2mapentity.Player{}, "fire", 0, true, 100},
	}

	for _, tt := range tests {
		got, has := pierceOf(tt.src, tt.kind)
		if got != tt.want || has != tt.wantHas {
			t.Errorf("%s: pierce %d,%v want %d,%v", tt.kind, got, has, tt.want, tt.wantHas)
		}

		res := d2combat.EffectiveResist(d2combat.ResistInput{Resist: 100, Ignore: true, Pierce: got, HasPierce: has})
		if res != tt.wantResp {
			t.Errorf("%s: immune monster resist %d want %d", tt.kind, res, tt.wantResp)
		}
	}
}
