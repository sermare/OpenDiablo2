package d2gamescreen

import (
	"errors"
	"fmt"
)

// Test console aids for the corpse and the pickup rules (w50).

// commandFillInv is "fillinv <code>": fills the inventory grid with that item until it is full.
func (v *Game) commandFillInv(args []string) error {
	if len(args) != 1 || v.gameControls == nil {
		return errors.New("usage: fillinv <code>")
	}

	n := 0

	for n < 200 {
		if _, err := v.gameControls.GiveItem(args[0]); err != nil {
			break
		}

		n++
	}

	v.Infof("FILLINV code=%s added=%d", args[0], n)

	return nil
}

// commandCorpseClick is "corpseclick": the same as a click on the hero's corpse.
func (v *Game) commandCorpseClick(_ []string) error {
	if v.death.corpse == nil {
		return fmt.Errorf("no corpse in this level")
	}

	v.OnPlayerInteract(v.death.corpse)

	return nil
}
