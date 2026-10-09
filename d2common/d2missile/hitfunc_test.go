package d2missile

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// finderWorld adds the optional Finder to the fake world.
type finderWorld struct {
	*fakeWorld
	found Target
	calls int
	gotR  int
}

func (w *finderWorld) NearestEnemy(_ Owner, _, _ float64, r int) Target {
	w.calls++
	w.gotR = r

	return w.found
}

func fireball() *Spec {
	sp := fireBolt()
	sp.Name, sp.SrvHitFunc = "fireball", 1
	sp.SHitPar[0] = 4

	return sp
}

func collect(s *Sim) *[]Event {
	var evs []Event

	s.OnEvent = func(e Event) { evs = append(evs, e) }

	return &evs
}

func count(evs []Event, k EventKind) (n int) {
	for _, e := range evs {
		if e.Kind == k {
			n++
		}
	}

	return n
}

// Hit function 1 returns 1 (0x5a7500): the hit function result replaces the
// damage bit, so the target takes NO direct damage; the area damage (one roll,
// radius sHitPar1 subtiles) is the only damage. Also where the missile runs
// out or hits a wall, because ProcessHitOrExpire runs the hit function then.
func TestHitFunc1EventsOnTargetExpiryAndWall(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeWorld)
		end   EventKind
	}{
		{"target", func(w *fakeWorld) { w.targets = []*fakeTarget{newTarget("m1", 10, 0)} }, ""},
		{"expiry", func(*fakeWorld) {}, EventExpire},
		{"wall", func(w *fakeWorld) { w.grid.Set(10, 0, d2path.FlagWall) }, EventWall},
	} {
		w := newWorld()
		tc.setup(w)

		sp := fireball()
		sp.Range = 30 // 30 frames * 0.94 = 28 subtiles: the target / wall at x=10 comes first for two cases

		if tc.end == EventExpire {
			sp.Range = 5
		}

		s := NewSim(w, nil)
		evs := collect(s)
		dmg := DamageDesc{Fire: Elem{Min: 768, Max: 768}}
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, Damage: dmg,
			Owner: Owner{ID: "hero", IsPlayer: true, Roller: fakeRoller{0}}})

		for i := 0; i < 40; i++ {
			w.frame++
			s.Step()
		}

		if count(*evs, EventHit) != 0 {
			t.Errorf("%s: hit function 1 must not roll direct damage", tc.name)
		}

		if count(*evs, EventArea) != 1 || len(s.Missiles()) != 0 {
			t.Fatalf("%s: area events %d, missiles %d", tc.name, count(*evs, EventArea), len(s.Missiles()))
		}

		for _, e := range *evs {
			if e.Kind == EventArea && (e.Radius != 4 || e.Damage.Fire != 768) {
				t.Errorf("%s: area %+v", tc.name, e)
			}
		}

		if tc.end != "" && count(*evs, tc.end) != 1 {
			t.Errorf("%s: no %s event", tc.name, tc.end)
		}
	}
}

func TestAreaRadiusFallsBackToCalcRadius(t *testing.T) {
	w := newWorld()
	sp := fireball()
	sp.SHitPar[0] = 0
	sp.Range = 3
	s := NewSim(w, nil)
	evs := collect(s)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, AreaRadius: 7})

	for i := 0; i < 5; i++ {
		w.frame++
		s.Step()
	}

	for _, e := range *evs {
		if e.Kind == EventArea && e.Radius != 7 {
			t.Errorf("radius %d", e.Radius)
		}
	}

	// neither source gives a radius: no area event at all (0x569510 needs radius > 0)
	s2 := NewSim(newWorld(), nil)
	ev2 := collect(s2)
	_, _ = s2.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

	for i := 0; i < 5; i++ {
		s2.World.(*fakeWorld).frame++
		s2.Step()
	}

	if count(*ev2, EventArea) != 0 {
		t.Error("area event without a radius")
	}
}

// Hit functions 2 and 4 return 3 (damage + destroy): they destroy the missile
// even when CollideKill is 0.
func TestHitFuncs2And4DestroyAndDamageWithoutCollideKill(t *testing.T) {
	for _, hf := range []int{2, 4} {
		w := newWorld()
		w.targets = []*fakeTarget{newTarget("m", 5, 0)}
		sp := fireBolt()
		sp.CollideKill, sp.SrvHitFunc = false, hf
		s := NewSim(w, nil)
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, Damage: DamageDesc{PhysMin: 256, PhysMax: 256}})

		if k := kinds(run(s, w, 20)); k[EventHit] != 1 || len(s.Missiles()) != 0 || k[EventExpire] != 0 {
			t.Errorf("hit func %d: events %v", hf, k)
		}
	}
}

// An Explosion missile with hit function 2 or 4 does take damage (the hit
// function's return 3 replaces the cleared damage bit); without a hit
// function it does not.
func TestExplosionFlagClearsDamageOnlyWithoutHitFunc(t *testing.T) {
	for _, tc := range []struct{ hf, hits int }{{0, 0}, {2, 1}, {4, 1}} {
		w := newWorld()
		w.targets = []*fakeTarget{newTarget("m", 5, 0)}
		sp := fireBolt()
		sp.Explosion, sp.SrvHitFunc = true, tc.hf
		s := NewSim(w, nil)
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, Damage: DamageDesc{PhysMin: 256, PhysMax: 256}})

		if got := kinds(run(s, w, 20))[EventHit]; got != tc.hits {
			t.Errorf("hit func %d: %d hits, want %d", tc.hf, got, tc.hits)
		}
	}
}

func TestPierceChargesRoll(t *testing.T) {
	for _, tc := range []struct {
		chance, roll, want int
	}{{0, 0, 0}, {50, 49, 4}, {50, 50, 0}, {100, 99, 4}, {1, 0, 4}} {
		if got := PierceCharges(tc.chance, fakeRoller{uint32(tc.roll)}); got != tc.want {
			t.Errorf("chance %d roll %d: %d charges, want %d", tc.chance, tc.roll, got, tc.want)
		}
	}

	if PierceCharges(100, nil) != 0 {
		t.Error("nil roller")
	}
}

// Charges exist only for missiles with the Pierce flag, from the owner's
// pierce chance, at most 4; with 4 charges a missile passes four targets and
// dies on the fifth.
func TestPierceChanceGrantsChargesAtCreation(t *testing.T) {
	w := newWorld()

	for i := 0; i < 6; i++ {
		w.targets = append(w.targets, newTarget(string(rune('a'+i)), 5+4*i, 0))
	}

	sp := fireBolt()
	sp.Pierce = true
	s := NewSim(w, nil)
	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 60, PierceChance: 50,
		Owner: Owner{Roller: fakeRoller{0}}})

	if m.Pierce != 4 {
		t.Fatalf("charges %d", m.Pierce)
	}

	k := kinds(run(s, w, 60))
	if k[EventHit] != 5 || k[EventPierce] != 4 || len(s.Missiles()) != 0 {
		t.Fatalf("events %v", k)
	}

	// no flag, no charges
	sp2 := fireBolt()
	m2, _ := NewSim(newWorld(), nil).Create(CreateParams{Spec: sp2, Level: 1, DestX: 60, PierceChance: 100,
		Owner: Owner{Roller: fakeRoller{0}}})

	if m2.Pierce != 0 {
		t.Errorf("charges without the Pierce flag: %d", m2.Pierce)
	}

	// an explicit count wins over the roll
	m3, _ := NewSim(newWorld(), nil).Create(CreateParams{Spec: sp, Level: 1, DestX: 60, Pierce: 1, PierceChance: 100,
		Owner: Owner{Roller: fakeRoller{0}}})
	if m3.Pierce != 1 {
		t.Errorf("explicit pierce %d", m3.Pierce)
	}
}

// The NextHit state 0x56 sits on the target, so two NextHit missiles share it.
func TestNextHitStateIsSharedBetweenMissiles(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("m", 5, 0)}
	sp := &Spec{Name: "nova", Vel: 12, Range: 30, CollideType: 3, NextHit: true, NextDelay: 4}
	s := NewSim(w, nil)
	evs := collect(s)

	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

	for i := 0; i < 30; i++ {
		w.frame++
		s.Step()
	}

	if got := count(*evs, EventHit); got != 1 {
		t.Fatalf("%d hits, want 1 (the second missile arrives while the target has state 0x56)", got)
	}
}

func guidedSpec() *Spec {
	return &Spec{ID: 5, Name: "guidedarrow", SrvDoFunc: 7, SrvHitFunc: 10, Vel: 24, MaxVel: 24, Range: 128,
		CollideType: 3, CollideKill: true, AlwaysExplode: true, Param: [5]int{5, 15}}
}

// Hit function 10, homing: the arrow flies through other units (4) and ends on
// the unit it was fired at.
func TestGuidedArrowIgnoresOtherUnits(t *testing.T) {
	w := newWorld()
	other := newTarget("other", 6, 0)
	goal := &posTarget{fakeTarget: newTarget("goal", 14, 0), px: 14.5, py: 0.5}
	w.targets = []*fakeTarget{other, goal.fakeTarget}
	s := NewSim(w, nil)
	evs := collect(s)
	dmg := DamageDesc{PhysMin: 256, PhysMax: 256}

	m, _ := s.Create(CreateParams{Spec: guidedSpec(), Level: 1, DestX: 14.5, DestY: 0.5, Home: goal, Damage: dmg})
	if m.HomeMode != 1 {
		t.Fatalf("mode %d", m.HomeMode)
	}

	for i := 0; i < 30; i++ {
		w.frame++
		s.Step()
	}

	var hit []string

	for _, e := range *evs {
		if e.Kind == EventHit {
			hit = append(hit, e.Target.ID())
		}
	}

	if len(hit) != 1 || hit[0] != "goal" || !m.Dead() {
		t.Fatalf("hits %v dead=%v", hit, m.Dead())
	}
}

// SrvDoFunc 7 re-aims only every Param1 frames and only for a target 4..24
// subtiles away; between re-aims the heading is fixed.
func TestGuidedArrowTurnsEveryParam1Frames(t *testing.T) {
	w := newWorld()
	goal := &posTarget{fakeTarget: newTarget("goal", 20, 10), px: 20.5, py: 10.5}
	w.targets = []*fakeTarget{goal.fakeTarget}
	s := NewSim(w, nil)
	// aimed along +x; target is 22 subtiles away at (20.5,10.5)
	m, _ := s.Create(CreateParams{Spec: guidedSpec(), Level: 1, DestX: 30, DestY: 0, Home: goal})
	startLife := m.Life

	var turns []int

	lx, ly := m.DX, m.DY

	for i := 0; i < 20; i++ {
		w.frame++
		s.Step()

		if m.DX != lx || m.DY != ly {
			turns = append(turns, startLife-m.Life)
			lx, ly = m.DX, m.DY
		}

		if m.Dead() {
			break
		}
	}

	if len(turns) == 0 {
		t.Fatal("never turned")
	}

	for _, f := range turns {
		// the re-aim happens at the start of the frame whose life is a multiple of 5;
		// life has been decremented once more by the time it is observed
		if (startLife-f+1)%5 != 0 {
			t.Errorf("turned after %d frames (life at the turn %d)", f, startLife-f+1)
		}
	}

	// out of the 4..24 window nothing turns: target 40 away
	far := &posTarget{fakeTarget: newTarget("far", 40, 30), px: 40.5, py: 30.5}
	m2, _ := NewSim(newWorld(), nil).Create(CreateParams{Spec: guidedSpec(), Level: 1, DestX: 30, DestY: 0, Home: far})
	dx := m2.DX

	s2 := NewSim(newWorld(), nil)
	m3, _ := s2.Create(CreateParams{Spec: guidedSpec(), Level: 1, DestX: 30, DestY: 0, Home: far})

	for i := 0; i < 4; i++ {
		s2.World.(*fakeWorld).frame++
		s2.Step()
	}

	if m3.DX != dx || m3.DY != m2.DY {
		t.Errorf("turned while the target was out of range: %v,%v", m3.DX, m3.DY)
	}
}

func TestGuidedArrowDiesWhenOwnerIsGone(t *testing.T) {
	w := newWorld()
	s := NewSim(w, nil)
	gone := false
	m, _ := s.Create(CreateParams{Spec: guidedSpec(), Level: 1, DestX: 30, Owner: Owner{Gone: func() bool { return gone }}})
	_ = run(s, w, 3)

	if m.Dead() {
		t.Fatal("died early")
	}

	gone = true
	k := kinds(run(s, w, 1))

	if !m.Dead() || k[EventVanish] != 1 {
		t.Fatalf("events %v", k)
	}
}

// Ground aimed Guided Arrow: it parks at its aim point (hit func 4 while it
// has life), and when its life runs out it looks for a target within Param2.
func TestGuidedArrowGroundRetargetsOnceThenEnds(t *testing.T) {
	base := newWorld()
	fw := &finderWorld{fakeWorld: base}
	s := NewSim(fw, nil)
	sp := guidedSpec()
	sp.Range = 40

	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, X: 0, Y: 0, DestX: 10, DestY: 0, ClampToDest: true})
	if m.HomeMode != 2 {
		t.Fatalf("mode %d", m.HomeMode)
	}

	for i := 0; i < 30; i++ {
		base.frame++
		s.Step()
	}

	if m.Dead() || !m.parked || math.Abs(m.X-10) > 1e-6 || fw.calls != 0 {
		t.Fatalf("not parked at the aim point: dead=%v parked=%v x=%v calls=%d", m.Dead(), m.parked, m.X, fw.calls)
	}

	for i := 0; i < 15 && fw.calls == 0; i++ {
		base.frame++
		s.Step()
	}

	// no enemy found: another ground leg of the original length (10), mode 6
	if fw.calls != 1 || fw.gotR != 15 || m.HomeMode != 6 || m.Dead() || m.parked || m.Life != 40 {
		t.Fatalf("retarget: calls=%d r=%d mode=%d dead=%v parked=%v life=%d", fw.calls, fw.gotR, m.HomeMode, m.Dead(), m.parked, m.Life)
	}

	for i := 0; i < 20 && !m.Dead(); i++ {
		base.frame++
		s.Step()
	}

	if !m.Dead() || math.Abs(m.X-20) > 0.5 || fw.calls != 1 {
		t.Fatalf("second leg: dead=%v x=%v calls=%d", m.Dead(), m.X, fw.calls)
	}
}

func TestGuidedArrowGroundRetargetsOntoAnEnemy(t *testing.T) {
	base := newWorld()
	goal := &posTarget{fakeTarget: newTarget("goal", 12, 6), px: 12.5, py: 6.5}
	fw := &finderWorld{fakeWorld: base, found: goal}
	base.targets = []*fakeTarget{goal.fakeTarget}
	s := NewSim(fw, nil)
	sp := guidedSpec()
	sp.Range = 40

	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 10, DestY: 0, ClampToDest: true})

	for i := 0; i < 45 && m.HomeMode != 5; i++ {
		base.frame++
		s.Step()
	}

	if m.HomeMode != 5 || m.Home == nil || m.Life != 40 {
		t.Fatalf("mode %d life %d", m.HomeMode, m.Life)
	}

	hits := 0

	s.OnEvent = func(e Event) {
		if e.Kind == EventHit {
			hits++
		}
	}

	for i := 0; i < 40 && !m.Dead(); i++ {
		base.frame++
		s.Step()
	}

	if hits != 1 || !m.Dead() {
		t.Fatalf("hits %d dead %v", hits, m.Dead())
	}
}

// Hit function 14 (Meteor): area damage plus HitSubMissile1 at 18 fixed
// offsets, every sHitPar2-th table entry, with the lifetime override.
func TestMeteorSpawnsFireAtTheTableOffsets(t *testing.T) {
	tbl := specs{"meteorfire": {ID: 2, Name: "meteorfire", Range: 90, CollideType: 3}}

	for _, tc := range []struct{ step, spawned int }{{1, 18}, {2, 9}, {0, 18}, {3, 6}} {
		w := newWorld()
		center := &Spec{ID: 1, Name: "meteorcenter", SrvDoFunc: 1, SrvHitFunc: 14, Range: 5, AlwaysExplode: true,
			HitSubMissile: [4]string{"meteorfire"}, LastCollide: true}
		center.SHitPar[1] = tc.step
		s := NewSim(w, tbl)
		evs := collect(s)

		_, _ = s.Create(CreateParams{Spec: center, Level: 1, X: 50, Y: 0, DestX: 50, DestY: 0, Stationary: true,
			HitSubRange: 33, AreaRadius: 3})

		for i := 0; i < 8; i++ {
			w.frame++
			s.Step()
		}

		n := 0

		for _, e := range *evs {
			if e.Kind == EventCreate && e.Missile.Spec.Name == "meteorfire" {
				n++

				if e.Missile.Total != 33 {
					t.Errorf("sub lifetime %d", e.Missile.Total)
				}
			}
		}

		if n != tc.spawned || count(*evs, EventArea) != 1 {
			t.Errorf("step %d: %d sub missiles, %d area events; want %d, 1", tc.step, n, count(*evs, EventArea), tc.spawned)
		}
	}

	// positions: the first two entries are (+2,-2) and (-2,-2)
	w := newWorld()
	center := &Spec{ID: 1, Name: "meteorcenter", SrvDoFunc: 1, SrvHitFunc: 14, Range: 2, HitSubMissile: [4]string{"meteorfire"}}
	s := NewSim(w, tbl)

	var pos [][2]float64

	s.OnEvent = func(e Event) {
		if e.Kind == EventCreate && e.Missile.Spec.Name == "meteorfire" {
			pos = append(pos, [2]float64{e.Missile.X, e.Missile.Y})
		}
	}

	_, _ = s.Create(CreateParams{Spec: center, Level: 1, X: 50, Y: 0, DestX: 50, DestY: 0, Stationary: true})

	for i := 0; i < 4; i++ {
		w.frame++
		s.Step()
	}

	if len(pos) != 18 || pos[0] != [2]float64{52, -2} || pos[1] != [2]float64{48, -2} || pos[17] != [2]float64{54, -2} {
		t.Errorf("positions %v", pos)
	}
}

// Collide type 0 (Meteor's center) never reacts to walls or units.
func TestCollideTypeZeroIgnoresEverything(t *testing.T) {
	w := newWorld()
	w.grid.Set(5, 0, d2path.FlagWall)
	w.targets = []*fakeTarget{newTarget("m", 3, 0)}
	sp := fireBolt()
	sp.CollideType = 0
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

	if k := kinds(run(s, w, 10)); k[EventHit]+k[EventWall]+k[EventVanish] != 0 {
		t.Fatalf("events %v", k)
	}
}
