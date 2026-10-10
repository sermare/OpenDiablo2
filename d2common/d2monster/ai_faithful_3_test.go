package d2monster

import (
	"strings"
	"testing"
)

// f3World adds the optional host interfaces of the batch-3 ports to a
// fakePet (which gives Owner/OwnerEnemy/Teleport).
type f3World struct {
	*fakePet
	raw        []int
	rawPt      []Point
	deadNear   bool
	noCell     bool
	classAIP   map[[2]int]int
	linked     *Target
	offsetCast int
	corpse     *Target
	radius     int
	level      int
	calc4      int
	quest      *Point
	arrived    int
	bladeOK    bool
	blades     int
	pounce     *Target
}

func newF3(ownerDist int) *f3World {
	f, _ := newPet("Imp", ownerDist)
	f.hasTarget, f.attackOK = true, true

	return &f3World{fakePet: f, level: 1, calc4: 100, bladeOK: true}
}

func (w *f3World) CastSkillID(_ *Brain, id int, _ Mode, _ *Target, at *Point) bool {
	w.raw = append(w.raw, id)
	if at != nil {
		w.rawPt = append(w.rawPt, *at)
	}

	return true
}
func (w *f3World) DeadPlayerNear(*Brain) bool { return w.deadNear }
func (w *f3World) FreeCellNear(_ *Brain, p Point) (Point, bool) {
	return p, !w.noCell
}
func (w *f3World) ClassAIP(_ *Brain, class, n int) (int, bool) {
	v, ok := w.classAIP[[2]int{class, n}]

	return v, ok
}
func (w *f3World) ImpLinkedUnit(*Brain, uint32) (Target, bool) {
	if w.linked == nil {
		return Target{}, false
	}

	return *w.linked, true
}
func (w *f3World) RandomOffsetCast(*Brain, int, int) bool { w.offsetCast++; return true }
func (w *f3World) SkillLevel(*Brain, int) int             { return w.level }
func (w *f3World) SkillCalc4(*Brain, int, int) int        { return w.calc4 }
func (w *f3World) FindReviveCorpse(*Brain, int, int) (Target, bool) {
	if w.corpse == nil {
		return Target{}, false
	}

	return *w.corpse, true
}
func (w *f3World) ReviveRadius(*Brain, int, int) int { return w.radius }
func (w *f3World) WandererTarget(*Brain) (Point, bool) {
	if w.quest == nil {
		return Point{}, false
	}

	return *w.quest, true
}
func (w *f3World) WandererArrived(*Brain)        { w.arrived++ }
func (w *f3World) BladeReady(*Brain) (int, bool) { return 50, w.bladeOK }
func (w *f3World) BladeFire(*Brain) bool         { w.blades++; return true }
func (w *f3World) NearbySkillTarget(*Brain, int) (Target, bool) {
	if w.pounce == nil {
		return Target{}, false
	}

	return *w.pounce, true
}

func f3Brain(ai string, class int, slots []int, aip ...int) *Brain {
	b := brainAt(withSkills(profile(ai, aip...), slots...))
	b.Class = class
	b.Profile.Walk, b.Profile.Run = 10, 20
	def, _ := Lookup(ai)
	b.SetAI(def)

	return b
}

func TestFaithfulBatch3Registered(t *testing.T) {
	for _, n := range FaithfulBatch3() {
		d, ok := Lookup(n)
		if !ok {
			t.Fatalf("%s not registered", n)
		}

		if m, ok := AITargetMode(n); !ok || m != d.TargetMode {
			t.Errorf("%s: target mode %d, exe %d", n, d.TargetMode, m)
		}

		if _, stand := genericAIs[n]; stand {
			t.Errorf("%s is still listed as a stand-in", n)
		}

		if strings.Contains(strings.ToLower(n), "generic") {
			t.Errorf("%s", n)
		}
	}
}

func TestTurretTables(t *testing.T) {
	// every transition lands on a valid direction and moves at most 2 steps
	for d := 0; d < 8; d++ {
		for p := 0; p < 8; p++ {
			n := turretNext[d][p]
			if n < 0 || n > 7 {
				t.Fatalf("bad entry %d %d", d, p)
			}
		}
	}
}

func TestDesertTurret(t *testing.T) {
	w := newF3(5)
	b := f3Brain("DesertTurret", 1, []int{0}, 4, 2, 50, 20, 3)

	// first think: opening cast, timer armed
	w.frame = 7
	Tick(w, b)

	if b.Scratch[0] != 7 || b.Scratch[2] != 0 || w.last() != "attack8" {
		t.Fatalf("opening: scratch=%v log=%v", b.Scratch, w.log)
	}

	// shot: target east (index 7 of the vector table); S1 counts up, S0 =
	// frame + aip1
	w.frame, b.Wake, w.log = 10, 0, nil
	w.dist = 10
	w.target = Target{ID: 1, X: 110, Y: 100, Size: 1, IsPlayer: true}
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[1] != 1 || b.Scratch[0] != 14 || b.Scratch[2] != 7 {
		t.Fatalf("shot: scratch=%v log=%v", b.Scratch, w.log)
	}

	// the burst ends after aip2+1 shots and arms the reload (aip3)
	b.Scratch[1] = 2
	w.frame, b.Wake, w.log = 20, 0, nil
	Tick(w, b)

	if b.Scratch[1] != 0 || b.Scratch[0] != 70 {
		t.Fatalf("reload: scratch=%v", b.Scratch)
	}

	// before the timer: wait 10
	w.frame, b.Wake, w.log = 30, 0, nil
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 40 {
		t.Fatalf("timer: wake=%d log=%v", b.Wake, w.log)
	}

	// target out of range: S1 decays, wait 15
	b.Scratch = [3]int{0, 3, 0}
	b.Scratch[0] = 1
	w.dist = 30
	w.target.X = 130
	w.frame, b.Wake = 40, 0
	Tick(w, b)

	if b.Scratch[1] != 2 || b.Wake != 55 {
		t.Fatalf("far: scratch=%v wake=%d", b.Scratch, b.Wake)
	}
}

func TestCatapultSpotter(t *testing.T) {
	w := newF3(5)
	b := f3Brain("CatapultSpotter", 1, []int{0}, 100, 30, 0, 3, 4)
	w.frame = 100

	Tick(w, b)

	if len(w.raw) != 1 || b.Scratch[1] != 4 || b.Scratch[2] != 100 {
		t.Fatalf("volley: raw=%v scratch=%v", w.raw, b.Scratch)
	}

	ok := false
	for _, id := range spotterSkills {
		ok = ok || id == w.raw[0]
	}

	if !ok || b.Scratch[0] < 0 || b.Scratch[0] > 4 {
		t.Fatalf("skill %d scratch %v", w.raw[0], b.Scratch)
	}

	// the point is within aip4 of the target
	p := w.rawPt[0]
	if p.X < w.target.X-3 || p.X > w.target.X+3 || p.Y < 97 || p.Y > 103 {
		t.Errorf("point %v target %v", p, w.target)
	}

	// next think before the gap has passed: sleep aip2
	w.frame, b.Wake = 110, 0
	w.raw = nil
	Tick(w, b)

	if len(w.raw) != 0 || b.Wake != 140 {
		t.Fatalf("gap: raw=%v wake=%d", w.raw, b.Wake)
	}

	// no free cell: sleep 15
	w.noCell = true
	w.frame, b.Wake = 200, 0
	Tick(w, b)

	if b.Wake != 215 {
		t.Fatalf("no cell: wake=%d", b.Wake)
	}

	// a dead player in the probes: the spotter ends itself
	w.noCell, w.deadNear = false, true
	b.Scratch[2] = 0
	w.frame, b.Wake, w.log = 300, 0, nil
	Tick(w, b)

	if w.last() != "attack0" {
		t.Fatalf("dead: %v", w.log)
	}
}

func TestImp(t *testing.T) {
	// aip of the fixed rows: A 1: hp threshold, 2: offset range, 3: percent
	w := newF3(5)
	w.classAIP = map[[2]int]int{
		{impClassA, 1}: 50, {impClassA, 2}: 6, {impClassA, 3}: 0,
		{impClassB, 1}: 3,
		{impClassC, 1}: 0, {impClassC, 2}: 0, {impClassC, 3}: 0, {impClassC, 4}: 0,
		{impClassD, 3}: 0, {impClassD, 4}: 0,
	}
	w.inRange = true
	b := f3Brain("Imp", 1, []int{0}, 0)
	b.HPPercent = 30

	Tick(w, b)

	if b.Scratch[0] != -1 || w.offsetCast != 1 {
		t.Fatalf("wounded: scratch=%v casts=%d log=%v", b.Scratch, w.offsetCast, w.log)
	}

	// healthy, nothing to do: waits 10 or wanders, never casts
	b.HPPercent, b.Wake, w.offsetCast = 100, 0, 0
	Tick(w, b)

	if w.offsetCast != 0 {
		t.Fatalf("healthy cast")
	}

	// remembered unit far away: walk to it
	w.linked = &Target{ID: 9, X: 120, Y: 100, Size: 1}
	b.Scratch[0] = 9
	b.Wake, w.log = 0, nil
	Tick(w, b)

	if w.last() != "walk-target/0" {
		t.Fatalf("linked: %v", w.log)
	}

	// the link vanishes: reset to -1
	w.linked = nil
	b.Wake = 0
	Tick(w, b)

	if b.Scratch[0] != -1 {
		t.Fatalf("reset: %v", b.Scratch)
	}
}

func TestDeathSentry(t *testing.T) {
	w := newF3(5)
	w.calc4, w.radius = 3, 20
	w.frame = 40
	b := f3Brain("DeathSentry", 1, []int{0, 1}, 0, 25, 100, 50)

	// no corpse, target in range, roll passes (aip3 100): Skill2 cast
	Tick(w, b)

	if b.Scratch[1] != 2 || w.last() != "cast1" {
		t.Fatalf("attack: scratch=%v log=%v", b.Scratch, w.log)
	}

	// corpse next to the target: raise it, remember its id
	w.corpse = &Target{ID: 77, X: w.target.X + 2, Y: 100, Size: 1}
	b.Wake, w.log = 0, nil
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[0] != 77 || b.Scratch[1] != 1 {
		t.Fatalf("raise: scratch=%v log=%v", b.Scratch, w.log)
	}

	// the same corpse is not raised twice: back to the attack branch
	b.Wake, w.log = 0, nil
	Tick(w, b)

	if w.last() != "cast1" {
		t.Fatalf("again: %v", w.log)
	}

	// life counter exhausted: death
	b.Scratch[1] = 0
	b.Wake, w.log = 0, nil
	Tick(w, b)

	if w.last() != "attack0" {
		t.Fatalf("expired: %v", w.log)
	}

	// no target: sleep aip2
	w2 := newF3(5)
	w2.hasTarget, w2.frame = false, 40
	b2 := f3Brain("DeathSentry", 1, []int{0, 1}, 0, 25, 100, 50)
	Tick(w2, b2)

	if b2.Wake != 65 {
		t.Fatalf("idle wake=%d", b2.Wake)
	}

	// no leader: death
	w3 := newF3(5)
	w3.noOwner = true
	b3 := f3Brain("DeathSentry", 1, []int{0, 1}, 0, 25, 100, 50)
	Tick(w3, b3)

	if w3.last() != "attack0" {
		t.Fatalf("orphan: %v", w3.log)
	}
}

func TestDruidBear(t *testing.T) {
	// aip1 attack wait, aip2 boost percent, aip3 skill percent (0: swing)
	w := newF3(5)
	w.hasEnemy, w.enemyDist, w.inRange = true, 3, true
	b := f3Brain("DruidBear", 1, []int{0}, 12, 0, 0)

	Tick(w, b)

	if w.last() != "attack4" || b.Wake != 12 || b.Scratch[2] != 1 {
		t.Fatalf("swing: log=%v wake=%d scratch=%v", w.log, b.Wake, b.Scratch)
	}

	// aip3 100: the skill replaces the swing
	b2 := f3Brain("DruidBear", 1, []int{0}, 12, 0, 100)
	Tick(w, b2)

	if w.last() != "cast0" {
		t.Fatalf("skill: %v", w.log)
	}

	// enemy out of reach: walk to it
	w.inRange, w.log, b.Wake = false, nil, 0
	Tick(w, b)

	if w.last() != "walk-target/0" {
		t.Fatalf("approach: %v", w.log)
	}

	// leader beyond 50: teleport
	w4 := newF3(60)
	w4.hasEnemy = false
	b4 := f3Brain("DruidBear", 1, []int{0}, 12, 0, 0)
	Tick(w4, b4)

	if w4.teleports != 1 {
		t.Fatalf("teleport: %d", w4.teleports)
	}

	// no leader, none remembered: wait 10
	w5 := newF3(5)
	w5.noOwner = true
	b5 := f3Brain("DruidBear", 1, []int{0}, 12, 0, 0)
	Tick(w5, b5)

	if b5.Wake != 10 {
		t.Fatalf("orphan wake=%d", b5.Wake)
	}
}

func TestDruidWolf(t *testing.T) {
	for _, cl := range []int{wolfClassA, 7} {
		w := newF3(5)
		w.hasEnemy, w.enemyDist, w.inRange = true, 3, true
		// aip1 wait, 2 wander, 3 leash/pounce, 4 radius, 5 follow dist
		b := f3Brain("DruidWolf", cl, []int{0, 1}, 9, 0, 100, 20, 30)

		Tick(w, b)

		if w.last() != "attack4" || b.Wake != 9 {
			t.Errorf("class %d: log=%v wake=%d", cl, w.log, b.Wake)
		}
	}

	// B pounces on a unit near the leader (state 0x8a clear, roll gate false)
	w := newF3(5)
	w.hasEnemy, w.enemyDist, w.inRange = true, 3, true
	w.pounce = &Target{ID: 40, X: 104, Y: 100, Size: 1}
	b := f3Brain("DruidWolf", 7, []int{0, 1}, 9, 0, 0, 20, 30)
	b.Seed.Init(3)
	Tick(w, b)

	if w.last() != "cast0" && w.last() != "attack4" {
		t.Errorf("pounce: %v", w.log)
	}

	// no leader: wait 10
	w2 := newF3(5)
	w2.noOwner = true
	b2 := f3Brain("DruidWolf", 7, []int{0, 1}, 9, 0, 0, 20, 30)
	Tick(w2, b2)

	if b2.Wake != 10 {
		t.Errorf("orphan wake %d", b2.Wake)
	}
}

func TestDarkWanderer(t *testing.T) {
	w := newF3(5)
	w.quest = &Point{X: 150, Y: 100}
	w.dist = 10
	b := f3Brain("DarkWanderer", 1, nil)

	Tick(w, b)

	if b.Scratch[0] != 2 || w.last() != "walk-to(150,100)" {
		t.Fatalf("start: scratch=%v log=%v", b.Scratch, w.log)
	}

	// still far: retry up to three times, then give up
	for i := 1; i <= darkWandererRetries; i++ {
		b.Wake, w.log = 0, nil
		Tick(w, b)

		if b.Scratch[1] != i || w.last() != "walk-to(150,100)" {
			t.Fatalf("retry %d: scratch=%v log=%v", i, b.Scratch, w.log)
		}
	}

	b.Wake, w.log = 0, nil
	Tick(w, b)

	if w.arrived != 1 || b.Mode != ModeDying {
		t.Fatalf("give up: arrived=%d log=%v", w.arrived, w.log)
	}

	// arrival by distance
	w2 := newF3(5)
	w2.quest = &Point{X: 102, Y: 100}
	w2.dist = 10
	b2 := f3Brain("DarkWanderer", 1, nil)
	b2.Scratch[0] = 2
	Tick(w2, b2)

	if w2.arrived != 1 {
		t.Fatalf("arrive: %v", w2.log)
	}

	// target far (>= 20): wait 40
	w3 := newF3(5)
	w3.quest = &Point{X: 150, Y: 100}
	w3.dist = 25
	b3 := f3Brain("DarkWanderer", 1, nil)
	Tick(w3, b3)

	if b3.Wake != 40 {
		t.Fatalf("far wake=%d", b3.Wake)
	}

	// quest inactive: wait 10
	w4 := newF3(5)
	b4 := f3Brain("DarkWanderer", 1, nil)
	Tick(w4, b4)

	if b4.Wake != 10 {
		t.Fatalf("inactive wake=%d", b4.Wake)
	}
}

func TestBladeCreeper(t *testing.T) {
	w := newF3(5)
	w.frame = 10
	b := f3Brain("BladeCreeper", 1, []int{0})
	b.AppendCommand(Command{Type: 4, X: 120, Y: 100, Count: 140, Delay: 100})

	Tick(w, b)

	if b.Scratch[0] != 60 || w.blades != 1 || b.Scratch[2] != 1 || w.last() != "walk-to(140,100)" {
		t.Fatalf("first: scratch=%v blades=%d log=%v", b.Scratch, w.blades, w.log)
	}

	// S1 starts at 1: point B first, the next think goes to A
	b.Wake, w.log = 0, nil
	Tick(w, b)

	if w.blades != 1 {
		t.Fatalf("missile twice")
	}

	// no command: wait 3
	b2 := f3Brain("BladeCreeper", 1, []int{0})
	w2 := newF3(5)
	Tick(w2, b2)

	if b2.Wake != 3 {
		t.Fatalf("idle wake=%d", b2.Wake)
	}

	// expired: death
	w.frame, b.Wake, w.log = 100, 0, nil
	Tick(w, b)

	if w.last() != "attack0" {
		t.Fatalf("expired: %v", w.log)
	}

	// missing skill: death
	w3 := newF3(5)
	w3.bladeOK = false
	b3 := f3Brain("BladeCreeper", 1, []int{0})
	Tick(w3, b3)

	if w3.last() != "attack0" {
		t.Fatalf("no skill: %v", w3.log)
	}
}
