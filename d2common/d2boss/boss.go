package d2boss

import (
	"fmt"
	"sort"
)

// Levels (levels.txt).
const (
	LevelTombEntrance = 66 // Tal Rasha's Tomb 1..7 are 66..72: the true tomb is picked by the Horadric Staff orifice
	LevelDurielLair   = 73
	LevelDurance3     = 102
	LevelFortress     = 103
	LevelChaos        = 108
	LevelThrone       = 131
	LevelWorldstone   = 132
)

// Objects (objects.txt Id, VERIFIED unless noted).
const (
	ObjDurielPortal   = 100 // "Portal to Duriel's Lair" (OperateFn 43)
	ObjOrifice        = 152 // "Where you place the Horadric staff" (OperateFn 25)
	ObjMephistoBridge = 341 // "mephisto bridge" (OperateFn 4), quest object 0x155 in QUEST_OnObjectOperated
	ObjHellgate       = 342 // "hellgate" portal (OperateFn 46)
	ObjSealVizier     = 392 // boss seal (OperateFn 54)
	ObjSealPlainA     = 393 // plain seal (OperateFn 52)
	ObjSealDeSeis     = 394 // boss seal (OperateFn 55)
	ObjSealPlainB     = 395 // plain seal (OperateFn 52)
	ObjSealInfector   = 396 // boss seal (OperateFn 56)
	ObjDiabloStart    = 255 // "diablo start point"
	ObjWorldstone     = 563 // "The Worldstone Chamber" portal (OperateFn 70)
	ObjTownPortal     = 60  // permanent town portal (used for Duriel's exit, UNVERIFIED)
)

// Monster classes (monstats.txt row; for rows below the "Expansion" separator
// the exe class is the same number).
const (
	ClassDuriel   = 211
	ClassMephisto = 242
	ClassDiablo   = 243
	ClassTyrael   = 251
	ClassVizier   = 306 // StormCaster, super unique "Grand Vizier of Chaos"
	ClassDeSeis   = 312 // OblivionKnight, "Lord De Seis"
	ClassInfector = 362 // VenomLord, "Infector of Souls"
)

// Super unique rows (SuperUniques.txt, VERIFIED).
const (
	SuperInfector = 36
	SuperDeSeis   = 37
	SuperVizier   = 38
	SuperBaalWave = 62 // "Baal Subject 1" .. 66
)

// Delays, in game frames (25 per second). UNVERIFIED unless noted.
const (
	tombPortalDelay = 50  // staff placed -> portal appears
	tyraelDelay     = 200 // Duriel dead -> Tyrael (the exe: QUEST_AddTimer(8), unit unknown)
	hellgateDelay   = 300 // Mephisto dead -> red portal (QUEST_AddTimer(0xc), unit unknown)
	diabloDelay     = 100 // last seal boss dead -> Diablo arrives
	baalPortalDelay = 100 // morph -> the Worldstone portal
)

// ActionKind says what the engine has to do.
type ActionKind int

// The action kinds.
const (
	// ActSpawnMonster: create a monster of Class (or super unique Super).
	ActSpawnMonster ActionKind = iota
	// ActSpawnNPC: create a friendly NPC.
	ActSpawnNPC
	// ActSpawnObject: create the object Class (objects.txt Id).
	ActSpawnObject
	// ActObjectMode: set the mode of the object Class (animation).
	ActObjectMode
	// ActMessage: a speech / sound message (Msg).
	ActMessage
	// ActWake: wake the monster Class.
	ActWake
	// ActPortal: a portal to Level opens (Class is its object).
	ActPortal
	// ActPurge: every other living, non-pet monster of Level goes into death
	// mode (the record FUN_005b2e60 sends, mode 0; see Seals.PurgeOnArrival).
	ActPurge
	// ActDropItem: the item with the code Key drops at the encounter (Name
	// says why).
	ActDropItem
)

func (k ActionKind) String() string {
	return [...]string{"spawn-monster", "spawn-npc", "spawn-object", "object-mode", "message", "wake", "portal", "purge", "drop-item"}[k]
}

// Action is one request to the engine.
type Action struct {
	Kind   ActionKind
	Name   string // human name for the log
	Key    string // monstats id hint for classes past the Expansion separator ("baalthrone")
	Class  int    // monster class or object id
	Super  int    // super unique row, -1 for none
	Level  int
	Group  int  // followers to create with the monster
	Boss   bool // a boss encounter monster (SetBoss in the exe)
	Asleep bool
	Msg    int
	Note   string
	// X, Y are subtile coordinates; 0,0 means "where the encounter happens"
	// (the engine picks the spot next to the hero or the preset position).
	X, Y int
}

func (a Action) String() string {
	s := fmt.Sprintf("%s %s class=%d", a.Kind, a.Name, a.Class)
	if a.Super >= 0 && a.Kind == ActSpawnMonster {
		s += fmt.Sprintf(" super=%d", a.Super)
	}

	if a.Group > 0 {
		s += fmt.Sprintf(" group=%d", a.Group)
	}

	if a.Msg != 0 {
		s += fmt.Sprintf(" msg=%d", a.Msg)
	}

	if a.Note != "" {
		s += " (" + a.Note + ")"
	}

	return s
}

// Events the manager receives.
type (
	// Operate is the hero using an object.
	Operate struct {
		Object    int
		X, Y      int
		HasStaff  bool // the hero carries the assembled Horadric Staff
		HasCube   bool
		QuestOpen bool // the quest allows it (Guardian state)
	}
	// Kill is a monster death.
	Kill struct {
		Class int
		Super int // super unique row or -1
		Name  string
		Level int
	}
)

// Encounter is one boss encounter.
type Encounter interface {
	Name() string
	// State is a short label of the state machine for the log.
	State() string
	OnEnter(m *Manager, level int)
	OnOperate(m *Manager, e Operate)
	OnKill(m *Manager, k Kill)
	OnTick(m *Manager, frame int)
}

// Manager runs the encounters.
type Manager struct {
	// Log receives one line per decision ("BOSS ...").
	Log func(string)

	Level int
	Frame int
	// Hero is the hero's subtile position (for wake distances).
	HeroX, HeroY int

	encounters []Encounter
	out        []Action
	timers     []timer
}

type timer struct {
	due int
	fn  func()
}

// New creates a manager with all encounters.
func New(log func(string)) *Manager {
	m := &Manager{Log: log}
	m.Add(NewTomb(), NewMephisto(), NewSeals(), NewThrone())

	return m
}

// Add registers encounters.
func (m *Manager) Add(e ...Encounter) { m.encounters = append(m.encounters, e...) }

// Encounter returns the encounter by name.
func (m *Manager) Encounter(name string) Encounter {
	for _, e := range m.encounters {
		if e.Name() == name {
			return e
		}
	}

	return nil
}

func (m *Manager) logf(format string, args ...interface{}) {
	if m.Log != nil {
		m.Log("BOSS " + fmt.Sprintf(format, args...))
	}
}

// emit queues an action and logs it.
func (m *Manager) emit(enc string, a Action) {
	if a.Super == 0 && a.Kind != ActSpawnMonster {
		a.Super = -1
	}

	m.out = append(m.out, a)
	m.logf("%s action: %s", enc, a)
}

func (m *Manager) after(frames int, fn func()) {
	m.timers = append(m.timers, timer{due: m.Frame + frames, fn: fn})
}

func (m *Manager) take() []Action {
	out := m.out
	m.out = nil

	return out
}

// Enter is the hero arriving in a level.
func (m *Manager) Enter(level int) []Action {
	m.logf("level %d -> %d", m.Level, level)
	m.Level = level

	for _, e := range m.encounters {
		e.OnEnter(m, level)
	}

	return m.take()
}

// Operate is the hero using an object.
func (m *Manager) Operate(e Operate) []Action {
	for _, enc := range m.encounters {
		enc.OnOperate(m, e)
	}

	return m.take()
}

// Killed is a monster death.
func (m *Manager) Killed(k Kill) []Action {
	if k.Level == 0 {
		k.Level = m.Level
	}

	for _, enc := range m.encounters {
		enc.OnKill(m, k)
	}

	return m.take()
}

// Tick advances the clock by frames game frames and runs due timers.
func (m *Manager) Tick(frames int) []Action {
	for i := 0; i < frames; i++ {
		m.Frame++

		due := m.timers
		m.timers = nil

		sort.SliceStable(due, func(a, b int) bool { return due[a].due < due[b].due })

		for _, t := range due {
			if t.due <= m.Frame {
				t.fn()
			} else {
				m.timers = append(m.timers, t)
			}
		}

		for _, e := range m.encounters {
			e.OnTick(m, m.Frame)
		}
	}

	return m.take()
}

// States returns "name=state" for all encounters (for the log).
func (m *Manager) States() string {
	s := ""

	for i, e := range m.encounters {
		if i > 0 {
			s += " "
		}

		s += e.Name() + "=" + e.State()
	}

	return s
}

func (m *Manager) heroDist(x, y int) int {
	dx, dy := m.HeroX-x, m.HeroY-y
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dx > dy {
		return dx + dy/2
	}

	return dy + dx/2
}

// ThroneStep is the host side of the throne statue's think function: the
// engine's ThroneHost calls it with the step (StepAnnounce, StepSpawn,
// StepMorph) and the wave counter. The returned actions are the engine's job.
func (m *Manager) ThroneStep(step, wave int) ([]Action, bool) {
	for _, e := range m.encounters {
		if t, ok := e.(*Throne); ok {
			ok := t.Step(m, step, wave)

			return m.take(), ok
		}
	}

	return nil, false
}

// WaveCleared reports that the monsters of the last throne wave are dead.
func (m *Manager) WaveCleared() bool {
	for _, e := range m.encounters {
		if t, ok := e.(*Throne); ok {
			return t.WaveCleared()
		}
	}

	return true
}

// BaalLeft is called when the walking Baal enters the Worldstone portal.
func (m *Manager) BaalLeft() []Action {
	for _, e := range m.encounters {
		if t, ok := e.(*Throne); ok {
			t.BaalLeft(m)
		}
	}

	return m.take()
}

// WaveSpawned tells the throne how many monsters the engine really created for
// the current wave (it may create fewer than the table says).
func (m *Manager) WaveSpawned(n int) {
	for _, e := range m.encounters {
		if t, ok := e.(*Throne); ok {
			t.Alive = n
			m.logf("baal wave %d: %d monsters created", t.Wave, n)
		}
	}
}

// Emit queues an action for the engine on behalf of an encounter that lives
// in another package (the Pandemonium event, d2common/d2uber).
func (m *Manager) Emit(enc string, a Action) { m.emit(enc, a) }

// After runs fn after the given number of frames.
func (m *Manager) After(frames int, fn func()) { m.after(frames, fn) }

// Logf writes a "BOSS ..." line.
func (m *Manager) Logf(format string, args ...interface{}) { m.logf(format, args...) }

// Do runs fn (which may Emit actions) and returns the actions it queued.
func (m *Manager) Do(fn func()) []Action {
	fn()

	return m.take()
}
