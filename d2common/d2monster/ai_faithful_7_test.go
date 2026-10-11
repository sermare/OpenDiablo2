package d2monster

import (
	"fmt"
	"strings"
	"testing"
)

// w7 adds the batch-7 host answers to the shared fake.
type w7 struct {
	*fakeWorld
	atDist     int  // distance AttackTarget reports (0 = same as the tick's)
	atOK       bool // AttackTarget finds a unit
	life       int  // target life percent
	maxHP      int
	maxMana    int
	flag20     bool
	auraStates map[int]int
	corpse     *Target
	classes    []int
	stats      map[int]int
	casts      []Target
	cleared    []int
}

func newW7(dist int, inRange bool) *w7 {
	return &w7{fakeWorld: newFake(dist, inRange), life: 100, maxHP: 1, auraStates: map[int]int{}, stats: map[int]int{}}
}

func (w *w7) AttackTarget(b *Brain) (Target, int, bool) {
	if w.atDist != 0 || w.atOK {
		return w.target, w.atDist, w.atOK
	}

	return w.fakeWorld.AttackTarget(b)
}

func (w *w7) Cast(b *Brain, slot int, t Target) bool {
	w.casts = append(w.casts, t)

	return w.fakeWorld.Cast(b, slot, t)
}

func (w *w7) LifePercent(Target) int              { return w.life }
func (w *w7) MaxHP(Target) int                    { return w.maxHP }
func (w *w7) MaxMana(Target) int                  { return w.maxMana }
func (w *w7) HasStatListFlag(Target, uint32) bool { return w.flag20 }

func (w *w7) SlotAuraState(_ *Brain, slot int) int {
	if v, ok := w.auraStates[slot]; ok {
		return v
	}

	return -1
}

func (w *w7) NearestCorpse(_ *Brain, classes []int, _ int) (Target, bool) {
	w.classes = classes

	if w.corpse == nil {
		return Target{}, false
	}

	return *w.corpse, true
}

func (w *w7) SetUnitState(_ *Brain, s int, on bool) {
	w.states[s] = on

	if !on {
		w.cleared = append(w.cleared, s)
	}
}

func (w *w7) UnitStat(_ *Brain, s int) int   { return w.stats[s] }
func (w *w7) SetUnitStat(_ *Brain, s, v int) { w.stats[s] = v }

func skillLvl(p *Profile, slot, lvl int) *Profile {
	p.Skills[slot] = SkillSlot{Name: "sk", Mode: ModeSkill1, Level: lvl}

	return p
}

func TestBatch7Registered(t *testing.T) {
	for _, n := range FaithfulBatch7() {
		d, ok := Lookup(n)
		if !ok {
			t.Fatalf("%s not registered", n)
		}

		if m, ok := AITargetMode(n); !ok || m != d.TargetMode {
			t.Errorf("%s: target mode %d, exe %d", n, d.TargetMode, m)
		}
	}

	if len(FaithfulBatch7()) != 12 {
		t.Errorf("batch 7 claims %d handlers", len(FaithfulBatch7()))
	}
}

// Fallen: a refused charge pops the alert, an accepted attack leaves it.
func TestFallenAlertLifecycle(t *testing.T) {
	b := brainAt(profile("Fallen", 30, 10, 100, 100))
	b.PushCommand(Command{Type: CmdAlert, Count: 1})

	w := newFake(14, false)
	w.failMove = true
	Tick(w, b)

	if b.QueueLen() != 0 {
		t.Fatalf("refused charge should pop the alert, q=%d", b.QueueLen())
	}

	b.PushCommand(Command{Type: CmdAlert, Count: 1})
	b.WakeNow(0)

	w = newFake(2, true)
	Tick(w, b)

	if w.last() != "attack4" || b.QueueLen() != 1 {
		t.Fatalf("attack keeps the alert: %v q=%d", w.log, b.QueueLen())
	}
}

// SkeletonMage: the tail reads the distance AttackTarget wrote.
func TestSkeletonMageUsesAttackTargetDistance(t *testing.T) {
	// tick distance 30, attack target at 3: nothing to approach, stall aip8
	w := newW7(30, false)
	w.atDist, w.atOK = 3, true
	b := brainAt(profile("SkeletonMage", 0, 10, 100, 0, 0, 0, 0, 7))
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 7 {
		t.Fatalf("log=%v wake=%d", w.log, b.Wake)
	}

	// no attack target: the tick distance is used and the mage approaches
	w = newW7(30, false)
	b = brainAt(profile("SkeletonMage", 0, 10, 100, 0, 0, 0, 0, 7))
	w.fakeWorld.attackOK = false
	Tick(w, b)

	if w.last() != "walk-target/10" {
		t.Fatalf("approach: %v", w.log)
	}
}

// Fetish phase 1 compares aip4 with the TARGET's life percent.
func TestFetishFleeReadsTargetLife(t *testing.T) {
	for _, tc := range []struct {
		targetLife int
		flee       bool
	}{{80, true}, {40, false}} {
		w := newW7(2, true)
		w.life = tc.targetLife
		b := brainAt(profile("Fetish", 100, 5, 0, 50))
		b.HPPercent = 100
		b.Scratch[0] = 1
		Tick(w, b)

		if got := b.Scratch[0] == 2; got != tc.flee {
			t.Errorf("target life %d: phase %d, flee want %v (log %v)", tc.targetLife, b.Scratch[0], tc.flee, w.log)
		}
	}
}

// FetishShaman: Skill1 fires below its own level; the buddy is a fetish corpse
// compared by squared distance.
func TestFetishShamanBatch7(t *testing.T) {
	// level 5 skill1, distance 3: alert cast (level 1 would not fire at 3)
	w := newW7(3, true)
	b := brainAt(skillLvl(profile("FetishShaman", 100, 0, 15, 0, 30), 0, 5))
	Tick(w, b)

	if w.last() != "cast0" {
		t.Fatalf("level gate: %v", w.log)
	}

	w = newW7(3, true)
	b = brainAt(skillLvl(profile("FetishShaman", 0, 0, 15, 0, 30), 0, 1))
	Tick(w, b)

	if w.last() == "cast0" {
		t.Fatalf("level 1 must not fire at distance 3: %v", w.log)
	}

	// corpse at squared distance 36 > aip3 15: walk near, no raise
	w = newW7(9, false)
	w.corpse = &Target{ID: 3, X: 106, Y: 100, Size: 1}
	b = brainAt(withSkills(profile("FetishShaman", 100, 0, 15, 0, 30), 2))
	Tick(w, b)

	if len(w.log) != 1 || !strings.HasPrefix(w.log[0], "walk-to(") {
		t.Fatalf("far corpse: %v", w.log)
	}

	if fmt.Sprint(w.classes) != "[141 396]" {
		t.Fatalf("classes %v", w.classes)
	}

	// aip2 == 1 drops the blowgun class
	w = newW7(9, false)
	w.corpse = &Target{ID: 3, X: 103, Y: 100, Size: 1}
	b = brainAt(withSkills(profile("FetishShaman", 100, 1, 15, 0, 30), 2))
	Tick(w, b)

	if w.last() != "cast2" || fmt.Sprint(w.classes) != "[141]" {
		t.Fatalf("raise: %v classes %v", w.log, w.classes)
	}
}

// BatDemon: Aggressive blocks the keep-climbing branch, the flight stat is
// scaled at take-off and restored at landing, a phase-2 walk switches phase
// whether or not it was queued.
func TestBatDemonBatch7(t *testing.T) {
	climb := func(aggr bool) string {
		w := newW7(9, false)
		b := brainAt(profile("BatDemon", 20, 20, 50, 50, 25))
		b.Scratch[0], b.Scratch[1] = 1, 2
		b.HPPercent = 40
		b.Aggressive = aggr
		Tick(w, b)

		return w.last()
	}

	if got := climb(false); got != "attack11" {
		t.Errorf("not aggressive keeps climbing, got %s", got)
	}

	if got := climb(true); got != "attack9" {
		t.Errorf("aggressive lands, got %s", got)
	}

	w := newW7(9, false)
	w.stats[batFlightStat] = 80
	b := brainAt(profile("BatDemon", 20, 20, 50, 50, 4))
	Tick(w, b)

	if w.stats[batFlightStat] != 120 || b.Scratch[2] != 40 || b.Scratch[0] != 1 {
		t.Fatalf("take-off: stat %d delta %d phase %d", w.stats[batFlightStat], b.Scratch[2], b.Scratch[0])
	}

	b.Scratch[1] = 2
	b.WakeNow(0)
	b.HPPercent = 100
	w.fakeWorld.inRange = true
	Tick(w, b)

	if w.stats[batFlightStat] != 80 || b.Scratch[0] != 3 {
		t.Fatalf("landing: stat %d phase %d", w.stats[batFlightStat], b.Scratch[0])
	}

	// phase 2: the walk is refused but the phase still becomes 3
	for seed := uint32(1); seed < 200; seed++ {
		w := newW7(9, false)
		w.failMove = true
		b := NewBrain(7, 1, Normal, profile("BatDemon", 0, 20, 50, 50, 25), seed)
		b.X, b.Y = 100, 100
		b.Scratch[0] = 2
		s := shadow(b)

		if s.Roll(100) < 33 {
			Tick(w, b)

			if b.Scratch[0] != 3 {
				t.Fatalf("seed %d: phase %d", seed, b.Scratch[0])
			}

			return
		}
	}

	t.Fatal("no seed produced the walk roll")
}

// Megademon: the out-of-reach cast needs distance below the skill's level.
func TestMegademonBatch7(t *testing.T) {
	mk := func(level int) (*w7, *Brain) {
		w := newW7(5, false)
		b := brainAt(skillLvl(profile("Megademon", 100, 0, 0, 100, 0, 40), 0, level))
		w.frame = 100

		return w, b
	}

	w, b := mk(8) // 5 < 8: cast and arm the cooldown
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[0] != 140 {
		t.Fatalf("level 8: %v cd %d", w.log, b.Scratch[0])
	}

	w, b = mk(3) // level 3 <= distance 5: approach instead
	Tick(w, b)

	if w.last() != "walk-target/7" {
		t.Fatalf("level 3: %v", w.log)
	}

	w, b = mk(8) // state 0xc up: cleared, no cast, approach
	w.states[0xc] = true
	Tick(w, b)

	if w.last() != "walk-target/7" || len(w.cleared) != 1 || w.cleared[0] != 0xc {
		t.Fatalf("state: %v cleared %v", w.log, w.cleared)
	}
}

// Succubus: skill1 needs the TARGET's life, skill3 the maxHP/maxMana test,
// and a target stat list with flag 0x20 blocks the whole group.
func TestSuccubusBatch7(t *testing.T) {
	prof := func() *Profile {
		return withSkills(profile("Succubus", 0, 0, 100, 20, 5, 5, 50, 0), 0, 2)
	}

	cases := []struct {
		name  string
		setup func(*w7)
		want  string
	}{
		{"target life above floor", func(w *w7) { w.life = 60 }, "cast0"},
		{"target life below floor, skill3 maxMana<maxHP", func(w *w7) { w.life = 30; w.maxHP = 100; w.maxMana = 50 }, "cast2"},
		{"target life below floor, mana>=hp player", func(w *w7) { w.life = 30; w.maxHP = 100; w.maxMana = 200 }, ""},
		{"stat list flag blocks", func(w *w7) { w.life = 60; w.flag20 = true }, ""},
	}

	for _, c := range cases {
		w := newW7(9, false)
		c.setup(w)
		b := brainAt(prof())
		b.Profile.AIP[2] = 0 // no approach so a miss ends in a stall
		Tick(w, b)

		got := ""
		if len(w.log) > 0 && strings.HasPrefix(w.log[0], "cast") {
			got = w.log[0]
		}

		if got != c.want {
			t.Errorf("%s: log %v want %q", c.name, w.log, c.want)
		}
	}
}

// AbyssKnight: the self buff needs an undefined-or-inactive aurastate, low
// life and the aip2 roll, and is cast without a target.
func TestAbyssKnightBuff(t *testing.T) {
	run := func(aura int, active bool) *w7 {
		w := newW7(12, false)
		if aura >= 0 {
			w.auraStates[slot2] = aura
		}

		w.states[aura] = active
		b := brainAt(withSkills(profile("AbyssKnight", 50, 100, 100, 10, 8, 50, 0, 10), 1))
		b.HPPercent = 30
		Tick(w, b)

		return w
	}

	if w := run(7, false); w.last() != "cast1" || w.casts[0] != (Target{}) {
		t.Errorf("buff down: %v casts %v", w.log, w.casts)
	}

	if w := run(7, true); w.last() == "cast1" {
		t.Errorf("buff up must not recast: %v", w.log)
	}

	if w := run(-1, false); w.last() == "cast1" {
		t.Errorf("no aurastate must not cast: %v", w.log)
	}
}

// Scarab and Brute pins live in ai2_test.go; this checks the Scarab rally
// ordering (distance and leader tested before the roll).
func TestScarabRallyOnlyWhenLeaderNear(t *testing.T) {
	p := profile("Scarab", 75, 50, 15, 35, 100)
	leader := brainAt(p)
	m := NewBrain(8, 1, Normal, p, testSeed)
	leader.AddMinion(m)

	Tick(newFake(25, false), leader) // too far: no alert

	if m.QueueLen() != 0 {
		t.Fatalf("far leader must not rally, q=%d", m.QueueLen())
	}
}
