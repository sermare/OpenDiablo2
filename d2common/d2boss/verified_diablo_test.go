package d2boss

import "testing"

// Pins of the second verification pass (Game.exe 1.14b): seal pairing, dummy offsets, Diablo's arrival, the lair gate.

func operateAll(m *Manager, o ...int) []Action {
	var all []Action

	for _, obj := range o {
		all = append(all, m.Operate(Operate{Object: obj, X: 1000, Y: 2000})...)
	}

	return all
}

// The default seal layout (0x5b4720/70/840, 0x5b3360): seal 392 -> hcIdx 36 Infector, 394 -> 37 De Seis, 396 -> 38 Vizier, each at the
// dummy offset from the seal; Seals.LegacyLayout keeps the earlier pairing and the seal position.
func TestSealPairingAndDummyOffsets(t *testing.T) {
	type want struct{ super, dx, dy int }

	exe := map[int]want{
		ObjSealVizier:   {SuperInfector, -0xc, -0x34},
		ObjSealDeSeis:   {SuperDeSeis, -0x27, 0x21},
		ObjSealInfector: {SuperVizier, 0x20, 0x10},
	}
	def := map[int]int{ObjSealVizier: SuperVizier, ObjSealDeSeis: SuperDeSeis, ObjSealInfector: SuperInfector}

	for _, on := range []bool{false, true} {
		for obj, w := range exe {
			m, _ := newTest()
			m.Encounter("diablo").(*Seals).LegacyLayout = !on

			as := operateAll(m, obj)
			if len(as) != 1 || as[0].Kind != ActSpawnMonster {
				t.Fatalf("on=%v seal %d: %v", on, obj, as)
			}

			if on && (as[0].Super != w.super || as[0].X != 1000+w.dx || as[0].Y != 2000+w.dy) {
				t.Errorf("exe seal %d: super %d at %d,%d, want %d at %d,%d", obj, as[0].Super, as[0].X, as[0].Y, w.super, 1000+w.dx, 2000+w.dy)
			}

			if !on && (as[0].Super != def[obj] || as[0].X != 1000 || as[0].Y != 2000) {
				t.Errorf("default seal %d changed: super %d at %d,%d", obj, as[0].Super, as[0].X, as[0].Y)
			}
		}
	}

	// the plain seals only set a flag in both layouts
	for _, obj := range []int{ObjSealPlainA, ObjSealPlainB} {
		m, _ := newTest()
		m.Encounter("diablo")

		if as := operateAll(m, obj); len(as) != 0 {
			t.Errorf("plain seal %d spawned %v", obj, as)
		}
	}

	if _, _, ok := SealDummyOffset(ObjSealPlainA); ok {
		t.Error("plain seal has a dummy offset")
	}
}

// Diablo's arrival (timer callback 0x5b2830): 1 frame timer + 10 frame count = 11 frames by default, 100 with LegacyLayout; the level
// stops populating itself after the arrival.
func TestDiabloArrivalDelay(t *testing.T) {
	for _, on := range []bool{false, true} {
		m, _ := newTest()
		s := m.Encounter("diablo").(*Seals)
		s.LegacyLayout = !on

		operateAll(m, ObjSealVizier, ObjSealPlainA, ObjSealDeSeis, ObjSealPlainB, ObjSealInfector)
		m.Killed(Kill{Class: ClassVizier, Super: SuperVizier})
		m.Killed(Kill{Class: ClassDeSeis, Super: SuperDeSeis})

		if s.SpawnsClosed() {
			t.Fatal("spawns closed before the arrival")
		}

		m.Killed(Kill{Class: ClassInfector, Super: SuperInfector})

		if s.SpawnsClosed() != on {
			t.Errorf("on=%v: SpawnsClosed = %v", on, s.SpawnsClosed())
		}

		delay := diabloDelay
		if on {
			delay = ExeDiabloDelay
		}

		if as := m.Tick(delay - 1); len(as) != 0 {
			t.Fatalf("on=%v: Diablo too early: %v", on, as)
		}

		as := m.Tick(1)
		if len(as) != 1 || as[0].Class != ClassDiablo {
			t.Fatalf("on=%v: Diablo did not arrive after %d frames: %v", on, delay, as)
		}
	}
}

// Default Tomb gate: the lair opens when the portal timer 0x59b450 has run (private +0xb = 1), not when the staff is placed.
func TestLairGateFollowsPortalTimer(t *testing.T) {
	m, _ := newTest()
	tomb := m.Encounter("duriel").(*Tomb)

	m.Operate(Operate{Object: ObjOrifice, HasStaff: true})

	if !tomb.LairWarpBlocked() {
		t.Fatal("lair open right after the staff")
	}

	m.Tick(tombPortalDelay)

	if tomb.LairWarpBlocked() {
		t.Fatal("lair still blocked after the portal appeared")
	}
}
