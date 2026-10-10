package d2monster

import "testing"

func TestFaithfulBatch4Registered(t *testing.T) {
	for _, n := range FaithfulBatch4() {
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
	}
}

func TestUnitTileDist(t *testing.T) {
	b := &Brain{X: 100, Y: 100}

	for _, c := range []struct{ x, y, want int }{
		{100, 100, 0}, {101, 100, 1}, {102, 100, 2}, {100, 103, 3}, {110, 100, 10},
		{110, 104, 12}, {90, 96, 12}, {103, 104, 5}, {104, 103, 5},
	} {
		if got := unitTileDist(b, c.x, c.y); got != c.want {
			t.Errorf("(%d,%d): %d want %d", c.x, c.y, got, c.want)
		}
	}
}

// f4World adds the batch-4 host interfaces to a fakePet.
type f4World struct {
	*fakePet
	cand       *Target
	vitalsCur  int
	vitalsMax  int
	auraRange  int
	auraAsked  int
	targetDead bool
}

func (w *f4World) NearbySkillTarget(_ *Brain, r int) (Target, bool) {
	w.auraAsked = r
	if w.cand == nil {
		return Target{}, false
	}

	return *w.cand, true
}
func (w *f4World) LeaderVitals(*Brain, int) (int, int)   { return w.vitalsCur, w.vitalsMax }
func (w *f4World) AuraRange(*Brain, int, int) int        { return w.auraRange }
func (w *f4World) TargetInDeathMode(*Brain, Target) bool { return w.targetDead }

func f4Brain(ai string, class int, slots []int, aip ...int) *Brain {
	b := brainAt(withSkills(profile(ai, aip...), slots...))
	b.Class = class
	def, _ := Lookup(ai)
	b.SetAI(def)

	return b
}

func TestAssassinSentry(t *testing.T) {
	// aip1 fire %, aip2 miss wait, aip3 idle wait, aip4 range
	cases := []struct {
		name    string
		aip     []int
		dist    int
		hasTgt  bool
		noOwner bool
		noSkill bool
		life    int // S1 before the think (-2: leave the pre-hook value)
		log     string
		wake    int
		s1      int
	}{
		{name: "fires in range", aip: []int{100, 7, 9, 20}, dist: 10, hasTgt: true, life: -2, log: "cast0", s1: 999},
		{name: "range edge is exclusive", aip: []int{100, 7, 9, 10}, dist: 10, hasTgt: true, life: -2, wake: 9, s1: 1000},
		{name: "no target waits aip3", aip: []int{100, 7, 9, 20}, hasTgt: false, life: -2, wake: 9, s1: 1000},
		{name: "missed roll waits aip2", aip: []int{0, 7, 9, 20}, dist: 10, hasTgt: true, life: -2, wake: 7, s1: 1000},
		{name: "no leader dies", aip: []int{100, 7, 9, 20}, dist: 10, hasTgt: true, noOwner: true, life: -2, log: "attack0", s1: -1},
		{name: "no skill dies", aip: []int{100, 7, 9, 20}, dist: 10, hasTgt: true, noSkill: true, life: -2, log: "attack0", s1: -1},
		{name: "life used up dies", aip: []int{100, 7, 9, 20}, dist: 10, hasTgt: true, life: 0, log: "attack0", s1: 0},
		{name: "last life tick still fires", aip: []int{100, 7, 9, 20}, dist: 10, hasTgt: true, life: 1, log: "cast0", s1: 0},
	}

	for _, c := range cases {
		f, _ := newPet("AssassinSentry", 5)
		f.hasTarget, f.attackOK, f.dist = c.hasTgt, c.hasTgt, c.dist
		f.noOwner = c.noOwner

		slots := []int{0}
		if c.noSkill {
			slots = nil
		}

		b := f4Brain("AssassinSentry", 1, slots, c.aip...)

		if c.life != -2 {
			b.preRan = true
			b.Scratch[1] = c.life
		}

		Tick(f, b)

		if c.log != "" && f.last() != c.log || c.log == "" && len(f.log) != 0 {
			t.Errorf("%s: log %v", c.name, f.log)
		}

		if c.log == "" && b.Wake != c.wake {
			t.Errorf("%s: wake %d want %d", c.name, b.Wake, c.wake)
		}

		if b.Scratch[1] != c.s1 {
			t.Errorf("%s: S1 %d want %d", c.name, b.Scratch[1], c.s1)
		}
	}

	// The pre-hook arms S0 with the frame of the first think.
	f, _ := newPet("AssassinSentry", 5)
	f.frame = 77
	b := f4Brain("AssassinSentry", 1, []int{0}, 100, 7, 9, 20)
	Tick(f, b)

	if b.Scratch[0] != 77 {
		t.Errorf("pre S0 = %d", b.Scratch[0])
	}

	// A burning state 0xc is cleared before the scan (no setter: no panic).
	f, _ = newPet("AssassinSentry", 5)
	f.states[0xc] = true
	f.hasTarget, f.attackOK, f.dist = true, true, 10
	b = f4Brain("AssassinSentry", 1, []int{0}, 100, 7, 9, 20)
	Tick(f, b)

	if f.last() != "cast0" {
		t.Errorf("state 0xc: %v", f.log)
	}
}

func TestCycleOfLife(t *testing.T) {
	ally := &Target{ID: 20, X: 104, Y: 100, Size: 1}

	// aip1 cast gap 50, aip2 reach 20, aip3 idle 12, aip4 step 4, aip5 leash 40
	aip := []int{50, 20, 12, 4, 40}

	cases := []struct {
		name      string
		class     int
		ownerDist int
		noOwner   bool
		cand      *Target
		cur, max  int
		frame     int
		last      int // S1
		hasTgt    bool
		log       string
		wake      int
		s1        int
		teleports int
	}{
		{name: "no leader waits", noOwner: true, ownerDist: 5, wake: 25},
		{name: "far from leader teleports", ownerDist: 50, wake: 5, teleports: 1},
		{name: "heals a hurt owner", class: cycleClassLife, ownerDist: 5, cand: ally, cur: 10, max: 100,
			frame: 100, log: "cast0", s1: 100},
		{name: "mana gate heals", class: cycleClassMana, ownerDist: 5, cand: ally, cur: 1, max: 5,
			frame: 100, log: "cast0", s1: 100},
		{name: "cast gap not elapsed walks to ally", class: cycleClassLife, ownerDist: 5, cand: ally, cur: 10, max: 100,
			frame: 100, last: 60, log: "walk-target/7", s1: 60},
		{name: "full life skips the cast", class: cycleClassLife, ownerDist: 5, cand: ally, cur: 100, max: 100,
			frame: 100, log: "walk-target/7"},
		{name: "other classes ignore vitals", class: 0x1a0, ownerDist: 5, cand: ally, cur: 100, max: 100,
			frame: 100, log: "cast0", s1: 100},
		{name: "nothing to do waits aip3", ownerDist: 5, frame: 10, wake: 22},
	}

	for _, c := range cases {
		f, _ := newPet("CycleOfLife", c.ownerDist)
		f.noOwner = c.noOwner
		f.hasTarget = c.hasTgt
		f.frame = c.frame
		f.inRange = true
		w := &f4World{fakePet: f, cand: c.cand, vitalsCur: c.cur, vitalsMax: c.max, auraRange: 12}

		b := f4Brain("CycleOfLife", c.class, []int{0}, aip...)
		b.Scratch[1] = c.last

		Tick(w, b)

		if c.log != "" && f.last() != c.log || c.log == "" && len(f.log) != 0 {
			t.Errorf("%s: log %v", c.name, f.log)
		}

		if c.log == "" && b.Wake != c.wake {
			t.Errorf("%s: wake %d want %d", c.name, b.Wake, c.wake)
		}

		if b.Scratch[1] != c.s1 {
			t.Errorf("%s: S1 %d want %d", c.name, b.Scratch[1], c.s1)
		}

		if f.teleports != c.teleports {
			t.Errorf("%s: teleports %d", c.name, f.teleports)
		}
	}

	// The aura range from the skill calc is clamped to 5..50 before the search.
	for _, r := range [][2]int{{0, 5}, {5, 5}, {6, 6}, {49, 49}, {50, 50}, {51, 50}, {500, 50}} {
		f, _ := newPet("CycleOfLife", 5)
		w := &f4World{fakePet: f, auraRange: r[0]}
		b := f4Brain("CycleOfLife", 0, []int{0}, aip...)

		Tick(w, b)

		if w.auraAsked != r[1] {
			t.Errorf("aura %d searched %d want %d", r[0], w.auraAsked, r[1])
		}
	}

	// A dead player target is dropped: no walk-away roll even when in range.
	f, _ := newPet("CycleOfLife", 5)
	f.hasTarget = true
	w := &f4World{fakePet: f, targetDead: true}
	b := f4Brain("CycleOfLife", 0, []int{0}, aip...)
	before := *b.Seed

	Tick(w, b)

	if *b.Seed != before || b.Wake != 12 {
		t.Errorf("dead target: wake=%d", b.Wake)
	}

	// Pre-hook: S0 = 0.
	b = f4Brain("CycleOfLife", 0, nil, aip...)
	b.Scratch[0] = 9
	f, _ = newPet("CycleOfLife", 5)
	Tick(f, b)

	if b.Scratch[0] != 0 {
		t.Errorf("pre S0 = %d", b.Scratch[0])
	}
}

// jjWorld is an NPC host plus the Arcane Sanctuary quest node.
type jjWorld struct {
	*npcHostWorld
	idle2, two bool
	coords     Point
	has        bool
}

func (w *jjWorld) ArcaneNpcState(*Brain) (bool, bool, Point, bool) {
	return w.idle2, w.two, w.coords, w.has
}

func newJarJar(ax, ay int) (*jjWorld, *Brain) {
	fw := newFake(30, false)
	fw.hasTarget = false
	w := &jjWorld{npcHostWorld: &npcHostWorld{fakeWorld: fw}, idle2: true}
	b := brainAt(profile("JarJar"))
	def, _ := Lookup("JarJar")
	b.SetAI(def)
	b.AppendCommand(Command{Type: CmdAnchor, X: ax, Y: ay})

	return w, b
}

func TestJarJar(t *testing.T) {
	cases := []struct {
		name  string
		setup func(w *jjWorld, b *Brain)
		log   string
		wake  int
		delay int // anchor Delay afterwards
	}{
		{name: "idle at home waits 120", setup: func(*jjWorld, *Brain) {}, wake: 120},
		{name: "drifted past 7 walks back", setup: func(_ *jjWorld, b *Brain) { b.Y = 90 },
			log: "walk-to(100,100)"},
		{name: "7 away still idle", setup: func(_ *jjWorld, b *Brain) { b.Y = 93 }, wake: 120},
		{name: "state two tightens to 2", setup: func(w *jjWorld, b *Brain) { w.two = true; b.Y = 97 },
			log: "walk-to(100,100)"},
		{name: "state two within 2 idles", setup: func(w *jjWorld, b *Brain) { w.two = true; b.Y = 98 }, wake: 120},
		{name: "quest active walks to anchor", setup: func(w *jjWorld, b *Brain) {
			w.idle2 = false
			b.Y = 110
			w.frame = 1000
		}, log: "walk-to(100,103)", delay: 1000},
		{name: "quest active throttles to 20", setup: func(w *jjWorld, b *Brain) {
			w.idle2 = false
			b.Y = 110
			w.frame = 1050
			b.FindCommand(CmdAnchor).Delay = 1000
		}, wake: 1070, delay: 1000},
		{name: "quest point is used, delay refreshed", setup: func(w *jjWorld, b *Brain) {
			w.idle2, w.has, w.coords = false, true, Point{140, 140}
			w.frame = 500
		}, log: "walk-to(140,140)", delay: 500},
		{name: "at the quest point waits 20", setup: func(w *jjWorld, b *Brain) {
			w.idle2, w.has, w.coords = false, true, Point{100, 101}
			w.frame = 500
		}, wake: 520},
		{name: "quest active at anchor, nobody near", setup: func(w *jjWorld, b *Brain) {
			w.idle2 = false
			b.Y = 103
			w.frame = 500
		}, wake: 520},
	}

	for _, c := range cases {
		w, b := newJarJar(100, 103)
		b.Y = 100
		c.setup(w, b)

		Tick(w, b)

		if c.log != "" && w.last() != c.log || c.log == "" && len(w.log) != 0 {
			t.Errorf("%s: log %v", c.name, w.log)
		}

		if c.log == "" && b.Wake != c.wake {
			t.Errorf("%s: wake %d want %d", c.name, b.Wake, c.wake)
		}

		if got := b.FindCommand(CmdAnchor).Delay; got != c.delay {
			t.Errorf("%s: anchor delay %d want %d", c.name, got, c.delay)
		}
	}

	// First think only records the anchor and waits 20.
	fw := newFake(30, false)
	fw.hasTarget = false
	b := brainAt(profile("JarJar"))
	def, _ := Lookup("JarJar")
	b.SetAI(def)
	Tick(fw, b)

	if n := b.FindCommand(CmdAnchor); n == nil || n.X != 100 || n.Y != 100 || b.Wake != 20 {
		t.Errorf("anchor: %+v wake=%d", n, b.Wake)
	}

	// With a player in the room the idle branch hands over to the NPC reaction
	// (face the player once, wait 20), which then overrides with the long wait.
	w, b := newJarJar(100, 103)
	w.havePlayer = true
	w.player = Target{ID: 1, X: 130, Y: 100, Size: 1, IsPlayer: true}
	Tick(w, b)

	if w.faced != 1 || b.Wake != 120 {
		t.Errorf("react: faced=%d wake=%d", w.faced, b.Wake)
	}
}
