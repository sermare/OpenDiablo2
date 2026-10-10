package d2gamescreen

import (
	"errors"
	"strconv"
	"sync"
)

const (
	// The real game saves every 8192 frames; at its 25 frames per second that
	// is 327.68 seconds, about 5.5 minutes.
	autosaveIntervalSeconds = 8192.0 / 25.0

	// autosaveTestGold is the gold OD2_AUTOSAVE=1 sets before it exits.
	autosaveTestGold = "31337"
)

// activeGame is the running game screen, so the application can save the hero
// when it exits without going through OnUnload.
var activeGame *Game

// activeGameMu guards activeGame: SaveActiveGame runs from the application's
// quit and console paths while a game screen may be created or unloaded.
var activeGameMu sync.Mutex

func setActiveGame(g *Game) {
	activeGameMu.Lock()
	activeGame = g
	activeGameMu.Unlock()
}

// clearActiveGame forgets g if it is still the active game.
func clearActiveGame(g *Game) {
	activeGameMu.Lock()
	if activeGame == g {
		activeGame = nil
	}
	activeGameMu.Unlock()
}

// SaveActiveGame saves the hero of the running game, if there is one. The
// application calls it when it quits.
func SaveActiveGame() {
	activeGameMu.Lock()
	g := activeGame
	activeGameMu.Unlock()

	if g != nil {
		g.saveBeforeExit()
	}
}

// saveBeforeExit saves the hero (a hero imported from a .d2s is also written
// back to a .d2s by the server). Errors are logged, never fatal.
func (v *Game) saveBeforeExit() {
	if v.soundEngine != nil {
		v.Infof("AUDIOSTAT %s", v.soundEngine.AudioStats())
	}

	if v.localPlayer == nil || v.gameClient == nil {
		return
	}

	if err := v.OnPlayerSave(); err != nil {
		v.Errorf("saving the hero: %v", err)
	}
}

// advanceAutosave saves the hero periodically, like the real game does.
func (v *Game) advanceAutosave(elapsed float64) {
	if v.localPlayer == nil {
		return
	}

	v.autosaveElapsed += elapsed
	if v.autosaveElapsed < autosaveIntervalSeconds {
		return
	}

	v.autosaveElapsed = 0

	v.saveBeforeExit()
}

// commandSetGold implements the "setgold <amount>" console command.
// commandSetExp raises the hero's experience to at least the given amount (the
// level-up logic then promotes him). Scenarios use it to send a hero into a
// level that the original expects a stronger character in (the Den of Evil
// with its full natural population).
func (v *Game) commandSetExp(args []string) error {
	if len(args) != 1 || v.localPlayer == nil || v.localPlayer.Stats == nil {
		return errors.New("usage: setexp <amount> (in a game)")
	}

	n, err := strconv.Atoi(args[0])
	if err != nil || n < 0 {
		return errors.New("setexp needs a non-negative number")
	}

	if v.localPlayer.Stats.Experience < n {
		v.localPlayer.Stats.Experience = n
	}

	v.Infof("experience set to %d", v.localPlayer.Stats.Experience)

	return nil
}

func (v *Game) commandSetGold(args []string) error {
	if len(args) != 1 || v.localPlayer == nil || v.gameControls == nil {
		return errors.New("usage: setgold <amount> (in a game)")
	}

	n, err := strconv.Atoi(args[0])
	if err != nil || n < 0 {
		return errors.New("setgold needs a non-negative number")
	}

	v.gameControls.AddGold(n - v.localPlayer.Gold)
	v.Infof("gold set to %d", n)

	return nil
}
