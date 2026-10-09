package d2gamescreen

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// zoneChangeText returns the "Entering The X" banner when the hero is now in a
// different level than the one last announced. Nothing is shown the first time
// (last == 0) or when the level has no Levels.txt row.
func zoneChangeText(last, cur int, details *d2records.LevelDetailRecord) (string, bool) {
	if last == 0 || last == cur || details == nil {
		return "", false
	}

	return fmt.Sprintf("Entering The %s", details.LevelDisplayName), true
}
