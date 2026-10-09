package d2missile

import "testing"

// Fire Blast: "bomb in air" (hit 36, 0x5a9ab0) spawns "bomb on ground" at its
// end; "bomb on ground" (hit 3, 0x5a7a20 -> 0x5a7890) deals area damage with
// the skill's aurarangecalc radius when it ends.
func TestFireBlastBombs(t *testing.T) {
	air := &Spec{ID: 385, Name: "bomb in air", SrvDoFunc: 1, SrvHitFunc: 36, Vel: 12, MaxVel: 12, Range: 20,
		CollideType: 6, LastCollide: true, AlwaysExplode: true, Explosion: true,
		HitSubMissile: [4]string{"bomb on ground"}}
	ground := &Spec{ID: 386, Name: "bomb on ground", SrvDoFunc: 1, SrvHitFunc: 3, Range: 5, CollideType: 6,
		CollideKill: true, AlwaysExplode: true, Explosion: true}
	tbl := specs{"bomb in air": air, "bomb on ground": ground}

	w := newWorld()
	s := NewSim(w, tbl)
	evs := collect(s)

	_, _ = s.Create(CreateParams{Spec: air, Level: 1, DestX: 40, Owner: Owner{Roller: fakeRoller{0}},
		AreaRadius: 5, Damage: DamageDesc{PhysMin: 256, PhysMax: 256}})

	for i := 0; i < 60; i++ {
		w.frame++
		s.Step()
	}

	areas := 0

	for _, e := range *evs {
		if e.Kind == EventArea {
			areas++

			if e.Radius != 1 && e.Radius != 5 {
				t.Errorf("radius %d", e.Radius)
			}
		}
	}

	// the air bomb ends with no area damage of its own; its child (created at
	// the end of the flight) damages once, when it expires
	if areas != 1 || len(s.Missiles()) != 0 {
		t.Errorf("%d area events, %d missiles left", areas, len(s.Missiles()))
	}
}
