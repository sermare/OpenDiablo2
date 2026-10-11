package d2monster

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// fakeWorld is a scripted world: one target, fixed distances, and a log of the
// actions the AI requested.
type fakeWorld struct {
	frame     int
	target    Target
	hasTarget bool
	dist      int
	inRange   bool
	attackOK  bool // AttackTarget finds the target
	dying     bool
	states    map[int]bool
	failMove  bool
	log       []string
	shouts    int
}

func newFake(dist int, inRange bool) *fakeWorld {
	return &fakeWorld{
		target: Target{ID: 1, X: 100 + dist, Y: 100, Size: 1, IsPlayer: true}, hasTarget: true,
		dist: dist, inRange: inRange, attackOK: true, states: map[int]bool{},
	}
}

func (f *fakeWorld) Frame() int { return f.frame }

func (f *fakeWorld) Nearest(*Brain) (Target, int, bool) { return f.target, f.dist, f.hasTarget }

func (f *fakeWorld) AttackTarget(*Brain) (Target, int, bool) {
	return f.target, f.dist, f.hasTarget && f.attackOK
}

func (f *fakeWorld) InRange(*Brain, Target, int) bool { return f.inRange }
func (f *fakeWorld) DyingNear(*Brain, int) bool       { return f.dying }
func (f *fakeWorld) HasState(_ *Brain, s int) bool    { return f.states[s] }
func (f *fakeWorld) SetSpeed(*Brain, int)             {}
func (f *fakeWorld) Shout(*Brain)                     { f.shouts++ }

func (f *fakeWorld) Attack(_ *Brain, m Mode, _ Target) bool {
	f.log = append(f.log, fmt.Sprintf("attack%d", m))

	return true
}

func (f *fakeWorld) Cast(_ *Brain, slot int, _ Target) bool {
	f.log = append(f.log, fmt.Sprintf("cast%d", slot))

	return true
}

func (f *fakeWorld) MoveTo(_ *Brain, p Point, t *Target, reach int, run bool) bool {
	if f.failMove {
		return false
	}

	kind := "walk"
	if run {
		kind = "run"
	}

	if t != nil {
		f.log = append(f.log, fmt.Sprintf("%s-target/%d", kind, reach))
	} else {
		f.log = append(f.log, fmt.Sprintf("%s-to(%d,%d)", kind, p.X, p.Y))
	}

	return true
}

func (f *fakeWorld) last() string {
	if len(f.log) == 0 {
		return ""
	}

	return f.log[len(f.log)-1]
}

const testSeed = 0xC0FFEE

func profile(ai string, aip ...int) *Profile {
	p := &Profile{AI: ai, ID: ai}
	copy(p.AIP[1:], aip)

	return p
}

func brainAt(p *Profile) *Brain {
	b := NewBrain(7, 1, Normal, p, testSeed)
	b.X, b.Y = 100, 100

	return b
}

// shadow returns an independent copy of the brain's RNG to predict rolls.
func shadow(b *Brain) *d2rand.Seed {
	s := *b.Seed

	return &s
}

func TestDistance(t *testing.T) {
	for _, c := range []struct{ dx, dy, want int }{
		{0, 0, 0}, {5, 0, 5}, {0, -5, 5}, {3, 4, 5}, {10, 10, 15}, {-7, 2, 8}, {1, 1, 1},
	} {
		if got := Distance(c.dx, c.dy); got != c.want {
			t.Errorf("Distance(%d,%d)=%d want %d", c.dx, c.dy, got, c.want)
		}
	}

	if got := EdgeDistance(10, 4, 3); got != Distance(7, 1) {
		t.Errorf("EdgeDistance = %d", got)
	}

	if got := EdgeDistance(2, -2, 5); got != 0 {
		t.Errorf("EdgeDistance clamp = %d", got)
	}
}

func TestParseMode(t *testing.T) {
	if m, ok := ParseMode(" s2 "); !ok || m != ModeSkill2 {
		t.Fatal("S2")
	}

	if _, ok := ParseMode("XX"); ok {
		t.Fatal("XX should not parse")
	}
}

func TestAggro(t *testing.T) {
	for _, c := range []struct{ in, want int }{{0, 35}, {20, 20}, {80, 55}} {
		if got := (&Profile{AIDist: c.in}).Aggro(); got != c.want {
			t.Errorf("Aggro(%d)=%d want %d", c.in, got, c.want)
		}
	}
}

func TestRegistryHasArchetypes(t *testing.T) {
	for _, n := range []string{"Skeleton", "skeletonbow", "CorruptArcher", "Fallen", "Idle", "None", "Goatman", "SkeletonMage"} {
		if _, ok := Lookup(n); !ok {
			t.Errorf("missing AI %s", n)
		}
	}

	if _, ok := Lookup("NoSuchAI"); ok {
		t.Error("NoSuchAI cannot be ported")
	}
}

func TestUnimplementedAIIdles(t *testing.T) {
	w := newFake(5, true)
	b := brainAt(profile("NoSuchAI"))

	if !Tick(w, b) || b.Wake != 25 || len(w.log) != 0 {
		t.Fatalf("unimplemented AI should just sleep: wake=%d log=%v", b.Wake, w.log)
	}
}

func TestTickNoTargetSleeps(t *testing.T) {
	for _, c := range []struct {
		name  string
		dist  int
		has   bool
		sleep int
	}{
		{"far", 40, true, 25}, {"nobody", 0, false, 25},
		{"mid", 30, true, 20}, {"edge", 25, true, 15}, {"near", 24, true, 10},
	} {
		w := newFake(c.dist, false)
		w.hasTarget = c.has

		b := brainAt(profile("Skeleton", 100, 15, 100, 50))
		b.Profile.AIDist = 10 // everything above is outside the aggro radius

		if c.name == "near" {
			b.Profile.AIDist = 5
		}

		if !Tick(w, b) {
			// Tick returns true only when the think function ran.
			if b.Wake != w.frame+c.sleep {
				t.Errorf("%s: wake=%d want %d", c.name, b.Wake, c.sleep)
			}
		} else {
			t.Errorf("%s: think should not run", c.name)
		}

		if len(w.log) != 0 {
			t.Errorf("%s: no action expected, got %v", c.name, w.log)
		}
	}
}

func TestTickRespectsWakeAndState(t *testing.T) {
	w := newFake(5, true)
	b := brainAt(profile("Skeleton", 100, 15, 100, 100))
	b.Wake = 10

	if Tick(w, b) {
		t.Fatal("must not run before Wake")
	}

	w.frame = 10
	w.states[StateStunLike] = true

	if Tick(w, b) || b.Wake != 13 {
		t.Fatalf("state 0x15 should sleep 3: wake=%d", b.Wake)
	}

	w.states = map[int]bool{}
	w.frame = 13
	b.Mode = ModeDead

	if Tick(w, b) {
		t.Fatal("dead monster must not think")
	}
}

func TestSkeletonDeterministic(t *testing.T) {
	// skeleton1 normal: approach 60, stall 15, attack 75, A1 75.
	for seed := uint32(0); seed < 40; seed++ {
		for _, inRange := range []bool{false, true} {
			w := newFake(12, inRange)
			b := NewBrain(seed, 1, Normal, profile("Skeleton", 60, 15, 75, 75), testSeed)
			b.X, b.Y = 100, 100
			sh := shadow(b)

			if !Tick(w, b) {
				t.Fatal("think should run")
			}

			var want string

			wantWake := -1

			if !inRange {
				if sh.Roll(100) < 60 {
					want = "walk-target/7"
				} else {
					wantWake = 15
				}
			} else if sh.Roll(100) < 75 {
				if sh.Roll(100) < 75 {
					want = "attack4"
				} else {
					want = "attack5"
				}
			} else {
				wantWake = 15
			}

			if want != "" && w.last() != want || want == "" && (len(w.log) != 0 || b.Wake != wantWake) {
				t.Fatalf("seed %d inRange=%v: log=%v wake=%d want %q/%d", seed, inRange, w.log, b.Wake, want, wantWake)
			}

			if *b.Seed != *sh {
				t.Fatalf("seed %d: RNG consumption differs from the original order", seed)
			}
		}
	}
}

func TestSkeletonForcedBranches(t *testing.T) {
	w := newFake(12, false)
	b := brainAt(profile("Skeleton", 100, 15, 100, 100))
	Tick(w, b)

	if w.last() != "walk-target/7" || b.Wake != waitForever {
		t.Fatalf("approach: %v wake=%d", w.log, b.Wake)
	}

	// the action finished, the engine wakes the unit
	w.inRange = true
	b.WakeNow(w.frame)
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("A1 expected, got %v", w.log)
	}

	b = brainAt(profile("Skeleton", 100, 15, 100, 0))
	Tick(w, b)

	if w.last() != "attack5" {
		t.Fatalf("A2 expected, got %v", w.log)
	}

	b = brainAt(profile("Skeleton", 0, 15, 0, 0))
	w.log = nil
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 15 {
		t.Fatalf("stall expected: %v wake=%d", w.log, b.Wake)
	}
}

func TestMoveFailureFallsBackToStall(t *testing.T) {
	w := newFake(12, false)
	w.failMove = true
	b := brainAt(profile("Skeleton", 100, 15, 0, 0))
	Tick(w, b)

	if b.Wake != 15 {
		t.Fatalf("failed walk should stall aip2: wake=%d", b.Wake)
	}
}

func TestGoatmanAlwaysA1(t *testing.T) {
	w := newFake(3, true)
	b := brainAt(profile("Goatman", 75, 10, 100))
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("goatman should A1: %v", w.log)
	}
}

func TestWraithRange(t *testing.T) {
	w := newFake(30, false)
	b := brainAt(profile("Wraith", 100, 12, 70))
	Tick(w, b)

	// walkToRange(step 12, desired 0): d=30, moves 12 toward the target (+x)
	if w.last() != "walk-to(112,100)" {
		t.Fatalf("wraith move: %v", w.log)
	}
}

func TestZombie(t *testing.T) {
	// out of range, far: wanders unless level 17
	w := newFake(30, false)
	b := brainAt(profile("Zombie", 30, 10, 0, 20))
	Tick(w, b)

	if len(w.log) != 1 || w.log[0][:7] != "walk-to" {
		t.Fatalf("zombie should wander: %v", w.log)
	}

	w = newFake(30, false)
	b = brainAt(profile("Zombie", 30, 10, 0, 20))
	b.LevelID = 17
	Tick(w, b)

	if w.last() != "run-target/0" {
		t.Fatalf("graveyard zombie should run: %v", w.log)
	}

	w = newFake(5, true)
	b = brainAt(profile("Zombie", 30, 10, 0, 100))
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("zombie attack: %v", w.log)
	}
}

func TestIdleAndNone(t *testing.T) {
	w := newFake(5, true)
	b := brainAt(profile("Idle"))
	Tick(w, b)

	if b.Wake != 200 || len(w.log) != 0 {
		t.Fatalf("idle wake=%d", b.Wake)
	}

	b = brainAt(profile("None"))
	Tick(w, b)

	if len(w.log) != 0 {
		t.Fatal("none acts")
	}
}

func TestSkeletonBow(t *testing.T) {
	// sk_archer1: 75/15/50/5/6. Target at 12: shoot with chance 75.
	w := newFake(12, false)
	b := brainAt(profile("SkeletonBow", 100, 15, 50, 5, 6))
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("archer should shoot: %v", w.log)
	}

	// target at 25 (>=20): walk to range with step aip4, desired aip5
	w = newFake(25, false)
	b = brainAt(profile("SkeletonBow", 75, 15, 100, 5, 6))
	b.Profile.AIDist = 40
	Tick(w, b)

	if w.last() != "walk-to(105,100)" {
		t.Fatalf("archer approach: %v", w.log)
	}
}

func TestWalkToRangeKitesAway(t *testing.T) {
	w := newFake(2, true)
	b := brainAt(profile("SkeletonBow", 0, 15, 100, 5, 6))
	c := &Ctx{B: b, W: w}
	c.WalkToRange(Target{X: 102, Y: 100}, 5, 6)

	// Size 1 gives EdgeDistance 1; away by min(|1-6|,5)=5 along -x.
	if w.last() != "walk-to(95,100)" {
		t.Fatalf("kite: %v", w.log)
	}
}

func TestCorruptArcher(t *testing.T) {
	p := profile("CorruptArcher", 100, 100, 14, 100, 20, 0, 0, 12)
	p.Skills[0] = SkillSlot{Name: "Fire Arrow", Mode: ModeAttack1}

	// d=3 <6, aip4=100 -> run away 12 diagonal-less (same y)
	w := newFake(3, true)
	b := brainAt(p)
	Tick(w, b)

	if w.last() != "run-to(88,100)" {
		t.Fatalf("archer kite: %v", w.log)
	}

	// d=10: not close, aip8=12 not exceeded, aip5=20 not exceeded: shoots skill1
	w = newFake(10, true)
	b = brainAt(p)
	Tick(w, b)

	if w.last() != "cast0" {
		t.Fatalf("archer cast: %v", w.log)
	}

	// d=30 (>aip5 20): run toward within aip5 (walk step fires first if rolled)
	w = newFake(30, false)
	b = brainAt(profile("CorruptArcher", 0, 100, 14, 0, 20, 0, 0, 12))
	b.Profile.AIDist = 40
	Tick(w, b)

	if w.last() != "run-target/20" {
		t.Fatalf("archer approach: %v", w.log)
	}

	// no attack target: circle or stall aip3
	w = newFake(10, true)
	w.attackOK = false
	b = brainAt(profile("CorruptArcher", 0, 100, 14, 0, 20, 0, 0, 12))
	Tick(w, b)

	if len(w.log) > 1 {
		t.Fatalf("no target branch: %v", w.log)
	}
}

func TestSkeletonMage(t *testing.T) {
	// skmage_pois1 N: 35/9/30/5/0/18/20/5
	w := newFake(15, false)
	b := brainAt(profile("SkeletonMage", 35, 9, 100, 5, 0, 18, 20, 5))
	Tick(w, b)

	if w.last() != "walk-target/9" {
		t.Fatalf("mage approaches to aip2: %v", w.log)
	}

	w = newFake(10, true)
	b = brainAt(profile("SkeletonMage", 100, 9, 0, 5, 0, 18, 20, 5))
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("mage fires: %v", w.log)
	}

	// aip5 0 never walks away; idle: stall aip8 if roll >= aip7
	w = newFake(10, true)
	b = brainAt(profile("SkeletonMage", 0, 9, 0, 5, 0, 18, 0, 5))
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 5 {
		t.Fatalf("mage stalls: %v wake=%d", w.log, b.Wake)
	}
}

func TestFallenPackRally(t *testing.T) {
	// fallen1: 30/10/50/20
	pf := func() *Profile { return profile("Fallen", 100, 10, 100, 20) }

	leader := brainAt(pf())
	m1, m2 := brainAt(pf()), brainAt(pf())
	m1.ID, m2.ID = 8, 9
	leader.AddMinion(m1)
	leader.AddMinion(m2)

	if !leader.IsGroupLeader() || m1.IsGroupLeader() {
		t.Fatal("leader detection")
	}

	// Leader sees the player 14 away, out of range: rally with S2 shout.
	w := newFake(14, false)
	Tick(w, leader)

	if w.last() != "attack9" {
		t.Fatalf("leader should shout: %v", w.log)
	}

	if m1.QueueLen() != 1 || m2.QueueLen() != 1 || leader.QueueLen() != 1 {
		t.Fatalf("alert not broadcast: %d %d %d", m1.QueueLen(), m2.QueueLen(), leader.QueueLen())
	}

	if m1.PeekCommand().Type != CmdAlert {
		t.Fatal("wrong command type")
	}

	// A minion with the alert out of range charges the target tile-exact.
	w = newFake(14, false)
	Tick(w, m1)

	if w.last() != "walk-target/0" {
		t.Fatalf("minion should charge: %v", w.log)
	}

	// In range it attacks (aip3 100); the command stays.
	w = newFake(2, true)
	m1.WakeNow(0)
	Tick(w, m1)

	if w.last() != "attack4" && w.last() != "attack5" {
		t.Fatalf("minion should attack: %v", w.log)
	}

	if m1.QueueLen() != 1 {
		t.Fatal("the exe never pops the alert on an attack (batch 7)")
	}

	// A minion with no command and a lone chase: within aip2 it walks to 7.
	solo := brainAt(profile("Fallen", 0, 20, 0, 20))
	w = newFake(15, false)
	Tick(w, solo)

	if w.last() != "walk-target/7" {
		t.Fatalf("fallen chase: %v", w.log)
	}
}

func TestFallenFleesDeath(t *testing.T) {
	w := newFake(6, true)
	w.dying = true
	b := brainAt(profile("Fallen", 30, 10, 50, 20))
	b.PushCommand(Command{Type: CmdAlert, Count: 1})
	Tick(w, b)

	// target is at +x, so the Fallen runs -x (diagonal flee is one axis here)
	if w.last() != "walk-to(88,99)" && w.last() != "walk-to(88,100)" {
		t.Fatalf("fallen flee: %v", w.log)
	}

	if b.QueueLen() != 0 || b.Scratch[0] != 1 {
		t.Fatalf("flee should clear commands and set scratch: q=%d s=%d", b.QueueLen(), b.Scratch[0])
	}
}

func TestFallenSleepsWhenBusyMode(t *testing.T) {
	w := newFake(6, true)
	b := brainAt(profile("Fallen", 30, 10, 50, 20))
	b.Mode = ModeGetHit
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 10 {
		t.Fatalf("fallen in hit recovery: %v wake=%d", w.log, b.Wake)
	}
}

func TestWanderDeterministic(t *testing.T) {
	a, b := brainAt(profile("Idle")), brainAt(profile("Idle"))
	wa, wb := newFake(5, true), newFake(5, true)

	(&Ctx{B: a, W: wa}).Wander(5)
	(&Ctx{B: b, W: wb}).Wander(5)

	if wa.last() != wb.last() || wa.last() == "" {
		t.Fatalf("wander not deterministic: %v %v", wa.log, wb.log)
	}

	// exactly four RNG steps (VERIFIED 0x5dcff0): parity, roll(n), two signs
	sh := d2rand.New(testSeed + 7)
	sh.Step()
	sh.Roll(5)
	sh.Step()
	sh.Step()

	if *a.Seed != *sh {
		t.Fatal("wander should consume exactly four steps")
	}
}

func TestMinionBookkeeping(t *testing.T) {
	l, m := brainAt(profile("Fallen")), brainAt(profile("Fallen"))
	l.AddMinion(m)

	if m.Leader != l || len(l.Minions) != 1 {
		t.Fatal("add")
	}

	l.RemoveMinion(m)

	if m.Leader != nil || len(l.Minions) != 0 || l.IsGroupLeader() {
		t.Fatal("remove")
	}

	m.PushCommand(Command{Type: 1})
	m.SetAI(unimplementedDef("x"))

	if m.QueueLen() != 0 {
		t.Fatal("SetAI should clear the queue")
	}
}
