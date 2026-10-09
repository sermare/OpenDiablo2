package d2monster

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// fake2 adds the optional World extensions to the scripted fakeWorld.
type fake2 struct {
	*fakeWorld
	src       Target
	srcOK     bool
	any       Target
	anyDist   int
	anyOK     bool
	decoy     Target
	decoyDist int
	decoyOK   bool
	town      bool
	noMode    map[Mode]bool
	fire      int
	cold      int
	light     int
	auras     int
	prey      bool
	sight     *bool
	flights   []string
}

func newFake2(dist int, inRange bool) *fake2 {
	return &fake2{fakeWorld: newFake(dist, inRange), noMode: map[Mode]bool{}}
}

func (f *fake2) UnitTarget(*Brain, uint32) (Target, int, bool) { return f.src, 5, f.srcOK }
func (f *fake2) NearestAny(*Brain) (Target, int, bool)         { return f.any, f.anyDist, f.anyOK }
func (f *fake2) NearestDecoy(*Brain) (Target, int, bool)       { return f.decoy, f.decoyDist, f.decoyOK }
func (f *fake2) InTown(*Brain) bool                            { return f.town }
func (f *fake2) HasMode(_ *Brain, m Mode) bool                 { return !f.noMode[m] }
func (f *fake2) Resists(Target) (int, int, int)                { return f.fire, f.cold, f.light }
func (f *fake2) StartAura(*Brain, int, int)                    { f.auras++ }
func (f *fake2) NearestPrey(*Brain, int, int) (Target, bool)   { return f.target, f.prey }
func (f *fake2) TakeOff(*Brain)                                { f.flights = append(f.flights, "takeoff") }
func (f *fake2) Land(*Brain)                                   { f.flights = append(f.flights, "land") }

func (f *fake2) ClearState(_ *Brain, s int) { delete(f.states, s) }

func (f *fake2) LineOfSight(*Brain, Target) bool {
	if f.sight == nil {
		return true
	}

	return *f.sight
}

// findBrain returns a brain whose own generator satisfies pred (the sequence
// of rolls it will make is predicted from a copy of the seed).
func findBrain(t *testing.T, p *Profile, pred func(s *d2rand.Seed) bool) *Brain {
	t.Helper()

	for id := uint32(1); id < 5000; id++ {
		b := NewBrain(id, 1, Normal, p, testSeed)
		b.X, b.Y = 100, 100

		if pred(shadow(b)) {
			return b
		}
	}

	t.Fatal("no brain with the wanted roll sequence")

	return nil
}

func TestParseForced(t *testing.T) {
	for _, c := range []struct {
		in   string
		want ForcedKind
		ok   bool
	}{
		{"fear", ForcedFear, true}, {"Terror", ForcedFear, true}, {"confuse", ForcedConfuse, true},
		{"attract", ForcedAttract, true}, {"charm", ForcedCharm, true}, {"convert", ForcedCharm, true},
		{"blind", ForcedBlind, true}, {"taunt", ForcedTaunt, true}, {"nope", ForcedNone, false},
	} {
		k, ok := ParseForced(c.in)
		if k != c.want || ok != c.ok {
			t.Errorf("ParseForced(%q) = %v,%v", c.in, k, ok)
		}
	}

	// the curse -> AI state mapping read from FUN_005c1270
	for k, want := range map[ForcedKind]int{ForcedFear: 11, ForcedBlind: 10, ForcedTaunt: 12, ForcedConfuse: 0, ForcedCharm: 0} {
		if k.AIState() != want {
			t.Errorf("%v.AIState() = %d want %d", k, k.AIState(), want)
		}
	}

	if ForcedFear.UnitStateID() != 56 || ForcedBlind.UnitStateID() != 23 || ForcedTaunt.UnitStateID() != 27 ||
		ForcedConfuse.UnitStateID() != 59 || ForcedAttract.UnitStateID() != 57 || ForcedCharm.UnitStateID() != 53 {
		t.Error("unit state ids (states.txt) wrong")
	}
}

func TestStateTable(t *testing.T) {
	for _, id := range []int{2, 3, 6, 8, 9, 10, 11, 12, 14, 17} {
		if d := StateDef(id); d == nil || !d.Implemented {
			t.Errorf("state %d is not ported", id)
		}
	}

	for _, id := range []int{0, 1, 4, 5, 7, 13, 15, 16, 18, -1} {
		if StateDef(id) != nil {
			t.Errorf("state %d should not be in the table", id)
		}
	}

	// target modes of the table (monster-ai.md): 10 and 11 standard, 17 acquire-only
	if StateDef(11).TargetMode != TargetStandard || StateDef(17).TargetMode != TargetOnly || StateDef(6).TargetMode != TargetNone {
		t.Error("target modes differ from the alternate table")
	}
}

// TestFearOverridesAndRestores walks the whole life of a fear: the think
// function is replaced, the monster flees (running, because the class can),
// strikes back when cornered, and gets its own AI, with clean scratch fields
// and an empty command queue, back when the duration is over.
func TestFearOverridesAndRestores(t *testing.T) {
	w := newFake2(5, false)
	w.states[StateTerror] = true

	p := profile("Skeleton", 60, 15, 75, 75)
	p.Run, p.Walk = 12, 8
	b := brainAt(p)
	b.Scratch[0] = 9
	b.PushCommand(Command{Type: CmdAlert})

	b.ApplyForced(ForcedFear, 0, 50, 1)

	if b.Def.Name != "State11" || b.Forced != ForcedFear || b.Scratch[0] != 0 || b.QueueLen() != 0 {
		t.Fatalf("not overridden/cleaned: def=%s forced=%v scratch=%v queue=%d", b.Def.Name, b.Forced, b.Scratch, b.QueueLen())
	}

	// first tick: run away from the source (it is east of the monster)
	if !Tick(w, b) || !strings.HasPrefix(w.last(), "run-to(") {
		t.Fatalf("first flee tick: %v", w.log)
	}

	var x, y int
	if _, err := fmtSscan(w.last(), &x, &y); err != nil || x >= 100 {
		t.Fatalf("flee destination (%d,%d) not away from the source: %v", x, y, w.log)
	}

	if b.Scratch[2] != 1 {
		t.Error("the flee should be marked started")
	}

	// later ticks keep fleeing
	w.frame, b.Wake = 5, 5
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "run-to(") {
		t.Fatalf("second flee tick: %v", w.log)
	}

	// cornered: the source is in reach -> strike back
	w.inRange = true
	w.frame, b.Wake = 6, 6
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("cornered monster should strike: %v", w.log)
	}

	// duration over: the class AI is back
	w.frame = 50
	Tick(w, b)

	if b.Def.Name != "Skeleton" || b.Forced != ForcedNone || b.Scratch != [3]int{} {
		t.Fatalf("not restored: def=%s forced=%v scratch=%v", b.Def.Name, b.Forced, b.Scratch)
	}
}

func fmtSscan(s string, x, y *int) (int, error) {
	s = strings.TrimSuffix(strings.TrimPrefix(s, "run-to("), ")")
	parts := strings.Split(s, ",")

	var err error

	*x, err = atoi(parts[0])
	if err != nil {
		return 0, err
	}

	*y, err = atoi(parts[1])

	return 2, err
}

func atoi(s string) (int, error) {
	n, neg := 0, false

	for i, ch := range s {
		if i == 0 && ch == '-' {
			neg = true

			continue
		}

		n = n*10 + int(ch-'0')
	}

	if neg {
		n = -n
	}

	return n, nil
}

func TestFearEndsWhenTerrorStateIsGone(t *testing.T) {
	w := newFake2(5, false) // HasState(56) is false: the unit state was removed early
	b := brainAt(profile("Skeleton", 60, 15, 75, 75))
	b.ApplyForced(ForcedFear, 0, 1000, 1)

	if !Tick(w, b) {
		t.Fatal("no tick")
	}

	if b.Def.Name != "Skeleton" || b.Forced != ForcedNone || b.Wake != 1 {
		t.Fatalf("def=%s forced=%v wake=%d", b.Def.Name, b.Forced, b.Wake)
	}
}

func TestFearWalksWhenClassCannotRun(t *testing.T) {
	w := newFake2(5, false)
	w.states[StateTerror] = true
	w.noMode[ModeRun] = true

	b := brainAt(profile("Skeleton", 60, 15, 75, 75))
	b.ApplyForced(ForcedFear, 0, 100, 1)
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("expected a walking flight: %v", w.log)
	}
}

func TestFearFarSourceIsIgnored(t *testing.T) {
	w := newFake2(40, false) // farther than the 30 trigger distance
	w.states[StateTerror] = true
	b := brainAt(profile("Skeleton"))
	b.Profile.AIDist = 50
	b.ApplyForced(ForcedFear, 0, 100, 1)

	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 10 {
		t.Fatalf("a far source should be ignored: wake=%d log=%v", b.Wake, w.log)
	}
}

func TestFearBoost(t *testing.T) {
	for _, c := range []struct{ walk, run, want int }{{8, 12, 50}, {8, 8, 0}, {0, 12, 0}, {4, 40, 0x78}, {10, 9, 0}} {
		if got := fleeBoost(&Profile{Walk: c.walk, Run: c.run}); got != c.want {
			t.Errorf("fleeBoost(%d,%d) = %d want %d", c.walk, c.run, got, c.want)
		}
	}
}

func TestBlindAndTaunt(t *testing.T) {
	w := newFake2(3, true)
	b := brainAt(profile("Skeleton", 60, 15, 75, 75))

	// blind (dim vision): strikes what is in reach, never pursues
	b.ApplyForced(ForcedBlind, 0, 100, 0)

	if b.Def.Name != "State10" {
		t.Fatalf("def %s", b.Def.Name)
	}

	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("blind monster in reach: %v", w.log)
	}

	w.inRange = false
	w.frame, b.Wake = 3, 3
	w.log = nil

	for i := 0; i < 30; i++ {
		w.frame++
		b.Wake = w.frame
		Tick(w, b)
	}

	for _, l := range w.log {
		if strings.HasPrefix(l, "walk-target") || strings.HasPrefix(l, "run-target") {
			t.Fatalf("a blind monster must not chase: %v", w.log)
		}
	}

	// without an attack animation it cannot strike
	w2 := newFake2(3, true)
	w2.noMode[ModeAttack1] = true
	b2 := brainAt(profile("Skeleton"))
	b2.ApplyForced(ForcedBlind, 0, 100, 0)
	Tick(w2, b2)

	if w2.last() == "attack4" {
		t.Fatal("no A1 animation, no strike")
	}

	// taunt: goes for the taunter
	w3 := newFake2(20, false)
	w3.src, w3.srcOK = Target{ID: 9, X: 130, Y: 100, Size: 1, IsPlayer: true}, true
	b3 := brainAt(profile("Skeleton"))
	b3.ApplyForced(ForcedTaunt, 0, 100, 9)
	Tick(w3, b3)

	if w3.last() != "walk-target/7" {
		t.Fatalf("taunted monster should walk to the taunter: %v", w3.log)
	}

	// next tick: the taunter is in reach -> strike it
	w3.inRange = true
	w3.frame, b3.Wake = 4, 4
	Tick(w3, b3)

	if w3.last() != "attack4" {
		t.Fatalf("taunted monster should strike the taunter: %v", w3.log)
	}

	// the taunter is gone: back to the class AI
	w3.srcOK = false
	w3.frame, b3.Wake = 8, 8
	Tick(w3, b3)

	if b3.Def.Name != "Skeleton" || b3.Forced != ForcedNone {
		t.Fatalf("taunt should end: %s", b3.Def.Name)
	}
}

func TestConfuseTargetsMonsters(t *testing.T) {
	w := newFake2(30, false)
	w.any, w.anyDist, w.anyOK = Target{ID: 99, X: 110, Y: 100, Size: 1}, 8, true

	b := brainAt(profile("Skeleton", 100, 15, 100, 100))

	Tick(w, b)

	if b.TargetID != 1 {
		t.Fatalf("a sane monster targets the hero, got %d", b.TargetID)
	}

	b.ApplyForced(ForcedConfuse, 0, 60, 0)

	if b.Def.Name != "Skeleton" {
		t.Fatalf("confuse keeps the think function, got %s", b.Def.Name)
	}

	w.frame, b.Wake = 1, 1
	Tick(w, b)

	if b.TargetID != 99 {
		t.Fatalf("a confused monster targets the nearest unit of any kind, got %d", b.TargetID)
	}

	// nobody around: it has nothing to attack and sleeps
	w.anyOK = false
	w.frame, b.Wake = 2, 2
	Tick(w, b)

	if b.HasTarget {
		t.Error("no target expected")
	}

	// the effect wears off
	w.frame = 60
	Tick(w, b)

	if b.Forced != ForcedNone || b.TargetID != 1 {
		t.Fatalf("confusion should be over: forced=%v target=%d", b.Forced, b.TargetID)
	}
}

func TestAttractDecoy(t *testing.T) {
	w := newFake2(20, false)
	w.decoy, w.decoyDist, w.decoyOK = Target{ID: 77, X: 108, Y: 100, Size: 1}, 8, true

	b := brainAt(profile("Skeleton", 100, 15, 100, 100))
	Tick(w, b)

	if b.TargetID != 77 {
		t.Fatalf("the decoy is nearer than the hero and inside the aggro radius: target %d", b.TargetID)
	}

	// decoy farther than the hero: the hero keeps the attention
	w.decoyDist = 25
	w.frame, b.Wake = 1, 1
	Tick(w, b)

	if b.TargetID != 1 {
		t.Fatalf("hero is nearer than the decoy: target %d", b.TargetID)
	}

	// the attracted monster itself is flagged for the duration
	b.ApplyForced(ForcedAttract, 0, 10, 3)

	if !b.Attracting || b.Forced != ForcedAttract {
		t.Fatal("attract flag")
	}

	b.ClearForced(10)

	if b.Attracting {
		t.Fatal("attract should end")
	}
}

func TestCharmFlagsAndReplacement(t *testing.T) {
	b := brainAt(profile("Skeleton", 60, 15, 75, 75))
	b.ApplyForced(ForcedCharm, 0, 100, 4)

	if !b.Allied || b.OwnerID != 4 {
		t.Fatal("converted monster must be allied to its owner")
	}

	// a curse replaces a conversion: the old one ends first
	b.ApplyForced(ForcedFear, 5, 100, 1)

	if b.Allied || b.Forced != ForcedFear || b.baseDef == nil || b.baseDef.Name != "Skeleton" {
		t.Fatalf("replacement did not restore first: allied=%v forced=%v", b.Allied, b.Forced)
	}

	b.ClearForced(7)

	if b.Def.Name != "Skeleton" || b.Wake != 7 {
		t.Fatalf("def=%s wake=%d", b.Def.Name, b.Wake)
	}
}

func TestStateLabel(t *testing.T) {
	b := brainAt(profile("Skeleton"))
	if b.StateLabel() != "Skeleton" {
		t.Fatal(b.StateLabel())
	}

	b.ApplyForced(ForcedConfuse, 0, 5, 0)

	if b.StateLabel() != "Skeleton+confuse" {
		t.Fatal(b.StateLabel())
	}
}

func TestState2WanderAndExit(t *testing.T) {
	// close target: back to the class AI
	w := newFake2(2, false)
	b := brainAt(profile("Skeleton"))
	b.ApplyForced(ForcedFear, 0, 100, 0)
	b.SetAI(StateDef(StateWander))
	Tick(w, b)

	if b.Def.Name != "Skeleton" {
		t.Fatalf("close target should end State2, def=%s", b.Def.Name)
	}

	// far target: first tick approaches (12 near the target), once
	w = newFake2(15, false)
	b = brainAt(profile("Skeleton"))
	b.SetAI(StateDef(StateWander))
	Tick(w, b)

	if len(w.log) != 1 || b.Scratch[0] != 1 {
		t.Fatalf("approach: %v scratch=%v", w.log, b.Scratch)
	}

	// afterwards it only shuffles or waits
	for i := 0; i < 20; i++ {
		w.frame += 10
		b.Wake = w.frame
		Tick(w, b)
	}

	for _, l := range w.log[1:] {
		if !strings.HasPrefix(l, "walk-to") {
			t.Fatalf("unexpected action %s", l)
		}
	}
}

func TestState6TownGuard(t *testing.T) {
	// in town: no hunting, only the idle shuffle
	w := newFake2(5, true)
	w.town = true
	b := brainAt(profile("Skeleton"))
	b.SetAI(StateDef(StateTownGuard))

	for i := 0; i < 40; i++ {
		w.frame += 10
		b.Wake = w.frame
		Tick(w, b)
	}

	for _, l := range w.log {
		if strings.HasPrefix(l, "attack") || strings.Contains(l, "target") {
			t.Fatalf("in town it must not hunt: %v", w.log)
		}
	}

	// not neutral: wait 5
	w = newFake2(5, true)
	b = brainAt(profile("Skeleton"))
	b.SetAI(StateDef(StateTownGuard))
	b.Mode = ModeAttack1
	Tick(w, b)

	if b.Wake != 5 {
		t.Fatalf("wake %d", b.Wake)
	}

	// outside, hero in reach: strikes on 80% of the rolls
	hits := 0

	for id := uint32(1); id <= 200; id++ {
		w = newFake2(5, true)
		b = NewBrain(id, 1, Normal, profile("Skeleton"), testSeed)
		b.X, b.Y = 100, 100
		b.SetAI(StateDef(StateTownGuard))
		Tick(w, b)

		if w.last() == "attack4" {
			hits++
		}
	}

	if hits < 130 || hits > 190 {
		t.Fatalf("expected about 80%% strikes, got %d/200", hits)
	}
}

func TestState8And14(t *testing.T) {
	w := newFake2(10, false)
	b := brainAt(profile("Skeleton"))
	b.SetAI(StateDef(StateAttack25))
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("state 8 within 25: %v", w.log)
	}

	w = newFake2(30, false)
	b = brainAt(profile("Skeleton"))
	b.SetAI(StateDef(StateAttack25))
	b.Profile.AIDist = 55
	Tick(w, b)

	if b.Wake != 50 {
		t.Fatalf("state 8 beyond 25 waits 50, wake=%d", b.Wake)
	}

	// state 14: in reach, 95%: A2 and two LCG steps
	b = findBrain(t, profile("Skeleton"), func(s *d2rand.Seed) bool { return s.Roll(100) < 0x5f })
	b.SetAI(StateDef(StateCharge))
	w = newFake2(2, true)
	before := shadow(b)
	Tick(w, b)

	if w.last() != "attack5" {
		t.Fatalf("state 14: %v", w.log)
	}

	before.Step()
	before.Step()

	if *before != *b.Seed {
		t.Error("state 14 must consume exactly two generator steps in reach")
	}
}

func TestState9ReleasesFollowers(t *testing.T) {
	leader := brainAt(profile("Skeleton"))
	m1 := NewBrain(8, 1, Normal, profile("Skeleton"), testSeed)
	m2 := NewBrain(9, 1, Normal, profile("Skeleton"), testSeed)
	leader.AddMinion(m1)
	leader.AddMinion(m2)

	// m1 leads a sub-group of its own
	sub := NewBrain(10, 1, Normal, profile("Skeleton"), testSeed)
	m1.AddMinion(sub)

	m1.SetAI(StateDef(StateRelease))

	// the target is far: it stays
	w := newFake2(30, false)
	m1.Profile.AIDist = 55
	Tick(w, m1)

	if m1.Def.Name != "State9" || m1.Wake != 10 {
		t.Fatalf("far target: def=%s wake=%d", m1.Def.Name, m1.Wake)
	}

	// the target is close: its followers are told, it leaves the leader
	w = newFake2(10, false)
	m1.Wake = 0
	Tick(w, m1)

	if m1.Def.Name != "Skeleton" || m1.Leader != nil || len(leader.Minions) != 1 {
		t.Fatalf("release failed: def=%s leader=%v minions=%d", m1.Def.Name, m1.Leader, len(leader.Minions))
	}

	if c := sub.PeekCommand(); c == nil || c.Type != CmdRelease {
		t.Fatal("the sub-group should have received the release command")
	}
}

func TestState3Formation(t *testing.T) {
	leader := brainAt(profile("Skeleton"))
	f := NewBrain(8, 1, Normal, profile("Skeleton"), testSeed)
	f.X, f.Y = 104, 98
	leader.AddMinion(f)
	leader.SetAI(StateDef(StateFormation))
	f.SetAI(StateDef(StateFormation))

	w := newFake2(30, false)
	leader.Profile.AIDist = 55
	f.Profile.AIDist = 55

	// follower with no order waits 15
	Tick(w, f)

	if f.Wake != 15 || f.Scratch[0] != 4 || f.Scratch[1] != -2 {
		t.Fatalf("follower: wake=%d offset=%v", f.Wake, f.Scratch)
	}

	// the leader marches and broadcasts its point (it may also wait: retry)
	var cmd *Command

	for i := 0; i < 60 && cmd == nil; i++ {
		w.frame += 10
		leader.Wake = w.frame
		Tick(w, leader)
		cmd = f.PeekCommand()
	}

	if cmd == nil || cmd.Type != 2 {
		t.Fatal("the leader should broadcast a march point")
	}

	// the follower walks to the point plus its offset and consumes the order
	w.log = nil
	f.Wake = 0
	Tick(w, f)

	if f.QueueLen() != 0 || !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("follower: %v queue=%d", w.log, f.QueueLen())
	}

	// an aggressive leader dissolves the march
	leader.Aggressive = true
	leader.Wake = 0
	Tick(w, leader)

	if leader.Def.Name != "Skeleton" {
		t.Fatal("aggressive leader should end the march")
	}

	if c := f.PeekCommand(); c == nil || c.Type != CmdRelease {
		t.Fatal("the follower should get the release command")
	}
}
