package d2boss

import "fmt"

// Throne is the Baal encounter in the Throne of Destruction (level 131) and
// the Worldstone Chamber (132), quest A5Q6 "Eve of Destruction" (id 36, slot
// 40).
//
// The wave cycle itself is the think function of the throne statue
// (d2monster.thinkBaalThrone, VERIFIED): announce (message + Baal Corpse
// Explode on the throne), 250 frames, spawn the wave (Baal Monster Spawn),
// 100 frames, announce the next; five waves (counter 0..4); with the counter
// above 4 the throne morphs. This type is the part of it the engine hosts:
// it receives the steps, composes the waves and keeps the bookkeeping.
//
// VERIFIED wave composition (SuperUniques.txt "Baal Subject 1..5", rows 62..66):
// the leader is the super unique of the row, followed by its group of the same
// class: classes 62 (5), 105 (3), 557 (5), 558 (8), 571 (5) (the Class column resolved
// to the hcIdx of MonStats.txt; an earlier version had 121/122/135, which are
// unrelated rows); the exe throne
// uses the wave counter as the index.
//
// VERIFIED after the last wave: the throne turns into class 0x22f (row 560
// "Baal Crab to Stairs") whose AI walks to the Worldstone Chamber portal,
// object 563, and leaves the level when within aip1 of it (d2monster
// thinkBaalToStairs).
//
// UNVERIFIED: that the next wave waits for the previous to be killed (the
// engine gates on it; the exe think function does not), the wave messages, the
// moment the portal object appears (taken as 100 frames after the morph), the
// fight in the Worldstone Chamber (Baal's own AI is d2monster.thinkBaalCrab,
// the class that dies in the chamber is the row 545 "Baal Crab").
type Throne struct {
	state     throneState
	Wave      int // waves sent so far
	Alive     int // monsters of the last wave still alive
	waveClass int
	portalUp  bool
	baalDead  bool
}

type throneState int

const (
	throneIdle throneState = iota
	throneWaiting
	throneWaves
	throneMorphed
	throneBaalGone
	throneBaalDead
)

var throneNames = [...]string{"idle", "waiting", "waves", "morphed", "baal-in-worldstone-chamber", "baal-dead"}

// Wave is one wave of the throne.
type Wave struct {
	Super int
	Class int
	Group int
	Name  string
}

// Waves are the five waves (VERIFIED from SuperUniques.txt, the names are the
// classes' monstats names).
var Waves = [5]Wave{
	{SuperBaalWave + 0, 62, 5, "Baal Subject 1 (WarpedShaman)"},
	{SuperBaalWave + 1, 105, 3, "Baal Subject 2 (BaalMummy)"},
	{SuperBaalWave + 2, 557, 5, "Baal Subject 3 (BaalHighPriest)"},
	{SuperBaalWave + 3, 558, 8, "Baal Subject 4 (VenomLord)"},
	{SuperBaalWave + 4, 571, 5, "Baal Subject 5 (BaalMinion1)"},
}

// Throne steps (the values of d2monster.ThroneStep).
const (
	StepAnnounce = 0
	StepSpawn    = 1
	StepMorph    = 2
)

// NewThrone creates the encounter.
func NewThrone() *Throne { return &Throne{waveClass: -1} }

// Name implements Encounter.
func (t *Throne) Name() string { return "baal" }

// State implements Encounter.
func (t *Throne) State() string {
	return fmt.Sprintf("%s-wave-%d-alive-%d", throneNames[t.state], t.Wave, t.Alive)
}

// OnEnter: the throne room places Baal's throne; the chamber is a plain level.
func (t *Throne) OnEnter(m *Manager, level int) {
	switch level {
	case LevelThrone:
		if t.state == throneIdle {
			t.state = throneWaiting
			m.logf("baal trigger: Throne of Destruction entered -> state %s", t.State())
			m.emit("baal", Action{Kind: ActSpawnMonster, Name: "Baal Throne", Class: 543, Key: "baalthrone", Super: -1,
				Boss: true, Note: "the throne statue runs the waves"})
		}
	case LevelWorldstone:
		if t.state == throneBaalGone {
			m.logf("baal trigger: Worldstone Chamber entered: Baal is here")
			m.emit("baal", Action{Kind: ActSpawnMonster, Name: "Baal", Class: 544, Key: "baalcrab", Super: -1, Boss: true,
				Note: "the final fight"})
		}
	}
}

// Step is the host side of the throne think function.
func (t *Throne) Step(m *Manager, step, wave int) bool {
	switch step {
	case StepAnnounce:
		if wave < len(Waves) {
			t.state = throneWaves
			m.logf("baal wave %d announced: %s", wave+1, Waves[wave].Name)
			m.emit("baal", Action{Kind: ActMessage, Name: fmt.Sprintf("wave %d announced", wave+1), Note: "message id unverified"})
		} else {
			m.logf("baal: no more waves (counter %d); the next step morphs the throne", wave)
		}

		return true
	case StepSpawn:
		if wave >= len(Waves) {
			return false
		}

		w := Waves[wave]
		t.Wave = wave + 1
		t.Alive = 1 + w.Group
		t.waveClass = w.Class
		m.logf("baal wave %d spawned: %s leader+%d (total %d)", wave+1, w.Name, w.Group, t.Alive)
		m.emit("baal", Action{Kind: ActSpawnMonster, Name: w.Name, Class: w.Class, Super: w.Super, Group: w.Group,
			Note: fmt.Sprintf("wave %d", wave+1)})

		return true
	case StepMorph:
		if t.state == throneMorphed {
			return true
		}

		t.state = throneMorphed
		m.logf("baal: the throne morphs into Baal (exe class 0x22f = Baal Crab to Stairs)")
		m.emit("baal", Action{Kind: ActSpawnMonster, Name: "Baal Crab to Stairs", Class: 559, Key: "baalcrabtostairs", Super: -1,
			Boss: true, Note: "replaces the throne"})
		m.after(baalPortalDelay, func() {
			t.portalUp = true
			m.emit("baal", Action{Kind: ActSpawnObject, Name: "The Worldstone Chamber portal", Class: ObjWorldstone,
				Level: LevelWorldstone, Note: "moment unverified"})
		})

		return true
	}

	return false
}

// WaveCleared reports that the last wave is dead (d2monster.WaveGate).
func (t *Throne) WaveCleared() bool { return t.Alive == 0 }

// BaalLeft is called when the walking Baal enters the portal.
func (t *Throne) BaalLeft(m *Manager) {
	if t.state != throneMorphed {
		return
	}

	t.state = throneBaalGone
	m.logf("baal enters the Worldstone Chamber portal -> state %s", t.State())
	m.emit("baal", Action{Kind: ActPortal, Name: "portal to the Worldstone Chamber", Class: ObjWorldstone, Level: LevelWorldstone})
}

// OnKill counts the wave and watches for Baal.
func (t *Throne) OnKill(m *Manager, k Kill) {
	if t.Alive > 0 && k.Class == t.waveClass && k.Level == LevelThrone {
		t.Alive--
		m.logf("baal wave %d: %d left", t.Wave, t.Alive)

		if t.Alive == 0 {
			m.logf("baal wave %d cleared", t.Wave)
		}

		return
	}

	if (k.Class == 544 || k.Name == "Baal") && (k.Level == LevelWorldstone || t.state == throneBaalGone) {
		t.state = throneBaalDead
		t.baalDead = true
		m.logf("baal killed -> state %s", t.State())
	}
}

// OnOperate implements Encounter.
func (t *Throne) OnOperate(*Manager, Operate) {}

// OnTick implements Encounter.
func (t *Throne) OnTick(*Manager, int) {}
