package d2monster

import (
	"testing"
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
