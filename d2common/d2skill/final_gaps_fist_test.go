package d2skill

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

func TestFistOfTheHeavensMarksTheTargetAndStrikesAtTheEnd(t *testing.T) {
	f := newFixture(map[string]int{"Fist of the Heavens": 10})
	f.u.base = f.u.levels
	f.addSkill(row{"skill": "Fist of the Heavens", "Id": "121", "srvdofunc": "80", "srvmissilea": "fistoftheheavensdelay",
		"aurafilter": "42375", "aurarangecalc": "20", "calc4": "3", "minmana": "1", "manashift": "8", "mana": "25",
		"HitShift": "8", "EType": "ltng", "EMin": "150", "EMax": "200"})
	f.addMissile(&d2missile.Spec{ID: 233, Name: "fistoftheheavensdelay", SrvDoFunc: 1, SrvHitFunc: 22, Range: 10, AlwaysExplode: true,
		Explosion: true, HitSubMissile: [4]string{"fistoftheheavensbolt"}})
	f.addMissile(&d2missile.Spec{ID: 234, Name: "fistoftheheavensbolt", SrvDoFunc: 1, SrvHitFunc: 7, Vel: 12, MaxVel: 12, Range: 50,
		CollideType: 3})

	mark := &testTarget{id: "mark", alive: true, level: 1, x: 10, y: 0}
	f.w.targets = []*testTarget{mark, {id: "a", alive: true, level: 1, x: 12, y: 0}, {id: "b", alive: true, level: 1, x: 14, y: 3},
		{id: "c", alive: true, level: 1, x: 16, y: 0}, {id: "d", alive: true, level: 1, x: 18, y: 0},
		{id: "far", alive: true, level: 1, x: 90, y: 0}}

	tg := Target{Unit: mark, UX: 10, UY: 0, X: 10, Y: 0}
	if st := f.p.Start(f.u, f.id("Fist of the Heavens"), tg); !st.OK {
		t.Fatalf("start refused: %q", st.Reason)
	}

	do := f.p.Do(f.u, f.id("Fist of the Heavens"), tg)
	if !do.OK || len(do.Missiles) != 1 {
		t.Fatalf("do ok=%v reason=%q missiles=%d", do.OK, do.Reason, len(do.Missiles))
	}

	if do.Missiles[0].Mark != d2missile.Target(mark) {
		t.Error("the delay missile must mark the target unit")
	}

	for i := 0; i < 12; i++ {
		f.w.frame++
		f.sim.Step()
	}

	struck, bolts := 0, 0

	for _, e := range f.evs {
		switch {
		case e.Kind == d2missile.EventHit && e.Missile.Spec.Name == "fistoftheheavensdelay" && e.Target.ID() == "mark" &&
			e.Damage.Lightning > 0:
			struck++
		case e.Kind == d2missile.EventCreate && e.Missile.Spec.Name == "fistoftheheavensbolt":
			bolts++
		}
	}

	if struck != 1 {
		t.Errorf("the marked unit was struck %d times, want 1", struck)
	}

	if bolts != 3 { // calc4 = 3 holy bolts at the enemies within aurarangecalc 20 (the far one is out of range)
		t.Errorf("%d holy bolts, want 3", bolts)
	}

	// no target unit, no fist
	f.u.cooldowns = map[int]int{}
	f.u.mana = 1000 << 8

	if do := f.p.Do(f.u, f.id("Fist of the Heavens"), Target{X: 10, Y: 0}); do.OK {
		t.Error("a cast without a target unit must fail")
	}
}
