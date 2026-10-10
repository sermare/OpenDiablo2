package d2missile

import (
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
)

// The numbers of these tests come from missiles.txt / skills.txt (patch_d2) and
// from the exe tables read for the functions in gaps.go and orb.go.

// exeOrbTable is the 64 int table at 0x6e3ae8 (== 0x6e3f28), read from the exe.
var exeOrbTable = [64]int{0, 2, 5, 8, 11, 14, 16, 19, 21, 23, 24, 26, 27, 28, 29, 29, 30, 29, 29, 28, 27, 26, 24, 23, 21, 19, 16, 14,
	11, 8, 5, 2, 0, -2, -5, -8, -11, -14, -16, -19, -21, -23, -24, -26, -27, -28, -29, -29, -30, -29, -29, -28, -27, -26, -24, -23,
	-21, -19, -16, -14, -11, -8, -5, -2}

func TestOrbTableMatchesExe(t *testing.T) {
	for i := 0; i < 64; i++ {
		if got := orbSin(i); got != exeOrbTable[i] {
			t.Errorf("sin[%d] = %d, exe %d", i, got, exeOrbTable[i])
		}

		// 0x6e3be8 / 0x6e4028 start at entry 16 of the same table
		if got := orbCos(i); got != exeOrbTable[(i+16)&63] {
			t.Errorf("cos[%d] = %d, exe %d", i, got, exeOrbTable[(i+16)&63])
		}
	}
}

// frozenorb / frozenorbbolt / frozenorbnova rows of missiles.txt.
func orbSpecs() (orb *Spec, tbl specs) {
	orb = &Spec{ID: 260, Name: "frozenorb", SrvDoFunc: 15, SrvHitFunc: 29, Vel: 10, MaxVel: 10, Range: 30,
		CollideType: 3, LastCollide: true, AlwaysExplode: true, Param: [5]int{1, 19}, SHitPar: [3]int{4},
		SubMissile: [3]string{"frozenorbbolt"}, HitSubMissile: [4]string{"frozenorbnova"}}
	tbl = specs{
		"frozenorbbolt": {ID: 261, Name: "frozenorbbolt", SrvDoFunc: 1, Vel: 18, MaxVel: 18, Range: 25, CollideType: 3},
		"frozenorbnova": {ID: 262, Name: "frozenorbnova", SrvDoFunc: 16, Vel: 24, MaxVel: 24, Range: 25, CollideType: 3,
			Param: [5]int{6, 2}},
	}

	return orb, tbl
}

func TestOrbBoltHeadingsFollowTheTable(t *testing.T) {
	orb, tbl := orbSpecs()
	w := newWorld()
	s := NewSim(w, tbl)
	evs := collect(s)

	m, _ := s.Create(CreateParams{Spec: orb, Level: 5, X: 40.5, Y: 0.5, DestX: 60, DestY: 0.5, Owner: Owner{Roller: fakeRoller{0}}})
	stepN(s, w, 8)

	bolts := creates(*evs, "frozenorbbolt")
	if len(bolts) != 8 { // Param1 = 1: a bolt every frame of elapsed life 0..7
		t.Fatalf("%d bolts, want 8", len(bolts))
	}

	// table index k*19 mod 64 (Param2 19): 0, 19, 38, 57, 12, 31, 50, 5
	for k, b := range bolts {
		idx := (k * 19) & 63
		wantX, wantY := float64(exeOrbTable[(idx+16)&63]), float64(exeOrbTable[idx])
		n := math.Hypot(wantX, wantY)

		if math.Abs(b.DX-wantX/n) > 1e-9 || math.Abs(b.DY-wantY/n) > 1e-9 {
			t.Errorf("bolt %d heads (%.3f,%.3f), table index %d gives (%.3f,%.3f)", k, b.DX, b.DY, idx, wantX/n, wantY/n)
		}
	}

	if m.Data28 != uint32((8*19)&63) {
		t.Errorf("index after 8 bolts is %d, want %d", m.Data28, (8*19)&63)
	}
}

func TestOrbRingAtExpiryOnly(t *testing.T) {
	orb, tbl := orbSpecs()
	orb.Range = 20
	dmg := DamageDesc{Cold: Elem{Min: 512, Max: 512}}

	// expiry (life 0): 64 entries with a stride of sHitPar1 = 4 -> 16 novas
	w := newWorld()
	s := NewSim(w, tbl)
	evs := collect(s)

	var born [][2]int32 // data fields of each nova at creation

	s.OnEvent = func(e Event) {
		*evs = append(*evs, e)

		if e.Kind == EventCreate && e.Missile.Spec.Name == "frozenorbnova" {
			born = append(born, [2]int32{int32(e.Missile.Data28), int32(e.Missile.Data2C)})
		}
	}

	_, _ = s.Create(CreateParams{Spec: orb, Level: 5, X: 40.5, Y: 0.5, DestX: 60, DestY: 0.5, Damage: dmg,
		Owner: Owner{Roller: fakeRoller{0}}})
	stepN(s, w, 25)

	novas := creates(*evs, "frozenorbnova")
	if len(novas) != 16 {
		t.Fatalf("%d novas, want 16", len(novas))
	}

	for k, n := range novas {
		i := k * 4
		wantA, wantB := exeOrbTable[(i+16)&63], exeOrbTable[i]

		if born[k] != [2]int32{int32(wantA), int32(wantB)} {
			t.Errorf("nova %d data %v, want (%d,%d)", k, born[k], wantA, wantB)
		}

		if n.Damage != dmg {
			t.Errorf("nova %d carries %+v, want the cast damage", k, n.Damage)
		}
	}

	// a wall before the end: life is not 0, the hit function returns 2 and the
	// orb just ends (no nova)
	w2 := newWorld()
	for y := -20; y < 20; y++ {
		w2.grid.Set(46, y, d2path.FlagWall)
	}

	s2 := NewSim(w2, tbl)
	evs2 := collect(s2)

	_, _ = s2.Create(CreateParams{Spec: orb, Level: 5, X: 40.5, Y: 0.5, DestX: 60, DestY: 0.5, Damage: dmg,
		Owner: Owner{Roller: fakeRoller{0}}})
	stepN(s2, w2, 25)

	if n := len(creates(*evs2, "frozenorbnova")); n != 0 {
		t.Errorf("%d novas at a wall, want 0 (hit function 29 needs life 0)", n)
	}
}

func TestOrbPassesThroughEnemies(t *testing.T) {
	orb, tbl := orbSpecs()
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("z", 45, 0)}
	s := NewSim(w, tbl)
	evs := collect(s)

	m, _ := s.Create(CreateParams{Spec: orb, Level: 5, X: 40.5, Y: 0.5, DestX: 60, DestY: 0.5,
		Damage: DamageDesc{Cold: Elem{Min: 512, Max: 512}}, Owner: Owner{Roller: fakeRoller{0}}})
	stepN(s, w, 12)

	hits := 0

	if m.Dead() {
		t.Fatal("the orb must fly on through an enemy (hit function 29 returns 2 while it has life)")
	}

	for _, e := range *evs {
		if e.Kind == EventHit && e.Missile == m {
			hits++

			if e.Damage.SumTotal(false) != 0 {
				t.Errorf("the orb itself dealt %+v", e.Damage)
			}
		}
	}

	if hits != 1 {
		t.Errorf("the orb passed the enemy with %d hits, want 1", hits)
	}
}

// frozenorbnova: Param1 6 ("frames"), Param2 2 ("frequency").
func TestNovaTurnsEveryOtherFrameForSix(t *testing.T) {
	_, tbl := orbSpecs()
	w := newWorld()
	s := NewSim(w, tbl)

	m, _ := s.Create(CreateParams{Spec: tbl["frozenorbnova"], Level: 5, X: 40.5, Y: 0.5, DestX: 70, DestY: 0.5,
		Data28: 30, Data2C: 0})

	// (a, b) -> ((a-b)/2, (a+b)/2) at elapsed 0, 2, 4 only
	want := [][2]int{{15, 15}, {15, 15}, {0, 15}, {0, 15}, {-7, 7}, {-7, 7}, {-7, 7}, {-7, 7}}

	for i, w2 := range want {
		px, py := m.X, m.Y

		stepN(s, w, 1)

		if int32(m.Data28) != int32(w2[0]) || int32(m.Data2C) != int32(w2[1]) {
			t.Fatalf("after frame %d data (%d,%d), want %v", i, int32(m.Data28), int32(m.Data2C), w2)
		}

		if i == 0 { // aimed at the integer position + (15,15) + 0.5 from where it stood
			tx, ty := math.Floor(px)+15.5-px, math.Floor(py)+15.5-py
			n := math.Hypot(tx, ty)

			if math.Abs(m.DX-tx/n) > 1e-9 || math.Abs(m.DY-ty/n) > 1e-9 {
				t.Errorf("heading (%.3f,%.3f), want (%.3f,%.3f)", m.DX, m.DY, tx/n, ty/n)
			}
		}
	}

	dx, dy := m.DX, m.DY

	stepN(s, w, 5)

	if m.DX != dx || m.DY != dy {
		t.Errorf("the nova kept turning after Param1 frames")
	}

	if dx >= 0 || dy <= 0 {
		t.Errorf("after three turns it should head up and back, got (%.2f,%.2f)", dx, dy)
	}
}

func TestPackedShorts(t *testing.T) {
	for _, c := range [][2]int{{0, 0}, {1, -1}, {-7, 7}, {32767, -32768}, {-300, 250}} {
		x, y := unpackShorts(packShorts(c[0], c[1]))
		if x != c[0] || y != c[1] {
			t.Errorf("%v round-trips to (%d,%d)", c, x, y)
		}
	}
}

// royalstrikechaosice: Param1 3 ("repath freq").
func TestChaosIceTurns(t *testing.T) {
	sp := &Spec{ID: 569, Name: "royalstrikechaosice", SrvDoFunc: 35, Vel: 16, MaxVel: 16, Range: 40, CollideType: 3,
		Param: [5]int{3}}

	// seeds of both parities for the first generator step
	var even, odd uint32

	for sd := uint32(1); even == 0 || odd == 0; sd++ {
		if d2rand.New(sd).Step()&1 == 0 {
			if even == 0 {
				even = sd
			}
		} else if odd == 0 {
			odd = sd
		}
	}

	// heading (x, y) -> bit 0: ((4x+y)/4, (4y-x)/4); bit 1: ((4x-y)/4, (4y+x)/4);
	// truncating division, a zero component becomes 1
	cases := []struct {
		name         string
		seed         uint32
		x, y         int
		wantX, wantY int
	}{
		{"even", even, 16, 0, 16, -4},
		{"odd", odd, 16, 0, 16, 4},
		{"even negative", even, -9, 5, -7, 7}, // (-36+5)/4 = -7, (20+9)/4 = 7
		{"odd negative", odd, -9, 5, -10, 2},  // (-36-5)/4 = -10, (20-9)/4 = 2
		{"zero becomes one", even, 1, 0, 1, 1},
	}

	for _, c := range cases {
		w := newWorld()
		s := NewSim(w, specs{})
		m, _ := s.Create(CreateParams{Spec: sp, Level: 5, X: 40.5, Y: 0.5, DestX: 70, DestY: 0.5,
			Data28: c.seed, Data2C: packShorts(c.x, c.y)})

		stepN(s, w, 1) // elapsed 0: a multiple of 3

		x, y := unpackShorts(m.Data2C)
		if x != c.wantX || y != c.wantY {
			t.Errorf("%s: heading (%d,%d), want (%d,%d)", c.name, x, y, c.wantX, c.wantY)
		}

		g := d2rand.New(c.seed)
		g.Step()

		if m.Data28 != g.Lo {
			t.Errorf("%s: seed %d, want %d", c.name, m.Data28, g.Lo)
		}

		before := m.Data2C

		stepN(s, w, 2) // elapsed 1, 2: no repath

		if m.Data2C != before {
			t.Errorf("%s: repathed off the Param1 period", c.name)
		}

		g2 := d2rand.New(m.Data28)
		g2.Step()

		stepN(s, w, 1) // elapsed 3: repaths again from the stored seed

		if m.Data28 != g2.Lo {
			t.Errorf("%s: no repath at elapsed 3", c.name)
		}
	}
}

// blade creeper: SrvDoFunc 20, SrvHitFunc 37, Range 10, NextHit 1 / NextDelay 25.
func TestBladeCreeperFollowsOwnerAndStrikesRepeatedly(t *testing.T) {
	sp := &Spec{ID: 392, Name: "blade creeper", SrvDoFunc: 20, SrvHitFunc: 37, Range: 10, LevRange: 5, CollideType: 3,
		NextHit: true, NextDelay: 25, Size: 3}
	w := newWorld()
	w.targets = []*fakeTarget{newTarget("z", 50, 0)}
	s := NewSim(w, specs{})
	evs := collect(s)

	ox, oy := 50.0, 0.0
	gone := false
	owner := Owner{ID: "creeper", Roller: fakeRoller{0},
		Pos: func() (float64, float64) { return ox, oy }, Gone: func() bool { return gone }}

	m, _ := s.Create(CreateParams{Spec: sp, Level: 1, X: 40.5, Y: 0.5, DestX: 41.5, DestY: 0.5, Owner: owner,
		Damage: DamageDesc{PhysMin: 256, PhysMax: 256}})

	stepN(s, w, 1)

	if int(m.X) != 50 || int(m.Y) != 0 || m.Life != 9 {
		t.Fatalf("at (%.1f,%.1f) life %d, want on the owner's cell with life 9", m.X, m.Y, m.Life)
	}

	stepN(s, w, 99) // far beyond its Range of 10 + 5 per level

	if m.Dead() {
		t.Fatal("the blade creeper must last as long as its owner")
	}

	hits := count(*evs, EventHit)
	if hits != 4 { // frames 1, 26, 51, 76 (NextDelay 25), never destroyed by the hit
		t.Errorf("%d hits in 100 frames, want 4", hits)
	}

	ox = 60
	stepN(s, w, 1)

	if int(m.X) != 60 {
		t.Errorf("did not follow the owner to x=60 (at %.1f)", m.X)
	}

	gone = true
	stepN(s, w, 1)

	if !m.Dead() {
		t.Error("the blade creeper outlived its owner")
	}
}

func TestBladeHitReturns(t *testing.T) {
	s := &Sim{}
	if got := s.bladeHit(newTarget("a", 0, 0)); got != resDamage {
		t.Errorf("with a target %d, want 2", got)
	}

	if got := s.bladeHit(nil); got != 0 {
		t.Errorf("without one %d, want 0", got)
	}
}

// ---- Rabies ----

// rabiesplague / rabiescontagion rows of missiles.txt.
func rabiesSpecs() (plague *Spec, tbl specs) {
	plague = &Spec{ID: 515, Name: "rabiesplague", SrvDoFunc: 30, Param: [5]int{4, 7}, Range: 10, LevRange: 5, CollideType: 3,
		Size: 3, SubMissile: [3]string{"rabiescontagion"}}
	tbl = specs{"rabiescontagion": {ID: 516, Name: "rabiescontagion", SrvDoFunc: 1, SrvHitFunc: 53, Vel: 5, MaxVel: 5,
		Range: 20, Activate: 2, CollideType: 3, LastCollide: true}}

	return plague, tbl
}

func TestRabiesPlagueSpawnsContagionFromTheInfected(t *testing.T) {
	plague, tbl := rabiesSpecs()
	w := newWorld()
	s := NewSim(w, tbl)
	evs := collect(s)

	ix, iy := 30.0, 5.0
	infected := Owner{ID: "monster", Roller: fakeRoller{0}, Pos: func() (float64, float64) { return ix, iy },
		StateExpire: func(state string) (int, bool) { return 1234, state == "rabies" }}
	caster := Owner{ID: "druid", IsPlayer: true, Roller: fakeRoller{0}}

	m, _ := s.Create(CreateParams{Spec: plague, Level: 4, X: 10, Y: 10, DestX: 11, DestY: 10, Owner: infected,
		MarkOwner: &caster, Skill: SkillInfo{AuraTargetState: "rabies", ElemLen: 130},
		Damage: DamageDesc{Poison: Elem{Min: 256, Max: 256, Len: 100}}})

	stepN(s, w, 20)

	if int(m.X) != 30 || int(m.Y) != 5 {
		t.Errorf("the plague is at (%.1f,%.1f), not on the infected monster", m.X, m.Y)
	}

	cs := creates(*evs, "rabiescontagion")
	if len(cs) != 5 { // elapsed 0, 4, 8, 12, 16 (the plague lives until the owner's life is reset: Range 10+5*3 = 25)
		t.Fatalf("%d contagion missiles, want 5", len(cs))
	}

	for _, c := range cs {
		if c.Owner.ID != "druid" {
			t.Errorf("contagion owned by %q, want the caster", c.Owner.ID)
		}

		if c.Data28 != 1234 {
			t.Errorf("contagion carries expiry frame %d, want 1234", c.Data28)
		}

		if c.Damage.Poison.Min != 256 {
			t.Errorf("contagion damage %+v lost", c.Damage)
		}
	}

	// without a caster the plague ends
	s2 := NewSim(newWorld(), tbl)
	m2, _ := s2.Create(CreateParams{Spec: plague, Level: 4, X: 10, Y: 10, DestX: 11, DestY: 10, Owner: infected})
	s2.Step()

	if !m2.Dead() {
		t.Error("a plague without its caster must end")
	}
}

func TestContagionHitHandsOnTheTimeLeft(t *testing.T) {
	_, tbl := rabiesSpecs()
	cases := []struct {
		name   string
		expire int
		frame  int
		want   int // frames handed on, 0 for none
	}{
		{"plenty left", 1000, 900, 100},
		{"exactly 10 left", 1000, 990, 10},
		{"9 left is too little", 1000, 991, 0},
		{"as long as the skill's ELen", 1000, 870, 130},
		{"more than the skill's ELen", 1000, 869, 0},
		{"already over", 1000, 1100, 0},
	}

	for _, c := range cases {
		w := newWorld()
		w.frame = c.frame
		w.targets = []*fakeTarget{newTarget("z", 43, 0)}
		s := NewSim(w, tbl)
		evs := collect(s)

		m, _ := s.Create(CreateParams{Spec: tbl["rabiescontagion"], Level: 4, X: 40.5, Y: 0.5, DestX: 60, DestY: 0.5,
			Owner: Owner{ID: "druid", Roller: fakeRoller{0}}, Data28: uint32(c.expire),
			Skill: SkillInfo{AuraTargetState: "rabies", ElemLen: 130}})

		for i := 0; i < 30 && !m.Dead(); i++ {
			s.Step()
		}

		var got []int

		for _, e := range *evs {
			if e.Kind == EventState {
				if e.Name != "rabies" || e.Target.ID() != "z" {
					t.Errorf("%s: state %q on %s", c.name, e.Name, e.Target.ID())
				}

				got = append(got, e.Frames)
			}
		}

		switch {
		case c.want == 0 && len(got) != 0:
			t.Errorf("%s: handed on %v, want nothing", c.name, got)
		case c.want != 0 && (len(got) != 1 || got[0] != c.want):
			t.Errorf("%s: handed on %v, want [%d]", c.name, got, c.want)
		}
	}
}

// ---- Howl, Shout, Battle Cry ----

type statefulTarget struct {
	*fakeTarget
	has bool
}

func (s *statefulTarget) HasStateNamed(string) bool { return s.has }

func TestHowlHit(t *testing.T) {
	// howl: Param2 1 (Plev+Slev+n), Param3 24 / Param4 5 (distance), Param5 75 /
	// Param6 25 (time); caster level 30, skill level 5: monsters below level 36 flee
	info := SkillInfo{AuraTargetState: "terror", Param: [7]int{0, 2, 1, 24, 5, 75, 25}}
	cases := []struct {
		name      string
		level     int
		player    bool
		has       bool
		wantState bool
	}{
		{"weak monster", 10, false, false, true},
		{"level 35 is below 36", 35, false, false, true},
		{"level 36 resists", 36, false, false, false},
		{"already afraid", 10, false, true, false},
		{"players never", 10, true, false, false},
	}

	for _, c := range cases {
		sp := &Spec{ID: 148, Name: "howl", SrvDoFunc: 1, SrvHitFunc: 17, Vel: 12, MaxVel: 12, Range: 12, CollideType: 3, LastCollide: true}
		w := newWorld()
		tg := &statefulTarget{fakeTarget: newTarget("z", 45, 0), has: c.has}
		tg.level, tg.player = c.level, c.player
		w.targets = []*fakeTarget{tg.fakeTarget}
		// the world hands out the plain fakeTarget; wrap it so Stateful is seen
		ww := &wrapWorld{fakeWorld: w, by: map[string]Target{"z": tg}}
		s := NewSim(ww, specs{})
		evs := collect(s)

		m, _ := s.Create(CreateParams{Spec: sp, Level: 5, X: 40.5, Y: 0.5, DestX: 60, DestY: 0.5, Skill: info,
			Owner: Owner{ID: "barb", IsPlayer: true, Level: 30, Roller: fakeRoller{0}}})

		var states []Event

		for i := 0; i < 12; i++ {
			s.Step()
		}

		for _, e := range *evs {
			if e.Kind == EventState {
				states = append(states, e)
			}
		}

		if !c.wantState {
			if len(states) != 0 {
				t.Errorf("%s: unexpected state %+v", c.name, states)
			}
		} else {
			if len(states) != 1 || states[0].Name != "terror" || states[0].Frames != 75+25*4 || states[0].Distance != 24+5*4 {
				t.Errorf("%s: states %+v, want terror for 175 frames over 44 subtiles", c.name, states)
			}
		}

		if m.Dead() && !c.wantState && !c.player {
			// the missile flies on through units (hit function returns 0)
			continue
		}
	}
}

// wrapWorld hands out wrapped targets by id.
type wrapWorld struct {
	*fakeWorld
	by map[string]Target
}

func (w *wrapWorld) Targets(x, y int) []Target {
	var out []Target

	for _, t := range w.fakeWorld.Targets(x, y) {
		if r, ok := w.by[t.ID()]; ok {
			out = append(out, r)
		} else {
			out = append(out, t)
		}
	}

	return out
}

func (w *wrapWorld) IsEnemy(o Owner, t Target) bool {
	if r, ok := t.(*statefulTarget); ok {
		return w.fakeWorld.IsEnemy(o, r.fakeTarget)
	}

	return w.fakeWorld.IsEnemy(o, t)
}

func TestHowlReturnsAndGuards(t *testing.T) {
	s := &Sim{}
	owner := Owner{ID: "barb"}
	tg := newTarget("z", 0, 0)

	cases := []struct {
		name string
		m    *Missile
		t    Target
		want int
	}{
		{"no owner", &Missile{}, tg, resKill},
		{"no target", &Missile{Owner: owner}, nil, 0},
		{"no state on the skill", &Missile{Owner: owner}, tg, resKill},
		{"gate not passed", &Missile{Owner: owner, Skill: SkillInfo{AuraTargetState: "terror"}}, tg, 0},
	}

	for _, c := range cases {
		if got := s.howlHit(c.m, c.t); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

func TestShoutBuffsAlliesOnly(t *testing.T) {
	// shout: CollideFriend 1, Range 15; the state and its time come from the skill
	sp := &Spec{ID: 149, Name: "shout", SrvDoFunc: 1, SrvHitFunc: 18, Vel: 30, MaxVel: 30, Range: 15, CollideType: 3,
		CollideFriend: true}
	w := newWorld()
	friend := newTarget("party", 45, 0)
	friend.friendly = true
	w.targets = []*fakeTarget{friend, newTarget("foe", 50, 0)}
	s := NewSim(w, specs{})
	evs := collect(s)

	var applied []string

	m, _ := s.Create(CreateParams{Spec: sp, Level: 5, X: 40.5, Y: 0.5, DestX: 70, DestY: 0.5,
		Owner: Owner{ID: "barb", Roller: fakeRoller{0}}, Skill: SkillInfo{AuraTargetState: "shout", AuraLen: 3000},
		OnEffect: func(e Event) { applied = append(applied, e.Target.ID()) }})

	stepN(s, w, 15)

	var states []Event

	for _, e := range *evs {
		if e.Kind == EventState {
			states = append(states, e)
		}
	}

	if len(states) != 1 || states[0].Target.ID() != "party" || states[0].Name != "shout" || states[0].Frames != 3000 {
		t.Fatalf("states %+v, want shout for 3000 frames on the ally only", states)
	}

	if len(applied) != 1 {
		t.Errorf("OnEffect called %d times, want 1", len(applied))
	}

	if m.Dead() && count(*evs, EventHit) != 0 {
		t.Error("the shout missile should not have been consumed by units")
	}
}

func TestBattleCryHit(t *testing.T) {
	// battlecry: aurafilter set (enemies), auratargetstate battlecry
	cases := []struct {
		name     string
		friendly bool
		filtered bool
		state    string
		wantHit  bool
		wantDead bool
	}{
		{"enemy", false, true, "battlecry", true, false},
		{"ally rejected by the filter", true, true, "battlecry", false, true},
		{"no filter takes anyone", true, false, "battlecry", true, false},
		{"skill without a state", false, true, "", false, true},
	}

	for _, c := range cases {
		sp := &Spec{ID: 219, Name: "battlecry", SrvDoFunc: 1, SrvHitFunc: 21, Vel: 12, MaxVel: 12, Range: 12, CollideType: 3,
			CollideFriend: true, LastCollide: true, NextHit: true, NextDelay: 4}
		w := newWorld()
		tg := newTarget("z", 45, 0)
		tg.friendly = c.friendly
		w.targets = []*fakeTarget{tg}
		s := NewSim(w, specs{})
		evs := collect(s)

		m, _ := s.Create(CreateParams{Spec: sp, Level: 5, X: 40.5, Y: 0.5, DestX: 70, DestY: 0.5,
			Owner: Owner{ID: "barb", Roller: fakeRoller{0}},
			Skill: SkillInfo{AuraTargetState: c.state, Filtered: c.filtered, AuraLen: 1200}})

		stepN(s, w, 12)

		got := 0

		for _, e := range *evs {
			if e.Kind == EventState && e.Frames == 1200 && e.Name == "battlecry" {
				got++
			}
		}

		if (got == 1) != c.wantHit || got > 1 {
			t.Errorf("%s: %d state events, want hit=%v", c.name, got, c.wantHit)
		}

		if c.wantDead && !m.Dead() {
			t.Errorf("%s: the missile should have ended (hit function returns 1)", c.name)
		}
	}
}

// ---- Fist of the Heavens ----

type fistWorld struct {
	*fakeWorld
	foes []Target
}

func (w *fistWorld) EnemiesWithin(_ Owner, _, _ float64, r int) []Target {
	var out []Target

	for _, f := range w.foes {
		pt := f.(*posTarget)
		if math.Hypot(pt.px-45.5, pt.py-0.5) <= float64(r) {
			out = append(out, f)
		}
	}

	return out
}

func TestFistOfTheHeavens(t *testing.T) {
	// fistoftheheavensdelay: Range 10, HitSubMissile1 fistoftheheavensbolt,
	// sHitPar1/2 empty -> aurarangecalc 20 and calc4 (# bolts)
	delay := &Spec{ID: 233, Name: "fistoftheheavensdelay", SrvDoFunc: 1, SrvHitFunc: 22, Range: 10, AlwaysExplode: true,
		Explosion: true, HitSubMissile: [4]string{"fistoftheheavensbolt"}}
	bolt := &Spec{ID: 234, Name: "fistoftheheavensbolt", SrvDoFunc: 1, SrvHitFunc: 7, Vel: 12, MaxVel: 12, Range: 50,
		CollideType: 3}
	tbl := specs{"fistoftheheavensbolt": bolt}

	mk := func(id string, x float64) *posTarget {
		return &posTarget{fakeTarget: newTarget(id, int(x), 0), px: x, py: 0.5}
	}

	mark := mk("mark", 45.5)
	foes := []Target{mark, mk("a", 50.5), mk("b", 52.5), mk("c", 55.5), mk("far", 80.5)}

	cases := []struct {
		name      string
		mark      Target
		radius    int
		count     int
		sHit      [3]int
		wantBolts int
		wantHit   bool
	}{
		{"calc4 bolts, radius 20", mark, 20, 2, [3]int{}, 2, true},
		{"all enemies in range", mark, 20, 9, [3]int{}, 4, true},
		{"table radius and count win", mark, 20, 9, [3]int{5, 1}, 1, true},
		{"marked unit gone", nil, 20, 9, [3]int{}, 0, false},
	}

	for _, c := range cases {
		w := &fistWorld{fakeWorld: newWorld(), foes: foes}
		s := NewSim(w, tbl)
		evs := collect(s)
		sp := *delay
		sp.SHitPar = c.sHit

		_, _ = s.Create(CreateParams{Spec: &sp, Level: 10, X: 45.5, Y: 0.5, DestX: 45.5, DestY: 0.5, Stationary: true,
			Owner: Owner{ID: "pal", IsPlayer: true, Roller: fakeRoller{0}}, Mark: c.mark, AreaRadius: c.radius, FuryCount: c.count,
			Damage: DamageDesc{Lightning: Elem{Min: 512, Max: 512}}})

		stepN(s, w.fakeWorld, 12)

		if n := len(creates(*evs, "fistoftheheavensbolt")); n != c.wantBolts {
			t.Errorf("%s: %d bolts, want %d", c.name, n, c.wantBolts)
		}

		hit := false

		for _, e := range *evs {
			if e.Kind == EventHit && e.Target != nil && e.Target.ID() == "mark" && e.Damage.Lightning == 512 {
				hit = true
			}
		}

		if hit != c.wantHit {
			t.Errorf("%s: marked unit struck = %v, want %v", c.name, hit, c.wantHit)
		}
	}
}

// ---- Bone Wall ----

func TestBoneWallMakerSummonsAtEachNewSubtile(t *testing.T) {
	// bonewallmaker: Vel 12, Range 7 + 2 per level, SrvDoFunc 13
	sp := &Spec{ID: 207, Name: "bonewallmaker", SrvDoFunc: 13, Vel: 12, MaxVel: 12, Range: 7, LevRange: 2, CollideType: 8,
		LastCollide: true, Size: 1, CanSlow: true}
	leader := newTarget("leader", 0, 0)
	owner := Owner{ID: "nec", IsPlayer: true, Roller: fakeRoller{0}}

	run := func(walls uint32, mark Target, gone *bool) (summons int, m *Missile) {
		w := newWorld()
		s := NewSim(w, specs{})
		evs := collect(s)
		o := owner

		if gone != nil {
			o.Gone = func() bool { return *gone }
		}

		m, _ = s.Create(CreateParams{Spec: sp, Level: 10, X: 40.5, Y: 0.5, DestX: 70, DestY: 0.5, Owner: o, Data2C: walls, Mark: mark})
		stepN(s, w, 30)

		for _, e := range *evs {
			if e.Kind == EventSummon {
				summons++

				if e.Target.ID() != "leader" || e.Missile != m {
					t.Errorf("summon event %+v", e)
				}
			}
		}

		return summons, m
	}

	if n, m := run(3, leader, nil); n != 3 || !m.Dead() || m.Data2C != 0 {
		t.Errorf("3 walls: %d summons, dead=%v, left=%d; want 3, true, 0", n, m.Dead(), m.Data2C)
	}

	// 12 per frame*... speed: Range 7+2*10 = 27 frames at 12*0.75 = 9/256*16 subtile per frame
	if n, m := run(100, leader, nil); n < 5 || !m.Dead() {
		t.Errorf("100 walls: %d summons, dead=%v; want one per entered subtile until the range runs out", n, m.Dead())
	}

	if n, m := run(3, nil, nil); n != 0 || m.Data2C != 3 {
		t.Errorf("without the leader: %d summons, %d left; want 0 and 3 (the counter only drops on a summon)", n, m.Data2C)
	}

	gone := true
	if n, m := run(3, leader, &gone); n != 0 || !m.Dead() {
		t.Errorf("owner gone: %d summons, dead=%v; want 0, true", n, m.Dead())
	}

	if n, m := run(0, leader, nil); n != 0 || !m.Dead() {
		t.Errorf("no walls to place: %d summons, dead=%v; want 0, true", n, m.Dead())
	}
}
