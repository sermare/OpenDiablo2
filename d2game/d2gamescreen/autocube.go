package d2gamescreen

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// Horadric Cube autotest, driven by environment variables so it needs no mouse:
//
//	OD2_AUTOCUBE=<names>      runs each scenario (gems, runes, reroll, socket,
//	                          craft, keys, invalid, runeword; "all" runs them all):
//	                          items go into the cube, the Transmute button is
//	                          pressed and the result is checked
//	OD2_AUTOCUBE_SEED=<n>     seeds the recipe rolls
//	OD2_AUTOCUBE_LADDER=1     makes the game a ladder game (ladder recipes)
//
// Every line carries the prefix "AUTOCUBE"; the last one is "AUTOCUBE done
// passed=N failed=M". OD2_AUTOEXIT quits at the end.
const autoCubeDelay = 5.0 // seconds after the hero appears

type autoCubeState struct {
	elapsed float64
	done    bool
}

// Level ids of the portals the cube opens (Levels.txt: Moo Moo Farm, the three
// Pandemonium areas and Uber Tristram).
const (
	cowLevel         = 39
	pandemoniumFirst = 133
	pandemoniumLast  = 135
	finaleLevel      = 136
)

// cubePortal opens the portal a cube recipe asked for. UNVERIFIED: which of the
// three Pandemonium areas the keys open is random here.
func (v *Game) cubePortal(kind string) error {
	level := 0

	switch kind {
	case "Cow Portal":
		level = cowLevel
	case "Pandemonium Portal":
		// nolint:gosec // not security relevant
		level = pandemoniumFirst + rand.Intn(pandemoniumLast-pandemoniumFirst+1)
	case "Pandemonium Finale Portal":
		level = finaleLevel
	default:
		return fmt.Errorf("unknown portal %q", kind)
	}

	v.Infof("CUBE portal %q opens to level %d", kind, level)

	return v.commandSpawnPortal([]string{strconv.Itoa(level)})
}

func (v *Game) advanceAutoCube(elapsed float64) {
	spec := os.Getenv("OD2_AUTOCUBE")
	if spec == "" || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	a := &v.autoCube
	a.elapsed += elapsed

	if a.done || a.elapsed < autoCubeDelay {
		return
	}

	a.done = true

	names := strings.Split(spec, ",")
	if spec == "all" {
		names = d2player.CubeScenarios
	}

	passed, failed := 0, 0

	for _, name := range names {
		if name = strings.TrimSpace(name); name == "" {
			continue
		}

		ok, detail := v.gameControls.AutoCube(name)
		if ok {
			passed++

			v.Infof("AUTOCUBE %s PASS: %s", name, detail)
		} else {
			failed++

			v.Infof("AUTOCUBE %s FAIL: %s", name, detail)
		}
	}

	v.Infof("AUTOCUBE done passed=%d failed=%d", passed, failed)
	v.autoTestExit()
}
