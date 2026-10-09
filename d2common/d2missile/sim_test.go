package d2missile

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2combat"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

type fakeRoller struct{ v uint32 }

func (f fakeRoller) Roll(n int32) uint32 {
	if f.v >= uint32(n) {
		return uint32(n) - 1
	}

	return f.v
}

type fakeTarget struct {
	id       string
	player   bool
	alive    bool
	level    int
	defense  int
	x, y     int
	friendly bool
}

func (t *fakeTarget) ID() string         { return t.id }
func (t *fakeTarget) IsPlayer() bool     { return t.player }
func (t *fakeTarget) Alive() bool        { return t.alive }
func (t *fakeTarget) Level() int         { return t.level }
func (t *fakeTarget) Defense(_ bool) int { return t.defense }
func newTarget(id string, x, y int) *fakeTarget {
	return &fakeTarget{id: id, alive: true, level: 1, x: x, y: y}
}

type fakeWorld struct {
	grid    *d2path.CellGrid
	targets []*fakeTarget
	frame   int
}

func (w *fakeWorld) Flags(x, y int) uint16 { return w.grid.Flags(x, y) }
func (w *fakeWorld) Frame() int            { return w.frame }
func (w *fakeWorld) IsEnemy(_ Owner, t Target) bool {
	if p, ok := t.(*posTarget); ok {
		return !p.friendly
	}

	return !t.(*fakeTarget).friendly
}

func (w *fakeWorld) Targets(x, y int) []Target {
	var out []Target

	for _, t := range w.targets {
		if t.alive && t.x == x && t.y == y {
			out = append(out, t)
		}
	}

	return out
}

type specs map[string]*Spec

func (s specs) ByID(id int) *Spec {
	for _, sp := range s {
		if sp.ID == id {
			return sp
		}
	}

	return nil
}
func (s specs) ByName(n string) *Spec { return s[n] }

func fireBolt() *Spec {
	return &Spec{ID: 58, Name: "firebolt", SrvDoFunc: 1, Vel: 20, MaxVel: 20, Range: 50, CollideType: 3,
		CollideKill: true, LastCollide: true, ExplosionMissile: "fireexplode", CanSlow: true}
}

func newWorld() *fakeWorld {
	// open field 0..200 x -20..20
	return &fakeWorld{grid: d2path.NewCellGrid(-5, -20, 205, 40)}
}

func run(s *Sim, w *fakeWorld, frames int) []Event {
	var evs []Event

	s.OnEvent = func(e Event) { evs = append(evs, e) }

	for i := 0; i < frames; i++ {
		w.frame++
		s.Step()
	}

	return evs
}

func kinds(evs []Event) map[EventKind]int {
	m := map[EventKind]int{}
	for _, e := range evs {
		m[e.Kind]++
	}

	return m
}

func TestCreateVelocityAndLifetime(t *testing.T) {
	w := newWorld()
	s := NewSim(w, nil)
	m, err := s.Create(CreateParams{Spec: fireBolt(), Level: 1, X: 0, Y: 0, DestX: 30, DestY: 0,
		Owner: Owner{ID: "hero", IsPlayer: true}})

	if err != nil {
		t.Fatal(err)
	}

	// vel = 20<<8, step = 75% = 3840, /4096 = 0.9375 subtile per frame
	if m.Velocity != 20<<8 || m.Life != 50 {
		t.Fatalf("velocity %d life %d", m.Velocity, m.Life)
	}

	run(s, w, 10)

	if math.Abs(m.X-9.375) > 1e-9 || m.Y != 0 {
		t.Fatalf("after 10 frames at (%v,%v)", m.X, m.Y)
	}

	evs := run(s, w, 40) // frames 11..50: expires at the 50th step

	if kinds(evs)[EventExpire] != 1 || !m.Dead() {
		t.Fatalf("events %v dead=%v", kinds(evs), m.Dead())
	}

	if math.Abs(m.X-46.875) > 1e-9 {
		t.Fatalf("flew to %v, want 50*0.9375", m.X)
	}

	if len(s.Missiles()) != 0 {
		t.Fatal("dead missile kept")
	}
}

func TestLevelVelocityAndRange(t *testing.T) {
	sp := &Spec{Vel: 10, VelLev: 8, Range: 20, LevRange: 2, CollideType: 3}
	s := NewSim(newWorld(), nil)
	m, _ := s.Create(CreateParams{Spec: sp, Level: 4, DestX: 5})
	// ((8*4)/8 + 10) << 8 = 14<<8 ; 20 + 2*4 frames
	if m.Velocity != 14<<8 || m.Life != 28 {
		t.Fatalf("vel %d life %d", m.Velocity, m.Life)
	}

	m, _ = s.Create(CreateParams{Spec: sp, Level: 4, DestX: 5, Velocity: 99 << 8, Range: 7})
	if m.Velocity != 99<<8 || m.Life != 7 {
		t.Fatalf("explicit %d %d", m.Velocity, m.Life)
	}
}

func TestTooFar(t *testing.T) {
	s := NewSim(newWorld(), nil)
	if _, err := s.Create(CreateParams{Spec: fireBolt(), DestX: 100}); err != ErrTooFar {
		t.Fatalf("err = %v", err)
	}

	if _, err := s.Create(CreateParams{Spec: fireBolt(), DestX: 99, DestY: -99}); err != nil {
		t.Fatal(err)
	}
}

func TestHitsTargetOnPath(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("m1", 10, 0)}
	s := NewSim(w, nil)
	dmg := DamageDesc{Fire: Elem{Min: 768, Max: 1536}}
	_, _ = s.Create(CreateParams{Spec: fireBolt(), Level: 1, DestX: 30, Damage: dmg,
		Owner: Owner{ID: "hero", IsPlayer: true, Roller: fakeRoller{0}}})

	evs := run(s, w, 20)
	k := kinds(evs)

	if k[EventHit] != 1 || k[EventExplode] != 1 || k[EventExpire] != 0 || len(s.Missiles()) != 0 {
		t.Fatalf("events %v", k)
	}

	for _, e := range evs {
		if e.Kind == EventHit {
			// roll 0 -> min; 3.0 fire damage in 8.8
			if e.Damage.Fire != 768 || e.Damage.Result&d2combat.ResultHit == 0 || e.Target.ID() != "m1" {
				t.Fatalf("hit %+v", e)
			}
		}
	}
}

func TestIgnoresFriendsAndDead(t *testing.T) {
	w := newWorld()
	friend := newTarget("pet", 5, 0)
	friend.friendly = true
	corpse := newTarget("corpse", 8, 0)
	corpse.alive = false
	w.targets = []*fakeTarget{friend, corpse}
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: fireBolt(), Level: 1, DestX: 30})

	k := kinds(run(s, w, 60))
	if k[EventHit] != 0 || k[EventExpire] != 1 {
		t.Fatalf("events %v", k)
	}
}

func TestCollideTypeFilters(t *testing.T) {
	for _, tc := range []struct {
		ct     int
		player bool
		hit    bool
	}{
		{1, true, true}, {1, false, false}, {2, true, false}, {2, false, true}, {3, true, true}, {3, false, true},
		{6, true, false}, {6, false, false},
	} {
		w := newWorld()
		tg := newTarget("t", 6, 0)
		tg.player = tc.player
		w.targets = []*fakeTarget{tg}
		sp := fireBolt()
		sp.CollideType = tc.ct
		s := NewSim(w, nil)
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

		if got := kinds(run(s, w, 20))[EventHit] == 1; got != tc.hit {
			t.Errorf("collide type %d player=%v: hit=%v want %v", tc.ct, tc.player, got, tc.hit)
		}
	}
}

func TestWallStopsMissile(t *testing.T) {
	w := newWorld()
	w.grid.Set(5, 0, d2path.FlagWall)
	w.targets = []*fakeTarget{newTarget("behind", 8, 0)}
	s := NewSim(w, nil)
	m, _ := s.Create(CreateParams{Spec: fireBolt(), Level: 1, DestX: 30})

	k := kinds(run(s, w, 20))
	if k[EventWall] != 1 || k[EventHit] != 0 || k[EventExplode] != 1 || !m.Dead() {
		t.Fatalf("events %v", k)
	}
}

func TestWalkBitOnlyBlocksTypeEight(t *testing.T) {
	// walk bit 0x1: in the block mask of type 8 only (0x185); the other types
	// let the missile in and the cached-flags test (&5) ends it silently
	for _, tc := range []struct {
		ct   int
		kind EventKind
	}{{3, EventVanish}, {6, EventVanish}, {1, EventVanish}, {2, EventVanish}, {5, EventVanish}, {7, EventVanish}, {8, EventWall}, {0, ""}} {
		w := newWorld()
		w.grid.Set(5, 0, d2path.FlagWalk) // water/hole style cell
		sp := fireBolt()
		sp.CollideType = tc.ct
		s := NewSim(w, nil)
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

		k := kinds(run(s, w, 20))
		if tc.kind != "" && k[tc.kind] != 1 || tc.kind == "" && (k[EventWall] != 0 || k[EventVanish] != 0) {
			t.Errorf("type %d: events %v want %q", tc.ct, k, tc.kind)
		}
	}
}

func TestWallBitBlocksEveryTypeButSevenAndZero(t *testing.T) {
	for _, tc := range []struct {
		ct   int
		kind EventKind
	}{{1, EventWall}, {2, EventWall}, {3, EventWall}, {5, EventWall}, {6, EventWall}, {8, EventWall}, {7, EventVanish}, {0, ""}} {
		w := newWorld()
		w.grid.Set(5, 0, d2path.FlagWall)
		sp := fireBolt()
		sp.CollideType = tc.ct
		s := NewSim(w, nil)
		m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

		k := kinds(run(s, w, 20))
		if tc.kind != "" && k[tc.kind] != 1 || tc.kind == "" && (k[EventWall] != 0 || k[EventVanish] != 0) {
			t.Errorf("type %d: events %v want %q", tc.ct, k, tc.kind)
		}

		if tc.kind == EventWall && (m.X != 4.5 || m.Y != 0.5) {
			t.Errorf("type %d stopped at (%v,%v), want the last free cell (4.5,0.5)", tc.ct, m.X, m.Y)
		}
	}
}

func TestWallEndsDuringActivateDelay(t *testing.T) {
	// the wall test (path blocked) is independent of the Activate delay
	w := newWorld()
	w.grid.Set(5, 0, d2path.FlagWall)
	sp := fireBolt()
	sp.Activate = 30
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

	if kinds(run(s, w, 20))[EventWall] != 1 {
		t.Fatal("wall must stop a missile that is still intangible to units")
	}
}

func TestStationaryMissileInAWallVanishes(t *testing.T) {
	w := newWorld()
	w.grid.Set(10, 0, d2path.FlagWall)
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: &Spec{Name: "fw", Range: 20, CollideType: 3}, Level: 1, X: 10.5, Y: 0.5, DestX: 11, Stationary: true})

	if k := kinds(run(s, w, 3)); k[EventVanish] != 1 || k[EventWall] != 0 {
		t.Fatalf("events %v", k)
	}
}

func TestLeavingTheGridStopsMissile(t *testing.T) {
	w := &fakeWorld{grid: d2path.NewCellGrid(0, -5, 6, 10)}
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: fireBolt(), Level: 1, X: 1, DestX: 30})

	if kinds(run(s, w, 20))[EventWall] != 1 {
		t.Fatal("missile must die at the edge of the grid (out-of-grid cells read 0x27)")
	}
}

func TestActivateDelay(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("m", 2, 0)}
	sp := fireBolt()
	sp.Activate = 5
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

	// target at x=2 is passed within the first 3 frames, inside the 5 frame delay
	if kinds(run(s, w, 10))[EventHit] != 0 {
		t.Fatal("collision during the activation delay")
	}
}

func TestPierceAndLastCollide(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("a", 5, 0), newTarget("b", 10, 0), newTarget("c", 15, 0)}
	sp := fireBolt()
	sp.Pierce = true
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, Pierce: 2})

	k := kinds(run(s, w, 40))
	// two pass-throughs, the third kills it
	if k[EventHit] != 3 || k[EventPierce] != 2 || len(s.Missiles()) != 0 {
		t.Fatalf("events %v", k)
	}
}

func TestNonKillingMissileHitsEachTargetOnce(t *testing.T) {
	// Howl: CollideKill 0, LastCollide 1
	w := newWorld()
	tg := newTarget("m", 5, 0)
	tg.x, tg.y = 5, 0
	w.targets = []*fakeTarget{tg}
	sp := &Spec{Name: "howl", Vel: 12, Range: 30, CollideType: 3, LastCollide: true}
	s := NewSim(w, nil)
	hooked := 0
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, OnHit: func(*Missile, Target) { hooked++ }})

	k := kinds(run(s, w, 40))
	if k[EventHit] != 1 || hooked != 1 || k[EventExpire] != 1 {
		t.Fatalf("events %v hooked %d", k, hooked)
	}
}

func TestToHitMissStillConsumesMissile(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("m", 6, 0)}
	w.targets[0].defense = 100000
	sp := fireBolt()
	sp.ToHit = true
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30,
		Owner: Owner{Level: 1, AttackRating: 10, Roller: fakeRoller{99}}})

	k := kinds(run(s, w, 60))
	if k[EventMiss] != 1 || k[EventHit] != 0 || k[EventExplode] != 0 { // CollideKill destroys the missile even on a miss (notes)
		t.Fatalf("events %v", k)
	}
}

func TestToHitHit(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("m", 6, 0)}
	sp := fireBolt()
	sp.ToHit = true
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30,
		Owner: Owner{Level: 10, AttackRating: 5000, Roller: fakeRoller{0}}})

	evs := run(s, w, 20)
	if kinds(evs)[EventHit] != 1 {
		t.Fatalf("events %v", kinds(evs))
	}

	for _, e := range evs {
		if e.Kind == EventHit && (e.Chance != 95 || e.Roll != 0) {
			t.Fatalf("chance %d roll %d", e.Chance, e.Roll)
		}
	}
}

func TestClampToDest(t *testing.T) {
	w := newWorld()
	s := NewSim(w, nil)
	m, _ := s.Create(CreateParams{Spec: fireBolt(), Level: 1, DestX: 10, ClampToDest: true})
	run(s, w, 30)

	if !m.Dead() || math.Abs(m.X-10) > 1e-9 {
		t.Fatalf("lob ended at %v dead=%v", m.X, m.Dead())
	}
}

func TestAngleAndAccel(t *testing.T) {
	w := newWorld()
	s := NewSim(w, nil)
	sp := &Spec{Vel: 12, MaxVel: 12, Accel: -1000, Range: 12, CollideType: 3}
	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 10, Angle: math.Pi / 2})

	if math.Abs(m.DX) > 1e-9 || math.Abs(m.DY-1) > 1e-9 {
		t.Fatalf("direction %v,%v", m.DX, m.DY)
	}

	run(s, w, 12)
	// accel acts on the 75%-scaled path velocity (2304); frames 5 and 10 lower it by 1000 each
	if m.pathVel != 12*256*75/100-2000 {
		t.Fatalf("path velocity %d", m.pathVel)
	}
}

func TestHitSubMissiles(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("m", 5, 0)}
	tbl := specs{"child": {ID: 1, Name: "child", Range: 10, CollideType: 3}}
	sp := fireBolt()
	sp.SrvHitFunc = 4
	sp.HitSubMissile = [4]string{"child", "child", "", "missing"}
	s := NewSim(w, tbl)

	var created []string

	s.OnEvent = func(e Event) {
		if e.Kind == EventCreate {
			created = append(created, e.Missile.Spec.Name)
		}
	}

	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

	for i := 0; i < 10; i++ {
		w.frame++
		s.Step()
	}

	if len(created) != 3 { // the bolt and two children
		t.Fatalf("created %v", created)
	}
}

func TestDamageDescRoll(t *testing.T) {
	d := DamageDesc{PhysMin: 256, PhysMax: 512, DamagePct: 50, Cold: Elem{Min: 512, Max: 1024, Len: 150}}
	got := d.Roll(fakeRoller{0})

	if got.Physical != 384 || got.Cold != 512 || got.ColdLen != 150 || got.Result != d2combat.ResultHit {
		t.Fatalf("%+v", got)
	}

	if (&DamageDesc{}).Empty() == false || d.Empty() {
		t.Fatal("Empty")
	}

	// a huge roll is clamped by the fake to range-1 -> max-1
	if got := d.Roll(fakeRoller{9999}); got.Cold != 1023 {
		t.Fatalf("cold %d", got.Cold)
	}
}

type posTarget struct {
	*fakeTarget
	px, py float64
}

func (p *posTarget) SubPos() (float64, float64) { return p.px, p.py }

func TestHomingStationaryAndHitEvery(t *testing.T) {
	w := newWorld()
	s := NewSim(w, nil)
	hero := Owner{ID: "hero", IsPlayer: true, Roller: fakeRoller{}}

	// homing: aimed away from the target, it still turns onto it
	tg := &posTarget{fakeTarget: newTarget("z", 20, 8), px: 20.5, py: 8.5}
	w.targets = append(w.targets, tg.fakeTarget)

	m, err := s.Create(CreateParams{Spec: fireBolt(), Level: 1, X: 0, Y: 0, DestX: 30, DestY: 0, Owner: hero, Home: tg,
		Damage: DamageDesc{PhysMin: 256, PhysMax: 512}})
	if err != nil {
		t.Fatal(err)
	}

	evs := run(s, w, 40)
	if kinds(evs)[EventHit] != 1 || !m.Dead() {
		t.Errorf("homing missile did not reach its target: %v", kinds(evs))
	}

	// stationary + HitEvery: a wall that damages every 5 frames, scaled
	w2 := newWorld()
	s2 := NewSim(w2, nil)
	z := newTarget("z", 10, 0)
	w2.targets = append(w2.targets, z)

	wall := &Spec{ID: 1, Name: "wall", SrvDoFunc: 1, Range: 21, CollideType: 3}
	m2, _ := s2.Create(CreateParams{Spec: wall, Level: 1, X: 10, Y: 0, DestX: 11, DestY: 0, Owner: hero,
		Stationary: true, HitEvery: 5, ScalePct: 50, Damage: DamageDesc{PhysMin: 1000, PhysMax: 1001}})

	var phys []int32

	s2.OnEvent = func(e Event) {
		if e.Kind == EventHit {
			phys = append(phys, e.Damage.Physical)
		}
	}

	for i := 0; i < 20; i++ {
		w2.frame++
		s2.Step()
	}

	if m2.X != 10 || m2.Y != 0 {
		t.Errorf("stationary missile moved to (%v,%v)", m2.X, m2.Y)
	}

	if len(phys) != 4 || phys[0] != 500 {
		t.Errorf("hits every 5 frames over 20 frames at 50%%: %v", phys)
	}
}

func TestPositiveAccelCapsAtUnscaledMaxVel(t *testing.T) {
	// Blessed Hammer: Vel 18, MaxVel 30, Accel 250 (verified: cap is MaxVel<<8, not scaled)
	w := newWorld()
	s := NewSim(w, nil)
	sp := &Spec{Vel: 18, MaxVel: 30, Accel: 250, Range: 400, CollideType: 3}
	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, DestX: 10})

	if m.pathVel != 18*256*75/100 {
		t.Fatalf("start %d", m.pathVel)
	}

	run(s, w, 5)

	if m.pathVel != 18*256*75/100+250 {
		t.Fatalf("after 5 frames %d", m.pathVel)
	}

	run(s, w, 400)

	if m.pathVel != 30<<8 || m.accel != 0 {
		t.Fatalf("cap %d accel %d", m.pathVel, m.accel)
	}
}

func TestAimAtStartIsDiagonal(t *testing.T) {
	s := NewSim(newWorld(), nil)
	m, _ := s.Create(CreateParams{Spec: fireBolt(), Level: 1, X: 3, Y: 3, DestX: 3, DestY: 3})

	if math.Abs(m.DX-m.DY) > 1e-9 || m.DX <= 0 {
		t.Fatalf("dir %v,%v", m.DX, m.DY)
	}
}

func TestMissAlwaysDestroysEvenWithoutCollideKillOrWithPierce(t *testing.T) {
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("a", 6, 0), newTarget("b", 10, 0)}
	w.targets[0].defense = 100000
	sp := fireBolt()
	sp.CollideKill, sp.ToHit, sp.Pierce = false, true, true
	s := NewSim(w, nil)
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30, Pierce: 5,
		Owner: Owner{Level: 1, AttackRating: 10, Roller: fakeRoller{99}}})

	k := kinds(run(s, w, 40))
	if k[EventMiss] != 1 || k[EventPierce] != 0 || len(s.Missiles()) != 0 {
		t.Fatalf("events %v", k)
	}
}

func TestExpiryAndWallRunHitSubMissiles(t *testing.T) {
	tbl := specs{"child": {ID: 1, Name: "child", Range: 10, CollideType: 3}}

	for _, wall := range []bool{false, true} {
		w := newWorld()
		if wall {
			w.grid.Set(5, 0, d2path.FlagWall)
		}

		sp := fireBolt()
		sp.Range, sp.SrvHitFunc = 10, 2
		sp.HitSubMissile = [4]string{"child"}
		s := NewSim(w, tbl)
		_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 30})

		creates := kinds(run(s, w, 30))[EventCreate]
		if creates != 1 { // the child is created while stepping: OnEvent was set by run() after the parent
			t.Fatalf("wall=%v creates=%d", wall, creates)
		}
	}
}
