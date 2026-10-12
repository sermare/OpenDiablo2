// Package d2uber is the Pandemonium event of Lord of Destruction (the "Uber
// Tristram" quest line added in 1.11): three keys open a portal to one of three
// uber areas, whose bosses drop the three organs, and the organs open the red
// portal to Tristram, where Uber Mephisto, Uber Diablo and Uber Baal wait.
//
// It is an encounter set for the d2boss Manager: the engine reports the level
// entered and the monsters killed, and receives the same Actions as for the
// story bosses (spawn a monster, open a portal, drop an item).
//
// Wave 22 re-audit (D2MOO, which tracks the exe: ACT1Q4_CreateUberRunPortal 0x594270 and
// ACT1Q4_CreateUberFinalePortal 0x594280, SUNIT_SpawnPresetUnitsInRoom 0x5559A0; notes w22-quest-world.md):
//   - VERIFIED shape: both cube portals open only in Harrogath on Hell (difficulty 2), as a permanent town portal;
//     the keys' portal goes to a random uber area that was not opened yet in this game (start 0, 1 or 2 of the
//     three, then the next unopened one, wrapping), a fourth one fails; the finale portal opens once per game;
//   - VERIFIED shape: Uber Mephisto, Uber Baal and Uber Diablo all spawn together at the Cain gibbet position of
//     Pandemonium Finale (no arrival delays); Lilith and Uber Izual stand at the sparkly chest of their area.
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
//     areas a key set opens (modelled as an injected choice), the order and delays in
//     which the three Tristram bosses arrive, the Standard of Heroes (item
//     code "std", not in the tables) and the red portal object.
package d2uber

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
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

// Item codes the event drops besides the organs and keys of d2cube.
const (
	CodeStandardOfHeroes = "std" // UNVERIFIED: not in the tables of the owner's install
)

// portalDelay is the delay in game frames (25 per second) between the last Tristram kill and the way out, UNVERIFIED.
const portalDelay = 50

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
		{LevelMatronsDen, "Matron's Den", Lilith, CodeHorn},
		{LevelForgottenSands, "Forgotten Sands", UberDuriel, CodeBaalEye},
		{LevelFurnaceOfPain, "Furnace of Pain", UberIzual, CodeMephBrain},
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

// KeyTreasureClasses names the Hell treasure classes of treasureclassex.txt that carry a Pandemonium key (VERIFIED in
// the table: only the Hell classes of the Countess, the Summoner and Nihlathak list pk1, pk2 and pk3).
// The keys therefore come out of the ordinary super unique treasure class roll (d2drop), at the table's chance, and
// not from Andariel, Duriel and Mephisto (their Hell classes carry the essences). The engine adds no separate key drop.
//
//nolint:gochecknoglobals // static lookup data
var KeyTreasureClasses = map[string]string{
	"Countess Item (H)": CodeKeyTerror,
	"Summoner (H)":      CodeKeyHate,
	"Nihlathak (H)":     CodeKeyDestr,
}

// Event is the Pandemonium event; it implements d2boss.Encounter.
type Event struct {
	// Pick chooses the uber area a key set opens (n = number of areas); the
	// default takes the first not yet entered. The engine may plug in a random
	// choice.
	Pick func(n int) int

	// Where, when set, says where the hero stands when the cube opens a portal: the level id and the difficulty
	// (0 normal, 1 nightmare, 2 hell). The portals open only in Harrogath on Hell (VERIFIED shape, D2MOO); nil
	// leaves them unrestricted (unit tests of the event alone).
	Where func() (level, difficulty int)

	entered map[int]bool
	// opened is bUberPortalRuns of the exe: the area portals already made in this game, finaleOpened the finale's.
	opened       [3]bool
	finaleOpened bool
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

// runOrder is the order the exe tries the three areas in for its random start 0, 1 or 2 (ACT1Q4_CreateUberRunPortal).
//
//nolint:gochecknoglobals // static lookup data
var runOrder = [3][3]int{{0, 1, 2}, {1, 2, 0}, {2, 0, 1}}

// allowed says whether the cube may open a portal where the hero stands (Harrogath, Hell).
func (e *Event) allowed() (bool, string) {
	if e.Where == nil {
		return true, ""
	}

	level, diff := e.Where()
	if level != LevelHarrogath || diff != 2 {
		return false, fmt.Sprintf("the hero stands in level %d on difficulty %d: the portal opens only in Harrogath on Hell", level, diff)
	}

	return true, ""
}

func (e *Event) refuse(m *d2boss.Manager, why string) []d2boss.Action {
	return m.Do(func() { m.Logf("uber trigger: the cube portal fails: %s", why) })
}

// OpenKeyPortal is the cube transmuting the three keys: a portal to an uber
// area appears (ACT1Q4_CreateUberRunPortal).
func (e *Event) OpenKeyPortal(m *d2boss.Manager) []d2boss.Action {
	if ok, why := e.allowed(); !ok {
		return e.refuse(m, why)
	}

	if e.opened[0] && e.opened[1] && e.opened[2] {
		return e.refuse(m, "all three area portals were opened in this game")
	}

	return m.Do(func() {
		start := 0
		if e.Pick != nil {
			start = e.Pick(len(Areas)) % len(Areas)
		}

		i := runOrder[start][2]

		for _, c := range runOrder[start] {
			if !e.opened[c] {
				i = c

				break
			}
		}

		e.opened[i] = true
		a := Areas[i]
		e.PortalsOpened++
		m.Logf("uber trigger: Terror, Hate and Destruction keys transmuted -> portal to %s (level %d)", a.Name, a.Level)
		m.Emit("uber", d2boss.Action{Kind: d2boss.ActPortal, Name: "Pandemonium portal to " + a.Name, Class: d2boss.ObjHellgate,
			Level: a.Level, Note: "random unopened area (VERIFIED shape, D2MOO)"})
	})
}

// OpenFinalePortal is the cube transmuting the three organs: the portal to
// Tristram opens, once per game (ACT1Q4_CreateUberFinalePortal).
func (e *Event) OpenFinalePortal(m *d2boss.Manager) []d2boss.Action {
	if ok, why := e.allowed(); !ok {
		return e.refuse(m, why)
	}

	if e.finaleOpened {
		return e.refuse(m, "the Tristram portal was opened in this game")
	}

	return m.Do(func() {
		e.finaleOpened = true
		e.PortalsOpened++
		m.Logf("uber trigger: Diablo's Horn, Baal's Eye and Mephisto's Brain transmuted -> red portal to Tristram")
		m.Emit("uber", d2boss.Action{Kind: d2boss.ActPortal, Name: "red portal to Tristram", Class: d2boss.ObjHellgate,
			Level: LevelTristram, Note: "the Pandemonium Finale portal"})
	})
}

// UseCube feeds a recipe the cube matched to the event; it reports whether the
// recipe belonged to it.
func (e *Event) UseCube(m *d2boss.Manager, portal string) ([]d2boss.Action, bool) {
	switch portal {
	case PortalPandemonium:
		return e.OpenKeyPortal(m), true
	case PortalFinale:
		return e.OpenFinalePortal(m), true
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
		m.Logf("uber trigger: Tristram entered -> Uber Mephisto, Uber Diablo and Uber Baal arrive together")

		for _, b := range []Boss{UberMephisto, UberBaal, UberDiablo} {
			m.Emit("uber", spawn(b, level, "all three at the Cain gibbet position (VERIFIED shape, D2MOO)"))

			e.bossesAlive[b.Class] = true
			e.arrived++
		}
	}
}

// OnOperate implements d2boss.Encounter.
func (e *Event) OnOperate(*d2boss.Manager, d2boss.Operate) {}

// OnKill implements d2boss.Encounter.
func (e *Event) OnKill(m *d2boss.Manager, k d2boss.Kill) {
	for _, a := range Areas {
		if k.Class == a.Boss.Class && k.Level == a.Level {
			e.organs[a.Drop] = true
			delete(e.bossesAlive, k.Class)
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
