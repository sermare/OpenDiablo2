package d2monster

import (
	"fmt"
	"strings"
	"testing"
)

// fakeBaal adds the Baal extensions to the scripted fakeWorld.
type fakeBaal struct {
	*fakeWorld
	steps    []string
	portal   Point
	portalOK bool
	portalD  int
	left     bool
	dismiss  bool
	corpse   Target
	corpseOK bool
	raw      []string
	idle     bool
	pulled   bool
	cleared  bool
	hasGate  bool
}

func (f *fakeBaal) Throne(_ *Brain, s ThroneStep, wave int) bool {
	f.steps = append(f.steps, fmt.Sprintf("%d:%d", s, wave))

	return true
}

func (f *fakeBaal) NearestObject(_ *Brain, class, radius int) (Point, int, bool) {
	if class != ObjectWorldstonePortal || radius != 25 {
		return Point{}, 0, false
	}

	return f.portal, f.portalD, f.portalOK
}

func (f *fakeBaal) LeaveLevel(*Brain) { f.left = true }
func (f *fakeBaal) Dismiss(*Brain)    { f.dismiss = true }

func (f *fakeBaal) NearestCorpse(*Brain, []int, int) (Target, bool) { return f.corpse, f.corpseOK }

func (f *fakeBaal) CastSkillID(_ *Brain, skill int, _ Mode, _ *Target, _ *Point) bool {
	f.raw = append(f.raw, fmt.Sprint(skill))

	return true
}

func (f *fakeBaal) TargetMode(*Brain, Target) Mode {
	if f.idle {
		return ModeNeutral
	}

	return ModeWalk
}

func (f *fakeBaal) PullTarget(*Brain, Target) bool { f.pulled = true; return true }

func TestBaalRegistered(t *testing.T) {
	for _, n := range []string{"BaalThrone", "BaalTaunt", "BaalToStairs", "BaalTentacle", "BaalCrab", "BaalCrabClone", "FallenShaman"} {
		if d, ok := Lookup(n); !ok || !d.Implemented {
			t.Errorf("%s is not registered", n)
		}
	}

	if StateDef(StateLeash) == nil || StateDef(StateImp) == nil {
		t.Error("forced states 13 and 16 are not registered")
	}
}

// The throne runs announce -> 250 frames -> wave -> 100 frames, five times, then
// morphs (the step list is the control flow of 0x5ee400).
func TestBaalThroneWaves(t *testing.T) {
	w := &fakeBaal{fakeWorld: newFake(80, false)}
	w.hasTarget = false
	b := brainAt(skills(profile("BaalThrone", 100), "Static"))

	for f := 0; f < 3000 && len(w.steps) < 13; f++ {
		w.frame = f
		Tick(w, b)
	}

	want := "0:0 1:0 0:1 1:1 0:2 1:2 0:3 1:3 0:4 1:4 0:5 2:5"
	if got := strings.Join(w.steps[:12], " "); got != want {
		t.Fatalf("steps\n got %s\nwant %s", got, want)
	}

	if b.Scratch[0] != 5 {
		t.Fatalf("wave counter %d, want 5", b.Scratch[0])
	}
}

func TestBaalThroneTimes(t *testing.T) {
	w := &fakeBaal{fakeWorld: newFake(80, false)}
	w.hasTarget = false
	b := brainAt(skills(profile("BaalThrone", 100), "Static"))

	w.frame = 10
	Tick(w, b)

	if b.Scratch[1] != 1 || b.Scratch[2] != 260 {
		t.Fatalf("after the announcement flags=%d next=%d (want 1, 260)", b.Scratch[1], b.Scratch[2])
	}

	w.frame = 259
	b.Wake = 0
	Tick(w, b)

	if len(w.steps) != 1 {
		t.Fatalf("acted before the wave was due: %v", w.steps)
	}

	w.frame = 260
	b.Wake = 0
	Tick(w, b)

	if b.Scratch[0] != 1 || b.Scratch[2] != 360 || b.Scratch[1] != 2 {
		t.Fatalf("after the wave: counter=%d next=%d flags=%d (want 1, 360, 2)", b.Scratch[0], b.Scratch[2], b.Scratch[1])
	}
}

// A hero within 64 makes the throne attack instead of sending waves.
func TestBaalThroneAttacksIntruder(t *testing.T) {
	w := &fakeBaal{fakeWorld: newFake(30, false)}
	b := brainAt(skills(profile("BaalThrone", 100), "Static"))
	b.Profile.AIDist = 64

	Tick(w, b)

	if w.last() != "cast0" || len(w.steps) != 0 {
		t.Fatalf("log=%v steps=%v", w.log, w.steps)
	}
}

type gateWorld struct {
	*fakeBaal
	cleared bool
}

func (g *gateWorld) WaveCleared(*Brain) bool { return g.cleared }

func TestBaalThroneWaveGate(t *testing.T) {
	g := &gateWorld{fakeBaal: &fakeBaal{fakeWorld: newFake(80, false)}}
	g.hasTarget = false
	b := brainAt(skills(profile("BaalThrone", 100), "Static"))

	run := func(upTo int) {
		for f := 0; f < upTo; f++ {
			g.frame = f
			Tick(g, b)
		}
	}

	run(400) // wave 0 is sent without a gate, wave 1 is announced
	if b.Scratch[0] != 1 {
		t.Fatalf("counter %d", b.Scratch[0])
	}

	run(700) // wave 1 is due but the first wave still lives
	if b.Scratch[0] != 1 {
		t.Fatalf("a wave was sent over the living one: counter %d", b.Scratch[0])
	}

	g.cleared = true

	run(900)
	if b.Scratch[0] != 2 {
		t.Fatalf("counter %d, want 2 after the clear", b.Scratch[0])
	}
}

func TestBaalToStairs(t *testing.T) {
	// aip1 = 5: inside the radius Baal enters; farther he walks.
	w := &fakeBaal{fakeWorld: newFake(80, false), portal: Point{140, 100}, portalOK: true, portalD: 30}
	b := brainAt(profile("BaalToStairs", 5))

	Tick(w, b)

	if w.left || w.last() != "walk-to(140,100)" {
		t.Fatalf("far from the portal: left=%v log=%v", w.left, w.log)
	}

	w.portalD = 3
	b.Wake = 0
	Tick(w, b)

	if !w.left {
		t.Fatal("Baal did not enter the portal")
	}

	// no portal in the scan radius: sleep 25
	w2 := &fakeBaal{fakeWorld: newFake(80, false)}
	b2 := brainAt(profile("BaalToStairs", 5))
	w2.frame = 100
	Tick(w2, b2)

	if b2.Wake != 125 {
		t.Fatalf("wake %d, want 125", b2.Wake)
	}
}

func TestBaalTaunt(t *testing.T) {
	// aip1 far 20, aip2 idle count 2, aip3 pull distance 40
	w := &fakeBaal{fakeWorld: newFake(10, false), idle: true}
	b := brainAt(profile("BaalTaunt", 20, 2, 40))

	for i := 0; i < 3; i++ {
		b.Wake = 0
		Tick(w, b)
	}

	if len(w.raw) != 1 || w.raw[0] != "284" {
		t.Fatalf("taunt casts %v after 3 idle thinks (want one Baal Taunt)", w.raw)
	}

	if b.Scratch[0] != 0 {
		t.Fatalf("idle count not reset: %d", b.Scratch[0])
	}

	// a moving hero resets the count
	w.idle = false
	b.Scratch[0] = 2
	b.Wake = 0
	Tick(w, b)

	if b.Scratch[0] != 0 {
		t.Fatal("moving hero did not reset the count")
	}

	// too far: pulled back
	w2 := &fakeBaal{fakeWorld: newFake(50, false)}
	b2 := brainAt(profile("BaalTaunt", 20, 2, 40))
	b2.Profile.AIDist = 55
	Tick(w2, b2)

	if !w2.pulled {
		t.Fatal("distant hero not pulled")
	}
}

func TestBaalTentacle(t *testing.T) {
	// aip1 100: always attacks while alive; lifetime (aip3 + roll(10)) * 25
	w := &fakeBaal{fakeWorld: newFake(5, true)}
	b := brainAt(profile("BaalTentacle", 100, 7, 4))
	w.frame = 10
	Tick(w, b)

	if w.last() != "attack5" {
		t.Fatalf("log %v", w.log)
	}

	life := b.Scratch[2] - 10
	if life < 4*25 || life > 13*25 || life%25 != 0 {
		t.Fatalf("lifetime %d frames", life)
	}

	w.frame = b.Scratch[2] + 1
	b.Wake = 0
	Tick(w, b)

	if !w.dismiss {
		t.Fatal("expired tentacle not dismissed")
	}

	// without a target it goes away at once
	w2 := &fakeBaal{fakeWorld: newFake(5, true)}
	w2.hasTarget = false
	Tick(w2, brainAt(profile("BaalTentacle", 100, 7, 4)))

	if !w2.dismiss {
		t.Fatal("targetless tentacle kept")
	}
}

// The verified constants of the three decision tables.
func TestBaalWeights(t *testing.T) {
	s := BaalSituation{InReach: true, Engaged: true, SelfHP: 100, TargetHP: 80}

	w := BaalWeights(Normal, s)
	if w[baalWait] != 75 || w[baalMelee] != 150 || w[baalCold] != 45 || w[baalTentacles] != 30 || w[baalBuff] != 20 {
		t.Errorf("reach table (normal): %v", w)
	}

	w = BaalWeights(Hell, BaalSituation{InReach: true, Engaged: true, SelfHP: 40, TargetHP: 20})
	if w[baalWait] != 50+60 || w[baalMelee] != 200 || w[baalCold] != 70 {
		t.Errorf("reach table (hell, hurt, weak target): %v", w)
	}

	w = BaalWeights(Hell, BaalSituation{InReach: true, Engaged: true, SelfHP: 100, TargetBlocked: true})
	if w[baalBuff] != 0 || w[baalCold] != 0 || w[baalNova] != 0 || w[baalInferno] != 10 {
		t.Errorf("blocked target: %v", w)
	}

	w = BaalWeights(Hell, BaalSituation{InReach: true, Engaged: false, SelfHP: 100})
	if w[baalNova] != 0 || w[baalCold] != 0 || w[baalInferno] != 0 {
		t.Errorf("not engaged: %v", w)
	}

	// engaged, out of reach, three targets, far target
	w = BaalWeights(Hell, BaalSituation{Engaged: true, SelfHP: 100, Targets: 4, Dist: 40, Clones: 0})
	if w[baalWait] != 100 || w[baalClone] != 20 || w[6] != 15 || w[baalCold] != 25+0 || w[baalTeleport] != 30 || w[baalNova] != 0 {
		t.Errorf("engaged table: %v", w)
	}

	w = BaalWeights(Normal, BaalSituation{Engaged: true, SelfHP: 100, Targets: 1, Dist: 5, Clones: 3, Nearby: 5})
	if w[baalClone] != 0 || w[baalWait] != 125 || w[baalNova] != 60 || w[baalCold] != 40 {
		t.Errorf("engaged table normal: %v", w)
	}

	// idle table
	w = BaalWeights(Normal, BaalSituation{SelfHP: 50, Targets: 1})
	if w[baalWait] != 175 || w[2] != 45 || w[baalMelee] != 25 || w[baalBuff] != 80 {
		t.Errorf("idle table: %v", w)
	}

	w = BaalWeights(Hell, BaalSituation{SelfHP: 100, Targets: 3, VeryFar: true})
	if w[baalHome] != 60 || w[2] != 20 {
		t.Errorf("idle table, away from home: %v", w)
	}
}

func TestBaalCrabCastsFromSlots(t *testing.T) {
	// the monstats order: Skill1 Nova, Skill2 Inferno, Skill3 Tentacle, Skill4 Cold Missiles
	p := skills(profile("BaalCrab"), "Baal Nova", "Baal Inferno", "Baal Tentacle", "Baal Cold Missiles")
	seen := map[string]bool{}

	for id := uint32(1); id < 400; id++ {
		w := &fakeBaal{fakeWorld: newFake(10, true)}
		b := NewBrain(id, 1, Hell, p, testSeed)
		b.Aggressive = true
		b.X, b.Y = 100, 100
		Tick(w, b)

		seen[w.last()] = true

		if b.Wake != 25 && !strings.HasPrefix(w.last(), "") {
			t.Fatal("unreachable")
		}

		if b.Wake != 25 {
			t.Fatalf("the action must be followed by a 25-frame wake, got %d (%v)", b.Wake, w.log)
		}

		if b.FindCommand(CmdAnchor) == nil {
			t.Fatal("no anchor command")
		}
	}

	for _, want := range []string{"cast0", "cast1", "cast3", "attack5"} {
		if !seen[want] {
			t.Errorf("action %s never chosen: %v", want, seen)
		}
	}
}

func TestBaalCrabCloneDismissedWithoutLeader(t *testing.T) {
	w := &fakeBaal{fakeWorld: newFake(10, true)}
	p := skills(profile("BaalCrabClone"), "Baal Nova", "Baal Inferno", "Baal Tentacle", "Baal Cold Missiles")
	b := brainAt(p)
	leader := brainAt(p)
	leader.Mode = ModeDying
	b.Leader = leader
	Tick(w, b)

	if !w.dismiss {
		t.Fatal("orphaned clone survives")
	}
}

// ---- FallenShaman ----

func TestFallenShaman(t *testing.T) {
	p := skills(profile("FallenShaman", 100, 100, 0, 24, 15), "Resurrect", "Shaman Fire")

	// a corpse in reach: Resurrect (slot 1) on it
	w := &fakeBaal{fakeWorld: newFake(8, false), corpse: Target{ID: 9, X: 110, Y: 100}, corpseOK: true}
	b := brainAt(p)
	Tick(w, b)

	if w.last() != "cast0" {
		t.Fatalf("expected a resurrection, log %v", w.log)
	}

	// no corpse: Shaman Fire (slot 2) at the target within aip5
	w2 := &fakeBaal{fakeWorld: newFake(8, false)}
	b2 := brainAt(p)
	Tick(w2, b2)

	if w2.last() != "cast1" {
		t.Fatalf("expected Shaman Fire, log %v", w2.log)
	}

	// in reach and aip3 100: melee
	p3 := skills(profile("FallenShaman", 100, 100, 100, 24, 15), "Resurrect", "Shaman Fire")
	w3 := &fakeBaal{fakeWorld: newFake(2, true)}
	Tick(w3, brainAt(p3))

	if w3.last() != "attack4" {
		t.Fatalf("expected the melee swing, log %v", w3.log)
	}
}

// ---- forced states ----

func TestState13Leash(t *testing.T) {
	w := &fakeBaal{fakeWorld: newFake(3, true)}
	p := profile("Boss", 100, 20, 100, 0, 0)
	b := brainAt(p)
	b.SetAI(StateDef(StateLeash))
	Tick(w, b)

	if a := b.FindCommand(CmdAnchor); a == nil || a.X != 100 || a.Y != 100 {
		t.Fatalf("anchor %+v", a)
	}

	if w.last() != "attack4" {
		t.Fatalf("log %v", w.log)
	}

	// out of reach and aip1 100: run to the target
	w2 := &fakeBaal{fakeWorld: newFake(15, false)}
	b2 := brainAt(p)
	b2.SetAI(StateDef(StateLeash))
	Tick(w2, b2)

	if w2.last() != "run-target/0" {
		t.Fatalf("log %v", w2.log)
	}
}

func TestState16RestoresClassAI(t *testing.T) {
	w := &fakeBaal{fakeWorld: newFake(10, true)}
	p := profile("Imp", 100, 100, 100)
	b := brainAt(p)
	b.ApplyForced(ForcedFear, 0, 1000, 1)
	b.SetAI(StateDef(StateImp)) // the forced fear replaced by the imp state for the test

	Tick(w, b) // no state 0x8f: class AI back

	if b.Def == StateDef(StateImp) {
		t.Fatal("imp state kept without the 0x8f state")
	}
}
