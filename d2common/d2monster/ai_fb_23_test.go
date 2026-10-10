package d2monster

import (
	"strings"
	"testing"
)

// fbWorld adds the FB extension answers the batch 2/3 tests need.
type fbWorld struct {
	*fakeWorld
	scan     FBXScanResult
	flags    []uint32
	killed   int
	leader   *OwnerInfo
	cellFree bool
}

func (w *fbWorld) FBXScan(*Brain, FBXScanQuery) FBXScanResult { return w.scan }
func (w *fbWorld) FBXSetFlags(_ *Brain, set, _ uint32)        { w.flags = append(w.flags, set) }
func (w *fbWorld) FBXKill(*Brain, *Target)                    { w.killed++ }
func (w *fbWorld) FBXSpawnCellsFree(*Brain) bool              { return w.cellFree }

func (w *fbWorld) Owner(*Brain) (OwnerInfo, bool) {
	if w.leader == nil {
		return OwnerInfo{}, false
	}

	return *w.leader, true
}

func fbRun(name string, aip []int, dist int, inRange bool, setup func(b *Brain, w *fbWorld)) (*Brain, *fbWorld) {
	w := &fbWorld{fakeWorld: newFake(dist, inRange), cellFree: true}
	p := withSkills(profile(name, aip...), 0, 1, 2, 3)
	b := brainAt(p)
	b.Def, _ = Lookup(name)

	if setup != nil {
		setup(b, w)
	}

	Tick(w, b)

	return b, w
}

func TestFaithfulBSimple(t *testing.T) {
	cases := []struct {
		name    string
		ai      string
		aip     []int
		dist    int
		inRange bool
		setup   func(b *Brain, w *fbWorld)
		want    string // last action, "" = sleep/none
		wake    int    // expected Wake when want == ""
		scratch [3]int
	}{
		{name: "VileDog first think sleeps 5", ai: "VileDog", aip: []int{100, 9, 100}, dist: 10, wake: 5, scratch: [3]int{1}},
		{name: "VileDog bites", ai: "VileDog", aip: []int{100, 9, 0}, dist: 3, inRange: true,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[0] = 1 }, want: "attack4", scratch: [3]int{1}},
		{name: "VileDog stalls", ai: "VileDog", aip: []int{0, 9, 0}, dist: 3, inRange: true,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[0] = 1 }, wake: 9, scratch: [3]int{1}},
		{name: "VileDog approaches", ai: "VileDog", aip: []int{0, 9, 100}, dist: 10,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[0] = 1 }, want: "walk-target/7", scratch: [3]int{1}},
		{name: "VileDog idles", ai: "VileDog", aip: []int{0, 9, 0}, dist: 10,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[0] = 1 }, wake: 10, scratch: [3]int{1}},
		{name: "PinHead stalls out of range", ai: "PinHead", aip: []int{0, 0, 0, 14}, dist: 10, wake: 14},
		{name: "PinHead walks", ai: "PinHead", aip: []int{0, 0, 101, 14}, dist: 10, want: "walk-target/7"},
		{name: "PinHead skill1", ai: "PinHead", aip: []int{0, 0, 0, 0, 100, 0}, dist: 3, inRange: true,
			want: "cast0", scratch: [3]int{1}},
		{name: "PinHead skill2", ai: "PinHead", aip: []int{0, 0, 0, 0, 0, 100}, dist: 3, inRange: true,
			want: "cast1", scratch: [3]int{1}},
		{name: "PinHead melee", ai: "PinHead", aip: []int{0, 0, 0, 0, 0, 0}, dist: 3, inRange: true,
			want: "attack4", scratch: [3]int{1}},
		{name: "QuillMother engaged walks", ai: "QuillMother", aip: []int{0, 0, 0, 0}, dist: 10,
			setup: func(b *Brain, _ *fbWorld) { b.Aggressive = true }, want: "walk-target/7"},
		{name: "QuillMother engaged bites", ai: "QuillMother", aip: []int{0, 0, 0, 0}, dist: 3, inRange: true,
			setup: func(b *Brain, _ *fbWorld) { b.Aggressive = true }, want: "attack4"},
		{name: "QuillMother stalls in range", ai: "QuillMother", aip: []int{0, 0, 7, 0}, dist: 3, inRange: true, wake: 7},
		{name: "QuillMother bites in range", ai: "QuillMother", aip: []int{101, 0, 7, 0}, dist: 3, inRange: true,
			want: "attack4"},
		{name: "Spirit attacks once", ai: "Spirit", aip: nil, dist: 3, inRange: true, want: "attack4", scratch: [3]int{1}},
		{name: "Spirit then waits 50", ai: "Spirit", aip: nil, dist: 3, inRange: true,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[0] = 1 }, wake: 50, scratch: [3]int{1}},
		{name: "Spirit out of range waits 10", ai: "Spirit", aip: nil, dist: 10, wake: 10},
		{name: "TrapMelee out of range", ai: "Trap-Melee", aip: []int{100, 9}, dist: 10, wake: 40},
		{name: "TrapMelee fires", ai: "Trap-Melee", aip: []int{100, 9}, dist: 3, inRange: true, want: "attack4"},
		{name: "TrapMelee pauses", ai: "Trap-Melee", aip: []int{0, 9}, dist: 3, inRange: true, wake: 9},
		{name: "TrapMissile shoots", ai: "Trap-Missile", aip: []int{20, 3, 11}, dist: 10, want: "attack4", scratch: [3]int{1, 1}},
		{name: "TrapMissile pauses", ai: "Trap-Missile", aip: []int{20, 3, 11}, dist: 10,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[1] = 1 }, wake: 11, scratch: [3]int{0, 0}},
		{name: "TrapNova casts", ai: "Trap-Nova", aip: []int{20, 3, 11}, dist: 10, want: "cast0", scratch: [3]int{1, 1}},
		{name: "TrapPoison spent", ai: "Trap-Poison", aip: []int{20, 1, 11}, dist: 10,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[0] = 1 }, want: "attack0"},
		{name: "SandMaggotQueen spawns animation", ai: "SandMaggotQueen", aip: []int{2, 6}, dist: 10, want: "attack8", scratch: [3]int{0, 1}},
		{name: "SandMaggotQueen cooldown", ai: "SandMaggotQueen", aip: []int{2, 6}, dist: 10,
			setup: func(b *Brain, _ *fbWorld) { b.Scratch[2] = 1 }, wake: 150, scratch: [3]int{}},
		{name: "SiegeBeast melee", ai: "SiegeBeast", aip: []int{0, 100, 0, 5}, dist: 3, inRange: true, want: "attack4"},
		{name: "SiegeBeast stall", ai: "SiegeBeast", aip: []int{0, 0, 0, 5}, dist: 3, inRange: true, wake: 5},
		{name: "SiegeTower sleeps aip1", ai: "SiegeTower", aip: []int{33}, dist: 10, wake: 33},
		{name: "ReanimatedHorde melee", ai: "ReanimatedHorde", aip: []int{100, 4}, dist: 3, inRange: true, want: "attack4"},
		{name: "ReanimatedHorde walks", ai: "ReanimatedHorde", aip: []int{0, 4, 0, 0, 100}, dist: 10, want: "walk-target/0"},
		{name: "ReanimatedHorde stalls", ai: "ReanimatedHorde", aip: []int{0, 4, 0, 0, 0, 0, 9}, dist: 10, wake: 9},
		{name: "Regurgitator bites", ai: "Regurgitator", aip: []int{100}, dist: 3, inRange: true, want: "attack4"},
		{name: "Regurgitator rests", ai: "Regurgitator", aip: []int{0}, dist: 3, inRange: true, wake: 15},
		{name: "Regurgitator walks", ai: "Regurgitator", aip: []int{0, 0, 100}, dist: 10, want: "walk-target/0"},
		{name: "ThornHulk walks", ai: "ThornHulk", aip: []int{0}, dist: 10, want: "walk-target/7"},
		{name: "ThornHulk melee A1", ai: "ThornHulk", aip: []int{101, 0, 0, 0}, dist: 3, inRange: true, want: "attack4",
			scratch: [3]int{0, -1}},
		{name: "ZakarumZealot walks", ai: "ZakarumZealot", aip: []int{0, 0, 0, 0}, dist: 10, want: "walk-target/7"},
		{name: "ZakarumZealot A2", ai: "ZakarumZealot", aip: []int{0, 101, 0, 0}, dist: 3, inRange: true, want: "attack5",
			scratch: [3]int{1}},
		{name: "Overseer melee", ai: "Overseer", aip: []int{0, 0, 0, 0, 0, 100, 101}, dist: 3, inRange: true, want: "attack5"},
		{name: "VileMother stalls", ai: "VileMother", aip: []int{0, 0, 0, 0, 0, 0, 0, 8}, dist: 3, inRange: true, wake: 8},
		{name: "OblivionKnight idle", ai: "OblivionKnight", aip: []int{0, 0, 0, 0, 0, 0, 0, 99}, dist: 10, wake: 10},
		{name: "PutridDefiler bites", ai: "PutridDefiler", aip: []int{0}, dist: 3, inRange: true, want: "attack4"},
		{name: "PutridDefiler idles", ai: "PutridDefiler", aip: []int{0, 0}, dist: 10, wake: 25},
		{name: "PutridDefiler backs off", ai: "PutridDefiler", aip: []int{30, 6}, dist: 10, want: "walk-to(94,100)"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, w := fbRun(c.ai, c.aip, c.dist, c.inRange, c.setup)

			got := ""
			if len(w.log) > 0 {
				got = strings.Join(w.log, ",")
			}

			if c.want != "" && got != c.want {
				t.Fatalf("log = %q want %q", got, c.want)
			}

			if c.want == "" && got != "" {
				t.Fatalf("expected a sleep, got actions %q", got)
			}

			if c.want == "" && b.Wake != c.wake {
				t.Fatalf("wake = %d want %d", b.Wake, c.wake)
			}

			if c.scratch != [3]int{} && b.Scratch != c.scratch {
				t.Fatalf("scratch = %v want %v", b.Scratch, c.scratch)
			}
		})
	}
}

// TestFaithfulBRollOrder checks the exact LCG draws of several ports against a
// shadow generator.
func TestFaithfulBRollOrder(t *testing.T) {
	for id := uint32(1); id < 30; id++ {
		w := &fbWorld{fakeWorld: newFake(3, true), cellFree: true}
		b := NewBrain(id, 1, Normal, withSkills(profile("Regurgitator", 50), 0), testSeed)
		b.X, b.Y = 100, 100
		b.Def, _ = Lookup("Regurgitator")
		s := shadow(b)
		r := int(s.Roll(100))

		Tick(w, b)

		wantAttack := r < 50
		if gotAttack := w.last() == "attack4"; gotAttack != wantAttack {
			t.Fatalf("seed %d: roll %d attack=%v log=%v", id, r, gotAttack, w.log)
		}

		// exactly one draw was consumed
		if b.Roll(100) != int(s.Roll(100)) {
			t.Fatalf("seed %d: RNG out of step after one roll", id)
		}
	}
}

func TestSarcophagusAndUberBaal(t *testing.T) {
	// Sarcophagus: the stall is 20 + one Roll(10) draw
	w := &fbWorld{fakeWorld: newFake(3, true), cellFree: false}
	b := brainAt(profile("Sarcophagus", 100, 0, 5))
	b.Def, _ = Lookup("Sarcophagus")
	s := shadow(b)
	want := 20 + int(s.Roll(10))

	Tick(w, b)

	if b.Wake != want {
		t.Fatalf("Sarcophagus wake = %d want %d", b.Wake, want)
	}

	// too many spawned: flag and death mode
	w = &fbWorld{fakeWorld: newFake(3, true)}
	b = brainAt(profile("Sarcophagus", 100, 0, 1))
	b.Def, _ = Lookup("Sarcophagus")
	b.Scratch[1], b.Scratch[2] = 5, 1
	Tick(w, b)

	if w.last() != "attack0" || len(w.flags) != 1 || w.flags[0] != 0x20000 {
		t.Fatalf("Sarcophagus end: log %v flags %v", w.log, w.flags)
	}

	// UberBaal queues nothing and never wakes
	w = &fbWorld{fakeWorld: newFake(3, true)}
	b = brainAt(profile("UberBaal"))
	b.Def, _ = Lookup("UberBaal")
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != waitForever {
		t.Fatalf("UberBaal: log %v wake %d", w.log, b.Wake)
	}
}

func TestTentacleLeader(t *testing.T) {
	// no leader: the tentacle dies
	b, w := fbRun("Tentacle", []int{0, 0, 3, 0, 5, 20}, 10, false, nil)
	if w.killed != 1 || b.Wake != 0 {
		t.Fatalf("leaderless tentacle should die, killed=%d", w.killed)
	}

	// leader alive: first think casts skill1, waits 8, arms the pause
	lead := &OwnerInfo{Target: Target{ID: 99, X: 90, Y: 100}, Mode: ModeNeutral}
	b, w = fbRun("Tentacle", []int{0, 0, 3, 0, 5, 20}, 10, false, func(_ *Brain, w *fbWorld) { w.leader = lead })

	if w.last() != "cast0" || b.Scratch[2] != 1 || b.Scratch[1] != 75 || b.Wake != 8 {
		t.Fatalf("Tentacle start: log %v scratch %v wake %d", w.log, b.Scratch, b.Wake)
	}

	// TentacleHead needs no leader and bites in range when the roll passes
	b, w = fbRun("TentacleHead", []int{100, 0, 3, 4, 5, 20}, 10, true, func(b *Brain, _ *fbWorld) {
		b.Scratch[2] = 2
		b.Scratch[1] = 1000
	})

	if w.last() != "attack4" {
		t.Fatalf("TentacleHead melee: %v (wake %d)", w.log, b.Wake)
	}
}

func TestFrogDemonPhases(t *testing.T) {
	// far target in phase 0: spit and wait 8
	b, w := fbRun("FrogDemon", []int{0, 0, 0, 0, 0, 0, 0, 0}, 20, false, nil)
	if w.last() != "cast0" || b.Scratch[2] != 1 || b.Wake != 8 {
		t.Fatalf("FrogDemon spit: %v scratch %v wake %d", w.log, b.Scratch, b.Wake)
	}

	// surfaced and in range: aip2 gate 101 forces A2
	_, w = fbRun("FrogDemon", []int{0, 101, 0, 0, 0, 0, 0, 0}, 3, true, func(b *Brain, _ *fbWorld) { b.Scratch[2] = 2 })
	if w.last() != "attack5" {
		t.Fatalf("FrogDemon A2: %v", w.log)
	}
}

func TestTrappedSoul(t *testing.T) {
	b, w := fbRun("TrappedSoul", nil, 3, true, nil)
	if w.last() != "attack9" || b.Scratch[0] != 1 || len(w.flags) != 1 {
		t.Fatalf("TrappedSoul shout: %v scratch %v flags %v", w.log, b.Scratch, w.flags)
	}

	// armed, far: waits 15 when it has not shouted yet
	b, w = fbRun("TrappedSoul", nil, 10, false, nil)
	if len(w.log) != 0 || b.Wake != 15 {
		t.Fatalf("TrappedSoul far: %v wake %d", w.log, b.Wake)
	}
}

func TestTrapArrows(t *testing.T) {
	// right-arrow trap: lined up on x within 3 and reload ready -> A1
	_, w := fbRun("Trap-RightArrow", []int{1, 50, 20, 10}, 10, false, func(b *Brain, w *fbWorld) { w.target.X = b.X + 1; w.frame = 100 })
	if w.last() != "attack4" {
		t.Fatalf("right arrow: %v", w.log)
	}

	// not lined up: sleeps 30
	b, w := fbRun("Trap-LeftArrow", []int{1, 50, 20, 10}, 10, false, func(b *Brain, w *fbWorld) { w.target.Y = b.Y + 9; w.frame = 100 })
	if len(w.log) != 0 || b.Wake != 130 {
		t.Fatalf("left arrow: %v wake %d", w.log, b.Wake)
	}
}

func TestFaithfulBRegistered(t *testing.T) {
	for _, n := range FaithfulB() {
		if d, ok := Lookup(n); !ok || !d.Implemented {
			t.Errorf("%s not registered", n)
		}
	}
}
