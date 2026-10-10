package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Director side of the pet wiring, tested without a map engine: units are
// built by hand from zero-value monsters (enough for Alive/SubtilePos).

func petUnit(id uint32, x, y int, ai, kind string) *unit {
	m := &d2mapentity.Monster{}
	m.Position.Set(float64(x), float64(y))

	b := &d2monster.Brain{ID: id, X: x, Y: y, Size: 1, Mode: d2monster.ModeNeutral,
		Def: &d2monster.AIDef{Name: ai, Implemented: true}}

	u := &unit{m: m, b: b}
	if kind != "" {
		u.ally = &allyState{kind: kind}
	}

	return u
}

func testDirector(us ...*unit) *Director {
	d := &Director{units: map[uint32]*unit{}, byEntity: map[string]*unit{}}

	for _, u := range us {
		d.units[u.b.ID] = u
		if u.b.ID > d.nextID {
			d.nextID = u.b.ID
		}
	}

	return d
}

func TestUsesPetAI(t *testing.T) {
	for _, tc := range []struct {
		name string
		u    *unit
		want bool
	}{
		{"necropet minion", petUnit(1, 0, 0, "NecroPet", "minion"), true},
		{"raven minion (case)", petUnit(1, 0, 0, "raven", "minion"), true},
		{"sentry trap keeps its driver", petUnit(1, 0, 0, "AssassinSentry", "trap"), false},
		{"sentry as minion kind is not pet AI", petUnit(1, 0, 0, "AssassinSentry", "minion"), false},
		{"vines totem", petUnit(1, 0, 0, "Vines", "totem"), false},
		{"hostile monster", petUnit(1, 0, 0, "NecroPet", ""), false},
		{"minion with other AI", petUnit(1, 0, 0, "Melee", "minion"), false},
	} {
		if got := usesPetAI(tc.u); got != tc.want {
			t.Errorf("%s: got %v", tc.name, got)
		}
	}

	u := petUnit(1, 0, 0, "NecroPet", "minion")
	u.b.Def.Implemented = false

	if usesPetAI(u) {
		t.Error("an unimplemented AI must not run")
	}

	u = petUnit(1, 0, 0, "NecroPet", "minion")
	u.b.Def = nil

	if usesPetAI(u) {
		t.Error("no AI definition")
	}
}

func TestOwnerEnemy(t *testing.T) {
	pet := petUnit(1, 10, 10, "NecroPet", "minion")
	friend := petUnit(2, 11, 10, "NecroPet", "minion") // closest, but friendly
	far := petUnit(3, 40, 10, "Melee", "")
	near := petUnit(4, 14, 10, "Melee", "")

	d := testDirector(pet, friend, far, near)

	tg, dist, ok := d.OwnerEnemy(pet.b, 24)
	if !ok || tg.ID != near.b.ID {
		t.Fatalf("nearest hostile expected, got %+v ok=%v", tg, ok)
	}

	if tg.X != 14 || tg.Y != 10 || dist <= 0 {
		t.Errorf("target %+v dist %d", tg, dist)
	}

	// a small radius excludes everything; friends never count
	if _, _, ok := d.OwnerEnemy(pet.b, 1); ok {
		t.Error("radius 1 should find nobody")
	}

	if _, _, ok := testDirector(pet, friend).OwnerEnemy(pet.b, 100); ok {
		t.Error("friends are not enemies")
	}
}

func TestDirectorPetsGuards(t *testing.T) {
	pet := petUnit(1, 0, 0, "NecroPet", "minion")
	foe := petUnit(2, 1, 1, "Melee", "")
	d := testDirector(pet, foe)
	p := directorPets{d: d}

	if !p.PetAlive(1) {
		t.Error("living minion")
	}

	if p.PetAlive(2) || p.PetAlive(99) {
		t.Error("a hostile or unknown id is not a pet")
	}

	p.RemovePet(2) // hostile: must be left alone

	if d.units[2] == nil {
		t.Error("RemovePet removed a hostile unit")
	}

	d.frame = 77
	if p.Frame() != 77 {
		t.Error("frame")
	}

	if d.MinionByBrainID(1) != pet.m || d.MinionByBrainID(2) != nil {
		t.Error("MinionByBrainID")
	}
}

func TestPetTickHook(t *testing.T) {
	thought := 0
	pet := petUnit(1, 0, 0, "NecroPet", "minion")
	pet.b.Def.TargetMode = d2monster.TargetNone
	pet.b.Def.Think = func(c *d2monster.Ctx) { thought++; c.Sleep(10) }

	d := testDirector(pet)

	if !d.petTick(pet) || thought != 1 {
		t.Fatalf("pet AI should have run once: thought=%d", thought)
	}

	d.frame = 5 // asleep until frame 10
	d.petTick(pet)

	if thought != 1 {
		t.Errorf("slept pet thought again: %d", thought)
	}

	d.frame = 10
	d.petTick(pet)

	if thought != 2 {
		t.Errorf("woken pet did not think: %d", thought)
	}

	// traps keep their driver: the hook declines and nothing runs
	sentry := petUnit(2, 0, 0, "AssassinSentry", "trap")
	sentry.b.Def.Think = func(*d2monster.Ctx) { thought++ }
	d.units[2] = sentry

	if d.petTick(sentry) || thought != 2 {
		t.Error("a trap must not be ticked by the pet hook")
	}
}
