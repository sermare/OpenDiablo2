package d2gamescreen

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2difficulty"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// OD2_AUTOFLOW drives the front-end flow without clicking, for the verify
// scenarios. It is a comma separated list of steps; each screen performs the
// step at the head of the list when it is one of its own, then the list moves
// on. (OD2_AUTOMENU already belongs to the NPC menu test, hence the name.)
//
//	single                       main menu: Single Player
//	host                         main menu: Multiplayer, TCP/IP, Host Game
//	join:Address                 main menu: Multiplayer, TCP/IP, Join Game, type the address, OK
//	new                          character select: Create New Character
//	create:Class:Name[:hardcore][:classic]
//	                             hero creation: pick the class, type the name, set the
//	                             checkboxes, press OK (refused names are logged, step done)
//	select:Name                  character select: pick a character
//	delete:Name                  character select: delete a character (confirmed)
//	play                         character select: OK
//	difficulty:N                 difficulty select: 0 Normal, 1 Nightmare, 2 Hell
//	gold:N                       in game: set the gold
//	wait:Seconds                 in game: wait
//	saveexit                     in game: Save and Exit (back to character select)
//	exit                         anywhere: quit the program
//
// Every step is logged as "AUTOFLOW step <n> <step>".
const autoFlowSettleSeconds = 1.5

type autoFlowState struct {
	sync.Mutex
	steps   []string
	index   int
	settle  float64 // seconds since the last step was taken
	waiting float64 // seconds of an active wait: step
	loaded  bool
}

//nolint:gochecknoglobals // the one flow of the process, set from the environment
var autoFlow autoFlowState

func (f *autoFlowState) load() {
	if f.loaded {
		return
	}

	f.loaded = true

	for _, s := range strings.Split(os.Getenv("OD2_AUTOFLOW"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			f.steps = append(f.steps, s)
		}
	}
}

// autoFlowActive reports whether OD2_AUTOFLOW still has steps to do.
func autoFlowActive() bool {
	autoFlow.Lock()
	defer autoFlow.Unlock()

	autoFlow.load()

	return autoFlow.index < len(autoFlow.steps)
}

// autoFlowNext returns the step at the head of the flow when its verb is one
// of verbs and the screen has had time to settle; elapsed is the time since the
// screen's last frame. The step is not consumed: call autoFlowDone for that.
func autoFlowNext(elapsed float64, verbs ...string) (args []string, ok bool) {
	autoFlow.Lock()
	defer autoFlow.Unlock()

	autoFlow.load()
	autoFlow.settle += elapsed

	if autoFlow.index >= len(autoFlow.steps) || autoFlow.settle < autoFlowSettleSeconds {
		return nil, false
	}

	parts := strings.Split(autoFlow.steps[autoFlow.index], ":")

	for _, v := range verbs {
		if parts[0] == v {
			return parts, true
		}
	}

	return nil, false
}

// autoFlowDone consumes the head step and logs it.
func autoFlowDone() {
	autoFlow.Lock()
	defer autoFlow.Unlock()

	log.Printf("AUTOFLOW step %d %s", autoFlow.index+1, autoFlow.steps[autoFlow.index])

	autoFlow.index++
	autoFlow.settle = 0
	autoFlow.waiting = 0
}

// autoFlowWait implements wait:S: true once the seconds have passed.
func autoFlowWait(elapsed float64, seconds string) bool {
	secs, err := strconv.ParseFloat(seconds, 64)
	if err != nil {
		return true
	}

	autoFlow.Lock()
	defer autoFlow.Unlock()

	autoFlow.waiting += elapsed

	return autoFlow.waiting >= secs
}

// autoFlowExit performs an exit step.
func autoFlowExit(elapsed float64) {
	if _, ok := autoFlowNext(elapsed, "exit"); ok {
		autoFlowDone()
		os.Exit(0)
	}
}

// advanceFlow performs the steps that belong to the character select.
func (v *CharacterSelect) advanceFlow(elapsed float64) {
	args, ok := autoFlowNext(elapsed, "new", "select", "delete", "play", "exit")
	if !ok || !v.loaded {
		return
	}

	switch args[0] {
	case "exit":
		autoFlowExit(0)
	case "new":
		autoFlowDone()
		v.onNewCharButtonClicked()
	case "select", "delete":
		idx := -1

		for i, st := range v.gameStates {
			if len(args) > 1 && strings.EqualFold(st.HeroName, args[1]) {
				idx = i
			}
		}

		if idx < 0 {
			v.Warningf("AUTOFLOW %s: no character %q in the list", args[0], strings.Join(args[1:], ":"))
			autoFlowDone()

			return
		}

		v.selectedCharacter = idx
		v.moveSelectionBox()

		if args[0] == "delete" {
			v.onDeleteCharacterConfirmClicked()
		}

		autoFlowDone()
	case "play":
		autoFlowDone()
		v.onOkButtonClicked()
	}
}

// advanceFlow performs the steps that belong to the main menu.
func (v *MainMenu) advanceFlow(elapsed float64) {
	args, ok := autoFlowNext(elapsed, "single", "host", "join", "exit")
	if !ok {
		return
	}

	if v.screenMode == ScreenModeTrademark {
		v.SetScreenMode(ScreenModeMainMenu) // the click that leaves the trademark screen

		return
	}

	switch args[0] {
	case "exit":
		autoFlowExit(0)
	case "host", "join":
		v.advanceMultiplayerFlow(args)
	default:
		autoFlowDone()
		v.onSinglePlayerClicked()
	}
}

// advanceMultiplayerFlow clicks through Multiplayer, TCP/IP, then Host Game or
// (typing the address of the host) Join Game, one screen per frame.
func (v *MainMenu) advanceMultiplayerFlow(args []string) {
	switch v.screenMode {
	case ScreenModeMainMenu:
		v.onMultiplayerClicked()
	case ScreenModeMultiplayer:
		v.onNetworkTCPIPClicked()
	case ScreenModeTCPIP:
		if args[0] == "host" {
			autoFlowDone()
			v.onTCPIPHostGameClicked()

			return
		}

		v.onTCPIPJoinGameClicked()
	case ScreenModeServerIP:
		autoFlowDone()
		v.tcpJoinGameEntry.SetText(strings.Join(args[1:], ":"))
		v.onBtnTCPIPOkClicked()
	}
}

// advanceFlow performs the create step of the hero creation screen.
func (v *SelectHeroClass) advanceFlow(elapsed float64) {
	args, ok := autoFlowNext(elapsed, "create", "exit")
	if !ok {
		return
	}

	if args[0] == "exit" {
		autoFlowExit(0)
		return
	}

	autoFlowDone()

	if len(args) < 3 {
		v.Warningf("AUTOFLOW create needs create:Class:Name")
		return
	}

	hero, found := heroByName(args[1])
	if !found {
		v.Warningf("AUTOFLOW create: unknown class %q", args[1])
		return
	}

	v.selectedHero = hero
	v.heroNameTextbox.SetText(args[2])
	v.expansionCheckbox.SetCheckState(true)
	v.hardcoreCheckbox.SetCheckState(false)

	for _, flag := range args[3:] {
		switch flag {
		case "hardcore":
			v.hardcoreCheckbox.SetCheckState(true)
		case "classic":
			v.expansionCheckbox.SetCheckState(false)
		}
	}

	v.onOkButtonClicked()
}

// advanceFlow performs the difficulty step.
func (v *DifficultySelect) advanceFlow(elapsed float64) {
	args, ok := autoFlowNext(elapsed, "difficulty", "exit")
	if !ok {
		return
	}

	if args[0] == "exit" {
		autoFlowExit(0)
		return
	}

	autoFlowDone()

	if n, err := strconv.Atoi(strings.Join(args[1:], "")); err == nil {
		v.Choose(difficultyLevel(n))
	}
}

// advanceFlow performs the in-game steps.
func (v *Game) advanceFlow(elapsed float64) {
	if v.localPlayer == nil || v.gameControls == nil || !autoFlowActive() {
		return
	}

	args, ok := autoFlowNext(elapsed, "gold", "wait", "saveexit", "exit", "difficulty")
	if !ok {
		return
	}

	switch args[0] {
	case "difficulty":
		autoFlowDone() // the hero had no difficulty to choose: the game is already running
	case "gold":
		autoFlowDone()

		if err := v.commandSetGold(args[1:]); err != nil {
			v.Warningf("AUTOFLOW gold: %v", err)
		}
	case "wait":
		if len(args) < 2 || autoFlowWait(elapsed, args[1]) {
			autoFlowDone()
		}
	case "saveexit":
		autoFlowDone()
		v.Infof("AUTOFLOW saveexit gold=%d", v.localPlayer.Gold)
		v.escapeMenu.SaveAndExit()
	case "exit":
		autoFlowDone()
		v.saveBeforeExit()
		os.Exit(0)
	}
}

// heroByName finds a hero by its class name ("Sorceress").
func heroByName(name string) (d2enum.Hero, bool) {
	for c := d2s.Amazon; c <= d2s.Assassin; c++ {
		if strings.EqualFold(c.String(), name) {
			return d2hero.HeroOfD2SClass(c)
		}
	}

	return d2enum.HeroNone, false
}

func difficultyLevel(n int) d2difficulty.Level { return d2difficulty.Level(n) }
