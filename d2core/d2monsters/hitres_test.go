package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

func TestHeroPhysicalDamage(t *testing.T) {
	tests := []struct {
		dmg, flat, res, want int
	}{
		{100, 0, 0, 100},
		{100, 10, 0, 90},
		{100, 10, 50, 45}, // flat first, then percent
		{100, 0, 50, 50},
		{10, 30, 50, 0}, // flat larger than the damage: negative component, Total floored
		{10, 10, 0, 0},
		{7, 0, 50, 3},      // truncation as before
		{100, 0, -50, 150}, // negative resist
		{100, 0, 100, 0},
	}

	for _, tt := range tests {
		if got := heroPhysicalDamage(tt.dmg, tt.flat, tt.res); got != tt.want {
			t.Errorf("dmg %d flat %d res %d: %d want %d", tt.dmg, tt.flat, tt.res, got, tt.want)
		}

		// differential against the previous formula (flat then percent, floor 0)
		old := d2combat.ApplyResist(tt.dmg-tt.flat, tt.res)
		if old < 0 {
			old = 0
		}

		if got := heroPhysicalDamage(tt.dmg, tt.flat, tt.res); got != old {
			t.Errorf("dmg %d flat %d res %d: %d differs from old %d", tt.dmg, tt.flat, tt.res, got, old)
		}
	}
}

func TestBlockAnimCooldown(t *testing.T) {
	d := &Director{}
	p := &d2mapentity.Player{Stats: &d2hero.HeroStatsState{Totals: &d2statlist.Totals{FasterBlock: 16}}}
	// cooldown = 15 + 16/8 = 17 frames: plays only when more than 17 frames passed
	steps := []struct {
		frame int
		want  string
	}{
		{100, " block_anim=true"},
		{117, " block_anim=false"}, // 17 frames since: not > 17
		{118, " block_anim=true"},  // 18 since the stamp at 100? stamp stays 100 after a refused one
		{130, " block_anim=false"}, // 12 since 118
		{136, " block_anim=true"},  // 18 since 118
	}

	for _, s := range steps {
		d.frame = s.frame
		if got := d.noteBlockAnim(p); got != s.want {
			t.Errorf("frame %d: %q want %q", s.frame, got, s.want)
		}
	}

	if d.Counters.BlockAnims != 3 {
		t.Errorf("block anims %d want 3", d.Counters.BlockAnims)
	}
}

func TestHeroAutoHitNeedsRunningAndMoving(t *testing.T) {
	p := &d2mapentity.Player{}
	if heroAutoHit(p) {
		t.Error("a standing hero must be rolled against")
	}

	p.SetIsRunning(true)

	if heroAutoHit(p) { // running flag without velocity: not in mode 3
		t.Error("running flag alone is not mode 3")
	}
}
