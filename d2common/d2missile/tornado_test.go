package d2missile

import "testing"

// SrvDoFunc 27 (0x5ad590, verified bytes): an area pulse every period frames
// (Param1, else the skill's calc4) with radius Param2 else aurarangecalc.
func TestTornadoPulses(t *testing.T) {
	sp := &Spec{ID: 478, Name: "tornado", SrvDoFunc: 27, Vel: 8, MaxVel: 8, Range: 75, CollideType: 3,
		NextHit: true, NextDelay: 25, Param: [5]int{0, 0}}
	w := newWorld()
	s := NewSim(w, nil)
	evs := collect(s)

	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 60, Owner: Owner{Roller: fakeRoller{0}},
		PulseEvery: 15, AreaRadius: 3, Damage: DamageDesc{PhysMin: 256, PhysMax: 256}})

	for i := 0; i < 74; i++ {
		w.frame++
		s.Step()
	}

	n := 0

	for _, e := range *evs {
		if e.Kind == EventArea {
			n++

			if e.Radius != 3 {
				t.Errorf("radius %d, want 3", e.Radius)
			}
		}
	}

	// remaining life 75..2 at the pulse test: multiples of 15 are 75,60,45,30,15
	if n != 5 {
		t.Errorf("%d pulses, want 5", n)
	}

	// the table's Param1/Param2 win over the skill values
	sp.Param = [5]int{25, 6}
	s2 := NewSim(newWorld(), nil)
	evs2 := collect(s2)
	_, _ = s2.Create(CreateParams{Spec: sp, Level: 1, DestX: 60, Owner: Owner{Roller: fakeRoller{0}},
		PulseEvery: 15, AreaRadius: 3})

	for i := 0; i < 74; i++ {
		s2.World.(*fakeWorld).frame++
		s2.Step()
	}

	n = 0

	for _, e := range *evs2 {
		if e.Kind == EventArea && e.Radius == 6 {
			n++
		}
	}

	if n != 3 { // 75, 50, 25
		t.Errorf("%d pulses with Param1 25, want 3", n)
	}
}
