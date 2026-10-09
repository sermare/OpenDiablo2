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
// VERIFIED in the second pass (feat/verify-diablo-hellforge, d2-re-notes/verify-diablo-hellforge.md):
//   - the boss seals are 392, 394, 396 (OperateFn 54, 55, 56 = 0x5b4720, 0x5b4770, 0x5b4840); each stores a position in the
//     quest data (+0x24, +0x2c, +0x34) = the seal's own coordinates plus a fixed offset, creates a "Dummy" object (131) there and then
//     runs the common seal code 0x5b3240. Operating the dummy (QUEST_OnObjectOperated -> 0x5b3360) matches its coordinates with the
//     three positions and spawns the super unique with hcIdx 36 (position 1, seal 392 = Infector of Souls), 37 (position 2, seal 394
//     = Lord De Seis) or 38 (position 3, seal 396 = Grand Vizier of Chaos) from the table at game root +0xae0 (+0xb28/2a/2c);
//   - the plain seals 393 and 395 (OperateFn 52 = 0x5b3240) only set their flag;
//   - Diablo: the "Dummy" object 255 (InitFn 55 = 0x5b31a0, placed in the level) records itself (private +6 = 1, +8 = its unit id);
//     when all five flags are set and the kill counter is 3 (checked in the seal code, the kill handler and that init), 0x5b2e60 runs
//     once and the quest timer 1 (callback 0x5b2830) is added with delay 1; the callback counts frames in +0x18 and, 10 frames later,
//     spawns Diablo (class 0xf3, 0x5b27b0) at the dummy 255 position (exact subtile, else radius 5, else radius 10), ORs 0x3000000
//     into his unit flags (+0xc4) and sets +0x11. After the arrival the level no longer populates itself (0x54ca00 asks 0x5b2e40).
//
// UNVERIFIED: which of the two plain seals belongs to which boss (it makes no difference in the exe), the cutscene/lock during the
// arrival (none found in the timer code; the 0x3000000 unit flags are not decoded), the delay of the dummy after the seal opens
// (objects.txt animation length), and the spawn of the minions around the seal bosses.
type Seals struct {
	// PurgeOnArrival makes the Diablo arrival emit ActPurge (off by default).
	// The exe sends every other living non-pet monster of level 0x6c a record
	// of mode 0 (FUN_005a5860(unit, 0) + MONAI_Proc_5a5630); mode 0 is the
	// death mode of the monster mode table, which 0x5a5630 special-cases
	// together with mode 0xc, so the remaining monsters are put to death.
	// VERIFIED: the loop and its filter; INFERRED: that mode 0 kills them.
	PurgeOnArrival bool

	// ExeLayout makes the encounter follow the exe's seal layout (off by default, the default keeps the earlier model): the boss of
	// seal 392 is the Infector, of 394 De Seis, of 396 the Vizier (VERIFIED, 0x5b4720/70/840 + 0x5b3360); the boss appears at the
	// dummy position (seal + SealDummyOffset) when the seal is operated, and Diablo arrives ExeDiabloDelay frames after the last
	// condition is met instead of diabloDelay.
	ExeLayout bool

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

// exeSealTable is the exe's pairing (VERIFIED, see the package comment): seal index -> boss.
var exeSealTable = [5]seal{
	{Object: ObjSealVizier, Name: "seal 1 (Infector of Souls)", Super: SuperInfector, Class: ClassInfector, Group: 9, Boss: "Infector of Souls"},
	{Object: ObjSealPlainA, Name: "seal 2 (plain)", Super: -1},
	{Object: ObjSealDeSeis, Name: "seal 3 (Lord De Seis)", Super: SuperDeSeis, Class: ClassDeSeis, Group: 5, Boss: "Lord De Seis"},
	{Object: ObjSealPlainB, Name: "seal 4 (plain)", Super: -1},
	{Object: ObjSealInfector, Name: "seal 5 (Grand Vizier of Chaos)", Super: SuperVizier, Class: ClassVizier, Group: 9, Boss: "Grand Vizier of Chaos"},
}

// ExeDiabloDelay is the exe's delay between the arrival conditions and Diablo's spawn: the timer callback 0x5b2830 is added with delay
// 1 and then counts 10 frames in the quest data (+0x18 < 10) before it spawns him (VERIFIED).
const ExeDiabloDelay = 11

// SealDummyOffset is the offset, in subtiles, from a boss seal object to the "Dummy" object (131) the exe creates for it and where the
// seal boss spawns (VERIFIED: add [data+0x24/0x28] -0xc/-0x34 in 0x5b4720, -0x27/+0x21 in 0x5b4770, +0x20/+0x10 in 0x5b4840).
// ok is false for the plain seals and other objects.
func SealDummyOffset(object int) (dx, dy int, ok bool) {
	switch object {
	case ObjSealVizier:
		return -0xc, -0x34, true
	case ObjSealDeSeis:
		return -0x27, 0x21, true
	case ObjSealInfector:
		return 0x20, 0x10, true
	}

	return 0, 0, false
}

// SpawnsClosed reports the exe's rule that, once the arrival has run (quest data +0x13, FUN_005b2e40), the Chaos Sanctuary no longer
// populates itself with random monsters (FUN_0054ca00 refuses level 108). Only meaningful with ExeLayout.
func (s *Seals) SpawnsClosed() bool { return s.ExeLayout && s.arrived }

func (s *Seals) table() *[5]seal {
	if s.ExeLayout {
		return &exeSealTable
	}

	return &sealTable
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
	for i, sl := range s.table() {
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
			x, y, note := o.X, o.Y, "seal boss at the seal"

			if dx, dy, ok := SealDummyOffset(o.Object); ok && s.ExeLayout && (x != 0 || y != 0) {
				x, y, note = x+dx, y+dy, "seal boss at the dummy object next to the seal"
			}

			m.emit("diablo", Action{Kind: ActSpawnMonster, Name: sl.Boss, Class: sl.Class, Super: sl.Super, Group: sl.Group,
				Boss: true, X: x, Y: y, Note: note})
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

	for _, sl := range s.table() {
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

	delay := diabloDelay
	if s.ExeLayout {
		delay = ExeDiabloDelay
	}

	m.after(delay, func() {
		s.diabloAlive = true
		m.emit("diablo", Action{Kind: ActSpawnMonster, Name: "Diablo", Class: ClassDiablo, Super: -1, Boss: true,
			Note: "at the diablo start point (object 255, exact subtile, else radius 5, 10)"})
	})
}

// OnTick implements Encounter.
func (s *Seals) OnTick(*Manager, int) {}
