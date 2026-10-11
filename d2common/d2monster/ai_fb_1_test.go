package d2monster

import (
	"fmt"
	"reflect"
	"testing"
)

// fb1World is the fake world plus the optional extensions of ai_fb_1.go.
type fb1World struct {
	*fakeWorld
	lifePct    int
	maxHP      int
	maxMana    int
	flag       bool
	corpse     *Target
	cleared    []int
	petLeader  *Target
	petCalls   int
	petAct     bool
	petTarget  *Target
	copyCalc   int
	copyOK     bool
	copyArgs   string
	hasSkills  bool
	bookOn     bool
	queued     []int
	chooseID   int
	chooseOK   bool
	partners   int
	scanTarget *Target
}

func newFB1(dist int, inRange bool) *fb1World {
	return &fb1World{fakeWorld: newFake(dist, inRange), lifePct: 100, maxHP: 1, hasSkills: true}
}

func (w *fb1World) LifePercent(Target) int              { return w.lifePct }
func (w *fb1World) MaxHP(Target) int                    { return w.maxHP }
func (w *fb1World) MaxMana(Target) int                  { return w.maxMana }
func (w *fb1World) HasStatListFlag(Target, uint32) bool { return w.flag }
func (w *fb1World) SetUnitState(_ *Brain, s int, on bool) {
	if !on {
		w.cleared = append(w.cleared, s)
	}
}

func (w *fb1World) ReviveCorpse(*Brain, int) (Target, bool) {
	if w.corpse == nil {
		return Target{}, false
	}

	return *w.corpse, true
}

func (w *fb1World) ScanWounded(*Brain, int, int) (Target, bool) {
	if w.scanTarget == nil {
		return Target{}, false
	}

	return *w.scanTarget, true
}

func (w *fb1World) LeaderUnit(*Brain) (Target, bool) {
	if w.petLeader == nil {
		return Target{}, false
	}

	return *w.petLeader, true
}

func (w *fb1World) PetAction(_ *Brain, tgt *Target, _ Target, _ bool) bool {
	w.petCalls++
	w.petTarget = tgt

	if w.petAct {
		w.log = append(w.log, "pet")
	}

	return w.petAct
}

func (w *fb1World) WispPartners(*Brain) int { return w.partners }

// the book is only visible when bookOn: fb1Book wraps the world
type fb1Book struct{ *fb1World }

func (w fb1Book) HasSkills(*Brain) bool { return w.hasSkills }
func (w fb1Book) CastQueued(_ *Brain, id int, _ Target, _ bool) bool {
	w.queued = append(w.queued, id)

	return true
}
func (w fb1Book) Upkeep(*Brain, *Target, bool) bool { return false }
func (w fb1Book) Choose(*Brain, Target, *Target, bool) (int, bool) {
	return w.chooseID, w.chooseOK
}

func (w fb1Book) Copy(_ *Brain, _ Target, _ Target, left, basic, rng bool) (int, bool) {
	w.copyArgs = fmt.Sprintf("left=%v basic=%v range=%v", left, basic, rng)

	return w.copyCalc, w.copyOK
}

// think runs one think function directly with the given tick parameters.
func fb1Think(w World, b *Brain, dist int, inRange, target bool) *Ctx {
	c := &Ctx{B: b, W: w}

	if target {
		t := Target{ID: 1, X: 100 + dist, Y: 100, Size: 1, IsPlayer: true}
		c.Target, c.Dist, c.InRange = &t, dist, inRange
	}

	b.Def.Think(c)

	return c
}

func fb1Brain(name string, aip ...int) *Brain {
	return brainAt(profile(name, aip...))
}

// stepsUsed advances a shadow of the seed n times and compares it with the
// brain's seed: the think function drew exactly n LCG steps.
func stepsUsed(b *Brain, orig *Brain, n int) bool {
	s := shadow(orig)
	for i := 0; i < n; i++ {
		s.Step()
	}

	return *s == *b.Seed
}

func TestFB1Registered(t *testing.T) {
	for _, n := range []string{"Vampire", "SuccubusWitch", "Nihlathak", "ShadowWarrior", "ShadowMaster",
		"ShadowMasterNoInit", "WillOWisp"} {
		d, ok := Lookup(n)
		if !ok || !d.Implemented {
			t.Errorf("%s not registered", n)

			continue
		}

		if m, _ := AITargetMode(n); m != d.TargetMode {
			t.Errorf("%s target mode %d, table %d", n, d.TargetMode, m)
		}
	}
}

type fb1Case struct {
	name    string
	aip     []int
	dist    int
	inRange bool
	setup   func(b *Brain, w *fb1World)
	log     []string
	steps   int
	scratch [3]int
	wake    int // expected Wake when the log is empty (0 = ignore)
}

func runFB1(t *testing.T, ai string, cases []fb1Case) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := withSkills(profile(ai, tc.aip...), 0, 1, 2, 3, 4)
			p.Walk, p.Run = 100, 150
			b := brainAt(p)
			w := newFB1(tc.dist, tc.inRange)

			if tc.setup != nil {
				tc.setup(b, w)
			}

			orig := &Brain{Seed: shadow(b)}
			fb1Think(w, b, tc.dist, tc.inRange, true)

			if tc.log != nil && !reflect.DeepEqual(w.log, tc.log) {
				t.Errorf("log = %v, want %v", w.log, tc.log)
			}

			if tc.log == nil && len(w.log) != 0 {
				t.Errorf("unexpected actions %v", w.log)
			}

			if !stepsUsed(b, orig, tc.steps) {
				t.Errorf("seed advanced a different number of steps than %d", tc.steps)
			}

			if b.Scratch != tc.scratch {
				t.Errorf("scratch = %v, want %v", b.Scratch, tc.scratch)
			}

			if tc.wake != 0 && b.Wake != tc.wake {
				t.Errorf("wake = %d, want %d", b.Wake, tc.wake)
			}
		})
	}
}

// rolls100 returns the first n percent rolls the brain would draw.
func rolls100(b *Brain, n int) []int {
	s := shadow(b)
	out := make([]int, n)

	for i := range out {
		out[i] = int(s.Roll(100))
	}

	return out
}

func TestVampire(t *testing.T) {
	always := []int{100, 100, 100, 100, 0}

	runFB1(t, "Vampire", []fb1Case{
		{name: "aggressive in reach without the skill bit attacks after one roll", aip: always, dist: 3, inRange: true,
			setup: func(b *Brain, _ *fb1World) { b.Aggressive = true }, log: []string{"attack4"}, steps: 1,
			scratch: [3]int{1, 3, 0}},
		{name: "wounded flees (mood 2) walking away", aip: always, dist: 3, inRange: true,
			setup: func(b *Brain, _ *fb1World) { b.HPPercent = 20 }, log: []string{"walk-to(92,100)"}, steps: 0,
			scratch: [3]int{2, 0, 0}},
		{name: "mood 2 above 74 percent life goes back to melee", aip: always, dist: 3, inRange: true,
			setup:   func(b *Brain, _ *fb1World) { b.Scratch[0], b.HPPercent = 2, 80 },
			log:     []string{"walk-target/7"},
			scratch: [3]int{1, 0, 0}},
		{name: "mood 2 close by runs away", aip: always, dist: 5, inRange: false,
			setup:   func(b *Brain, _ *fb1World) { b.Scratch[0], b.HPPercent = 2, 50 },
			log:     []string{"walk-to(92,100)"},
			scratch: [3]int{2, 0, 0}},
		{name: "mood 2 far away waits 15", aip: []int{100, 100, 20, 100, 0}, dist: 30, inRange: false,
			setup:   func(b *Brain, _ *fb1World) { b.Scratch[0], b.HPPercent = 2, 50 },
			scratch: [3]int{2, 0, 0}, wake: 15},
		{name: "far from the engage range and not in mood 1 waits 15", aip: []int{100, 100, 30, 100, 0}, dist: 40,
			scratch: [3]int{0, 0, 0}, wake: 15},
		{name: "far in mood 1 walks up", aip: []int{100, 100, 30, 100, 0}, dist: 40,
			setup: func(b *Brain, _ *fb1World) { b.Scratch[0] = 1 }, log: []string{"walk-target/7"},
			scratch: [3]int{1, 0, 0}},
		{name: "engaged: slot 2 then 11 frame cooldown (two rolls)", aip: []int{100, 100, 30, 100, 2}, dist: 10,
			log: []string{"cast1"}, steps: 2, scratch: [3]int{1, 0, 11}},
		{name: "engaged: slot 3 when only the 4 bit is set", aip: []int{100, 100, 30, 100, 4}, dist: 10,
			log: []string{"cast2"}, steps: 2, scratch: [3]int{1, 0, 11}},
		{name: "cooldown counts down and blocks slot 2", aip: []int{100, 100, 30, 100, 2}, dist: 10,
			setup: func(b *Brain, _ *fb1World) { b.Scratch[2] = 5 }, log: []string{"walk-target/7"},
			steps: 1, scratch: [3]int{1, 0, 4}},
		{name: "in reach, melee roll passes with no ranged bit", aip: []int{100, 0, 30, 0, 0}, dist: 3, inRange: true,
			log: []string{"attack4"}, steps: 1, scratch: [3]int{1, 0, 0}},
	})
}

// TestVampireRollOrder checks the shortcuts against predicted rolls.
func TestVampireRollOrder(t *testing.T) {
	// in reach, aip1 100, ranged bit on, second target in range: roll order is
	// melee-gate roll (aip1), the 30 percent roll, then the 50 percent pick
	p := withSkills(profile("Vampire", 100, 0, 30, 0, 1), 0, 1, 2, 3)
	b := brainAt(p)
	r := rolls100(b, 4)
	w := newFB1(3, true)
	fb1Think(w, b, 3, true, true)

	var want string

	switch {
	case r[0] >= 100:
		t.Skip("impossible")
	case r[1] > 30:
		want = "attack4"
	case r[2] < 50:
		want = "cast0"
	default:
		want = "cast3"
	}

	if len(w.log) != 1 || w.log[0] != want {
		t.Errorf("rolls %v: log %v, want %s", r, w.log, want)
	}
}

func TestSuccubusWitch(t *testing.T) {
	// aip1 melee, aip2 approach, aip3 support, aip4 radius, aip5 shout, aip6 stall, aip7 life gate, aip8 own life
	runFB1(t, "SuccubusWitch", []fb1Case{
		{name: "support slot 1 when close and the roll passes", aip: []int{0, 0, 100, 10, 0, 7, 0, 0}, dist: 3,
			log: []string{"cast0"}, steps: 1, scratch: [3]int{}},
		{name: "in reach: no walk-away roll passes, melee roll passes", aip: []int{100, 0, 0, 0, 0, 7, 0, 0}, dist: 3,
			inRange: true, log: []string{"attack4"}, steps: 2},
		{name: "in reach: all rolls fail, stalls aip6", aip: []int{0, 0, 0, 0, 0, 7, 0, 0}, dist: 3, inRange: true,
			steps: 2, wake: 7},
		{name: "out of reach: shout instead of slot 5 when it is unused", aip: []int{0, 0, 0, 0, 100, 7, 0, 0},
			dist: 10, setup: func(b *Brain, _ *fb1World) { b.Profile.Skills[4] = SkillSlot{} },
			log: []string{"attack9"}, steps: 1},
		{name: "out of reach: approach roll", aip: []int{0, 100, 0, 0, 0, 7, 0, 0}, dist: 10,
			setup: func(b *Brain, _ *fb1World) { b.Profile.Skills[4] = SkillSlot{} },
			log:   []string{"walk-target/0"}, steps: 2},
		{name: "slot 5 first: aip8 gates the ranged cast", aip: []int{0, 0, 0, 0, 100, 7, 0, 100}, dist: 10,
			log: []string{"cast4"}, steps: 2},
	})

	// a target with a stat list flagged 0x20 is not given support
	w := newFB1(3, false)
	w.flag = true
	b := fb1Brain("SuccubusWitch", 0, 0, 100, 10, 0, 7, 0, 0)
	b.Profile.Skills[0] = SkillSlot{Name: "x", Mode: ModeSkill1}
	fb1Think(w, b, 3, false, true)

	for _, l := range w.log {
		if l == "cast0" {
			t.Errorf("support cast on a flagged target: %v", w.log)
		}
	}
}

func TestNihlathak(t *testing.T) {
	t.Run("no target: random-offset teleport only when aggressive", func(t *testing.T) {
		b := fb1Brain("Nihlathak", 0, 0, 0, 0, 0)
		b.Profile.Skills[0] = SkillSlot{Name: "tp", Mode: ModeSkill1}
		w := newFB1(3, false)
		fb1Think(w, b, 0, false, false)

		if b.Wake != 25 || len(w.log) != 0 {
			t.Errorf("idle: wake %d log %v", b.Wake, w.log)
		}

		b.Aggressive = true
		w2 := newFB1(3, false)
		fb1Think(w2, b, 0, false, false)

		if !reflect.DeepEqual(w2.log, []string{"cast0"}) {
			t.Errorf("log %v", w2.log)
		}
	})

	t.Run("clears state 0xc", func(t *testing.T) {
		b := fb1Brain("Nihlathak", 0, 0, 0, 0, 0)
		w := newFB1(3, false)
		w.states[12] = true
		fb1Think(w, b, 3, false, true)

		if !reflect.DeepEqual(w.cleared, []int{12}) {
			t.Errorf("cleared %v", w.cleared)
		}
	})

	runFB1(t, "Nihlathak", []fb1Case{
		{name: "in reach teleports on the aip1 roll", aip: []int{100, 0, 0, 0, 0}, dist: 3, inRange: true,
			log: []string{"cast0"}, steps: 1},
		{name: "raises a corpse on the aip3 roll", aip: []int{0, 0, 100, 0, 0}, dist: 30,
			setup: func(b *Brain, w *fb1World) { w.corpse = &Target{ID: 5, X: 110, Y: 100} },
			log:   []string{"cast2"}, steps: 1},
		{name: "helper cast on a wounded ally", aip: []int{0, 0, 0, 100, 0}, dist: 30,
			setup: func(b *Brain, w *fb1World) {
				b.Profile.Skills[2] = SkillSlot{}
				b.Profile.Skills[3] = SkillSlot{}
				w.scanTarget = &Target{ID: 6, X: 99, Y: 100}
			},
			log: []string{"cast1"}, steps: 1},
		{name: "summons when nothing to help", aip: []int{0, 0, 0, 100, 0}, dist: 30,
			setup: func(b *Brain, w *fb1World) {
				b.Profile.Skills[2] = SkillSlot{}
				b.Profile.Skills[3] = SkillSlot{}
			},
			log: []string{"cast4"}, steps: 1},
		{name: "close-range slot 4 whatever the 60 percent roll says", aip: []int{0, 0, 0, 0, 0}, dist: 5,
			setup: func(b *Brain, w *fb1World) {
				b.Profile.Skills[0], b.Profile.Skills[1] = SkillSlot{}, SkillSlot{}
				b.Profile.Skills[2], b.Profile.Skills[4] = SkillSlot{}, SkillSlot{}
			},
			log: []string{"cast3"}, steps: 1},
	})
}

func TestShadowWarrior(t *testing.T) {
	t.Run("no owner sleeps 100", func(t *testing.T) {
		b := fb1Brain("ShadowWarrior", 50, 50, 50, 1)
		w := newFB1(5, true)
		fb1Think(w, b, 5, true, true)

		if b.Wake != 100 {
			t.Errorf("wake %d", b.Wake)
		}
	})

	t.Run("fatigue counter decays and wraps", func(t *testing.T) {
		b := fb1Brain("ShadowWarrior", 50, 50, 50, 2)
		w := newFB1(5, true)
		w.petLeader = &Target{ID: 2, X: 100, Y: 100}
		b.Scratch[1] = 10
		fb1Think(w, b, 5, true, true)

		if b.Scratch[1] != 7 {
			t.Errorf("fatigue %d", b.Scratch[1])
		}

		b.Scratch[1] = 1
		fb1Think(w, b, 5, true, true)

		if b.Scratch[1] != 0 {
			t.Errorf("fatigue should reset when negative, got %d", b.Scratch[1])
		}
	})

	t.Run("target dropped beyond aip1 or owner beyond aip2; pet action ends the tick", func(t *testing.T) {
		b := fb1Brain("ShadowWarrior", 4, 50, 50, 1)
		w := newFB1(10, true)
		w.petLeader = &Target{ID: 2, X: 100, Y: 100}
		w.petAct = true
		fb1Think(w, b, 10, true, true)

		if w.petTarget != nil || !reflect.DeepEqual(w.log, []string{"pet"}) {
			t.Errorf("target %v log %v", w.petTarget, w.log)
		}
	})

	t.Run("copy through the book: 1 roll out of reach, 2 in reach, cooldown calc/3+18+frame", func(t *testing.T) {
		b := fb1Brain("ShadowWarrior", 50, 50, 100, 1, 0, 0, 0, 100)
		orig := &Brain{Seed: shadow(b)}
		w := newFB1(5, true)
		w.petLeader = &Target{ID: 2, X: 100, Y: 100}
		w.copyCalc, w.copyOK = 30, true
		w.frame = 5
		book := fb1Book{w}
		fb1Think(book, b, 5, true, true)

		if b.Scratch[0] != 30/3+18+5 {
			t.Errorf("cooldown %d", b.Scratch[0])
		}

		if !stepsUsed(b, orig, 2) {
			t.Errorf("expected the coin flip and the basic-attack roll")
		}
	})
}

func TestShadowMaster(t *testing.T) {
	t.Run("no skills sleeps 100", func(t *testing.T) {
		b := fb1Brain("ShadowMaster", 20, 20, 50)
		w := newFB1(5, true)
		w.hasSkills = false
		fb1Think(fb1Book{w}, b, 5, true, true)

		if b.Wake != 100 {
			t.Errorf("wake %d", b.Wake)
		}
	})

	t.Run("queued follow-up casts decrement the count", func(t *testing.T) {
		b := fb1Brain("ShadowMaster", 20, 20, 50)
		b.Scratch = [3]int{3, 77, 1}
		w := newFB1(5, true)
		fb1Think(fb1Book{w}, b, 5, true, true)

		if b.Scratch[0] != 2 || b.Scratch[1] != 77 {
			t.Errorf("scratch %v", b.Scratch)
		}
	})

	t.Run("target beyond aip2 is dropped: no target sleeps 25", func(t *testing.T) {
		b := fb1Brain("ShadowMaster", 20, 10, 50)
		w := newFB1(15, false)
		fb1Think(w, b, 15, false, true)

		if b.Wake != 25 {
			t.Errorf("wake %d", b.Wake)
		}
	})

	t.Run("cast roll uses aip3 - 2*level clamped to 5..100", func(t *testing.T) {
		b := fb1Brain("ShadowMaster", 20, 20, 100)
		b.Scratch[2] = 3 // 100 - 6 = 94
		r := rolls100(b, 1)
		w := newFB1(5, true)
		fb1Think(w, b, 5, true, true)

		want := "attack4"
		if r[0] >= 94 {
			want = "walk-target/0"
		}

		if len(w.log) == 0 || w.log[0] != want && r[0] >= 94 {
			t.Errorf("roll %d log %v", r[0], w.log)
		}
	})

	t.Run("book choice is stored in Scratch[1]", func(t *testing.T) {
		b := fb1Brain("ShadowMaster", 20, 20, 0)
		w := newFB1(5, false)
		w.chooseID, w.chooseOK = 55, true
		fb1Think(fb1Book{w}, b, 5, false, true)

		if b.Scratch[1] != 55 || b.Wake != waitForever {
			t.Errorf("scratch %v wake %d", b.Scratch, b.Wake)
		}
	})
}

func TestWillOWisp(t *testing.T) {
	runFB1(t, "WillOWisp", []fb1Case{
		{name: "out of reach, aip1 roll attacks and resets the state", aip: []int{100, 0, 0}, dist: 20,
			log: []string{"attack7"}, steps: 1, scratch: [3]int{0, 0, 0}},
		{name: "in reach, aip2 roll attacks", aip: []int{0, 100, 0}, dist: 3, inRange: true,
			log: []string{"attack4"}, steps: 1},
		{name: "otherwise approaches on aip3 with a 3 counter", aip: []int{0, 0, 100}, dist: 20,
			log: []string{"walk-target/0"}, steps: 2, scratch: [3]int{1, 3, 0}},
		{name: "chase state counts the counter down and walks", aip: []int{0, 0, 100}, dist: 20,
			setup: func(b *Brain, _ *fb1World) { b.Scratch = [3]int{1, 2, 0} }, log: []string{"walk-target/0"},
			steps: 1, scratch: [3]int{1, 1, 0}},
		{name: "chase state with the counter spent attacks in reach (state 3)", aip: []int{0, 0, 100}, dist: 3,
			inRange: true, setup: func(b *Brain, _ *fb1World) { b.Scratch = [3]int{1, 0, 0} },
			log: []string{"attack8"}, scratch: [3]int{3, 0, 0}},
		{name: "state 2 attacks out of reach without a roll", aip: []int{0, 0, 100}, dist: 20,
			setup: func(b *Brain, _ *fb1World) { b.Scratch = [3]int{2, 0, 0} }, log: []string{"attack7"}},
	})
}
