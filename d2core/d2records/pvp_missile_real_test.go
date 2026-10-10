package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// These tests cast the missile based storm skills at a hostile hero through the
// real tables, fly the missiles and feed every hit on the hero through the PvP
// scale the way d2core/d2skills hurtPlayer does (area events and missile hits
// alike, d2combat.PvPCarry). They guard the 9f-pvp-skills-ear scenario: a
// Meteor's burning ground ticks (3.34..3.66 fire each, 8.8 fixed point) scaled
// to 17 percent one tick at a time came out as 0, so the ground never hurt a
// hero and the fight ran out of damage.

// pvpTally is what a cast did to the hero.
type pvpTally struct {
	areas, hits        int
	raw                [5]int64 // exact 8.8 totals per type
	scaled             int      // whole points through the carry
	legacy             int      // whole points the old per hit scaling gave
	minTick, maxTick   int32    // fire of the missile hits (burning ground)
	areaMin, areaMax   int32    // fire/cold of the area event
	firstArea, lastHit d2combat.Damage
}

func (p *pvpTally) add(carry *d2combat.PvPCarry, d d2combat.Damage) {
	v := [5]int32{d.Physical, d.Fire, d.Lightning, d.Magic, d.Cold}
	for i, x := range v {
		p.raw[i] += int64(x)
	}

	p.scaled += carry.Scale(v, d2combat.PvPPercent()).Total()

	// the old hurtPlayer: whole points first, then the integer scale
	w := func(x int32) int {
		if x <= 0 {
			return 0
		}

		return (int(x) + 128) >> 8
	}
	old := d2combat.PvPParts{Phys: w(d.Physical), Fire: w(d.Fire), Light: w(d.Lightning), Magic: w(d.Magic), Cold: w(d.Cold)}
	p.legacy += old.Scale(d2combat.PvPPercent()).Total()
}

func tallyHeroHits(r *simRun) *pvpTally {
	var carry d2combat.PvPCarry

	t := &pvpTally{}

	for _, e := range r.evs {
		switch {
		case e.Kind == d2missile.EventArea:
			t.areas++
			t.firstArea = e.Damage
			t.add(&carry, e.Damage)
		case e.Kind == d2missile.EventHit && e.Target != nil && e.Target.IsPlayer() && e.Damage.SumTotal(true) > 0:
			t.hits++
			t.lastHit = e.Damage

			if e.Damage.Fire > 0 && (t.minTick == 0 || e.Damage.Fire < t.minTick) {
				t.minTick = e.Damage.Fire
			}

			if e.Damage.Fire > t.maxTick {
				t.maxTick = e.Damage.Fire
			}

			t.add(&carry, e.Damage)
		}
	}

	return t
}

func TestRealStormSkillsHurtAHostileHero(t *testing.T) {
	tests := []struct {
		skill    string
		lvl      int
		frames   int
		minAreas int // area events (Meteor's impact)
		minHits  int // missile hits on the hero (burning ground, shards)
	}{
		{"Meteor", 20, 400, 1, 50}, // meteorcenter Range 60, then 18 meteorfire for 315 frames
		{"Meteor", 1, 400, 1, 1},
		{"Blizzard", 20, 140, 0, 0}, // 25 shards scatter over radius 7: a hit on the hero is chance
		{"Volcano", 20, 220, 0, 0},
	}

	for _, tt := range tests {
		r := realCastFoe(t, tt.skill, tt.lvl, true)
		r.step(tt.frames)

		got := tallyHeroHits(r)
		t.Logf("%s L%d: %d areas %d hits scaled %d legacy %d raw %v", tt.skill, tt.lvl, got.areas, got.hits, got.scaled, got.legacy, got.raw)

		if got.areas < tt.minAreas || got.hits < tt.minHits {
			t.Errorf("%s L%d: %d area hits, %d missile hits on the hero, want >= %d and >= %d", tt.skill, tt.lvl,
				got.areas, got.hits, tt.minAreas, tt.minHits)
		}

		var exact int64
		for _, v := range got.raw {
			exact += v * int64(d2combat.PvPPercent()) / 100
		}

		// the carry loses less than one point per damage type in total
		if int64(got.scaled)<<8 > exact || exact-int64(got.scaled)<<8 > 5<<8 {
			t.Errorf("%s L%d: scaled %d points, exact %.2f", tt.skill, tt.lvl, got.scaled, float64(exact)/256)
		}

		if got.areas+got.hits > 0 && got.scaled < 1 {
			t.Errorf("%s L%d: %d hits on the hero scaled to 0", tt.skill, tt.lvl, got.areas+got.hits)
		}
	}
}

func TestRealMeteorPvPDamageRanges(t *testing.T) {
	r := realCastFoe(t, "Meteor", 20, true)
	r.step(400)

	got := tallyHeroHits(r)
	center := r.created("meteorcenter")[0].Damage.Fire

	// the impact is the skill's damage (meteorcenter carries the descriptor)
	if center.Max <= 0 || got.firstArea.Fire < center.Min || got.firstArea.Fire > center.Max {
		t.Errorf("area fire %d outside the skill damage %+v", got.firstArea.Fire, center)
	}

	// the burning ground is the missile's own rows: (15+92)<<3 .. (25+92)<<3
	if got.hits == 0 || got.minTick < (15+92)<<3 || got.maxTick > (25+92)<<3 {
		t.Errorf("%d burning ground hits with fire %d..%d, want %d..%d", got.hits, got.minTick, got.maxTick, (15+92)<<3, (25+92)<<3)
	}

	// at 17 percent every tick is under one point: per tick scaling gave nothing,
	// the carry gives the sum of the fractions
	areaScaled := int(int64(got.firstArea.Fire) * int64(d2combat.PvPPercent()) / 100 >> 8)
	wantTicks := int64(got.hits) * int64(got.minTick) * int64(d2combat.PvPPercent()) / 100 >> 8

	if got.legacy-areaScaled > 0 {
		t.Errorf("legacy per tick scaling already gave %d points from the ground", got.legacy-areaScaled)
	}

	if int64(got.scaled-areaScaled) < wantTicks-1 || got.scaled-areaScaled < 20 {
		t.Errorf("burning ground gave %d points from %d ticks (>= %d expected)", got.scaled-areaScaled, got.hits, wantTicks)
	}
}
