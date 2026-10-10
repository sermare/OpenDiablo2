package d2records

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
)

// Real-table (D2_TABLES) checks of the last eleven skills whose missile
// functions were read from the exe: Frozen Orb's nova (do16), Howl (hit17),
// Shout / Battle Cry (hit18 / 21 behind SRVDO_068's nova ring), Fist of the
// Heavens (hit22), Rabies (do30 / hit53 behind SRVDO_121) and Blade Sentinel
// (do20 / hit37). The hero (clvl 30) stands at (50, 50); the enemy (clvl 20)
// at (55, 50).

// novaPair applies SrvDoFunc 16's step ((a-b)/2, (a+b)/2) n times.
func novaPair(a, b, n int) (int, int) {
	for i := 0; i < n; i++ {
		a, b = (a-b)/2, (a+b)/2
	}

	return a, b
}

func TestRealFrozenOrbNovaCurls(t *testing.T) {
	r := realCast(t, "Frozen Orb", 20)

	// frozenorbnova: Param1 6 ("frames"), Param2 2 ("frequency"); frozenorb: sHitPar1 4
	var born [][2]int32

	r.sim.OnEvent = func(e d2missile.Event) {
		r.evs = append(r.evs, e)

		if e.Kind == d2missile.EventCreate && e.Missile.Spec.Name == "frozenorbnova" {
			born = append(born, [2]int32{int32(e.Missile.Data28), int32(e.Missile.Data2C)})
		}
	}

	r.step(34) // frozenorb Range 30: it ends and releases the ring

	novas := r.created("frozenorbnova")
	if len(novas) != 16 {
		t.Fatalf("%d novas, want 16 (64 table entries, stride sHitPar1 4)", len(novas))
	}

	if sp := novas[0].Spec; sp.SrvDoFunc != 16 || sp.Param[0] != 6 || sp.Param[1] != 2 {
		t.Fatalf("nova spec do%d params %v", sp.SrvDoFunc, sp.Param)
	}

	if len(r.created("frozenorbbolt")) != 30 { // Param1 1: a bolt every frame of the orb's 30
		t.Errorf("%d bolts, want 30", len(r.created("frozenorbbolt")))
	}

	r.step(20)

	for i, n := range novas {
		// elapsed 0, 2, 4 turn it: three applications, then it flies straight
		wa, wb := novaPair(int(born[i][0]), int(born[i][1]), 3)

		if int32(n.Data28) != int32(wa) || int32(n.Data2C) != int32(wb) {
			t.Errorf("nova %d heading %d,%d, want %d,%d after three turns", i, int32(n.Data28), int32(n.Data2C), wa, wb)
		}
	}
}

func TestRealHowlFrightensWeakMonsters(t *testing.T) {
	r := realCast(t, "Howl", 20)
	r.step(20)

	// Param5 75 + (20-1) * Param6 25 frames; the enemy (clvl 20) is below 1 + 30 + 20
	want := "foe:terror:550"

	if len(r.states) == 0 || r.states[0] != want {
		t.Fatalf("states %v, want %s first", r.states, want)
	}

	for _, e := range r.evs {
		if e.Kind == d2missile.EventState && e.Distance != 24+5*19 {
			t.Errorf("flee distance %d, want %d (Param3 24 + 19 * Param4 5)", e.Distance, 24+5*19)
		}
	}
}

func TestRealShoutFamilyRings(t *testing.T) {
	// the ring is the same 64 missiles as Howl's; Shout buffs allies only (the
	// audit world has none), Battle Cry debuffs the enemy for ln12 = 300 + 19 * 60
	shout := realCast(t, "Shout", 20)
	if len(shout.res.Missiles) != 64 {
		t.Fatalf("shout: %d missiles", len(shout.res.Missiles))
	}

	shout.step(20)

	if len(shout.states) != 0 {
		t.Errorf("shout buffed an enemy: %v", shout.states)
	}

	selfStates := 0

	for _, e := range shout.res.Effects {
		if e.Kind == "self_state" {
			selfStates++
		}
	}

	if selfStates != 1 {
		t.Errorf("shout: %d self states", selfStates)
	}

	bc := realCast(t, "Battle Cry", 20)
	if len(bc.res.Missiles) != 64 {
		t.Fatalf("battle cry: %d missiles", len(bc.res.Missiles))
	}

	bc.step(20)

	if want := "foe:battlecry:1440"; len(bc.states) == 0 || bc.states[0] != want {
		t.Errorf("battle cry states %v, want %s first", bc.states, want)
	}

	for _, e := range bc.res.Effects {
		if e.Kind == "self_state" {
			t.Errorf("battle cry has no aurastate but gave %+v", e)
		}
	}

	for _, name := range []string{"Battle Orders", "Battle Command"} {
		o := realCast(t, name, 20)
		if len(o.res.Missiles) != 64 {
			t.Errorf("%s: %d missiles", name, len(o.res.Missiles))
		}
	}
}

func TestRealFistOfTheHeavens(t *testing.T) {
	r := realCast(t, "Fist of the Heavens", 20)

	if len(r.res.Missiles) != 1 || r.res.Missiles[0].Spec.Name != "fistoftheheavensdelay" ||
		r.res.Missiles[0].Spec.SrvHitFunc != 22 || r.res.Missiles[0].Mark == nil {
		t.Fatalf("cast %v", r.res.Missiles)
	}

	r.step(12) // Range 10

	struck := 0

	for _, e := range r.evs {
		if e.Kind == d2missile.EventHit && e.Missile.Spec.Name == "fistoftheheavensdelay" {
			struck++

			// docs/skills-coverage.md: elem 675.0-725.0 at level 20
			if lo, hi := e.Damage.Lightning, e.Damage.Lightning; lo < 675<<8 || hi > 725<<8 {
				t.Errorf("lightning %d, want 675..725 points", lo>>8)
			}
		}
	}

	if struck != 1 {
		t.Errorf("target struck %d times, want 1", struck)
	}
}

func TestRealRabiesBiteInfects(t *testing.T) {
	r := realCast(t, "Rabies", 20)

	if r.res.Melee == nil || !r.res.Melee.Hit {
		t.Fatalf("no bite: %+v", r.res.Melee)
	}

	// ELen 100 + 19 * 10 = 290 frames; the audit foe cannot own the plague
	if len(r.states) != 1 || r.states[0] != "foe:rabies:290" {
		t.Errorf("states %v, want foe:rabies:290", r.states)
	}
}

func TestRealBladeCreeperStaysWhereItIsCast(t *testing.T) {
	rm := loadRealRecords(t)
	reg := rm.SkillTable()
	sk := reg.ByName("Blade Sentinel")
	r := realCast(t, "Blade Sentinel", 20)

	if sk == nil {
		t.Fatal("no Blade Sentinel")
	}

	frame := 0
	r.p.Frame = func() int { return frame }

	m := r.p.CastTrap(r.u, sk.ID, "blade creeper", 60, 40, d2skill.Target{X: 61, Y: 40})
	if m == nil {
		t.Fatal("no blade creeper")
	}

	if m.Spec.SrvDoFunc != 20 || m.Spec.SrvHitFunc != 37 {
		t.Fatalf("blade creeper do%d hit%d", m.Spec.SrvDoFunc, m.Spec.SrvHitFunc)
	}

	step := func(n int) {
		for i := 0; i < n; i++ {
			frame++
			r.sim.Step()
		}
	}

	step(5)

	if x, y := int(m.X), int(m.Y); x != 60 || y != 40 {
		t.Errorf("at (%d,%d), want to stay at the trap (60,40)", x, y)
	}

	step(m.Total + 5)

	if !m.Dead() {
		t.Error("the blade creeper outlived its table lifetime ", m.Total)
	}
}
