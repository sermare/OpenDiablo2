package d2inventory

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero/herogen"
)

// TestHerogenBeltTable proves the hero generator's copy of the belt sizes (it does not import this package)
// is the table the game uses.
func TestHerogenBeltTable(t *testing.T) {
	if herogen.BeltBoxesByType() != BeltBoxesByType {
		t.Errorf("herogen belt table %v, game table %v", herogen.BeltBoxesByType(), BeltBoxesByType)
	}
}
