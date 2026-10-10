package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// Glacial Spike (hit function 13) splashes: the cast gives the missile an
// area radius of Param1 + Param2*(lvl-1) and the sim fires one area event.
func TestGlacialSpikeSplashRadius(t *testing.T) {
	for _, tc := range []struct {
		name        string
		p1, p2, lvl int
		wantRadius  int
	}{
		{"table row (4, 0)", 4, 0, 1, 4},
		{"table row at 20", 4, 0, 20, 4},
		{"per-level growth", 3, 2, 5, 11},
	} {
		f := newFixture(map[string]int{"Gl": tc.lvl})
		f.reg.Add(&Skill{ID: 800, Name: "Gl", SrvMissile: "glacialspike", ManaShift: 8,
			DamageSpec: DamageSpec{EType: "cold", EMin: 1, EMax: 1}, Params: [9]int{0, tc.p1, tc.p2}})
		f.u.levels["Gl"] = tc.lvl
		f.p.Missiles.(missileTable)["glacialspike"] = &d2missile.Spec{ID: 801, Name: "glacialspike", SrvDoFunc: 1,
			SrvHitFunc: 13, Vel: 16, MaxVel: 16, Range: 40, CollideType: 3, CollideKill: true}
		f.sim.Table = f.p.Missiles

		var radii []int

		f.sim.OnEvent = func(e d2missile.Event) {
			if e.Kind == d2missile.EventArea {
				radii = append(radii, e.Radius)
			}
		}

		if st, r := f.cast("Gl", 60, 0); !st.OK || !r.OK || len(r.Missiles) != 1 {
			t.Fatalf("%s: cast %+v %+v", tc.name, st, r)
		}

		for i := 0; i < 60; i++ {
			f.w.frame++
			f.sim.Step()
		}

		if len(radii) != 1 || radii[0] != tc.wantRadius {
			t.Errorf("%s: area radii %v, want [%d]", tc.name, radii, tc.wantRadius)
		}
	}
}
