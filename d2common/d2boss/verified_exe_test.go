package d2boss

import "testing"

// Pins of facts read from Game.exe 1.14b (docs/boss-encounters.md, verify-bosses).

// The throne wave table at 0x6e4ab8 holds the super unique hcIdx 61..65 and
// DATATBL_LoadSuperUniquesTable maps hcIdx -> first row; in SuperUniques.txt
// those are the rows "Baal Subject 1..5" = SuperBaalWave+0..4. The groups are
// MinGrp = MaxGrp of those rows.
func TestWaveTableMatchesExe(t *testing.T) {
	wantGroup := [5]int{5, 3, 5, 8, 5}
	wantClass := [5]int{62, 105, 121, 122, 135}

	for i, w := range Waves {
		if w.Super != SuperBaalWave+i || w.Group != wantGroup[i] || w.Class != wantClass[i] {
			t.Errorf("wave %d = %+v", i, w)
		}
	}
}

// The five seal flags are indexed by object 392..396 in order (FUN_005b3240).
func TestSealFlagsFollowObjectOrder(t *testing.T) {
	for i, obj := range []int{392, 393, 394, 395, 396} {
		if sealTable[i].Object != obj {
			t.Errorf("seal %d is object %d, want %d", i, sealTable[i].Object, obj)
		}
	}
}

// PurgeOnArrival (FUN_005b2e60) is off by default and adds one ActPurge.
func TestPurgeOnArrival(t *testing.T) {
	for _, on := range []bool{false, true} {
		m, _ := newTest()
		m.Encounter("diablo").(*Seals).PurgeOnArrival = on

		for _, o := range []int{ObjSealVizier, ObjSealPlainA, ObjSealDeSeis, ObjSealPlainB, ObjSealInfector} {
			m.Operate(Operate{Object: o})
		}

		m.Killed(Kill{Class: ClassVizier, Super: SuperVizier})
		m.Killed(Kill{Class: ClassDeSeis, Super: SuperDeSeis})
		as := m.Killed(Kill{Class: ClassInfector, Super: SuperInfector})

		purges := 0

		for _, a := range as {
			if a.Kind == ActPurge {
				purges++

				if a.Level != LevelChaos {
					t.Errorf("purge level %d", a.Level)
				}
			}
		}

		if (purges == 1) != on || purges > 1 {
			t.Errorf("PurgeOnArrival=%v: %d purge actions (%v)", on, purges, as)
		}
	}
}

// The lair warp gate (0x59b700): blocked until the staff is placed (inferred).
func TestLairWarpGate(t *testing.T) {
	m, _ := newTest()
	tomb := m.Encounter("duriel").(*Tomb)

	if !tomb.LairWarpBlocked() {
		t.Fatal("lair open before the staff")
	}

	m.Operate(Operate{Object: ObjOrifice, HasStaff: true})

	if tomb.LairWarpBlocked() {
		t.Fatal("lair still blocked with the staff placed")
	}
}
