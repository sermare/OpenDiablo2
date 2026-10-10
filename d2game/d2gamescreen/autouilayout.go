package d2gamescreen

import (
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// autoUILayout implements OD2_AUTOUILAYOUT=1: it opens every panel in turn and logs the rectangles the interface
// places ("UILAYOUT <panel> <name> <x> <y> <w> <h>", numbers only; see d2player.UILayout), compared by the
// verify.d scenario with scripts/verify.d/ui-layout.golden. OD2_AUTOSHOT_DELAY screenshots can be taken with
// OD2_AUTOPANEL=<panel> in separate runs.
func (v *Game) autoUILayout() {
	if os.Getenv("OD2_AUTOUILAYOUT") == "" {
		return
	}

	for _, name := range d2player.UILayoutPanels {
		switch name {
		case "hud", "minipanel":
		case "waypoint":
			v.openWaypointPanel(v.currentLevel())
		default:
			if err := v.gameControls.AutoPanel(name); err != nil {
				v.Infof("UILAYOUT panel=%s unavailable: %v", name, err)
				continue
			}
		}

		for _, r := range v.gameControls.UILayout(name) {
			v.Infof("UILAYOUT %s", r)
		}

		_ = v.gameControls.AutoPanel("close")
	}

	v.Infof("AUTOUILAYOUT done")
}
