package d2gamescreen

import (
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// Container autotest, driven by environment variables so it needs no mouse:
//
//	OD2_AUTOPANEL=<names>       opens each of stash, cube, belt, inventory in turn and
//	                            logs its items with their grid positions
//	OD2_AUTOSTASH=1             walks to the stash object of the town, as a click on it does,
//	                            and logs whether the stash opened
//	OD2_AUTOBELT=<1..4,...>     drinks the belt potion of each column (hotkeys 1 to 4) at
//	                            40% life and mana and logs the restored amounts; the belt
//	                            and the stats are restored, nothing is saved
//	OD2_AUTOPANEL_HOLD=<sec>    keeps the last panel on screen that long
//
// Every line carries the prefix "AUTOPANEL". The positions are those of the
// hero's containers, so an import can be checked against the save's own parse
// (nokkasorc.json: location_id, alt_position_id 1 = inventory, 4 = cube, 5 =
// stash, position_x, position_y). OD2_AUTOEXIT quits at the end.
const (
	autoPanelDelay = 4.0 // seconds after the hero appears
)

type autoPanelState struct {
	elapsed float64
	done    bool
	holdEnd float64

	stashStarted, stashDone bool
	stashTimer              float64
}

func autoPanelEnabled() bool {
	return os.Getenv("OD2_AUTOPANEL") != "" || os.Getenv("OD2_AUTOBELT") != "" || os.Getenv("OD2_AUTOSTASH") != ""
}

// autoStash walks to the stash object like a click on it does and waits for
// the stash to open. It returns true once it is finished.
func (v *Game) autoStash(elapsed float64) bool {
	a := &v.autoPanel

	if os.Getenv("OD2_AUTOSTASH") == "" || a.stashDone {
		return true
	}

	if !a.stashStarted {
		a.stashStarted = true

		for _, e := range v.gameClient.MapEngine.Entities() {
			if ob, ok := e.(*d2mapentity.Object); ok && ob.Record().Index == stashObjectID {
				x, y := ob.GetPositionF()
				v.Infof("AUTOPANEL stash object %q found at (%.1f,%.1f); clicking it", ob.Label(), x, y)
				v.walkToObject(ob)

				a.stashTimer = 0

				return false
			}
		}

		v.Infof("AUTOPANEL stash object (id %d) is not in this area", stashObjectID)

		a.stashDone = true

		return true
	}

	a.stashTimer += elapsed

	switch {
	case v.gameControls.IsStashOpen():
		v.Infof("AUTOPANEL stash object opened the stash after %.1fs", a.stashTimer)
	case a.stashTimer > interactTimeout+2:
		v.Infof("AUTOPANEL stash did not open within %.0fs", a.stashTimer)
	default:
		return false
	}

	a.stashDone = true

	return true
}

func (v *Game) advanceAutoPanel(elapsed float64) {
	if !autoPanelEnabled() || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	a := &v.autoPanel
	a.elapsed += elapsed

	if a.done {
		if a.holdEnd > 0 && a.elapsed >= a.holdEnd {
			v.autoTestExit()
		}

		return
	}

	if a.elapsed < autoPanelDelay {
		return
	}

	if !v.autoStash(elapsed) {
		return
	}

	a.done = true

	for _, name := range strings.Split(os.Getenv("OD2_AUTOPANEL"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			v.autoPanelLog(name)
		}
	}

	for _, c := range strings.Split(os.Getenv("OD2_AUTOBELT"), ",") {
		col, err := strconv.Atoi(strings.TrimSpace(c))
		if err != nil || col < 1 || col > 4 {
			continue
		}

		for _, line := range v.gameControls.AutoBeltUse(col - 1) {
			v.Infof("AUTOPANEL belt use %s", line)
		}
	}

	checked, bad := v.gameControls.SpecRoundTripMismatches()
	v.Infof("AUTOPANEL spec round trip: checked=%d mismatched=%d", checked, bad)

	if hold, err := strconv.ParseFloat(os.Getenv("OD2_AUTOPANEL_HOLD"), 64); err == nil && hold > 0 {
		a.holdEnd = a.elapsed + hold
		return
	}

	v.autoTestExit()
}

// autoPanelLog opens a panel and logs its items.
func (v *Game) autoPanelLog(name string) {
	if err := v.gameControls.AutoPanel(name); err != nil {
		v.Infof("AUTOPANEL panel=%s: %v", name, err)
		return
	}

	lines, size, err := v.gameControls.ContainerPositions(name)
	if err != nil {
		v.Infof("AUTOPANEL panel=%s: %v", name, err)
		return
	}

	v.Infof("AUTOPANEL panel=%s size=%s items=%d", name, size, len(lines))

	for i, l := range lines {
		v.Infof("AUTOPANEL panel=%s item #%d %s", name, i, l)
	}
}
