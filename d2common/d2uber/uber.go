// Package d2uber is the Pandemonium event of Lord of Destruction (the "Uber
// Tristram" quest line added in 1.11): three keys open a portal to one of three
// uber areas, whose bosses drop the three organs, and the organs open the red
// portal to Tristram, where Uber Mephisto, Uber Diablo and Uber Baal wait.
//
// It is an encounter set for the d2boss Manager: the engine reports the level
// entered and the monsters killed, and receives the same Actions as for the
// story bosses (spawn a monster, open a portal, drop an item).
//
// Evidence (tables of the 1.14b patch_d2 data, read from the owner's install):
//   - VERIFIED: monstats rows ubermephisto, uberdiablo, uberizual, uberandariel
//     (display name Lilith), uberduriel, uberbaal exist with exe classes 704,
//     705, 706, 707, 708, 709, level 110 in all difficulties, their AI names
//     (UberMephisto, UberDiablo, UberIzual, Andariel, Duriel, UberBaal) and
//     skill lists; the treasure classes "Uber Andariel", "Uber Duriel" and
//     "Uber Izual" each hold exactly one organ (dhn, bey, mbr); levels.txt rows
//     133, 134, 135 ("Pandemonium 1..3") and 136 ("Pandemonium Finale"); the
//     CubeMain.txt rows that open the portals (see d2cube).
//   - UNVERIFIED (community knowledge, nothing in the data): which of the three
//     areas a key set opens (modelled as an injected choice), the keys' drop
//     sources (Hell Andariel, Duriel and Mephisto), the order and delays in
//     which the three Tristram bosses arrive, the Standard of Heroes (item
//     code "std", not in the tables) and the red portal object.
package d2uber

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2cube"
)

// Levels (levels.txt).
const (
	LevelMatronsDen     = 133 // Lilith
	LevelForgottenSands = 134 // Uber Duriel
	LevelFurnaceOfPain  = 135 // Uber Izual
	LevelTristram       = 136 // "Pandemonium Finale": Uber Mephisto, Diablo, Baal
	LevelHarrogath      = 109 // where the red portal of Tristram leads back to
)

// Monster classes (exe numbering = monstats hcIdx, VERIFIED).
const (
	ClassUberMephisto = 704
	ClassUberDiablo   = 705
	ClassUberIzual    = 706
	ClassLilith       = 707 // monstats id "uberandariel"
	ClassUberDuriel   = 708
	ClassUberBaal     = 709
)

// Normal-game bosses that drop the keys (UNVERIFIED), by exe class.
const (
	classAndariel = 156
	classDuriel   = 211
	classMephisto = 242
)

// Item codes the event drops besides the organs and keys of d2cube.
const (
	CodeStandardOfHeroes = "std" // UNVERIFIED: not in the tables of the owner's install
)

// Delays in game frames (25 per second), UNVERIFIED.
const (
	diabloDelay = 75
	baalDelay   = 150
	portalDelay = 50
)

// Area is one uber area.
type Area struct {
	Level int
	Name  string
	Boss  Boss
	Drop  string // the organ the boss carries (treasure class, VERIFIED)
}

// Boss is a spawnable uber boss (monstats row).
type Boss struct {
	Key   string // monstats id
	Name  string
	Class int
	AI    string
}

// Bosses of the three areas and of Tristram.
//
//nolint:gochecknoglobals // static lookup data
var (
	Lilith       = Boss{"uberandariel", "Lilith", ClassLilith, "Andariel"}
	UberDuriel   = Boss{"uberduriel", "Uber Duriel", ClassUberDuriel, "Duriel"}
	UberIzual    = Boss{"uberizual", "Uber Izual", ClassUberIzual, "UberIzual"}
	UberMephisto = Boss{"ubermephisto", "Uber Mephisto", ClassUberMephisto, "UberMephisto"}
	UberDiablo   = Boss{"uberdiablo", "Uber Diablo", ClassUberDiablo, "UberDiablo"}
	UberBaal     = Boss{"uberbaal", "Uber Baal", ClassUberBaal, "UberBaal"}

	// Areas are the keys' destinations.
	Areas = []Area{
		{LevelMatronsDen, "Matron's Den", Lilith, d2cube.CodeHorn},
		{LevelForgottenSands, "Forgotten Sands", UberDuriel, d2cube.CodeBaalEye},
		{LevelFurnaceOfPain, "Furnace of Pain", UberIzual, d2cube.CodeMephBrain},
	}
)

// AreaOf returns the uber area of a level.
func AreaOf(level int) (Area, bool) {
	for _, a := range Areas {
		if a.Level == level {
			return a, true
		}
	}

	return Area{}, false
}

// KeyDrop returns the key a normal-game boss drops in Hell (UNVERIFIED).
func KeyDrop(class, difficulty int) (string, bool) {
	if difficulty < 2 {
		return "", false
	}

	switch class {
	case classAndariel:
		return d2cube.CodeKeyTerror, true
	case classDuriel:
		return d2cube.CodeKeyHate, true
	case classMephisto:
		return d2cube.CodeKeyDestr, true
	}

	return "", false
}

// Event is the Pandemonium event; it implements d2boss.Encounter.
type Event struct {
	// Pick chooses the uber area a key set opens (n = number of areas); the
	// default takes the first not yet entered. The engine may plug in a random
	// choice.
	Pick func(n int) int

	entered map[int]bool
	// organs the hero has collected from kills (by code), for the log.
	organs map[string]bool
	// Tristram
	bossesAlive map[int]bool
	tristram    bool
	arrived     int // Tristram bosses spawned so far
	finished    bool
	// PortalsOpened counts the portals the cube opened.
	PortalsOpened int
}

// New creates the event and registers it with the manager.
func New(m *d2boss.Manager) *Event {
	e := &Event{entered: map[int]bool{}, organs: map[string]bool{}, bossesAlive: map[int]bool{}}
	m.Add(e)

	return e
}

// Name implements d2boss.Encounter.
func (e *Event) Name() string { return "uber" }

// State implements d2boss.Encounter.
func (e *Event) State() string {
	switch {
	case e.finished:
		return "done"
	case e.tristram:
		return "tristram"
	case len(e.entered) > 0:
		return fmt.Sprintf("areas-%d", len(e.entered))
	}

	return "idle"
}

// OpenKeyPortal is the cube transmuting the three keys: a portal to an uber
// area appears.
func (e *Event) OpenKeyPortal(m *d2boss.Manager) []d2boss.Action {
	return m.Do(func() {
		n := len(Areas)

		i := 0

		switch {
		case e.Pick != nil:
			i = e.Pick(n) % n
		default:
			for i < n-1 && e.entered[Areas[i].Level] {
				i++
			}
		}

		a := Areas[i]
		e.PortalsOpened++
		m.Logf("uber trigger: Terror, Hate and Destruction keys transmuted -> portal to %s (level %d)", a.Name, a.Level)
		m.Emit("uber", d2boss.Action{Kind: d2boss.ActPortal, Name: "Pandemonium portal to " + a.Name, Class: d2boss.ObjHellgate,
			Level: a.Level, Note: "area chosen by the engine, UNVERIFIED"})
	})
}

// OpenFinalePortal is the cube transmuting the three organs: the red portal to
// Tristram opens.
func (e *Event) OpenFinalePortal(m *d2boss.Manager) []d2boss.Action {
	return m.Do(func() {
		e.PortalsOpened++
		m.Logf("uber trigger: Diablo's Horn, Baal's Eye and Mephisto's Brain transmuted -> red portal to Tristram")
		m.Emit("uber", d2boss.Action{Kind: d2boss.ActPortal, Name: "red portal to Tristram", Class: d2boss.ObjHellgate,
			Level: LevelTristram, Note: "the Pandemonium Finale portal"})
	})
}

// UseCube feeds a recipe the cube matched to the event; it reports whether the
// recipe belonged to it.
func (e *Event) UseCube(m *d2boss.Manager, r d2cube.Recipe) ([]d2boss.Action, bool) {
	if r.Kind != d2cube.ResultPortal {
		return nil, false
	}

	switch r.Portal {
	case d2cube.PortalPandemonium:
		return e.OpenKeyPortal(m), true
	case d2cube.PortalFinale:
		return e.OpenFinalePortal(m), true
	case d2cube.PortalNone:
	}

	return nil, false
}

func spawn(b Boss, level int, note string) d2boss.Action {
	return d2boss.Action{Kind: d2boss.ActSpawnMonster, Name: b.Name, Key: b.Key, Class: b.Class, Super: -1, Level: level, Boss: true, Note: note}
}

// OnEnter implements d2boss.Encounter.
func (e *Event) OnEnter(m *d2boss.Manager, level int) {
	if a, ok := AreaOf(level); ok && !e.entered[level] {
		e.entered[level] = true
		m.Logf("uber trigger: %s entered -> %s arrives", a.Name, a.Boss.Name)
		m.Emit("uber", spawn(a.Boss, level, "level 110, AI "+a.Boss.AI))

		e.bossesAlive[a.Boss.Class] = true
	}

	if level == LevelTristram && !e.tristram {
		e.tristram = true
		m.Logf("uber trigger: Tristram entered -> Uber Mephisto, Uber Diablo and Uber Baal arrive")
		m.Emit("uber", spawn(UberMephisto, level, "first of the three"))

		e.bossesAlive[ClassUberMephisto] = true
		e.arrived++

		m.After(diabloDelay, func() {
			e.bossesAlive[ClassUberDiablo] = true
			e.arrived++
			m.Emit("uber", spawn(UberDiablo, level, "delay UNVERIFIED"))
		})
		m.After(baalDelay, func() {
			e.bossesAlive[ClassUberBaal] = true
			e.arrived++
			m.Emit("uber", spawn(UberBaal, level, "delay UNVERIFIED"))
		})
	}
}

// OnOperate implements d2boss.Encounter.
func (e *Event) OnOperate(*d2boss.Manager, d2boss.Operate) {}

// OnKill implements d2boss.Encounter.
func (e *Event) OnKill(m *d2boss.Manager, k d2boss.Kill) {
	for _, a := range Areas {
		if k.Class == a.Boss.Class && k.Level == a.Level {
			e.organs[a.Drop] = true
			m.Logf("uber trigger: %s dead -> drops %s", a.Boss.Name, a.Drop)
			m.Emit("uber", d2boss.Action{Kind: d2boss.ActDropItem, Name: a.Boss.Name + " drops " + a.Drop, Key: a.Drop, Class: k.Class, Level: k.Level})
		}
	}

	if !e.tristram || k.Level != LevelTristram {
		return
	}

	switch k.Class {
	case ClassUberMephisto, ClassUberDiablo, ClassUberBaal:
		delete(e.bossesAlive, k.Class)
		m.Logf("uber trigger: %s dead in Tristram (%d of 3 still to come or alive)", k.Name, len(e.bossesAlive))

		if len(e.bossesAlive) == 0 && !e.finished && e.allArrived() {
			e.finished = true
			m.Logf("uber trigger: all three Tristram bosses dead -> the event is won")
			m.After(portalDelay, func() {
				m.Emit("uber", d2boss.Action{Kind: d2boss.ActDropItem, Name: "Standard of Heroes", Key: CodeStandardOfHeroes, Level: LevelTristram,
					Note: "UNVERIFIED item code and source"})
				m.Emit("uber", d2boss.Action{Kind: d2boss.ActPortal, Name: "red portal out of Tristram", Class: d2boss.ObjHellgate, Level: LevelHarrogath})
			})
		}
	}
}

// allArrived is true once Diablo and Baal were spawned (the kill of Mephisto
// before the others arrived does not end the event).
func (e *Event) allArrived() bool { return e.arrived >= 3 }

// OnTick implements d2boss.Encounter.
func (e *Event) OnTick(*d2boss.Manager, int) {}
