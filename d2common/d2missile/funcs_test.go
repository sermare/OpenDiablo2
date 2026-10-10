package d2missile

import (
	"math"
	"testing"
)

// The specs below carry numbers copied from missiles.txt (patch_d2) and the
// skills they belong to; the expected positions of the scatter tests come from
// the exe's seed generator (multiplier 0x6AC690C5, high word 0x29A, verified in
// d2rand) evaluated by hand with the formulas of 0x5a7280 / 0x5ad6f0.

func ownerWith(roll uint32) Owner { return Owner{ID: "hero", Roller: fakeRoller{roll}} }

func creates(evs []Event, name string) (out []*Missile) {
	for _, e := range evs {
		if e.Kind == EventCreate && e.Missile.Spec.Name == name {
			out = append(out, e.Missile)
		}
	}

	return out
}

func stepN(s *Sim, w *fakeWorld, n int) {
	for i := 0; i < n; i++ {
		w.frame++
		s.Step()
	}
}

func blizzardSpecs(centerDo int) specs {
	return specs{
		"blizzardcenter": {ID: 158, Name: "blizzardcenter", SrvDoFunc: centerDo, Range: 100,
			SubMissile: [3]string{"blizzard1"}, LastCollide: true},
		"blizzard1": {ID: 159, Name: "blizzard1", SrvDoFunc: 3, Range: 9, CollideType: 3, LastCollide: true,
			AlwaysExplode: true, Size: 2},
	}
}

// SrvDoFunc 10 (Blizzard, 0x5ac5b0 -> 0x5a7280): calc1 = Param1 = 7 is the
// radius, calc2 = Param3 = 4 the period; Range 100 gives 25 shards.
func TestBlizzardCenterScatter(t *testing.T) {
	tab := blizzardSpecs(10)
	w := newWorld()
	s := NewSim(w, tab)
	evs := collect(s)

	_, err := s.Create(CreateParams{Spec: tab["blizzardcenter"], Owner: ownerWith(0), SkillID: 59, Level: 20,
		X: 40.5, Y: 0.5, DestX: 40.5, DestY: 0.5, SpawnRadius: 7, SpawnEvery: 4})
	if err != nil {
		t.Fatal(err)
	}

	stepN(s, w, 100)

	shards := creates(*evs, "blizzard1")
	if len(shards) != 25 {
		t.Fatalf("%d shards, want 25 (elapsed 0, 4, ..., 96)", len(shards))
	}

	// seed = integer x + elapsed, two rolls of 2*(7-1) = 12, offset by -6
	for i, want := range [][2]int{{36, 4}, {40, -6}, {44, -4}, {40, 1}} {
		if x, y := int(math.Floor(shards[i].X)), int(math.Floor(shards[i].Y)); x != want[0] || y != want[1] {
			t.Errorf("shard %d at (%d,%d), want %v", i, x, y, want)
		}
	}

	for _, sh := range shards {
		if x, y := int(sh.X), int(sh.Y); x < 34 || x > 46 || y < -6 || y > 6 {
			t.Errorf("shard outside radius 6: %v", sh)
		}

		if sh.Velocity != 0 || sh.Life != sh.Total && sh.Spec.Range != 9 {
			t.Errorf("shard %v should be stationary, range 9", sh)
		}
	}
}

func TestScatterTableCases(t *testing.T) {
	cases := []struct {
		name          string
		radius, every int
		wall          uint16 // bit set on every cell
		do            int
		want          int
	}{
		{"no period", 7, 0, 0, 10, 0},
		{"radius 1 is the centre cell", 1, 25, 0, 10, 4},
		{"walls refuse blizzard", 7, 25, 0x04, 10, 0},
		{"walk bit refuses blizzard", 7, 25, 0x01, 10, 0},
		{"missile bit is fine for blizzard", 7, 25, 0x40, 10, 4},
		{"missile bit refuses eruption", 7, 25, 0x40, 25, 0},
		{"eruption in the open", 7, 25, 0, 25, 4},
	}

	for _, tc := range cases {
		tab := blizzardSpecs(tc.do)
		w := newWorld()

		for x := 20; x < 60; x++ {
			for y := -15; y < 15; y++ {
				w.grid.Set(x, y, tc.wall)
			}
		}

		s := NewSim(w, tab)
		evs := collect(s)
		_, _ = s.Create(CreateParams{Spec: tab["blizzardcenter"], Owner: ownerWith(0), SkillID: 59, Level: 1,
			X: 40.5, Y: 0.5, DestX: 40.5, DestY: 0.5, SpawnRadius: tc.radius, SpawnEvery: tc.every})
		stepN(s, w, 100)

		got := creates(*evs, "blizzard1")
		if len(got) != tc.want {
			t.Errorf("%s: %d shards, want %d", tc.name, len(got), tc.want)
		}

		if tc.name == "radius 1 is the centre cell" {
			for _, g := range got {
				if int(g.X) != 40 || int(g.Y) != 0 {
					t.Errorf("radius 1 shard at %v", g)
				}
			}
		}
	}
}

// SrvDoFunc 28 (Volcano, 0x5ad6f0): Param3 2 < elapsed < Param4 128, period
// calc4 = 2, radius aurarangecalc = 12, seeded with data field 0x28 which is
// advanced to the seed's low word after every spawn.
func TestVolcanoSpread(t *testing.T) {
	tab := specs{
		"volcano": {ID: 479, Name: "volcano", SrvDoFunc: 28, Range: 150, Param: [5]int{0, 0, 2, 128, 30},
			SubMissile: [3]string{"volcano debris 2"}},
		"volcano debris 2": {ID: 481, Name: "volcano debris 2", SrvDoFunc: 1, SrvHitFunc: 51, Vel: 14, MaxVel: 14,
			Range: 16, AlwaysExplode: true, HitSubMissile: [4]string{"volcano small fire"}},
		"volcano small fire": {ID: 482, Name: "volcano small fire", SrvDoFunc: 1, Range: 100},
	}
	w := newWorld()
	s := NewSim(w, tab)
	evs := collect(s)

	m, _ := s.Create(CreateParams{Spec: tab["volcano"], Owner: ownerWith(0), SkillID: 244, Level: 5, X: 50.5, Y: 0.5,
		DestX: 50.5, DestY: 0.5, PulseEvery: 2, AreaRadius: 12, Data28: 77})

	stepN(s, w, 3) // elapsed 0, 1, 2: not inside (2, 128) yet
	if n := len(creates(*evs, "volcano debris 2")); n != 0 {
		t.Fatalf("%d debris before elapsed 3", n)
	}

	stepN(s, w, 1) // elapsed 3: odd
	stepN(s, w, 1) // elapsed 4
	deb := creates(*evs, "volcano debris 2")

	if len(deb) != 1 || int(deb[0].destX) != 52 || int(deb[0].destY) != 5 || m.Data28 != 2767673767 {
		t.Fatalf("first debris %v dest (%v,%v) data28 %d", deb, deb[0].destX, deb[0].destY, m.Data28)
	}

	stepN(s, w, 2) // elapsed 5, 6
	stepN(s, w, 2) // 7, 8
	deb = creates(*evs, "volcano debris 2")

	for i, want := range [][2]int{{52, 5}, {56, 10}, {50, 1}} {
		if int(deb[i].destX) != want[0] || int(deb[i].destY) != want[1] {
			t.Errorf("debris %d dest (%v,%v), want %v", i, deb[i].destX, deb[i].destY, want)
		}
	}

	stepN(s, w, 150)

	// elapsed 4, 6, ..., 126
	if n := len(creates(*evs, "volcano debris 2")); n != 62 {
		t.Errorf("%d debris, want 62", n)
	}

	// a debris lands on its aim point and spawns the small fire (hit function 51)
	if n := len(creates(*evs, "volcano small fire")); n == 0 {
		t.Errorf("no small fire from the landed debris")
	}
}

// SrvDoFunc 14 and hit functions 26 / 27 (Grim Ward): Param1 6 / Param2 30;
// the start missile creates the ward with the skill's calc1 (min 5) frames.
func TestGrimWardMissiles(t *testing.T) {
	ward := &Spec{ID: 256, Name: "grimwardlarge", SrvDoFunc: 14, SrvHitFunc: 27, Param: [5]int{6, 30}, Range: 200,
		AlwaysExplode: true}
	start := &Spec{ID: 255, Name: "grimwardlargestart", SrvDoFunc: 1, SrvHitFunc: 26, Range: 8, AlwaysExplode: true,
		HitSubMissile: [4]string{"grimwardlarge"}}
	tab := specs{"grimwardlarge": ward, "grimwardlargestart": start}

	w := newWorld()
	s := NewSim(w, tab)
	evs := collect(s)
	_, _ = s.Create(CreateParams{Spec: ward, Owner: ownerWith(0), SkillID: 150, Level: 4, X: 10.5, Y: 0.5, DestX: 10.5,
		DestY: 0.5, Stationary: true})
	stepN(s, w, 200)

	n := 0

	for _, e := range *evs {
		if e.Kind == EventPeriodic {
			if e.Helper != 30 {
				t.Errorf("helper %d", e.Helper)
			}

			n++
		}
	}

	if n != 34 { // elapsed 0, 6, ..., 198
		t.Errorf("%d periodic dispatches, want 34", n)
	}

	// a missile without a skill is destroyed at once (return 2)
	w2 := newWorld()
	s2 := NewSim(w2, tab)
	m2, _ := s2.Create(CreateParams{Spec: ward, Owner: ownerWith(0), X: 10.5, Y: 0.5, DestX: 10.5, DestY: 0.5, Stationary: true})
	stepN(s2, w2, 1)

	if !m2.Dead() {
		t.Error("ward without a skill should vanish")
	}

	for _, tc := range []struct{ par, calc, want int }{{0, 1000, 1000}, {0, 2, 5}, {0, 0, 5}, {77, 1000, 77}} {
		s3 := NewSim(newWorld(), tab)
		start.SHitPar[0] = tc.par
		evs3 := collect(s3)
		_, _ = s3.Create(CreateParams{Spec: start, Owner: ownerWith(0), SkillID: 150, Level: 4, X: 10.5, Y: 0.5,
			DestX: 10.5, DestY: 0.5, Stationary: true, HitSubRange: tc.calc})

		stepN(s3, s3.World.(*fakeWorld), 8)

		got := creates(*evs3, "grimwardlarge")
		if len(got) != 1 || got[0].Total != tc.want {
			t.Errorf("sHitPar1 %d calc1 %d: ward %v, want life %d", tc.par, tc.calc, got, tc.want)
		}
	}
}

// SrvDoFunc 23 (Firestorm's emitter): SubMissile1 in every subtile the emitter
// enters, one frame later; Param1 lengthens the sub missile by loops.
func TestFirestormEmitter(t *testing.T) {
	storm := &Spec{ID: 457, Name: "firestorm", SrvDoFunc: 1, Range: 15, SubLoop: true, SubStart: 12, SubStop: 36}
	maker := &Spec{ID: 458, Name: "firestormmaker", SrvDoFunc: 23, Vel: 8, MaxVel: 8, Range: 40,
		SubMissile: [3]string{"firestorm"}}
	tab := specs{"firestorm": storm, "firestormmaker": maker}

	w := newWorld()
	s := NewSim(w, tab)
	evs := collect(s)
	_, _ = s.Create(CreateParams{Spec: maker, Owner: ownerWith(0), SkillID: 225, Level: 3, X: 10.5, Y: 0.5, DestX: 60.5, DestY: 0.5})
	stepN(s, w, 40)

	got := creates(*evs, "firestorm")
	if len(got) < 13 || len(got) > 15 {
		t.Fatalf("%d fires", len(got))
	}

	for i, g := range got {
		if int(g.X) != 11+i {
			t.Errorf("fire %d in cell %d, want %d (one per entered subtile)", i, int(g.X), 11+i)
		}

		if g.Total != 15 {
			t.Errorf("fire life %d without Param1", g.Total)
		}
	}

	maker.Param[0] = 2
	s2 := NewSim(newWorld(), tab)
	evs2 := collect(s2)
	_, _ = s2.Create(CreateParams{Spec: maker, Owner: ownerWith(0), SkillID: 225, Level: 3, X: 10.5, Y: 0.5, DestX: 60.5, DestY: 0.5})
	stepN(s2, s2.World.(*fakeWorld), 5)

	if g := creates(*evs2, "firestorm"); len(g) == 0 || g[0].Total != 15+(36-12)*2 {
		t.Errorf("Param1 2: fire %v, want life %d", g, 15+(36-12)*2)
	}

	// no SubMissile1: the emitter is destroyed (return 2)
	bare := &Spec{ID: 9, Name: "bare", SrvDoFunc: 24, Vel: 8, MaxVel: 8, Range: 40}
	s3 := NewSim(newWorld(), specs{"bare": bare})
	m3, _ := s3.Create(CreateParams{Spec: bare, Owner: ownerWith(0), X: 10.5, Y: 0.5, DestX: 60.5, DestY: 0.5})
	stepN(s3, s3.World.(*fakeWorld), 1)

	if !m3.Dead() {
		t.Error("emitter without a sub missile should vanish")
	}
}

type markWorld struct {
	*fakeWorld
	marks [][4]int
}

func (w *markWorld) MarkCell(x, y, size int, flag uint16) {
	w.marks = append(w.marks, [4]int{x, y, size, int(flag)})
}

// SrvDoFunc 3 and 5 stamp bit 0x40 into the cell under the missile; 5 also
// runs the fire's frame animation (SubStart 12, SubStop 36, Range 90).
func TestCellStampAndFireFrames(t *testing.T) {
	cloud := &Spec{ID: 39, Name: "poisonjavcloud", SrvDoFunc: 3, Range: 60, Size: 1}
	fire := &Spec{ID: 69, Name: "firewall", SrvDoFunc: 5, Range: 90, Size: 1, SubLoop: true, SubStart: 12, SubStop: 36}
	tab := specs{"poisonjavcloud": cloud, "firewall": fire}

	w := &markWorld{fakeWorld: newWorld()}
	s := NewSim(w, tab)
	_, _ = s.Create(CreateParams{Spec: cloud, Owner: ownerWith(0), X: 20.5, Y: 3.5, DestX: 20.5, DestY: 3.5, Stationary: true})
	stepN(s, w.fakeWorld, 10)

	if len(w.marks) != 10 || w.marks[0] != [4]int{20, 3, 1, 0x40} {
		t.Errorf("marks %v", w.marks)
	}

	w2 := newWorld()
	s2 := NewSim(w2, tab)
	m, _ := s2.Create(CreateParams{Spec: fire, Owner: ownerWith(0), X: 20.5, Y: 3.5, DestX: 20.5, DestY: 3.5, Stationary: true})

	// the fade starts when the remaining life equals SubStart (12): step 79
	stepN(s2, w2, 78)

	if m.Frame != 0 {
		t.Fatalf("frame %d before the fade", m.Frame)
	}

	var seen []int

	for i := 0; i < 8; i++ {
		stepN(s2, w2, 1)
		seen = append(seen, m.Frame>>8)
	}

	// life 12 -> start-3 = 9; below 12 -> back off by 2, floored at 0
	want := []int{9, 7, 5, 3, 1, 0, 0, 0}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("frames %v, want %v", seen, want)
		}
	}

	// a frame sitting at start-1 re-rolls to start-1 + rand(stop-start) with the
	// missile's own seed (first missile: seed 1, first roll mod 24 = 15)
	s3 := NewSim(newWorld(), tab)
	m3, _ := s3.Create(CreateParams{Spec: fire, Owner: ownerWith(0), X: 20.5, Y: 3.5, DestX: 20.5, DestY: 3.5, Stationary: true})
	m3.Frame = 11 << 8
	stepN(s3, s3.World.(*fakeWorld), 1)

	if m3.Frame != (11+15)<<8 {
		t.Errorf("rerolled frame %d, want %d", m3.Frame>>8, 11+15)
	}
}

type stateOwner struct{ states map[int]bool }

// Hit function 8 (Blaze, 0x5a7c20): the caster in state 13 is not hurt (0);
// everyone else takes the damage (2) and the fire stays.
func TestBlazeHit(t *testing.T) {
	blaze := &Spec{ID: 67, Name: "blaze", SrvDoFunc: 5, SrvHitFunc: 8, CollideType: 3, Range: 90}
	hero := newTarget("hero", 5, 0)
	foe := newTarget("foe", 6, 0)

	cases := []struct {
		name  string
		has   func(int) bool
		tgt   Target
		wantR int
	}{
		{"caster while blazing", func(st int) bool { return st == 13 }, hero, 0},
		{"caster not blazing", func(int) bool { return false }, hero, resDamage},
		{"caster, other state", func(st int) bool { return st == 12 }, hero, resDamage},
		{"caster, no state check", nil, hero, resDamage},
		{"enemy while the caster blazes", func(st int) bool { return st == 13 }, foe, resDamage},
		{"no target", func(int) bool { return true }, nil, resDamage},
	}

	for _, tc := range cases {
		s := NewSim(newWorld(), specs{"blaze": blaze})
		m := &Missile{Spec: blaze, Owner: Owner{ID: "hero", HasState: tc.has}}

		if r, ok := s.hitFunc(m, tc.tgt); !ok || r != tc.wantR {
			t.Errorf("%s: %d ok=%v, want %d", tc.name, r, ok, tc.wantR)
		}
	}
}

// Hit function 9 (Immolation Arrow, 0x5a7cf0): a disc of fire missiles (calc1
// = Param1 = 3, life SHitCalc1 = 100), skipping wall cells, one area damage of
// radius calc2 = Param2 = 4 with the chill and freeze stats cleared; returns 3.
func TestImmolationArrowHit(t *testing.T) {
	fire := &Spec{ID: 230, Name: "immolationfire", SrvDoFunc: 5, CollideType: 3, Range: 100}
	arrow := &Spec{ID: 85, Name: "immolationarrow", SrvDoFunc: 1, SrvHitFunc: 9, Vel: 24, Range: 40,
		HitSubMissile: [4]string{"immolationfire"}}
	tab := specs{"immolationfire": fire, "immolationarrow": arrow}

	cases := []struct {
		name         string
		par1, par2   int
		wallCells    [][2]int
		wantFire     int
		wantAreaRadi int
	}{
		{"skill radii (L20: 3 and 4)", 0, 0, nil, 29, 4},
		{"two walls in the disc", 0, 0, [][2]int{{31, 0}, {30, 3}}, 27, 4},
		{"sHitPar1 2 / sHitPar2 6", 2, 6, nil, 13, 6},
	}

	for _, tc := range cases {
		w := newWorld()
		for _, c := range tc.wallCells {
			w.grid.Set(c[0], c[1], 0x04)
		}

		s := NewSim(w, tab)
		evs := collect(s)
		arrow.SHitPar = [3]int{tc.par1, tc.par2}

		m, _ := s.Create(CreateParams{Spec: arrow, Owner: ownerWith(0), SkillID: 27, Level: 20, X: 30.5, Y: 0.5, DestX: 60.5,
			DestY: 0.5, DiscRadius: 3, DiscLife: 100, AreaRadius: 4,
			Damage: DamageDesc{Fire: Elem{Min: 256, Max: 256}, Cold: Elem{Min: 256, Max: 256, Len: 50}, FreezeLen: 30}})

		ret, ok := s.hitFunc(m, nil)

		if !ok || ret != resKill|resDamage {
			t.Errorf("%s: ret %d ok=%v, want 3", tc.name, ret, ok)
		}

		fires := creates(*evs, "immolationfire")
		if len(fires) != tc.wantFire {
			t.Errorf("%s: %d fires, want %d", tc.name, len(fires), tc.wantFire)
		}

		for _, f := range fires {
			if f.Total != 100 {
				t.Errorf("%s: fire life %d, want 100", tc.name, f.Total)
			}
		}

		var area *Event

		for i := range *evs {
			if (*evs)[i].Kind == EventArea {
				area = &(*evs)[i]
			}
		}

		if area == nil || area.Radius != tc.wantAreaRadi || area.Damage.ColdLen != 0 || area.Damage.FreezeLen != 0 ||
			area.Damage.Cold != 256 {
			t.Errorf("%s: area %+v, want radius %d, chill/freeze cleared", tc.name, area, tc.wantAreaRadi)
		}
	}
}

type boulderTarget struct {
	*fakeTarget
	bit bool
}

func (b boulderTarget) BoulderBit() bool { return b.bit }

// Hit functions 48 / 47 (Molten Boulder): the emerge missile ends by creating
// the boulder toward the original aim point; the boulder only damages (2) the
// units it rolls through and explodes (area damage + fire path at every
// sHitPar2-th Meteor offset) with no target or a unit with the class bit.
func TestMoltenBoulderHits(t *testing.T) {
	path := &Spec{ID: 455, Name: "moltenboulderfirepath", SrvDoFunc: 5, CollideType: 8, Range: 37}
	boulder := &Spec{ID: 452, Name: "moltenboulder", SrvDoFunc: 6, SrvHitFunc: 47, Vel: 4, MaxVel: 4, Range: 100,
		CollideType: 8, SHitPar: [3]int{0, 1}, SubMissile: [3]string{"moltenboulderfirepath"},
		HitSubMissile: [4]string{"moltenboulderfirepath"}}
	emerge := &Spec{ID: 453, Name: "moltenboulderemerge", SrvDoFunc: 1, SrvHitFunc: 48, Vel: 3, MaxVel: 3, Range: 5,
		CollideType: 8, HitSubMissile: [4]string{"moltenboulder"}}
	tab := specs{"moltenboulderfirepath": path, "moltenboulder": boulder, "moltenboulderemerge": emerge}

	w := newWorld()
	s := NewSim(w, tab)
	evs := collect(s)
	m, _ := s.Create(CreateParams{Spec: emerge, Owner: ownerWith(0), SkillID: 229, Level: 8, X: 20.5, Y: 0.5, DestX: 45.5,
		DestY: 0.5})
	stepN(s, w, 12)

	got := creates(*evs, "moltenboulder")
	if !m.Dead() || len(got) != 1 || got[0].destX != 45.5 || got[0].destY != 0.5 || got[0].X < 20.5 {
		t.Fatalf("emerge %v dead=%v boulders %v", m, m.Dead(), got)
	}

	player := &fakeTarget{id: "p", alive: true, player: true}
	plain := boulderTarget{newTarget("m1", 0, 0), false}
	marked := boulderTarget{newTarget("m2", 0, 0), true}

	for _, tc := range []struct {
		name    string
		t       Target
		wantRet int
		fires   int
	}{
		{"rolls through a monster", plain, resDamage, 0},
		{"rolls through a player", player, resDamage, 0},
		{"explodes on the class bit", marked, resKill, 18},
		{"explodes at the end of its path", nil, resKill, 18},
	} {
		w := newWorld()
		s := NewSim(w, tab)
		evs := collect(s)
		bm := &Missile{Spec: boulder, Owner: ownerWith(0), AreaRadius: 5, X: 30.5, Y: 0.5}

		ret, _ := s.hitFunc(bm, tc.t)
		if ret != tc.wantRet || len(creates(*evs, "moltenboulderfirepath")) != tc.fires {
			t.Errorf("%s: ret %d fires %d, want %d / %d", tc.name, ret, len(creates(*evs, "moltenboulderfirepath")),
				tc.wantRet, tc.fires)
		}

		if tc.fires > 0 && count(*evs, EventArea) != 1 {
			t.Errorf("%s: no area damage", tc.name)
		}
	}

	// sHitPar2 3: every third of the 18 offsets
	boulder.SHitPar = [3]int{0, 3}
	s4 := NewSim(newWorld(), tab)
	evs4 := collect(s4)
	_, _ = s4.hitFunc(&Missile{Spec: boulder, Owner: ownerWith(0), AreaRadius: 5, X: 30.5, Y: 0.5}, nil)

	if n := len(creates(*evs4, "moltenboulderfirepath")); n != 6 {
		t.Errorf("sHitPar2 3: %d fires, want 6", n)
	}
}

// Hit functions 51 and 56: sub missiles at the missile, the missile ends.
func TestDebrisAndArmageddonHits(t *testing.T) {
	sub := func(n string) *Spec { return &Spec{Name: n, SrvDoFunc: 1, Range: 20} }
	deb := &Spec{ID: 481, Name: "volcano debris 2", SrvHitFunc: 51,
		HitSubMissile: [4]string{"a", "b", "c", "d"}}
	ctl := &Spec{ID: 600, Name: "armageddoncontrol", SrvHitFunc: 56, HitSubMissile: [4]string{"a"}}
	tab := specs{"a": sub("a"), "b": sub("b"), "c": sub("c"), "d": sub("d")}

	s := NewSim(newWorld(), tab)
	evs := collect(s)
	ret, ok := s.hitFunc(&Missile{Spec: deb, Owner: ownerWith(0), X: 30.5, Y: 0.5}, nil)

	if !ok || ret != resKill || len(creates(*evs, "a"))+len(creates(*evs, "b"))+len(creates(*evs, "c")) != 3 ||
		len(creates(*evs, "d")) != 0 {
		t.Errorf("hit 51: ret %d, events %d", ret, len(*evs))
	}

	s2 := NewSim(newWorld(), tab)
	evs2 := collect(s2)
	ret, ok = s2.hitFunc(&Missile{Spec: ctl, Owner: ownerWith(0), AreaRadius: 8, X: 30.5, Y: 0.5}, nil)

	if !ok || ret != resKill || count(*evs2, EventArea) != 1 || len(creates(*evs2, "a")) != 1 {
		t.Errorf("hit 56: ret %d, events %v", ret, kinds(*evs2))
	}
}

// Sub missiles with damage columns of their own (burning ground) get their own
// damage from the parent's ChildDamage map and pass it on.
func TestChildDamageMap(t *testing.T) {
	fire := &Spec{ID: 240, Name: "meteorfire", SrvDoFunc: 5, Range: 90}
	center := &Spec{ID: 101, Name: "meteorcenter", SrvDoFunc: 1, SrvHitFunc: 14, Range: 5, SHitPar: [3]int{0, 6},
		HitSubMissile: [4]string{"meteorfire"}}
	tab := specs{"meteorfire": fire, "meteorcenter": center}

	own := DamageDesc{Fire: Elem{Min: 15 << 3, Max: 25 << 3}}
	impact := DamageDesc{Fire: Elem{Min: 869 << 8, Max: 927 << 8}}

	s := NewSim(newWorld(), tab)
	evs := collect(s)
	_, _ = s.Create(CreateParams{Spec: center, Owner: ownerWith(0), SkillID: 56, Level: 20, X: 30.5, Y: 0.5, DestX: 30.5,
		DestY: 0.5, Damage: impact, AreaRadius: 6, ChildDamage: map[string]DamageDesc{"meteorfire": own}})
	stepN(s, s.World.(*fakeWorld), 6)

	fires := creates(*evs, "meteorfire")
	if len(fires) != 3 { // offsets 0, 6, 12 of 18
		t.Fatalf("%d fires, want 3", len(fires))
	}

	for _, f := range fires {
		if f.Damage != own || f.ChildDamage == nil {
			t.Errorf("fire damage %+v", f.Damage)
		}
	}

	// no map entry: the child inherits the parent's damage as before
	s2 := NewSim(newWorld(), tab)
	evs2 := collect(s2)
	_, _ = s2.Create(CreateParams{Spec: center, Owner: ownerWith(0), SkillID: 56, Level: 20, X: 30.5, Y: 0.5, DestX: 30.5,
		DestY: 0.5, Damage: impact, AreaRadius: 6})
	stepN(s2, s2.World.(*fakeWorld), 6)

	if f := creates(*evs2, "meteorfire"); len(f) != 3 || f[0].Damage != impact {
		t.Errorf("inherited damage %v", f)
	}
}

// The Tornado pulse tests the ELAPSED life (0x64b640 = +0xe - +0x10, whose
// setters 0x64b580 / 0x64b5e0 were read): with Range 74 and a period of 15 the
// pulses fall at elapsed 0, 15, 30, 45, 60 (five), not at the remaining life
// 60, 45, 30, 15 (four).
func TestTornadoUsesElapsedLife(t *testing.T) {
	sp := &Spec{ID: 478, Name: "tornado", SrvDoFunc: 27, Vel: 8, MaxVel: 8, Range: 74, CollideType: 3}
	w := newWorld()
	s := NewSim(w, nil)
	evs := collect(s)

	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 60, Owner: ownerWith(0), PulseEvery: 15, AreaRadius: 3})
	stepN(s, w, 73)

	if n := count(*evs, EventArea); n != 5 {
		t.Errorf("%d pulses, want 5", n)
	}
}
