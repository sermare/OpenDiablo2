package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func TestDirectorResists(t *testing.T) {
	var tot d2statlist.Totals

	tot.ResistShown[d2statlist.ResFire], tot.ResistShown[d2statlist.ResCold], tot.ResistShown[d2statlist.ResLight] = 40, -20, 75

	hero := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Totals: &tot}}

	rec := &d2records.MonStatRecord{ResistanceFireNormal: 100, ResistanceColdNormal: 30, ResistanceLightningNormal: -10}
	mon := &d2mapentity.Monster{Stat: rec}

	d := &Director{
		targets: map[uint32]*d2mapentity.Player{7: hero},
		units:   map[uint32]*unit{3: {m: mon, b: &d2monster.Brain{}}},
	}

	if f, c, l := d.Resists(d2monster.Target{ID: 7, IsPlayer: true}); f != 40 || c != -20 || l != 75 {
		t.Errorf("hero %d %d %d", f, c, l)
	}

	if f, c, l := d.Resists(d2monster.Target{ID: unitTargetBase + 3}); f != 100 || c != 30 || l != -10 {
		t.Errorf("monster %d %d %d", f, c, l)
	}

	if f, c, l := d.Resists(d2monster.Target{ID: mercTargetBase + 1}); f != 0 || c != 0 || l != 0 {
		t.Errorf("merc %d %d %d", f, c, l)
	}

	if f, _, _ := d.Resists(d2monster.Target{ID: 99, IsPlayer: true}); f != 0 {
		t.Error("unknown player")
	}
}
