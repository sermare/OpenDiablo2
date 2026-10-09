package d2boss

import "fmt"

// Seals is the Chaos Sanctuary (level 108) of act 4, quest A4Q2 "Terror's End"
// (id 23, slot 26).
//
// VERIFIED (quest node init 0x5b30d0, 0x4c bytes of private data;
// QUEST_A4_TerrorsEnd_OnMonsterKilled 0x5b2f10, FUN_005b2e60, FUN_005b3360):
//   - five seals, kept as five flags (private +0xc..+0x10);
//   - three seal bosses: the quest private data holds three positions and three
//     super unique ids (game table +0xb28/+0xb2a/+0xb2c); operating the seal
//     at one of the positions spawns the matching super unique there (the
//     super uniques are Grand Vizier of Chaos, Lord De Seis and Infector of
//     Souls: SuperUniques.txt rows 38, 37, 36 with groups of 9, 5 and 9);
//   - a kill counter (+0x48): when it reaches 3 and all five flags are set the
//     one-shot "arrival" (private +0x13) runs: every other monster of the level
//     (not Diablo, not dead, not a pet) is sent a record (FUN_005a5860/5a5630,
//     meaning not determined) and the quest timer 1 starts;
//   - a kill of class 0xf3 (Diablo) sets private +0x14 (Diablo dead), stores his
//     room, plays the speech 0x40 (not shown) and completes the quest.
//
// UNVERIFIED: that the two plain seals (objects 393, 395) belong to the
// Vizier and the Infector (the usual game layout: Vizier two seals, De Seis
// one, Infector two), that the boss seals are 392/394/396, and that the
// seal operate functions (52, 54, 55, 56) only open the seal; the delay before
// Diablo; his position (the "diablo start point" object, 255).
type Seals struct {
	// PurgeOnArrival makes the Diablo arrival emit ActPurge (off by default).
	// The exe sends every other living non-pet monster of level 0x6c a record
	// of mode 0 (FUN_005a5860(unit, 0) + MONAI_Proc_5a5630); mode 0 is the
	// death mode of the monster mode table, which 0x5a5630 special-cases
	// together with mode 0xc, so the remaining monsters are put to death.
	// VERIFIED: the loop and its filter; INFERRED: that mode 0 kills them.
	PurgeOnArrival bool

	open        [5]bool
	killed      map[int]bool // seal boss super unique rows that died
	spawned     map[int]bool
	diabloAlive bool
	arrived     bool
	diabloDead  bool
}

type seal struct {
	Object int
	Name   string
	Super  int // super unique row spawned when this seal opens, -1 for a plain seal
	Class  int
	Group  int
	Boss   string
}

var sealTable = [5]seal{
	{Object: ObjSealVizier, Name: "seal 1 (Grand Vizier of Chaos)", Super: SuperVizier, Class: ClassVizier, Group: 9, Boss: "Grand Vizier of Chaos"},
	{Object: ObjSealPlainA, Name: "seal 2 (Vizier's second)", Super: -1},
	{Object: ObjSealDeSeis, Name: "seal 3 (Lord De Seis)", Super: SuperDeSeis, Class: ClassDeSeis, Group: 5, Boss: "Lord De Seis"},
	{Object: ObjSealPlainB, Name: "seal 4 (Infector's first)", Super: -1},
	{Object: ObjSealInfector, Name: "seal 5 (Infector of Souls)", Super: SuperInfector, Class: ClassInfector, Group: 9, Boss: "Infector of Souls"},
}

// NewSeals creates the encounter.
func NewSeals() *Seals {
	return &Seals{killed: map[int]bool{}, spawned: map[int]bool{}}
}

// Name implements Encounter.
func (s *Seals) Name() string { return "diablo" }

// State implements Encounter.
func (s *Seals) State() string {
	switch {
	case s.diabloDead:
		return "diablo-dead"
	case s.diabloAlive:
		return "diablo-alive"
	case s.arrived:
		return "arriving"
	}

	return fmt.Sprintf("seals-%d-of-5-bosses-%d-of-3", s.count(), len(s.killed))
}

// OnEnter implements Encounter.
func (s *Seals) OnEnter(m *Manager, level int) {
	if level == LevelChaos {
		m.logf("diablo trigger: Chaos Sanctuary entered (5 seals, 3 seal bosses)")
	}
}

// OnOperate: a seal is used.
func (s *Seals) OnOperate(m *Manager, o Operate) {
	for i, sl := range sealTable {
		if sl.Object != o.Object {
			continue
		}

		if s.open[i] {
			return
		}

		s.open[i] = true
		m.logf("diablo trigger: %s opened (%d/5)", sl.Name, s.count())

		if sl.Super >= 0 && !s.spawned[sl.Super] {
			s.spawned[sl.Super] = true
			m.emit("diablo", Action{Kind: ActSpawnMonster, Name: sl.Boss, Class: sl.Class, Super: sl.Super, Group: sl.Group,
				Boss: true, X: o.X, Y: o.Y, Note: "seal boss at the seal"})
		}

		s.maybeArrive(m)

		return
	}
}

func (s *Seals) count() int {
	n := 0

	for _, o := range s.open {
		if o {
			n++
		}
	}

	return n
}

// OnKill: seal bosses count, Diablo ends the quest.
func (s *Seals) OnKill(m *Manager, k Kill) {
	if k.Class == ClassDiablo {
		s.diabloAlive, s.diabloDead = false, true
		m.logf("diablo killed -> %s", s.State())

		return
	}

	for _, sl := range sealTable {
		if sl.Super >= 0 && k.Super == sl.Super && !s.killed[sl.Super] {
			s.killed[sl.Super] = true
			m.logf("diablo trigger: seal boss %s dead (%d/3)", sl.Boss, len(s.killed))
			s.maybeArrive(m)

			return
		}
	}
}

// maybeArrive runs the one-shot arrival when all five seals are open and the
// three seal bosses are dead (FUN_005b2e60's guard).
func (s *Seals) maybeArrive(m *Manager) {
	if s.arrived || len(s.killed) < 3 || s.count() < 5 {
		return
	}

	s.arrived = true
	m.logf("diablo trigger: all 5 seals open and 3 seal bosses dead -> Diablo is summoned")
	m.emit("diablo", Action{Kind: ActMessage, Name: "Diablo summoned", Note: "the remaining monsters of the level are sent a record (mode 0, inferred death)"})

	if s.PurgeOnArrival {
		m.emit("diablo", Action{Kind: ActPurge, Name: "remaining monsters of the Chaos Sanctuary", Level: LevelChaos,
			Note: "FUN_005b2e60: mode 0 record to every living non-pet monster except class 0xf3"})
	}

	m.after(diabloDelay, func() {
		s.diabloAlive = true
		m.emit("diablo", Action{Kind: ActSpawnMonster, Name: "Diablo", Class: ClassDiablo, Super: -1, Boss: true,
			Note: "at the diablo start point (object 255), delay unverified"})
	})
}

// OnTick implements Encounter.
func (s *Seals) OnTick(*Manager, int) {}
