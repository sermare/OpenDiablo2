package d2missile

import (
	"math"
	"testing"
)

// Frozen Orb stand-in: bolts every Param1 frames in a spiral of Param2 steps
// over 16 directions, then a ring of HitSubMissile1 where the orb ends. The
// orb itself carries no damage.
func TestFrozenOrbEmitsBoltsThenRing(t *testing.T) {
	orb := &Spec{ID: 260, Name: "frozenorb", SrvDoFunc: 15, SrvHitFunc: 29, Vel: 10, MaxVel: 10, Range: 20,
		CollideType: 3, AlwaysExplode: true, Param: [5]int{4, 19},
		SubMissile: [3]string{"frozenorbbolt"}, HitSubMissile: [4]string{"frozenorbnova"}}
	tbl := specs{
		"frozenorbbolt": {ID: 261, Name: "frozenorbbolt", SrvDoFunc: 1, Vel: 18, MaxVel: 18, Range: 25, CollideType: 3},
		"frozenorbnova": {ID: 262, Name: "frozenorbnova", SrvDoFunc: 1, Vel: 24, MaxVel: 24, Range: 25, CollideType: 3},
	}
	w := newWorld()
	s := NewSim(w, tbl)

	var created []*Missile

	s.OnEvent = func(e Event) {
		if e.Kind == EventCreate {
			created = append(created, e.Missile)
		}
	}

	dmg := DamageDesc{Cold: Elem{Min: 512, Max: 512}}
	_, _ = s.Create(CreateParams{Spec: orb, Level: 5, DestX: 60, Damage: dmg, Owner: Owner{Roller: fakeRoller{0}}})

	if created[0].Damage != (DamageDesc{}) {
		t.Fatalf("the orb must not carry damage: %+v", created[0].Damage)
	}

	for i := 0; i < 40; i++ {
		w.frame++
		s.Step()
	}

	bolts, nova := 0, 0
	dirs := map[int]bool{}

	for _, m := range created[1:] {
		if m.Damage != dmg {
			t.Errorf("%s carries %+v, want the cast damage", m.Spec.Name, m.Damage)
		}

		switch m.Spec.Name {
		case "frozenorbbolt":
			bolts++
		case "frozenorbnova":
			nova++

			dirs[int(math.Round(math.Atan2(m.DY, m.DX)/(2*math.Pi/16)))&15] = true
		}
	}

	if bolts != 5 { // life 20, a bolt every 4th frame of age 0..19
		t.Errorf("%d bolts, want 5", bolts)
	}

	if nova != 16 || len(dirs) != 16 {
		t.Errorf("ring: %d missiles in %d directions, want 16 in 16", nova, len(dirs))
	}
}
