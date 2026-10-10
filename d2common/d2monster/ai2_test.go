package d2monster

import (
	"strings"
	"testing"
)

// runOne ticks a brain once on a fresh fake world and returns the actions.
func runOne(t *testing.T, w *fakeWorld, b *Brain) []string {
	t.Helper()

	if !Tick(w, b) {
		t.Fatalf("think did not run (wake=%d)", b.Wake)
	}

	return w.log
}

func TestNewArchetypesRegistered(t *testing.T) {
	for _, n := range []string{"Brute", "Mummy", "Scarab", "Bighead", "CorruptRogue", "PantherJavelin",
		"Andariel", "Smith", "Griswold", "BloodRaven"} {
		if d, ok := Lookup(n); !ok || !d.Implemented {
			t.Errorf("AI %s is not registered", n)
		}
	}

	// the former Tentacle/FrogDemon TODOs are ported in ai_fb_3.go
	for _, n := range []string{"Tentacle", "FrogDemon"} {
		if _, ok := Lookup(n); !ok {
			t.Errorf("%s should be ported", n)
		}
	}
}

// TestBruteAip3Twice pins the observed quirk: in range, with aip3=100 both
// rolls pass so only A1 happens; the A2 split cannot be driven by aip4.
func TestBruteAip3Twice(t *testing.T) {
	w := newFake(3, true)
	b := brainAt(profile("Brute", 0, 15, 100, 0)) // aip4=0 would mean "always A2" if it were used

	if got := runOne(t, w, b); len(got) != 1 || got[0] != "attack4" {
		t.Fatalf("aip3=100: want A1, got %v", got)
	}

	// aip3=0: the first test fails, the second (circle) fails, sleep 15
	w = newFake(3, true)
	b = brainAt(profile("Brute", 0, 15, 0, 100))
	runOne(t, w, b)

	if len(w.log) != 0 || b.Wake != 15 {
		t.Fatalf("aip3=0: want sleep 15 and no action, got log=%v wake=%d", w.log, b.Wake)
	}
}

func TestBruteSplitUsesSameColumnTwice(t *testing.T) {
	// With aip3 = 50 the monster attacks A1 or A2 only after two passing rolls;
	// replay the RNG to prove both rolls compare against aip3.
	for seed := uint32(1); seed < 60; seed++ {
		w := newFake(3, true)
		b := NewBrain(7, 1, Normal, profile("Brute", 0, 15, 50, 99), seed)
		b.X, b.Y = 100, 100
		s := shadow(b)

		r1 := int(s.Roll(100)) < 50
		r2 := int(s.Roll(100)) < 50

		Tick(w, b)

		want := ""

		switch {
		case r1 && r2:
			want = "attack4"
		case r1:
			want = "attack5"
		}

		got := ""
		if len(w.log) > 0 {
			got = w.log[0]
		}

		if strings.HasPrefix(got, "walk") { // circle is a walk-to-point
			got = ""
		}

		if got != want {
			t.Fatalf("seed %d: r1=%v r2=%v want %q got %v", seed, r1, r2, want, w.log)
		}
	}
}

func TestBruteChaseOutOfRange(t *testing.T) {
	w := newFake(30, false)
	b := brainAt(profile("Brute", 0, 15, 50, 50))
	b.HPPercent = 20

	if got := runOne(t, w, b); len(got) != 1 || got[0] != "walk-target/7" {
		t.Fatalf("got %v", got)
	}
}

func TestMummy(t *testing.T) {
	// awake distance 5, attack 100%, A1 100%
	w := newFake(3, true)
	b := brainAt(profile("Mummy", 5, 60, 100, 100, 10))

	if got := runOne(t, w, b); len(got) != 1 || got[0] != "attack4" {
		t.Fatalf("in range: %v", got)
	}

	// asleep far away: wander% 100 -> a wander step
	w = newFake(30, false)
	b = brainAt(profile("Mummy", 5, 100, 100, 100, 10))
	runOne(t, w, b)

	if len(w.log) != 1 || !strings.HasPrefix(w.log[0], "walk-to") {
		t.Fatalf("wander: %v", w.log)
	}

	// asleep, no wander: stall aip5
	w = newFake(30, false)
	b = brainAt(profile("Mummy", 5, 0, 100, 100, 10))
	runOne(t, w, b)

	if len(w.log) != 0 || b.Wake != 10 {
		t.Fatalf("stall: %v wake=%d", w.log, b.Wake)
	}

	// awake and out of range: lunge
	w = newFake(4, false)
	b = brainAt(profile("Mummy", 5, 0, 100, 100, 10))

	if got := runOne(t, w, b); len(got) != 1 || got[0] != "walk-target/7" {
		t.Fatalf("lunge: %v", got)
	}

	// failed attack roll in range: sleep, then the walk request (kept as observed);
	// the fake accepts it so the unit is busy, a real world refuses it within reach
	w = newFake(3, true)
	b = brainAt(profile("Mummy", 5, 0, 0, 100, 10))
	w.failMove = true
	runOne(t, w, b)

	if b.Wake != 10 || len(w.log) != 0 {
		t.Fatalf("stall in range: %v wake=%d", w.log, b.Wake)
	}
}

func TestScarabGroupAlert(t *testing.T) {
	// leader within 20 alerts its followers with aip5=100
	w := newFake(10, false)
	p := profile("Scarab", 75, 50, 15, 35, 100)
	leader := brainAt(p)
	f := NewBrain(8, 1, Normal, p, testSeed)
	f.X, f.Y = 101, 100
	leader.AddMinion(f)

	runOne(t, w, leader)

	if cmd := f.PeekCommand(); cmd == nil || cmd.Type != CmdAlert {
		t.Fatal("follower should hold the alert")
	}

	// the follower in range with a command jabs / attacks without the aip1 gate
	w = newFake(3, true)
	p2 := profile("Scarab", 0, 100, 15, 0, 0)
	f.Profile = p2
	runOne(t, w, f)

	if len(w.log) != 1 || w.log[0] != "attack4" {
		t.Fatalf("follower attack: %v", w.log)
	}
}

func TestScarabJabAndStall(t *testing.T) {
	w := newFake(3, true)
	p := profile("Scarab", 100, 50, 15, 100, 0)
	p.Skills[0] = SkillSlot{Name: "Jab", Mode: ModeAttack1}
	b := brainAt(p)

	if got := runOne(t, w, b); len(got) != 1 || got[0] != "cast0" {
		t.Fatalf("jab: %v", got)
	}

	w = newFake(3, true)
	b = brainAt(profile("Scarab", 0, 50, 15, 100, 0))
	runOne(t, w, b)

	if len(w.log) != 0 || b.Wake != 15 {
		t.Fatalf("stall: %v wake %d", w.log, b.Wake)
	}
}

func TestScarabTogglesCircleAndWalk(t *testing.T) {
	w := newFake(12, false)
	b := brainAt(profile("Scarab", 75, 50, 15, 35, 0))

	runOne(t, w, b)

	if !strings.HasPrefix(w.last(), "walk-to") || b.Scratch[0] != 1 {
		t.Fatalf("first out-of-range tick should circle: %v", w.log)
	}

	b.Wake = 0
	runOne(t, w, b)

	if w.last() != "walk-target/7" || b.Scratch[0] != 0 {
		t.Fatalf("second tick should walk to the target: %v", w.log)
	}
}

func TestBighead(t *testing.T) {
	// healthy and in range: melee
	w := newFake(3, true)
	b := brainAt(profile("Bighead", 88, 40, 0, 60))

	if got := runOne(t, w, b); got[0] != "attack4" {
		t.Fatalf("healthy: %v", got)
	}

	// healthy, near but out of range, fire%=100: A2 cast
	w = newFake(10, false)
	b = brainAt(profile("Bighead", 88, 40, 100, 60))

	if got := runOne(t, w, b); got[0] != "attack5" {
		t.Fatalf("healthy fire: %v", got)
	}

	// healthy, fire%=0: walk
	w = newFake(10, false)
	b = brainAt(profile("Bighead", 88, 40, 0, 60))

	if got := runOne(t, w, b); got[0] != "walk-target/7" {
		t.Fatalf("healthy approach: %v", got)
	}

	// hurt and too close: walk away
	w = newFake(2, true)
	b = brainAt(profile("Bighead", 88, 40, 0, 60))
	b.HPPercent = 10

	if got := runOne(t, w, b); !strings.HasPrefix(got[0], "walk-to(") {
		t.Fatalf("hurt close: %v", got)
	}

	// hurt and far: walk back toward the target
	w = newFake(25, false)
	b = brainAt(profile("Bighead", 88, 40, 0, 60))
	b.HPPercent = 10

	if got := runOne(t, w, b); got[0] != "walk-target/6" {
		t.Fatalf("hurt far: %v", got)
	}

	// hurt mid range: fire at 100%
	w = newFake(10, false)
	b = brainAt(profile("Bighead", 88, 40, 0, 100))
	b.HPPercent = 10

	if got := runOne(t, w, b); got[0] != "attack5" {
		t.Fatalf("hurt fire: %v", got)
	}

	// hurt mid range, never fires, never circles: sleep 10
	w = newFake(10, false)
	b = brainAt(profile("Bighead", 88, 0, 0, 0))
	b.HPPercent = 10
	runOne(t, w, b)

	if len(w.log) != 0 || b.Wake != 10 {
		t.Fatalf("hurt idle: %v wake=%d", w.log, b.Wake)
	}
}

func TestCorruptRogue(t *testing.T) {
	// far: run at the target
	w := newFake(30, false)
	b := brainAt(profile("CorruptRogue", 60, 15, 75, 100, 20))

	if got := runOne(t, w, b); got[0] != "run-target/3" {
		t.Fatalf("far: %v", got)
	}

	// in range, attack 100%
	w = newFake(3, true)
	b = brainAt(profile("CorruptRogue", 60, 15, 100, 100, 20))

	if got := runOne(t, w, b); got[0] != "attack4" {
		t.Fatalf("attack: %v", got)
	}

	// in range, attack 0%: stall aip2
	w = newFake(3, true)
	b = brainAt(profile("CorruptRogue", 60, 15, 0, 100, 20))
	runOne(t, w, b)

	if b.Wake != 15 || len(w.log) != 0 {
		t.Fatalf("stall: wake=%d", b.Wake)
	}

	// out of range: approach% 100, run% 0 -> walk; run% 100 -> run
	w = newFake(10, false)
	b = brainAt(profile("CorruptRogue", 100, 15, 75, 100, 0))

	if got := runOne(t, w, b); got[0] != "walk-target/7" {
		t.Fatalf("walk: %v", got)
	}

	w = newFake(10, false)
	b = brainAt(profile("CorruptRogue", 100, 15, 75, 100, 100))

	if got := runOne(t, w, b); got[0] != "run-target/3" {
		t.Fatalf("run: %v", got)
	}
}

type allyWorld struct {
	*fakeWorld
	ally Target
	d    int
}

func (a allyWorld) NearestAlly(*Brain) (Target, int, bool) { return a.ally, a.d, true }

func TestPantherJavelin(t *testing.T) {
	p := profile("PantherJavelin", 70, 100, 12, 0, 15, 20)

	// in throwing range: throw
	w := newFake(10, false)
	b := brainAt(p)

	if got := runOne(t, w, b); got[0] != "attack4" {
		t.Fatalf("throw: %v", got)
	}

	// too close with walk-away 100%: back off 16
	w = newFake(5, true)
	b = brainAt(profile("PantherJavelin", 70, 100, 12, 100, 15, 20))

	if got := runOne(t, w, b); !strings.HasPrefix(got[0], "walk-to(") {
		t.Fatalf("back off: %v", got)
	}

	// too far with approach 100%: move near the target
	w = newFake(30, false)
	b = brainAt(profile("PantherJavelin", 100, 100, 12, 0, 15, 20))

	if got := runOne(t, w, b); !strings.HasPrefix(got[0], "walk-to(") {
		t.Fatalf("approach: %v", got)
	}

	// throw% 0: stall aip5
	w = newFake(10, false)
	b = brainAt(profile("PantherJavelin", 70, 0, 12, 0, 15, 20))
	runOne(t, w, b)

	if b.Wake != 15 {
		t.Fatalf("stall wake=%d", b.Wake)
	}

	// nothing to shoot at but an ally far away: regroup with it
	f := newFake(30, false)
	f.attackOK = false
	b = brainAt(profile("PantherJavelin", 0, 100, 12, 0, 15, 20))
	aw := allyWorld{f, Target{ID: 99, X: 130, Y: 100}, 30}

	if !Tick(aw, b) || f.last() != "walk-target/7" {
		t.Fatalf("regroup: %v", f.log)
	}
}

func TestAndariel(t *testing.T) {
	p := profile("Andariel", 30, 10, 30, 50)
	p.Skills[0] = SkillSlot{Name: "AndrialSpray", Mode: ModeSkill1}
	p.Skills[1] = SkillSlot{Name: "AndyPoisonBolt", Mode: ModeSkill2}

	// melee: spray at aip1=100
	q := *p
	q.AIP[1] = 100
	w := newFake(3, true)

	if got := runOne(t, w, brainAt(&q)); got[0] != "cast0" {
		t.Fatalf("spray: %v", got)
	}

	// melee: otherwise A1
	q.AIP[1] = 0
	w = newFake(3, true)

	if got := runOne(t, w, brainAt(&q)); got[0] != "attack4" {
		t.Fatalf("melee: %v", got)
	}

	// ranged: stall
	q = *p
	q.AIP[2] = 100
	w = newFake(20, false)
	b := brainAt(&q)
	runOne(t, w, b)

	if b.Wake != 5 || len(w.log) != 0 {
		t.Fatalf("stall: wake=%d %v", b.Wake, w.log)
	}

	// ranged: fire-or-engage, spray 100% then poison bolt
	q = *p
	q.AIP = [9]int{0, 30, 0, 100, 100}
	w = newFake(20, false)

	if got := runOne(t, w, brainAt(&q)); got[0] != "cast0" {
		t.Fatalf("spray from range: %v", got)
	}

	q.AIP[4] = 0
	w = newFake(20, false)

	if got := runOne(t, w, brainAt(&q)); got[0] != "cast1" {
		t.Fatalf("poison bolt: %v", got)
	}

	// ranged: nothing rolled, engage on foot
	q.AIP = [9]int{0, 30, 0, 0, 0}
	w = newFake(20, false)

	if got := runOne(t, w, brainAt(&q)); got[0] != "walk-target/7" {
		t.Fatalf("engage: %v", got)
	}
}

func TestSmithAndGriswold(t *testing.T) {
	w := newFake(3, true)
	if got := runOne(t, w, brainAt(profile("Smith"))); got[0] != "attack4" {
		t.Fatalf("smith melee: %v", got)
	}

	w = newFake(20, false)
	if got := runOne(t, w, brainAt(profile("Smith"))); got[0] != "walk-target/7" {
		t.Fatalf("smith walk: %v", got)
	}

	// Griswold: in range 80% A1; replay the RNG
	for seed := uint32(1); seed < 40; seed++ {
		w = newFake(3, true)
		b := NewBrain(7, 1, Normal, profile("Griswold"), seed)
		b.X, b.Y = 100, 100
		hit := int(shadow(b).Roll(100)) < 80
		Tick(w, b)

		if hit != (len(w.log) == 1 && w.log[0] == "attack4") {
			t.Fatalf("seed %d: roll says %v, log %v", seed, hit, w.log)
		}

		if !hit && b.Wake != 10 {
			t.Fatalf("seed %d: should sleep 10", seed)
		}
	}
}

func TestBloodRavenLeashAndShots(t *testing.T) {
	p := profile("BloodRaven")
	p.AIDist = 55
	p.Skills[0] = SkillSlot{Name: "BRArrow", Mode: ModeAttack2}
	p.Skills[1] = SkillSlot{Name: "BRQuick", Mode: ModeAttack1}

	// first tick: anchor is recorded at the spawn point
	w := newFake(30, false)
	w.target = Target{ID: 1, X: 130, Y: 100, Size: 1, IsPlayer: true}
	b := brainAt(p)
	runOne(t, w, b)

	a := b.FindCommand(CmdAnchor)
	if a == nil || a.X != 100 || a.Y != 100 {
		t.Fatalf("anchor: %+v", a)
	}

	// target ignored beyond 45
	w = newFake(50, false)
	b = brainAt(p)
	runOne(t, w, b)

	if b.Wake != 5 || len(w.log) != 0 {
		t.Fatalf("far target: wake=%d %v", b.Wake, w.log)
	}

	// dragged away from home: runs back to the anchor and keeps going until close
	w = newFake(30, false)
	b = brainAt(p)
	b.AppendCommand(Command{Type: CmdAnchor, X: 20, Y: 100})
	runOne(t, w, b)

	if w.last() != "run-to(20,100)" || b.Scratch[brFleeHome] != 1 {
		t.Fatalf("leash: %v scratch=%v", w.log, b.Scratch)
	}

	// back near home but still "fleeing": keep running until within 5
	b.X = 28
	b.Wake = 0
	runOne(t, w, b)

	if w.last() != "run-to(20,100)" {
		t.Fatalf("keep returning: %v", w.log)
	}

	// the exe's leash test uses the TARGET's distance to the anchor (batch 5):
	// a target still 50 or more from it keeps her running home
	b.X = 22
	b.Wake = 0
	runOne(t, w, b)

	if b.Scratch[brFleeHome] != 1 {
		t.Fatal("a target far from the anchor keeps the flag set")
	}

	w.target = Target{ID: 1, X: 60, Y: 100, Size: 1, IsPlayer: true}
	w.dist = 38
	b.Wake = 0
	runOne(t, w, b)

	if b.Scratch[brFleeHome] != 0 {
		t.Fatal("fleeing flag should clear within 5 of the anchor")
	}
}

func TestBloodRavenFiresAtRandomSpot(t *testing.T) {
	p := profile("BloodRaven")
	p.Skills[0] = SkillSlot{Name: "BRArrow", Mode: ModeAttack2}

	w := newFake(15, false)
	b := brainAt(p)
	b.Scratch[brCharge] = 100 // guaranteed shot
	rec := &recordingWorld{fakeWorld: w}

	if !Tick(rec, b) {
		t.Fatal("no think")
	}

	if len(rec.casts) != 1 {
		t.Fatalf("expected one cast, got %v", w.log)
	}

	c := rec.casts[0]
	if c.ID != 0 {
		t.Error("shot must target a ground point (ID 0)")
	}

	dx, dy := c.X-w.target.X, c.Y-w.target.Y
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	// batch 5 (0x5e5260 disassembly): r = roll(15)+5 on one axis, a modulo r
	// roll on the other, signs from two coin flips
	hi, lo := dx, dy
	if lo > hi {
		hi, lo = lo, hi
	}

	if hi < 5 || hi > 19 || lo >= hi {
		t.Errorf("spot (%d,%d) is %d,%d from the target", c.X, c.Y, dx, dy)
	}

	if b.Scratch[brShots] != 1 || b.Scratch[brCharge] != 0 {
		t.Errorf("counters: %v", b.Scratch)
	}
}

// recordingWorld remembers the targets of casts.
type recordingWorld struct {
	*fakeWorld
	casts []Target
}

func (r *recordingWorld) Cast(b *Brain, slot int, t Target) bool {
	r.casts = append(r.casts, t)

	return r.fakeWorld.Cast(b, slot, t)
}

func TestBloodRavenShotCap(t *testing.T) {
	p := profile("BloodRaven")
	p.Skills[0] = SkillSlot{Name: "BRArrow", Mode: ModeAttack2}

	for diff, cap := range map[Difficulty]int{Normal: 8, Nightmare: 10, Hell: 12} {
		w := newFake(15, false)
		b := NewBrain(7, 1, diff, p, testSeed)
		b.X, b.Y = 100, 100
		b.Scratch[brCharge] = 100
		b.Scratch[brShots] = cap

		rec := &recordingWorld{fakeWorld: w}
		Tick(rec, b)

		if len(rec.casts) != 0 {
			t.Errorf("diff %d: cap %d reached, no more arrows expected", diff, cap)
		}

		b.Scratch[brShots] = cap - 1
		b.Wake = 0
		b.Scratch[brCharge] = 100
		Tick(rec, b)

		if len(rec.casts) != 1 {
			t.Errorf("diff %d: one below the cap should still fire", diff)
		}
	}
}
