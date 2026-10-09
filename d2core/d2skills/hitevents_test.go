package d2skills

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2state"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// seqRoller returns scripted rolls and counts steps.
type seqRoller struct {
	vals  []uint32
	steps int
}

func (s *seqRoller) Roll(_ int32) uint32 {
	v := s.vals[s.steps%len(s.vals)]
	s.steps++

	return v
}

func TestPlanHitEvents(t *testing.T) {
	life := 400 << 8

	tests := []struct {
		name        string
		rolls       []uint32
		crush, wnd  int
		class       d2combat.CrushingBlowDefender
		missile     bool
		phys        int
		wantRemoved int
		wantWounds  bool
		wantSteps   int
	}{
		{"no stats, no roll", []uint32{0}, 0, 0, d2combat.CBNormalMonster, false, 0, 0, false, 0},
		{"normal monster 1p: life/4", []uint32{0, 99}, 50, 50, d2combat.CBNormalMonster, false, 0, life / 4, false, 2},
		{"boss: life/8", []uint32{0, 0}, 50, 50, d2combat.CBBossMonster, false, 0, life / 8, true, 2},
		{"special: life/10", []uint32{0, 0}, 50, 50, d2combat.CBSpecial, false, 0, life / 10, true, 2},
		{"missile doubles the divisor", []uint32{0, 0}, 50, 50, d2combat.CBNormalMonster, true, 0, life / 8, true, 2},
		{"raw phys resist halves it", []uint32{0, 0}, 50, 50, d2combat.CBNormalMonster, false, 50, life/4 - life/4*50/100, true, 2},
		{"crush fails, wounds roll", []uint32{99, 0}, 50, 50, d2combat.CBNormalMonster, false, 0, 0, true, 2},
		{"wounds only", []uint32{10}, 0, 50, d2combat.CBNormalMonster, false, 0, 0, true, 1},
	}

	for _, tt := range tests {
		r := &seqRoller{vals: tt.rolls}
		got := planHitEvents(r, tt.crush, tt.wnd, tt.class, 1, tt.missile, life, tt.phys, 30, false, 100)

		if got.CrushRemoved != tt.wantRemoved || got.Wounds.Triggered != tt.wantWounds || r.steps != tt.wantSteps {
			t.Errorf("%s: removed=%d wounds=%v steps=%d, want %d %v %d", tt.name, got.CrushRemoved, got.Wounds.Triggered,
				r.steps, tt.wantRemoved, tt.wantWounds, tt.wantSteps)
		}

		if got.Wounds.Triggered {
			// level 30: (14*9+15*18)+40 = 436 on a monster, state lasts 200 frames
			if got.Wounds.RegenStat != -436 || got.Wounds.Expires != 300 {
				t.Errorf("%s: wounds %+v want regen -436 until 300", tt.name, got.Wounds)
			}
		}
	}
}

func TestCrushingBlowTinyLife(t *testing.T) {
	// life 3 (8.8) / 4 removes 0: no kill, so open wounds still rolls
	r := &seqRoller{vals: []uint32{0, 0}}
	got := planHitEvents(r, 100, 100, d2combat.CBNormalMonster, 1, false, 3, 0, 10, false, 0)

	if got.CrushKilled || !got.Wounds.Triggered {
		t.Errorf("tiny life: %+v", got)
	}
}

func newTestEngine() *Engine {
	return &Engine{sets: map[string]*d2state.Set{}, heroes: map[string]*heroUnit{}}
}

func TestOpenWoundsRefresh(t *testing.T) {
	e := newTestEngine()
	m := &d2mapentity.Monster{}
	p := &d2mapentity.Player{}

	e.frame = 10
	e.applyOpenWounds(m, p, d2combat.RollOpenWounds(&seqRoller{vals: []uint32{0}}, 100, 436, e.frame))

	e.frame = 100
	e.applyOpenWounds(m, p, d2combat.RollOpenWounds(&seqRoller{vals: []uint32{0}}, 100, 999, e.frame))

	in := e.setOf(m.ID()).Get(e.frame, StateOpenWounds)
	if in == nil {
		t.Fatal("state missing")
	}

	// the value does not stack nor change; only the expiry moves
	if in.Until != 300 || in.Level != 1 || in.Mods[0].Value != -436 {
		t.Errorf("state %+v", in)
	}

	if e.setOf(m.ID()).Get(301, StateOpenWounds) != nil {
		t.Error("state should end after 200 frames")
	}
}

func TestPhysNullifiedVsUndead(t *testing.T) {
	e := newTestEngine()
	p := &d2mapentity.Player{}
	undead := &d2mapentity.Monster{Stat: &d2records.MonStatRecord{}}
	undead.Stat.IsUndeadLow = true
	living := &d2mapentity.Monster{Stat: &d2records.MonStatRecord{}}
	living.Stat.ResistancePhysicalNormal = 40
	undead.Stat.ResistancePhysicalNormal = 40

	// without state 0x2f nothing changes
	if got := e.resistFrom(undead, p, "phys"); got != 40 {
		t.Fatalf("no state: phys resist %d want 40", got)
	}

	e.setOf(p.ID()).Apply(0, d2state.Instance{Name: StateUndeadPhysNullify})

	tests := []struct {
		m    *d2mapentity.Monster
		kind string
		want int
	}{
		{undead, "phys", 0}, // positive physical resist voided
		{living, "phys", 40},
	}

	for _, tt := range tests {
		if got := e.resistFrom(tt.m, p, tt.kind); got != tt.want {
			t.Errorf("%v %s: %d want %d", tt.m.Stat.IsUndeadLow, tt.kind, got, tt.want)
		}
	}

	// negative physical resist is kept
	undead.Stat.ResistancePhysicalNormal = -50
	if got := e.resistFrom(undead, p, "phys"); got != -50 {
		t.Errorf("negative resist %d want -50", got)
	}
}

func TestCrushingClass(t *testing.T) {
	boss := &d2mapentity.Monster{Stat: &d2records.MonStatRecord{}}
	boss.Stat.IsSpecialBoss = true

	if crushingClass(boss) != d2combat.CBBossMonster {
		t.Error("boss class")
	}

	if crushingClass(&d2mapentity.Monster{Stat: &d2records.MonStatRecord{}}) != d2combat.CBNormalMonster {
		t.Error("normal class")
	}
}

func TestHurtReduceComponentDifferential(t *testing.T) {
	// the per-type step must equal the old ApplyResist for monsters (no flat, no absorb)
	for _, dmg := range []int{0, 1, 255, 256, 1000, 123456} {
		for _, res := range []int{-100, -50, 0, 25, 75, 99, 100, 120} {
			got, heal := d2combat.ReduceComponent(dmg, 0, res, false, false, 0, 0)
			if want := d2combat.ApplyResist(dmg, res); got != want || heal != 0 {
				t.Errorf("dmg %d res %d: %d,%d want %d", dmg, res, got, heal, want)
			}
		}
	}
}
