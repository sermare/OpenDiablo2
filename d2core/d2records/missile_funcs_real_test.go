package d2records

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// These tests cast skills through the real tables (D2_TABLES) and fly the
// missiles that the missile movement / hit functions read from the exe create:
// SRVDO_028 -> blizzardcenter / meteorcenter, SrvDoFunc 10 shards, hit
// function 14 burning ground, SRVDO_123 -> volcano, hit function 9 fire discs.
// The hero stands at (50, 50) and aims at (55, 50).

type simRun struct {
	t   *testing.T
	sim *d2missile.Sim
	w   *auditWorld
	evs []d2missile.Event
	res d2skill.DoResult
	// states records every ApplyState call as "target:state:frames"
	states []string
	p      *d2skill.Pipeline
	u      d2skill.Unit
}

func realCast(t *testing.T, skill string, lvl int) *simRun { return realCastFoe(t, skill, lvl, false) }

// realCastFoe is realCast with the target at (55, 50) being a hostile hero (PvP)
// or a monster.
func realCastFoe(t *testing.T, skill string, lvl int, player bool) *simRun {
	t.Helper()

	rm := loadRealRecords(t)
	reg, mt := rm.SkillTable(), rm.MissileTable()
	sk := reg.ByName(skill)

	if sk == nil {
		t.Fatalf("no skill %q", skill)
	}

	r := &simRun{t: t}
	r.w = &auditWorld{grid: d2path.NewCellGrid(-50, -50, 300, 200), foe: &auditFoe{x: 55, y: 50, player: player}}
	r.sim = d2missile.NewSim(r.w, mt)
	r.sim.OnEvent = func(e d2missile.Event) { r.evs = append(r.evs, e) }

	p := &d2skill.Pipeline{Skills: reg, Missiles: mt, Sim: r.sim, Grid: r.w.grid, Frame: func() int { return 0 },
		Opt: d2skill.Options{IgnoreTown: true, StaticFieldMinPct: 25}}
	p.ApplyState = func(_ d2skill.Unit, t d2missile.Target, st string, frames int) {
		r.states = append(r.states, fmt.Sprintf("%s:%s:%d", t.ID(), st, frames))
	}
	p.After = func(int, func()) {}

	u := &auditUnit{reg: reg, self: sk.ID, lvl: lvl, player: true, rng: d2rand.New(1), mana: 5000 << 8, cd: map[int]int{}}
	r.p, r.u = p, u
	tg := d2skill.Target{X: 55, Y: 50, Unit: r.w.foe, UX: 55, UY: 50, Corpse: true, CX: 55, CY: 50, CorpseID: "c1", CorpseHP: 200, CorpseKey: "zombie", CorpseLevel: 5}

	if st := p.Start(u, sk.ID, tg); !st.OK {
		t.Fatalf("%s: start refused %q", skill, st.Reason)
	}

	r.res = p.Do(u, sk.ID, tg)
	if !r.res.OK {
		t.Fatalf("%s: do refused %q", skill, r.res.Reason)
	}

	return r
}

func (r *simRun) step(n int) {
	for i := 0; i < n; i++ {
		r.sim.Step()
	}
}

func (r *simRun) created(name string) (out []*d2missile.Missile) {
	for _, e := range r.evs {
		if e.Kind == d2missile.EventCreate && e.Missile.Spec.Name == name {
			out = append(out, e.Missile)
		}
	}

	return out
}

func TestRealBlizzard(t *testing.T) {
	for _, lvl := range []int{1, 20} {
		r := realCast(t, "Blizzard", lvl)

		// calc1 = par1 = 7 (radius), calc2 = par3 = 4 (frames between shards),
		// blizzardcenter Range 100: elapsed 0, 4, ..., 96
		if len(r.res.Missiles) != 1 || r.res.Missiles[0].Spec.Name != "blizzardcenter" ||
			r.res.Missiles[0].SpawnRadius != 7 || r.res.Missiles[0].SpawnEvery != 4 {
			t.Fatalf("L%d: cast %+v", lvl, r.res.Missiles)
		}

		r.step(100)

		shards := r.created("blizzard1")
		if len(shards) != 25 {
			t.Errorf("L%d: %d shards, want 25", lvl, len(shards))
		}

		for _, s := range shards {
			if abs(int(s.X)-55) > 6 || abs(int(s.Y)-50) > 6 || s.Damage.Cold.Max <= 0 || s.Spec.Range != 9 {
				t.Errorf("L%d: shard %v dmg %+v", lvl, s, s.Damage.Cold)
			}
		}
	}
}

func TestRealMeteorBurningGround(t *testing.T) {
	r := realCast(t, "Meteor", 20)
	r.step(70) // meteorcenter Range 60

	if len(r.created("meteorcenter")) != 1 {
		t.Fatal("no meteorcenter")
	}

	// hit function 14: area damage of radius aurarangecalc (ln12 = 6), then
	// meteorfire at every sHitPar2 (1) offset of the 18 entry table
	n := 0

	for _, e := range r.evs {
		if e.Kind == d2missile.EventArea {
			n++

			if e.Radius != 6 {
				t.Errorf("area radius %d, want 6", e.Radius)
			}
		}
	}

	if n != 1 {
		t.Errorf("%d area hits", n)
	}

	fires := r.created("meteorfire")
	if len(fires) != 18 {
		t.Fatalf("%d meteorfire, want 18", len(fires))
	}

	// life Param3 + (lvl-1)*Param4 = 30 + 19*15 frames (record +0x150 / +0x154)
	// and damage from the missile's own columns, not the impact: EMin 15 plus
	// tiers 4/5/6 (levels 2-8, 9-16, 17-20 = 28 + 40 + 24 = 92), EMax 25 + 92,
	// both << HitShift 3
	for _, f := range fires {
		if f.Total != 30+19*15 || f.Damage.Fire.Min != (15+92)<<3 || f.Damage.Fire.Max != (25+92)<<3 {
			t.Errorf("meteorfire life %d fire %+v", f.Total, f.Damage.Fire)
		}
	}

	imp := r.created("meteorcenter")[0].Damage.Fire
	if imp.Min < 100*fires[0].Damage.Fire.Min {
		t.Errorf("impact %+v should dwarf the fire tick %+v", imp, fires[0].Damage.Fire)
	}
}

func TestRealVolcano(t *testing.T) {
	r := realCast(t, "Volcano", 20)
	if len(r.res.Missiles) != 1 || r.res.Missiles[0].Spec.Name != "volcano" {
		t.Fatalf("cast %+v", r.res.Missiles)
	}

	m := r.res.Missiles[0]
	// Param3 2 < elapsed < Param4 128, period calc4 = par2 = 2 (Param1 empty),
	// radius aurarangecalc = par1
	if m.PulseEvery != 2 || m.AreaRadius < 1 {
		t.Errorf("period %d radius %d", m.PulseEvery, m.AreaRadius)
	}

	r.step(150)

	deb := r.created("volcano debris 2")
	if len(deb) != 62 { // elapsed 4, 6, ..., 126
		t.Errorf("%d debris, want 62", len(deb))
	}

	for _, d := range deb {
		dx, dy := d.Dest()
		if abs(int(dx)-55) > m.AreaRadius || abs(int(dy)-50) > m.AreaRadius {
			t.Errorf("debris aimed outside the radius %d: %v", m.AreaRadius, d)
		}
	}

	if len(r.created("volcano small fire")) == 0 {
		t.Error("landed debris created no fire")
	}
}

func TestRealEruption(t *testing.T) {
	r := realCast(t, "Eruption", 20)
	if len(r.res.Missiles) != 1 || r.res.Missiles[0].Spec.Name != "erruption center" {
		t.Fatalf("cast %+v", r.res.Missiles)
	}

	m := r.res.Missiles[0]
	if m.SpawnRadius != 7 || m.SpawnEvery != 6 { // calc1 = par1 = 7, calc2 = par2 = 6
		t.Errorf("radius %d every %d", m.SpawnRadius, m.SpawnEvery)
	}

	r.step(80)

	// Range 80: elapsed 0, 6, ..., 78 (14 tries; cells with a wall, walk or
	// missile bit are skipped, the open field has none)
	if n := len(r.created("erruption crack 1")); n != 14 {
		t.Errorf("%d cracks, want 14", n)
	}
}

func TestRealImmolationArrow(t *testing.T) {
	r := realCast(t, "Immolation Arrow", 20)
	if len(r.res.Missiles) != 1 {
		t.Fatalf("cast %+v", r.res.Missiles)
	}

	// calc1 = par1 = 3 (disc radius), calc2 = par2 = 4 (damage radius), the
	// fire lives SHitCalc1 = 100 frames
	m := r.res.Missiles[0]
	if m.DiscRadius != 3 || m.AreaRadius != 4 || m.DiscLife != 100 {
		t.Fatalf("disc %d area %d life %d", m.DiscRadius, m.AreaRadius, m.DiscLife)
	}

	r.step(60)

	fires := r.created("immolationfire")
	if len(fires) != 29 { // integer points with dx*dx + dy*dy <= 9
		t.Fatalf("%d fires, want 29", len(fires))
	}

	for _, f := range fires {
		// own damage: EMin 7 / Emax 9 plus tiers 5 per level step, << HitShift 2
		if f.Total != 100 || f.Damage.Fire.Max <= 0 || f.Damage.Fire.Max >= 100<<8 {
			t.Errorf("immolationfire life %d fire %+v", f.Total, f.Damage.Fire)
		}
	}
}

func TestRealGrimWardLife(t *testing.T) {
	r := realCast(t, "Grim Ward", 4)
	_ = r // the ward order is an effect; its life is calc1 = ln34 = Param3 1000
	found := false

	for _, e := range r.res.Effects {
		if e.Kind == "ward" {
			found = true

			if e.Ward.Life != 1000 {
				t.Errorf("ward life %d, want 1000 (ln34 with Param3 1000, Param4 0)", e.Ward.Life)
			}
		}
	}

	if !found {
		t.Fatal("no ward effect")
	}
}
