package d2gamescreen

import (
	"errors"
	"fmt"
)

// Console commands of the item comparison scenarios (scripts/verify.d/9k-item-compare.sh).

// compareCommands are bound with the other console commands (OnLoad) and unbound with them (OnUnload).
func (v *Game) compareCommands() []struct {
	name string
	desc string
	args []string
	fn   func([]string) error
} {
	return []struct {
		name string
		desc string
		args []string
		fn   func([]string) error
	}{
		{"hoveritem", "moves the pointer onto the first item with this base code of inv|stash|cube|vendor (scripts)",
			[]string{"where", "code"}, v.commandHoverItem},
		{"comparelog", "logs the comparison lines of the first item with this base code of inv|stash|cube|vendor (scripts)",
			[]string{"where", "code"}, v.commandCompareLog},
		{"hoverground", "moves the pointer onto the ground item whose label contains the text (scripts)",
			[]string{"label"}, v.commandHoverGround},
		{"openvendor", "opens the trade window of a vendor next to the hero (scripts)", []string{"vendor"}, v.commandOpenVendor},
		{"itemtext", "logs the whole tooltip text of every item in the inventory (scripts)", []string{"tag"}, v.commandItemText},
		{"stashgive", "creates an identified item and puts it into the stash (scripts)", []string{"code"}, v.commandStashGive},
	}
}

func (v *Game) commandHoverItem(args []string) error {
	if v.gameControls == nil {
		return errors.New("not in a game")
	}

	return v.gameControls.HoverItem(args[0], args[1])
}

func (v *Game) commandCompareLog(args []string) error {
	if v.gameControls == nil {
		return errors.New("not in a game")
	}

	_, err := v.gameControls.CompareLog(args[0], args[1])

	return err
}

func (v *Game) commandStashGive(args []string) error {
	if v.gameControls == nil {
		return errors.New("not in a game")
	}

	return v.gameControls.GiveItemToStash(args[0])
}

func (v *Game) commandHoverGround(args []string) error {
	if v.gameControls == nil {
		return errors.New("not in a game")
	}

	return v.gameControls.HoverGround(args[0])
}

func (v *Game) commandOpenVendor(args []string) error {
	if v.gameControls == nil || v.gameClient == nil {
		return errors.New("not in a game")
	}

	for _, e := range v.gameClient.MapEngine.Entities() {
		if e.Label() == args[0] && v.openTrade(e, 1) {
			return nil
		}
	}

	return fmt.Errorf("no vendor %s here", args[0])
}

// commandItemText is "itemtext <tag>": one ITEMTEXT log line per inventory item with its whole tooltip, so that
// the tooltips before and after a save and reload can be compared (scripts/verify.d/w51-rare-*.sh).
func (v *Game) commandItemText(args []string) error {
	if v.gameControls == nil {
		return errors.New("not in a game")
	}

	for i, text := range v.gameControls.InventoryTooltips() {
		v.Infof("ITEMTEXT tag=%s #%d %s", args[0], i, text)
	}

	return nil
}
