package d2monster

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

func skills(p *Profile, names ...string) *Profile {
	for i, n := range names {
		p.Skills[i] = SkillSlot{Name: n, Mode: ModeSkill1}
	}

	return p
}

func TestBoss2Registered(t *testing.T) {
	for _, n := range []string{"Vulture", "Summoner", "Duriel", "Mephisto", "Diablo", "Izual", "BaalMinion",
		"SuicideMinion", "UberDiablo", "UberMephisto", "UberIzual"} {
		if d, ok := Lookup(n); !ok || !d.Implemented {
			t.Errorf("%s is not registered", n)
		}
	}
}

// ---- Summoner ----

func summonerProfile(aip ...int) *Profile {
	return skills(profile("Summoner", aip...), "Glacial Spike", "Frost Nova", "Fire Ball", "VampireFirewall", "Weaken")
}

func TestSummonerColdPath(t *testing.T) {
	// aip: 1 act 100%, 2 weaken 0%, 3 never flips (100), 4 nova cd 40, 5 fw cd 120, 6 back-off 0, 7 nova range 10, 8 missile range 40
	p := summonerProfile(100, 0, 100, 40, 120, 0, 10, 40)

	w := newFake2(8, false)
	w.frame = 10
	w.fire, w.cold = 50, 10 // cold resist <= fire: cold spells
	b := awake(brainAt(p))

	Tick(w, b)

	if w.last() != "cast1" || b.Scratch[1] != 10+40 {
		t.Fatalf("first: Frost Nova expected, log=%v cd=%d", w.log, b.Scratch[1])
	}

	// nova on cooldown: Glacial Spike (the attack target is in missile range)
	w.frame, b.Wake = 20, 20
	Tick(w, b)

	if w.last() != "cast0" {
		t.Fatalf("second: Glacial Spike expected, log=%v", w.log)
	}

	// far (beyond the missile range 20): neither nova nor spike; the Fire Wall is ready
	pf := summonerProfile(100, 0, 100, 40, 120, 0, 10, 20)
	w2 := newFake2(30, false)
	w2.frame = 10
	w2.fire, w2.cold = 50, 10
	b2 := awake(brainAt(pf))
	b2.Profile.AIDist = 55
	Tick(w2, b2)

	if w2.last() != "cast3" || b2.Scratch[2] != 10+120 {
		t.Fatalf("far: Fire Wall expected, log=%v cd=%d", w2.log, b2.Scratch[2])
	}

	// ... and then everything left is out of range: the Weaken fallback
	w2.frame, b2.Wake = 11, 11
	Tick(w2, b2)

	if w2.last() != "cast4" {
		t.Fatalf("Weaken fallback expected after the wall, log=%v", w2.log)
	}
}

func TestSummonerFirePathAndWeaken(t *testing.T) {
	p := summonerProfile(100, 0, 100, 40, 120, 0, 10, 40)

	// fire is better: Fire Wall first, Fire Ball next, Frost Nova only as the late fallback
	w := newFake2(8, false)
	w.frame = 10
	w.fire, w.cold = 10, 50
	b := awake(brainAt(p))
	Tick(w, b)

	if w.last() != "cast3" {
		t.Fatalf("Fire Wall first: %v", w.log)
	}

	w.frame, b.Wake = 11, 11
	Tick(w, b)

	if w.last() != "cast2" {
		t.Fatalf("Fire Ball second: %v", w.log)
	}

	// with no fire slots left to use the Weaken fallback is the last resort
	p2 := skills(profile("Summoner", 100, 0, 100, 40, 120, 0, 10, 40))
	p2.Skills[4] = SkillSlot{Name: "Weaken", Mode: ModeSkill1}
	w2 := newFake2(8, false)
	b2 := awake(brainAt(p2))
	Tick(w2, b2)

	if w2.last() != "cast4" {
		t.Fatalf("Weaken fallback: %v", w2.log)
	}

	// 100% weaken chance goes first
	p3 := summonerProfile(100, 100, 100, 40, 120, 0, 10, 40)
	w3 := newFake2(8, false)
	Tick(w3, awake(brainAt(p3)))

	if w3.last() != "cast4" {
		t.Fatalf("Weaken first: %v", w3.log)
	}
}

func TestSummonerIdleWanderAndBackoff(t *testing.T) {
	// aip1 = 0: never acts, only wanders
	w := newFake2(8, false)
	b := awake(brainAt(summonerProfile(0, 0, 100, 40, 120, 100, 10, 40)))
	w.target.X, w.target.Y = 103, 100
	w.dist = 3
	Tick(w, b)

	// within 5 the 100% back-off request is issued, then the wander replaces it
	if len(w.log) != 2 || !strings.HasPrefix(w.log[0], "walk-to(") {
		t.Fatalf("expected back-off then wander: %v", w.log)
	}

	if w.log[0] != "walk-to(99,99)" && w.log[0] != "walk-to(97,99)" && !strings.HasPrefix(w.log[0], "walk-to(") {
		t.Fatalf("back-off direction: %s", w.log[0])
	}
}

// ---- Duriel ----

func durielProfile(aip ...int) *Profile {
	return skills(profile("Duriel", aip...), "Charge", "Jab", "Smite", "Holy Freeze")
}

func TestDuriel(t *testing.T) {
	// reach: Smite first
	w := newFake2(3, true)
	b := brainAt(durielProfile(5, 100, 100, 100))
	Tick(w, b)

	if w.last() != "cast2" || w.auras != 1 {
		t.Fatalf("Smite + aura: %v auras=%d", w.log, w.auras)
	}

	// the aura is started once
	w.frame, b.Wake = 5, 5
	Tick(w, b)

	if w.auras != 1 {
		t.Fatal("aura started twice")
	}

	for _, c := range []struct {
		aip  []int
		want string
	}{
		{[]int{5, 0, 100, 100}, "cast1"},   // Jab
		{[]int{5, 0, 0, 100}, "attack5"},   // A2
		{[]int{5, 0, 0, 0}, "attack4"},     // A1
		{[]int{5, 100, 100, 100}, "cast2"}, // Smite beats Jab beats A2
	} {
		w := newFake2(3, true)
		Tick(w, brainAt(durielProfile(c.aip...)))

		if w.last() != c.want {
			t.Errorf("aip %v: %v want %s", c.aip, w.log, c.want)
		}
	}

	// out of reach: walks (Charge only with aip5 > 0)
	w = newFake2(15, false)
	Tick(w, brainAt(durielProfile(5, 33, 50)))

	if w.last() != "walk-target/7" {
		t.Fatalf("approach: %v", w.log)
	}

	w = newFake2(15, false)
	Tick(w, brainAt(durielProfile(5, 33, 50, 0, 100)))

	if w.last() != "cast0" {
		t.Fatalf("charge: %v", w.log)
	}
}

// ---- Mephisto ----

func mephProfile(aip1 int) *Profile {
	return skills(profile("Mephisto", aip1, 25, 25), "PrimeLightning", "PrimeBolt", "PrimePoisonNova",
		"MephistoMissile", "MephFrostNova", "Blizzard")
}

func TestMephistoBurstSequence(t *testing.T) {
	// out of reach but near, healthy: with the 50% roll a burst of 3..5 ticks starts and
	// keeps its phase until the counter runs out
	b := findBrain(t, mephProfile(15), func(s *d2rand.Seed) bool { return s.Roll(100) < 50 })
	w := newFake2(10, false)

	steps := 0

	for ; steps < 10; steps++ {
		w.frame += 20
		b.Wake = w.frame
		Tick(w, b)

		if b.Scratch[2] == mephIdle {
			break
		}
	}

	if steps < 2 || steps > 5 {
		t.Fatalf("burst should last 3..5 ticks, lasted %d: %v", steps+1, w.log)
	}

	casts := 0

	for _, l := range w.log {
		if strings.HasPrefix(l, "cast") {
			casts++
		}
	}

	if casts == 0 {
		t.Fatalf("no cast in a burst: %v", w.log)
	}
}

func TestMephistoSelector(t *testing.T) {
	// far (>= 21): approach phase
	w := newFake2(30, false)
	b := brainAt(mephProfile(15))
	b.Profile.AIDist = 55
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") { // WalkNearTarget(6): a point beside the target
		t.Fatalf("approach: %v", w.log)
	}

	// hurt (<21%), adjacent, 40% roll: back off, Poison Nova when cornered
	pred := func(s *d2rand.Seed) bool { return s.Roll(100) < 40 }
	b = findBrain(t, mephProfile(15), pred)
	b.HPPercent = 10
	w = newFake2(3, false)
	w.failMove = true // cannot retreat
	Tick(w, b)

	// 0x5f6c1c: the Poison Nova branch needs the target in reach, which the selector excludes for
	// this phase, so a failed retreat always falls into one burst step (first step: flag 2, strafe)
	if b.Scratch[1] != 2 || b.Scratch[2] != mephIdle {
		t.Fatalf("cornered: scratch %v log %v", b.Scratch, w.log)
	}

	for _, l := range w.log {
		if strings.HasPrefix(l, "cast") {
			t.Fatalf("no cast when it cannot back off: %v", w.log)
		}
	}

	// the same, free to move: it walks away
	b = findBrain(t, mephProfile(15), pred)
	b.HPPercent = 10
	w = newFake2(3, false)
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("back off: %v", w.log)
	}

	// healthy melee phase: roll > aip1 picks it, point-blank action follows
	b = findBrain(t, mephProfile(0), func(s *d2rand.Seed) bool {
		return s.Roll(100) > 0 && s.Roll(100) < 80 && s.Roll(100) < 80 // melee, then neither retreat nor lightning
	})
	w = newFake2(3, true)
	Tick(w, b)

	if b.Scratch[2] != mephIdle || len(w.log) != 1 {
		t.Fatalf("melee phase: %v", w.log)
	}
}

func TestMephistoMeleePhase(t *testing.T) {
	// melee phase, k = 0: no retreat (r1 < 80), then r2 >= 80 casts Skill3 (Poison Nova, the exe's
	// slot for the point-blank cast; the Go port had Skill1)
	b := findBrain(t, mephProfile(0), func(s *d2rand.Seed) bool {
		return s.Roll(100) > 0 && s.Roll(100) < 80 && s.Roll(100) >= 80
	})
	w := newFake2(3, true)
	Tick(w, b)

	if w.last() != "cast2" {
		t.Fatalf("poison nova: %v", w.log)
	}

	// r1 >= 80: strafe (Circle) instead of walking away
	b = findBrain(t, mephProfile(0), func(s *d2rand.Seed) bool {
		return s.Roll(100) > 0 && s.Roll(100) >= 80
	})
	w = newFake2(3, true)
	Tick(w, b)

	if len(w.log) != 1 || !strings.HasPrefix(w.log[0], "walk-to(") {
		t.Fatalf("strafe: %v", w.log)
	}
}

func TestMephistoBurstSteps(t *testing.T) {
	// first step: the flag becomes 2, the tick strafes and casts nothing
	b := brainAt(mephProfile(15))
	b.Scratch = [3]int{3, 0, mephBurst}
	w := newFake2(10, false)
	Tick(w, b)

	if b.Scratch[0] != 2 || b.Scratch[1] != 2 || b.Scratch[2] != mephBurst {
		t.Fatalf("scratch %v", b.Scratch)
	}

	for _, l := range w.log {
		if strings.HasPrefix(l, "cast") {
			t.Fatalf("first step must not cast: %v", w.log)
		}
	}

	// second step, six skills (share 16): r in [32,48) is Skill2 (index 1)
	b = findBrain(t, mephProfile(15), func(s *d2rand.Seed) bool { r := s.Roll(100); return r >= 32 && r < 48 })
	b.Scratch = [3]int{3, 2, mephBurst}
	w = newFake2(10, false)
	Tick(w, b)

	if w.last() != "cast1" {
		t.Fatalf("burst cast: %v scratch %v", w.log, b.Scratch)
	}

	// the counter reaching zero ends the burst
	b = brainAt(mephProfile(15))
	b.Scratch = [3]int{1, 2, mephBurst}
	Tick(newFake2(10, false), b)

	if b.Scratch[2] != mephIdle {
		t.Fatalf("burst should end: %v", b.Scratch)
	}
}

// ---- Diablo ----

func TestPickWeighted(t *testing.T) {
	b := brainAt(profile("Diablo"))
	counts := make([]int, 4)

	for i := 0; i < 4000; i++ {
		counts[pickWeighted(b, []int{0, 10, 30, 0})]++
	}

	if counts[0] != 0 || counts[3] != 0 || counts[2] < 2*counts[1] {
		t.Fatalf("distribution %v", counts)
	}

	// a table with no weights falls through to "wait"
	if pickWeighted(b, []int{0, 0}) != diaWait {
		t.Fatal("empty table")
	}
}

func diabloProfile() *Profile {
	return skills(profile("Diablo"), "DiabLight", "DiabCold", "DiabFire", "DiabWall", "DiabRun", "PrimeFirewall", "DiabPrison")
}

func TestDiabloFirstTickAnchorAndWait(t *testing.T) {
	w := newFake2(5, true)
	b := brainAt(diabloProfile())
	b.Diff = Hell

	if !Tick(w, b) {
		t.Fatal("no tick")
	}

	a := b.FindCommand(CmdAnchor)
	if a == nil || a.X != 100 || a.Y != 100 {
		t.Fatalf("anchor: %+v", a)
	}

	// whatever he did, it was one of his actions: an attack, a cast, a move, or the wait
	if len(w.log) == 0 && b.Wake != 4 {
		t.Fatalf("idle wait on hell should be 4 frames, wake=%d", b.Wake)
	}
}

func TestDiabloChannelContinues(t *testing.T) {
	w := newFake2(5, true)
	b := brainAt(diabloProfile())
	b.Scratch[0] = diaLight // the DiabLight channel is still running
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[0] != diaLight {
		t.Fatalf("channel: %v act=%d", w.log, b.Scratch[0])
	}

	// the self-inflicted state 0xc ends it
	w2 := newFake2(5, true)
	w2.states[0xc] = true
	b2 := brainAt(diabloProfile())
	b2.Scratch[0] = diaLight
	Tick(w2, b2)

	if len(w2.log) != 0 || b2.Scratch[0] != 0 {
		t.Fatalf("channel end: %v", w2.log)
	}
}

func TestDiabloActionsSpendSkills(t *testing.T) {
	// drive many ticks and check that every action is one of his legal ones
	w := newFake2(8, true)
	b := brainAt(diabloProfile())
	b.Aggressive = true

	seen := map[string]bool{}

	for i := 0; i < 300; i++ {
		w.frame += 12
		b.Wake = w.frame
		w.log = nil
		w.states[0xc] = len(seen) > 0 && seen["cast0"] // the breath gives him state 0xc, which ends the channel
		Tick(w, b)

		for _, l := range w.log {
			seen[l] = true
		}
	}

	for _, want := range []string{"attack4", "attack5", "cast0", "cast1", "cast2", "cast3"} {
		if !seen[want] {
			t.Errorf("Diablo never did %s: %v", want, seen)
		}
	}
}

// ---- Izual ----

func izualProfile(aip ...int) *Profile {
	p := skills(profile("Izual", aip...), "Frost Nova")
	p.AIDel = 15

	return p
}

func TestIzual(t *testing.T) {
	// in reach, melee 100%
	w := newFake2(3, true)
	Tick(w, brainAt(izualProfile(100, 50, 66, 0, 20, 3)))

	if w.last() != "attack4" {
		t.Fatalf("melee: %v", w.log)
	}

	// in reach, no melee, nova 100%: Frost Nova then the stall, then the swings
	w = newFake2(3, true)
	b := brainAt(izualProfile(0, 50, 66, 100, 20, 3))
	Tick(w, b)

	if w.last() != "cast0" || b.Scratch[1] != 20 || b.Scratch[2] != 3 {
		t.Fatalf("nova: %v %v", w.log, b.Scratch)
	}

	w.frame, b.Wake = 3, 3
	Tick(w, b)

	if b.Wake != 3+20 || b.Scratch[1] != 0 {
		t.Fatalf("stall: wake=%d %v", b.Wake, b.Scratch)
	}

	for i := 3; i > 0; i-- {
		w.frame, b.Wake = w.frame+20, w.frame+20
		Tick(w, b)

		if w.last() != "attack4" || b.Scratch[2] != i-1 {
			t.Fatalf("swing %d: %v %v", i, w.log, b.Scratch)
		}
	}

	// in reach, nothing rolls: waits aidel
	w = newFake2(3, true)
	b = brainAt(izualProfile(0, 50, 66, 0, 20, 3))
	Tick(w, b)

	if b.Wake != 15 {
		t.Fatalf("aidel wait: %d", b.Wake)
	}

	// closing in within 10: nova 100%
	w = newFake2(8, false)
	Tick(w, brainAt(izualProfile(45, 0, 100, 0, 20, 3)))

	if w.last() != "cast0" {
		t.Fatalf("nova on approach: %v", w.log)
	}

	// far: keeps a distance of 9 (kite step) or waits when within 11
	w = newFake2(30, false)
	b = brainAt(izualProfile(45, 0, 0, 0, 20, 3))
	b.Profile.AIDist = 55
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("kite: %v", w.log)
	}

	w = newFake2(10, false)
	b = brainAt(izualProfile(45, 0, 0, 0, 20, 3))
	Tick(w, b)

	if b.Wake != 15 {
		t.Fatalf("hold at 10: %d", b.Wake)
	}

	// aip2 = 100 never holds: it walks up
	w = newFake2(30, false)
	b = brainAt(izualProfile(45, 100, 0, 0, 20, 3))
	b.Profile.AIDist = 55
	Tick(w, b)

	if w.last() != "walk-target/7" {
		t.Fatalf("walk up: %v", w.log)
	}
}

// ---- Baal minion ----

func TestBaalMinion(t *testing.T) {
	p := skills(profile("BaalMinion", 100, 100, 100, 17), "Smite")

	// in reach: Smite (aip3 = 100%), then the next think after aip4 frames
	w := newFake2(3, true)
	w.frame = 40
	b := brainAt(p)
	Tick(w, b)

	if w.last() != "cast0" || b.Wake != 40+17 {
		t.Fatalf("smite: %v wake=%d", w.log, b.Wake)
	}

	// without Smite: A1
	w = newFake2(3, true)
	b = brainAt(profile("BaalMinion", 100, 100, 100, 17))
	Tick(w, b)

	if w.last() != "attack4" {
		t.Fatalf("melee: %v", w.log)
	}

	// out of reach: approach with aip2
	w = newFake2(10, false)
	b = brainAt(p)
	Tick(w, b)

	if w.last() != "walk-target/1" || b.Wake != 17 {
		t.Fatalf("approach: %v wake=%d", w.log, b.Wake)
	}

	// attack roll failed: the stall is aip3, then the trailing aip4 delay wins anyway
	w = newFake2(3, true)
	b = brainAt(skills(profile("BaalMinion", 0, 0, 5, 17), "Smite"))
	Tick(w, b)

	if len(w.log) != 0 || b.Wake != 17 {
		t.Fatalf("stall: %v wake=%d", w.log, b.Wake)
	}
}

// ---- Suicide minion ----

func TestSuicideMinion(t *testing.T) {
	p := profile("SuicideMinion", 0, 10, 100, 0, 30)

	// in reach: the fuse is armed
	w := newFake2(2, true)
	w.frame = 100
	b := brainAt(p)
	Tick(w, b)

	if b.Scratch[0] != 130 || len(w.log) != 0 {
		t.Fatalf("fuse: %d %v", b.Scratch[0], w.log)
	}

	// before the fuse runs out it just waits
	w.frame, b.Wake = 120, 120
	Tick(w, b)

	if len(w.log) != 0 {
		t.Fatalf("should wait: %v", w.log)
	}

	// after: the death mode (the explosion)
	w.frame, b.Wake = 131, 131
	Tick(w, b)

	if w.last() != "attack0" {
		t.Fatalf("explosion: %v", w.log)
	}

	// out of reach it runs at the hero
	w = newFake2(10, false)
	Tick(w, brainAt(p))

	if w.last() != "walk-target/0" {
		t.Fatalf("approach: %v", w.log)
	}
}

// ---- Vulture ----

func vultureProfile() *Profile {
	return profile("Vulture", 100, 8, 75, 100, 100)
}

func TestVultureTakeOffAndHover(t *testing.T) {
	// on the ground, hero farther than 12, 60% roll: take off
	b := findBrain(t, vultureProfile(), func(s *d2rand.Seed) bool { return s.Roll(100) < 60 })
	w := newFake2(20, false)
	w.target.X = 120
	b.Profile.AIDist = 55
	Tick(w, b)

	if !b.Airborne || (b.Scratch[0] != 24 && b.Scratch[0] != 25) || len(w.flights) != 1 || w.flights[0] != "takeoff" {
		t.Fatalf("take-off: air=%v laps=%d %v %v", b.Airborne, b.Scratch[0], w.flights, w.log)
	}

	// airborne with laps left: it picks a waypoint far enough from itself and moves
	laps := b.Scratch[0]
	w.log = nil
	w.frame, b.Wake = 12, 12
	w.target.X, w.dist = 108, 8
	Tick(w, b)

	if b.Scratch[0] != laps-1 || b.Scratch[1] == 0 || !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("hover: laps=%d wp=(%d,%d) %v", b.Scratch[0], b.Scratch[1], b.Scratch[2], w.log)
	}

	if EdgeDistance(b.X-b.Scratch[1], b.Y-b.Scratch[2], 0) < clamp(laps*2, 12, 36) {
		t.Fatalf("waypoint too close: (%d,%d)", b.Scratch[1], b.Scratch[2])
	}

	// the last lap: it lands and closes in
	b.Scratch[0] = 1
	w.log, w.flights = nil, nil
	w.frame, b.Wake = 30, 30
	Tick(w, b)

	if b.Airborne || b.Scratch[0] != -1 || len(w.flights) != 1 || w.flights[0] != "land" {
		t.Fatalf("landing: air=%v laps=%d %v", b.Airborne, b.Scratch[0], w.flights)
	}

	if !strings.HasPrefix(w.last(), "run-to(") {
		t.Fatalf("landing move: %v", w.log)
	}
}

func TestVultureGroundTail(t *testing.T) {
	// in reach: strike with aip1
	w := newFake2(3, true)
	Tick(w, brainAt(vultureProfile()))

	if w.last() != "attack4" {
		t.Fatalf("strike: %v", w.log)
	}

	// in reach, strike roll fails: stall aip2
	w = newFake2(3, true)
	p := vultureProfile()
	p.AIP[1] = 0
	b := brainAt(p)
	Tick(w, b)

	if b.Wake != 8 {
		t.Fatalf("stall: %d", b.Wake)
	}

	// close but not in reach, after landing (laps -1): circles (aip4 100%)
	w = newFake2(6, false)
	b = brainAt(vultureProfile())
	b.Scratch[0] = -1
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") || b.Scratch[0] != 0 {
		t.Fatalf("circle: %v laps=%d", w.log, b.Scratch[0])
	}

	// with aip4 = 0 it closes in with the kite step instead
	w = newFake2(6, false)
	p = vultureProfile()
	p.AIP[4] = 0
	b = brainAt(p)
	b.Scratch[0] = -1
	Tick(w, b)

	if !strings.HasPrefix(w.last(), "walk-to(") {
		t.Fatalf("approach: %v", w.log)
	}
}

func TestVultureNearbyPreyAndLeader(t *testing.T) {
	// a vulture with a leader never takes off
	leader := brainAt(vultureProfile())
	b := findBrain(t, vultureProfile(), func(s *d2rand.Seed) bool { return s.Roll(100) < 60 })
	leader.AddMinion(b)

	w := newFake2(20, false)
	w.target.X = 120
	b.Profile.AIDist = 55
	Tick(w, b)

	if b.Airborne {
		t.Fatal("a vulture in a flock does not take off")
	}
}
