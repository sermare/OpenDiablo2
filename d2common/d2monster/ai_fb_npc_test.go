package d2monster

import "testing"

func fbnNew(ai string, aip ...int) (*fakeWorld, *Brain) {
	w := newFake(6, true)
	b := brainAt(profile(ai, aip...))
	def, _ := Lookup(ai)
	b.SetAI(def)

	return w, b
}

func TestFBNTownThinkers(t *testing.T) {
	// Vendor: one roll; below 20 the wave (S1), else a 30 frame wait.
	for seed := uint32(1); seed < 12; seed++ {
		w, b := fbnNew("Vendor")
		b.Seed.Init(seed)
		want := shadow(b).Roll(100) < 20

		Tick(w, b)

		if want && w.last() != "attack8" || !want && (len(w.log) != 0 || b.Wake != 30) {
			t.Errorf("vendor seed %d: want wave=%v log=%v wake=%d", seed, want, w.log, b.Wake)
		}
	}

	// TownRogue: shoots within 24, otherwise waits 50, no roll.
	for _, c := range []struct {
		dist int
		log  string
		wake int
	}{{10, "attack4", waitForever}, {24, "attack4", waitForever}, {25, "", 50}} {
		w, b := fbnNew("TownRogue")
		w.dist = c.dist
		w.target.X = 100 + c.dist
		before := *b.Seed

		Tick(w, b)

		if w.last() != c.log || (c.log == "" && b.Wake != c.wake) || *b.Seed != before {
			t.Errorf("rogue dist %d: log=%v wake=%d", c.dist, w.log, b.Wake)
		}
	}

	// Towner: first think records the anchor and waits 20, then waits 12.
	w, b := fbnNew("Towner")
	Tick(w, b)

	if n := b.FindCommand(CmdAnchor); n == nil || n.X != 100 || n.Y != 100 || b.Wake != 20 {
		t.Fatalf("towner anchor: %+v wake=%d", n, b.Wake)
	}

	b.WakeNow(0)
	Tick(w, b)

	if b.Wake != 12 {
		t.Errorf("towner idle wake = %d", b.Wake)
	}

	// Navi without an NPC server shoots a target within 24 or waits 50.
	w, b = fbnNew("Navi")
	Tick(w, b)

	if w.last() != "attack4" {
		t.Errorf("navi: %v", w.log)
	}

	// NpcStationary and Wussie without their hosts just wait.
	for _, c := range []struct {
		ai   string
		wake int
	}{{"NpcStationary", 20}, {"Wussie", 25}} {
		w, b = fbnNew(c.ai)
		Tick(w, b)

		if len(w.log) != 0 || b.Wake != c.wake {
			t.Errorf("%s: log=%v wake=%d", c.ai, w.log, b.Wake)
		}
	}

	// NpcOutOfTown: the first think queues the type 3 walk node and waits 1.
	w, b = fbnNew("NpcOutOfTown")
	Tick(w, b)

	if n := b.FindCommand(3); n == nil || n.X != 103 || n.Y != 103 || b.Wake != 1 {
		t.Errorf("outoftown node %+v wake=%d", n, b.Wake)
	}
}

func TestFBNNpcBarb(t *testing.T) {
	// in reach: A1 then a wait of aip1
	w, b := fbnNew("NpcBarb", 17, 100, 50)
	Tick(w, b)

	if w.last() != "attack4" || b.Wake != 17 {
		t.Errorf("barb attack: %v wake=%d", w.log, b.Wake)
	}

	// out of reach, inside aip3, roll below aip2: a run at the target
	w, b = fbnNew("NpcBarb", 17, 100, 50)
	w.inRange = false
	sh := shadow(b)
	first := sh.Roll(100) < 100

	Tick(w, b)

	if !first || w.last() != "run-target/0" {
		t.Errorf("barb charge: %v", w.log)
	}

	// no target: shuffle around the post, two steps drawn for the first try
	w, b = fbnNew("NpcBarb", 17, 0, 0)
	w.hasTarget = false
	sh = shadow(b)
	p, q := sh.Step(), sh.Step()

	Tick(w, b)

	want := "walk-to(" + itoa(100+int(p%20)-40) + "," + itoa(100+int(q%20)-10) + ")"
	if w.last() != want || *b.Seed != *sh {
		t.Errorf("barb shuffle: %v want %s", w.log, want)
	}

	// every walk fails: three tries drawing 2+3+2 steps, then a 15 frame wait
	w, b = fbnNew("NpcBarb", 17, 0, 0)
	w.hasTarget, w.failMove = false, true
	sh = shadow(b)

	for i := 0; i < 7; i++ {
		sh.Step()
	}

	Tick(w, b)

	if b.Wake != 15 || *b.Seed != *sh {
		t.Errorf("barb idle: wake=%d", b.Wake)
	}
}

// npcHostWorld implements the optional NPC interfaces for the greeting tests.
type npcHostWorld struct {
	*fakeWorld
	player      Target
	havePlayer  bool
	interact    bool
	faced       int
	gestures    bool
	gestureCall int
}

func (n *npcHostWorld) HasInteractEntry(*Brain) bool      { return n.interact }
func (n *npcHostWorld) InteractListHasPlayer(*Brain) bool { return false }
func (n *npcHostWorld) InteractBusy(*Brain, Target) bool  { return false }
func (n *npcHostWorld) FacePlayer(*Brain, Target)         { n.faced++ }
func (n *npcHostWorld) RandomIdle(*Brain) bool            { return false }
func (n *npcHostWorld) PlayGestures(*Brain) bool          { n.gestureCall++; return n.gestures }
func (n *npcHostWorld) NearestPlayerNearby(*Brain) (Target, bool, bool) {
	return n.player, true, n.havePlayer
}

func TestFBNNpcGreeting(t *testing.T) {
	// a player 10 away (inside 3..23): the Npc steps up to it with the kiting
	// walk after setting its anchor; after the anchor exists it walks.
	fw := newFake(10, true)
	w := &npcHostWorld{fakeWorld: fw, player: fw.target, havePlayer: true}
	b := brainAt(profile("Npc"))
	def, _ := Lookup("Npc")
	b.SetAI(def)

	Tick(w, b)

	if b.Wake != 20 || b.FindCommand(CmdAnchor) == nil {
		t.Fatalf("anchor step: wake=%d", b.Wake)
	}

	b.WakeNow(0)
	Tick(w, b)

	if len(fw.log) != 1 {
		t.Errorf("approach: %v", fw.log)
	}

	// a far player: the greeting timer starts and the NPC faces it
	fw = newFake(40, true)
	w = &npcHostWorld{fakeWorld: fw, player: fw.target, havePlayer: true}
	b = brainAt(profile("Npc"))
	b.SetAI(def)
	b.AppendCommand(Command{Type: CmdAnchor, X: 100, Y: 100})
	fw.target.IsPlayer = true
	w.player = fw.target

	Tick(w, b)

	if w.faced != 1 || b.Scratch[1] != 0x3c || b.Wake != 20 {
		t.Errorf("far player: faced=%d timer=%d wake=%d", w.faced, b.Scratch[1], b.Wake)
	}

	// someone talking to it: nothing but the busy countdown
	w.interact = true
	b.WakeNow(0)
	b.Scratch[0] = 50

	Tick(w, b)

	if b.Scratch[0] != 49 {
		t.Errorf("busy countdown = %d", b.Scratch[0])
	}
}
