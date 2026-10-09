package d2monster

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// fakePet adds the PetWorld methods to fakeWorld.
type fakePet struct {
	*fakeWorld
	owner     OwnerInfo
	noOwner   bool
	teleports int
	enemy     Target
	hasEnemy  bool
	enemyDist int
}

func (f *fakePet) Owner(*Brain) (OwnerInfo, bool) { return f.owner, !f.noOwner }
func (f *fakePet) Teleport(*Brain) bool           { f.teleports++; return true }
func (f *fakePet) OwnerEnemy(*Brain, int) (Target, int, bool) {
	return f.enemy, f.enemyDist, f.hasEnemy
}

func newPet(ai string, ownerDist int, aip ...int) (*fakePet, *Brain) {
	fw := newFake(5, true)
	fw.hasTarget = false

	f := &fakePet{fakeWorld: fw,
		owner: OwnerInfo{Target: Target{ID: 1, X: 100 + ownerDist, Y: 100, Size: 1, IsPlayer: true}, Mode: ModeNeutral},
		enemy: Target{ID: 50, X: 104, Y: 100, Size: 1},
	}
	b := brainAt(profile(ai, aip...))
	def, _ := Lookup(ai)
	b.SetAI(def)

	return f, b
}

func TestNecroPetLeash(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dist      int
		teleports int
		wantLog   string
	}{
		{"beyond 50: teleport", 60, 1, ""},
		{"29..50: walk back to the owner", 40, 0, "walk-target/8"},
		{"within 29, no enemy, far from owner: run or walk up", 20, 0, ""},
		{"within 29, next to the owner: stand", 4, 0, ""},
	} {
		f, b := newPet("NecroPet", tc.dist)

		Tick(f, b)

		if f.teleports != tc.teleports {
			t.Errorf("%s: teleports = %d", tc.name, f.teleports)
		}

		if tc.wantLog != "" && f.last() != tc.wantLog {
			t.Errorf("%s: log = %v, want %s", tc.name, f.log, tc.wantLog)
		}
	}

	// the close pet idles 5 frames, the far one follows
	f, b := newPet("NecroPet", 4)
	Tick(f, b)

	if len(f.log) != 0 || b.Wake != 5 {
		t.Errorf("idle pet: log=%v wake=%d", f.log, b.Wake)
	}

	f, b = newPet("NecroPet", 20)
	Tick(f, b)

	if k := f.last(); k != "run-target/8" && k != "walk-target/8" {
		t.Errorf("follow: log=%v", f.log)
	}
}

func TestNecroPetFightsOwnersEnemies(t *testing.T) {
	// enemy in reach: 80% attack (A1), else sleep 10
	attacks, sleeps := 0, 0

	for i := 0; i < 400; i++ {
		f, b := newPet("NecroPet", 6)
		f.hasEnemy, f.enemyDist = true, 3
		b.Seed = shadowSeed(i)

		Tick(f, b)

		if f.last() == "attack4" {
			attacks++
		} else if len(f.log) == 0 && b.Wake == 10 {
			sleeps++
		} else {
			t.Fatalf("unexpected action %v wake %d", f.log, b.Wake)
		}
	}

	if attacks < 290 || attacks > 350 { // 80% of 400 = 320
		t.Errorf("attacks = %d of 400, want about 80%%", attacks)
	}

	// enemy out of reach: walk or run toward it
	f, b := newPet("NecroPet", 6)
	f.hasEnemy, f.enemyDist, f.inRange = true, 15, false

	Tick(f, b)

	if k := f.last(); k != "run-target/6" && k != "walk-target/6" {
		t.Errorf("approach: %v", f.log)
	}

	// an enemy only matters while the pet is within 29 of its owner
	f, b = newPet("NecroPet", 35)
	f.hasEnemy, f.enemyDist = true, 3
	Tick(f, b)

	if f.last() != "walk-target/8" {
		t.Errorf("leashed pet engaged: %v", f.log)
	}

	// the run/walk split follows roll > 14 (about 85% running)
	runs := 0

	for i := 0; i < 400; i++ {
		f, b = newPet("NecroPet", 6)
		f.hasEnemy, f.enemyDist, f.inRange = true, 15, false
		b.Seed = shadowSeed(i)

		Tick(f, b)

		if f.last() == "run-target/6" {
			runs++
		}
	}

	if runs < 320 || runs > 360 {
		t.Errorf("runs = %d of 400, want about 85%%", runs)
	}
}

func TestNecroPetOwnerGone(t *testing.T) {
	// never had an owner: idle
	f, b := newPet("NecroPet", 4)
	f.noOwner = true
	Tick(f, b)

	if f.teleports != 0 || b.Wake != 10 {
		t.Errorf("no owner: teleports=%d wake=%d", f.teleports, b.Wake)
	}

	// owner remembered (left the level): teleport back to it
	f, b = newPet("NecroPet", 4)
	Tick(f, b)

	if b.Scratch[2] != 1 {
		t.Fatalf("owner id not remembered: %v", b.Scratch)
	}

	f.noOwner = true
	b.Wake = 0
	Tick(f, b)

	if f.teleports != 1 {
		t.Errorf("remembered owner: teleports=%d", f.teleports)
	}
}

func TestNecroPetCasterCasts(t *testing.T) {
	f, b := newPet("NecroPet", 6)
	f.hasEnemy, f.enemyDist = true, 4
	b.Scratch[0] = 1 // casting pet

	for i := 0; i < 50 && f.last() != "cast0"; i++ {
		b.Wake = 0
		f.log = nil
		Tick(f, b)
	}

	if f.last() != "cast0" {
		t.Errorf("caster never cast: %v", f.log)
	}
}

func TestSentryAndVines(t *testing.T) {
	// a sentry fires skill 1 at a target in reach, and always when aip1 = 100
	f, _ := newPet("AssassinSentry", 0, 100, 10)
	f.hasTarget, f.attackOK, f.inRange = true, true, true
	b := brainAt(profile("AssassinSentry", 100, 10))
	def, _ := Lookup("AssassinSentry")
	b.SetAI(def)

	Tick(f, b)

	if f.last() != "cast0" {
		t.Errorf("sentry: %v", f.log)
	}

	// nothing in reach: sleep aip2
	f.inRange, b.Wake, f.log = false, 0, nil
	Tick(f, b)

	if len(f.log) != 0 || b.Wake != 10 {
		t.Errorf("idle sentry: log=%v wake=%d", f.log, b.Wake)
	}

	// vines strike when someone is in reach
	for _, name := range []string{"Vines", "CycleOfLife"} {
		f, _ = newPet(name, 0, 100, 20)
		f.hasTarget, f.attackOK, f.inRange = true, true, true
		b = brainAt(profile(name, 100, 20))
		def, _ = Lookup(name)
		b.SetAI(def)

		Tick(f, b)

		if f.last() != "cast0" {
			t.Errorf("%s: %v", name, f.log)
		}
	}
}

func TestRavenFollowsAndPecks(t *testing.T) {
	// far from the owner and nothing to fight: fly back
	f, b := newPet("Raven", 20, 10, 6)
	Tick(f, b)

	if f.last() != "run-target/8" {
		t.Errorf("raven follow: %v", f.log)
	}

	// enemy out of reach: dive at it
	f, b = newPet("Raven", 6, 10, 6)
	f.hasEnemy, f.enemyDist, f.inRange = true, 12, false
	Tick(f, b)

	if f.last() != "run-target/1" {
		t.Errorf("raven dive: %v", f.log)
	}

	// in reach with aip1 = 100 it always attacks
	f, b = newPet("Raven", 6, 100, 6)
	f.hasEnemy, f.enemyDist = true, 2
	Tick(f, b)

	if f.last() != "attack4" {
		t.Errorf("raven peck: %v", f.log)
	}

	// close to the owner and idle: sleep aip2
	f, b = newPet("Raven", 3, 10, 6)
	Tick(f, b)

	if len(f.log) != 0 || b.Wake != 6 {
		t.Errorf("raven idle: log=%v wake=%d", f.log, b.Wake)
	}
}

// shadowSeed gives a distinct, deterministic RNG per iteration.
func shadowSeed(i int) *d2rand.Seed { return d2rand.New(uint32(i)*7919 + 13) }
